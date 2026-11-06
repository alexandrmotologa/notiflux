# Notiflux Architecture & System Design

Notiflux is a self-hosted notification engine packaged as a single static binary with an embedded management dashboard and drop-in client widget.

## Design Goals

1. **Low Resource Footprint**: Idle memory consumption remains below 25MB RAM, and steady-state load under 10,000 active SSE subscribers remains below 45MB RAM.
2. **Zero External Infrastructure Dependencies**: All state, workflow rules, templates, and delivery logs persist inside an embedded SQLite database using Write-Ahead Logging (WAL) mode. No Redis, RabbitMQ, Kafka, or MongoDB instances are required.
3. **Multi-Channel Dispatch**: Single trigger API call dispatches across In-App, Email (SMTP and Resend API), Web Push (RFC 8292 VAPID), Telegram Bot API, Slack Webhooks, and outbound HTTPS Webhooks.
4. **Reliable Client Delivery**: Drop-in client widget (< 3KB) receives immediate notification updates using Server-Sent Events (SSE) with automatic backoff reconnection and 15-second heartbeat keep-alives.

---

## System Components

```
                   Trigger Event (POST /v1/trigger)
                                  │
                                  ▼
                     ┌──────────────────────────┐
                     │   Authentication Layer   │
                     └────────────┬─────────────┘
                                  │
                                  ▼
                     ┌──────────────────────────┐
                     │   Idempotency Engine     │
                     └────────────┬─────────────┘
                                  │
                                  ▼
                     ┌──────────────────────────┐
                     │  Workflow & Preferences  │
                     └────────────┬─────────────┘
                                  │
                                  ▼
                     ┌──────────────────────────┐
                     │ Template Compiler Engine │
                     └────────────┬─────────────┘
                                  │
          ┌───────────────────────┴───────────────────────┐
          │                                               │
          ▼                                               ▼
┌──────────────────────────┐                  ┌──────────────────────────┐
│   Outbound Dispatchers   │                  │   In-App Feed & Storage  │
│ - SMTP / Resend Email    │                  │ - SQLite WAL Mode        │
│ - Web Push (RFC 8292)    │                  └───────────┬──────────────┘
│ - Telegram Bot API       │                              │
│ - Slack Webhooks         │                              ▼
│ - Signed HTTP Webhooks   │                  ┌──────────────────────────┐
└──────────────────────────┘                  │ Real-Time SSE Hub        │
                                              └───────────┬──────────────┘
                                                          │
                                                          ▼
                                              ┌──────────────────────────┐
                                              │ Client Widget (< 3KB)    │
                                              └──────────────────────────┘
```

---

## Data Layer: SQLite in WAL Mode

Notiflux uses pure Go SQLite (`modernc.org/sqlite`) without CGO dependencies. This design choice enables static compilation across Linux, macOS, and Windows targets.

Database configuration settings:
- `PRAGMA journal_mode = WAL;`: Allows concurrent readers while a write transaction is executing.
- `PRAGMA busy_timeout = 5000;`: Blocks writers up to 5000ms during lock contention before throwing an error.
- `PRAGMA synchronous = NORMAL;`: Reduces disk sync frequency while preserving database integrity in WAL mode.
- `PRAGMA foreign_keys = ON;`: Enforces relational constraints and cascading deletions across workflows and subscribers.

---

## Real-Time In-App Delivery: Server-Sent Events (SSE)

Unlike WebSockets, which require a protocol upgrade and custom framing, Server-Sent Events operate over standard HTTP/1.1 or HTTP/2 connections.

Key properties:
- **Connection Transport**: Streams `text/event-stream` chunks.
- **Heartbeat Mechanism**: Sends an empty comment (`: ping\n\n`) every 15 seconds to keep intermediary reverse proxies (such as Nginx, Cloudflare, or Traefik) from dropping idle connections.
- **Client Auto-Reconnect**: Browser `EventSource` automatically reconnects on disconnect with exponential backoff.
- **Multiplexing**: The internal SSE broker maintains subscriber-keyed subscriber maps protected by read-write mutexes (`sync.RWMutex`), providing instantaneous fan-out without thread leaks.

---

## Web Push Implementation: RFC 8292 & RFC 8291

Notiflux implements native Web Push without third-party push gateways:
- **VAPID (RFC 8292)**: Generates and signs JSON Web Tokens (JWT) using ECDSA on curve P-256 (prime256v1).
- **Encryption (RFC 8291)**: Encrypts payloads using `aes128gcm` with ECDH shared secret derivation between the daemon private key and the browser public key (`p256dh`).

---

## Embedded Management Studio

The management studio is built with React 19, TypeScript, and Tailwind CSS. The compilation step creates static assets in `web/dist`, which are then embedded directly into the Go executable via `//go:embed`.

The single binary serves:
- REST API at `/v1/*`
- Real-time SSE stream at `/v1/feed/stream`
- Client widget library at `/widget/notiflux-bell.js` and `/widget/notiflux-bell.css`
- Administration cockpit at `/studio` (and `/`)
