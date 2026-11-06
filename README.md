# Notiflux

Self-hosted multi-channel notification engine and real-time in-app notification center running under 45MB RAM.

Notiflux provides a single-binary alternative to Novu, Knock, and Courier. It runs with zero external infrastructure dependencies, using embedded SQLite in Write-Ahead Logging (WAL) mode for message storage, user preferences, and delivery logs.

## Highlights

- Single static Go binary with zero CGO dependencies.
- Sub-45MB RAM consumption under 10,000 active Server-Sent Events subscribers.
- Multi-channel delivery across In-App, Email (SMTP and Resend API), Web Push (RFC 8292 VAPID), Telegram Bot API, Slack Webhooks, and outbound HTTPS Webhooks.
- Real-time client widget (< 3KB) available as both a Vanilla Web Component (`<notiflux-bell>`) and a React 19 component (`<NotificationBell />`).
- Embedded management studio written in React 19 and Tailwind CSS, served directly from the binary.
- Built-in idempotency key engine to prevent duplicate notification deliveries.
- User preference management with per-channel opt-out and quiet-hours handling.

---

## Architecture Overview

```
                      Backend Application (Node, Python, Go, Ruby)
                                           │
                                           │ POST /v1/trigger
                                           ▼
┌──────────────────────────────────────────────────────────────────────────────────┐
│                               Notiflux Daemon                                    │
│                                                                                  │
│  ┌──────────────────────┐         ┌───────────────────────────────────────────┐  │
│  │ Authentication Layer │ ──────► │ Idempotency Engine & Rate Limiter         │  │
│  └──────────────────────┘         └─────────────────────┬─────────────────────┘  │
│                                                         │                        │
│                                                         ▼                        │
│  ┌────────────────────────────────────────────────────────────────────────────┐  │
│  │ User Preference Engine (Channel Opt-Ins, Mute Flags, Quiet Hours)          │  │
│  └──────────────────────────────────────────────┬─────────────────────────────┘  │
│                                                 │ Active channels                │
│                                                 ▼                                │
│  ┌────────────────────────────────────────────────────────────────────────────┐  │
│  │ Mustache Template Compiler & Markdown Engine                               │  │
│  └──────────────────────────────────────────────┬─────────────────────────────┘  │
│                                                 │                                │
│                   ┌─────────────────────────────┴────────────────────────────┐   │
│                   ▼                                                          ▼   │
│  ┌─────────────────────────────────┐        ┌─────────────────────────────────┐  │
│  │ Outbound Dispatcher Pool        │        │ In-App Feed & Storage           │  │
│  │ - SMTP & Resend API             │        │ - Pure Go SQLite (WAL mode)     │  │
│  │ - Web Push (RFC 8292 VAPID)     │        └────────────────┬────────────────┘  │
│  │ - Telegram Bot API              │                         │                   │
│  │ - Slack Webhooks                │                         ▼                   │
│  │ - Signed HTTPS Webhooks         │        ┌─────────────────────────────────┐  │
│  └─────────────────────────────────┘        │ Server-Sent Events (SSE) Hub    │  │
│                                             └────────────────┬────────────────┘  │
│                                                              │                   │
│  ┌───────────────────────────────────────────────────────────▼────────────────┐  │
│  │ Client Widget: <notiflux-bell> (Served via /widget/notiflux-bell.js)       │  │
│  └────────────────────────────────────────────────────────────────────────────┘  │
│                                                                                  │
│  ┌────────────────────────────────────────────────────────────────────────────┐  │
│  │ Embedded Management Studio (/studio)                                       │  │
│  └────────────────────────────────────────────────────────────────────────────┘  │
└──────────────────────────────────────────────────────────────────────────────────┘
```

---

## Quickstart

### 1. Run via Go

```bash
git clone https://github.com/alexandrmotologa/notiflux.git
cd notiflux
go run ./cmd/notiflux serve --port 8085 --db ./notiflux.db
```

### 2. Run via Docker

```bash
docker run -d \
  -p 8085:8085 \
  -v notiflux_data:/data \
  -e NOTIFLUX_DB_PATH=/data/notiflux.db \
  -e NOTIFLUX_API_KEY=secret-token \
  alexandrmotologa/notiflux:latest
```

The administration studio is accessible at `http://localhost:8085/studio`.

---

## Ingesting Notifications

Trigger a workflow across multiple channels with a single HTTP request:

```bash
curl -X POST http://localhost:8085/v1/trigger \
  -H "Authorization: Bearer secret-token" \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: req_98124801" \
  -d '{
    "workflow": "order.shipped",
    "subscriber": {
      "id": "usr_1001",
      "email": "customer@example.com",
      "telegram_chat_id": "987654321"
    },
    "payload": {
      "order_id": "#8942",
      "carrier": "DHL Express",
      "action_url": "https://store.example.com/orders/8942"
    }
  }'
```

---

## Drop-In Notification Bell Widget

Notiflux serves a pre-bundled client widget directly from `/widget/notiflux-bell.js`.

### Vanilla HTML / Web Component (< 3KB)

```html
<!-- Load the widget script -->
<script type="module" src="http://localhost:8085/widget/notiflux-bell.js"></script>

<!-- Place the element anywhere in your layout -->
<notiflux-bell
  api-url="http://localhost:8085"
  subscriber-id="usr_1001"
  theme="dark">
</notiflux-bell>
```

### React 19

```tsx
import { NotificationBell } from "@notiflux/react";

export function Header({ currentUserId }: { currentUserId: string }) {
  return (
    <header className="flex justify-between items-center p-4">
      <h1>My Application</h1>
      <NotificationBell
        apiUrl="http://localhost:8085"
        subscriberId={currentUserId}
        theme="dark"
        onNotificationClick={(notification) => {
          if (notification.action_url) {
            window.location.href = notification.action_url;
          }
        }}
      />
    </header>
  );
}
```

---

## Configuration Reference

Notiflux accepts configuration via CLI flags or environment variables:

| Environment Variable | Flag | Default | Description |
| :--- | :--- | :--- | :--- |
| `NOTIFLUX_PORT` | `--port` | `8085` | HTTP server port |
| `NOTIFLUX_DB_PATH` | `--db` | `notiflux.db` | Path to SQLite database file |
| `NOTIFLUX_API_KEY` | `--api-key` | `""` | Master API key for administrative operations |
| `NOTIFLUX_SMTP_HOST` | `--smtp-host` | `""` | Outbound SMTP server hostname |
| `NOTIFLUX_SMTP_PORT` | `--smtp-port` | `587` | Outbound SMTP server port |
| `NOTIFLUX_SMTP_USER` | `--smtp-user` | `""` | SMTP authentication username |
| `NOTIFLUX_SMTP_PASS` | `--smtp-pass` | `""` | SMTP authentication password |
| `NOTIFLUX_SMTP_FROM` | `--smtp-from` | `""` | Default sender email address |
| `NOTIFLUX_RESEND_API_KEY` | `--resend-api-key` | `""` | Optional Resend REST API key |
| `NOTIFLUX_VAPID_PUBLIC_KEY` | `--vapid-public-key` | `""` | RFC 8292 VAPID public key |
| `NOTIFLUX_VAPID_PRIVATE_KEY` | `--vapid-private-key`| `""` | RFC 8292 VAPID private key |
| `NOTIFLUX_VAPID_SUBJECT` | `--vapid-subject` | `""` | VAPID contact mailto URI |
| `NOTIFLUX_TELEGRAM_BOT_TOKEN` | `--telegram-token` | `""` | Telegram Bot API token |

---

## Utility Commands

### Generate VAPID Keys
To generate an ECDSA P-256 key pair for Web Push notifications:

```bash
notiflux vapid
```

Output:
```
VAPID Public Key:  BNcRdreALRFXTkOOUHK1T9S...
VAPID Private Key: MC4CAQAwBQYDK2VwBCIEI...
```

---

## Documentation

- [System Architecture](file:///docs/ARCHITECTURE.md)
- [REST API Reference](file:///docs/API.md)
- [Master Implementation Plan](file:///MASTER_PLAN.md)

---

## License

MIT License (c) 2026 Alexandr Motologa
