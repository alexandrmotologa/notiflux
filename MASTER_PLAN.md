# notiflux: Engineering Specification & Implementation Plan

> Self-hosted multi-channel notification engine and real-time in-app notification center (< 45MB RAM).  
> Single binary alternative to Novu, Knock, and Courier written in Go and React 19.

---

## 1. Project Overview

### Problem Statement
Modern web and mobile applications require notification delivery across multiple user channels:
- Real-time in-app notification bells with live unread badge counters.
- Transactional emails through SMTP or provider REST APIs (e.g. Resend).
- Mobile and desktop web push notifications adhering to RFC 8292 (VAPID).
- Chat notifications for Telegram and Slack.
- Outbound developer webhooks with cryptographic signing.

Existing tools introduce operational hurdles:
- Commercial SaaS options (Knock, Courier, MagicBell) cost upwards of $250 to $600 per month, bill per subscriber, and transmit sensitive user data outside private infrastructure.
- Open-source platforms like Novu require more than 10 Docker containers (MongoDB, Redis, RabbitMQ, multiple Node services) and draw over 4GB of RAM at idle.

### Solution Architecture
`notiflux` delivers an all-in-one notification daemon in a single static binary:
- Core daemon built in Go using `go-chi/chi/v5` and pure Go SQLite (`modernc.org/sqlite`).
- RAM consumption remains below 45MB during steady operations.
- Built-in multi-channel dispatchers: In-App, Email (SMTP + Resend API), Web Push (RFC 8292 VAPID), Telegram Bot API, Slack Incoming Webhooks, and signed HTTP webhooks.
- Drop-in client notification widget under 3KB (Vanilla Web Component and React 19) connected via Server-Sent Events (SSE).
- Embedded React 19 management studio with live workflow builders, template editors, delivery logs, and recipient preference controls.

---

## 2. System Architecture

```
                      [ Application Backends (Node, Python, Go, Ruby, etc.) ]
                                                 │
                                                 │ POST /v1/trigger (Idempotency-Key)
                                                 ▼
┌────────────────────────────────────────────────────────────────────────────────────────┐
│                                   notiflux Core Daemon                                 │
│                                                                                        │
│  ┌───────────────────────┐          ┌──────────────────────┐                           │
│  │ API Key Authorization │ ───────► │ Idempotency Engine   │                           │
│  └───────────────────────┘          └──────────┬───────────┘                           │
│                                                │                                       │
│                                                ▼                                       │
│  ┌──────────────────────────────────────────────────────────────────┐                  │
│  │ Workflow Router & Recipient Preference Engine                     │                  │
│  │ - Evaluates channel mute flags, global opt-outs, quiet hours     │                  │
│  └─────────────────────────────────────┬────────────────────────────┘                  │
│                                        │ Active channels                               │
│                                        ▼                                               │
│  ┌──────────────────────────────────────────────────────────────────┐                  │
│  │ Template Compiler & Markdown Processor                            │                  │
│  │ - Safe Mustache variable interpolation ({{user.name}})            │                  │
│  │ - Markdown to styled HTML converter                              │                  │
│  └─────────────────────────────────────┬────────────────────────────┘                  │
│                                        │ Rendered messages                             │
│                  ┌─────────────────────┴────────────────────┐                          │
│                  ▼                                          ▼                          │
│  ┌─────────────────────────────────┐      ┌─────────────────────────────────────────┐  │
│  │ Outbound Dispatcher Pool        │      │ In-App Feed & Storage                   │  │
│  │ - Email: SMTP & Resend API      │      │ - SQLite WAL Engine (WAL mode, Pragmas) │  │
│  │ - Web Push: RFC 8292 VAPID      │      └────────────────────┬────────────────────┘  │
│  │ - Telegram Bot API              │                           │                       │
│  │ - Slack Webhooks                │                           ▼                       │
│  │ - Outbound Signed Webhooks      │      ┌─────────────────────────────────────────┐  │
│  └─────────────────────────────────┘      │ Real-Time SSE Stream Hub                │  │
│                                           │ - 15s heartbeats, reconnection handling │  │
│                                           └────────────────────┬────────────────────┘  │
│                                                                │                       │
│  ┌─────────────────────────────────────────────────────────────▼────────────────────┐  │
│  │ Client: <notiflux-bell> / <NotificationBell /> Drop-in Widget                    │  │
│  │ - Served directly via /widget/notiflux-bell.js (< 3KB)                           │  │
│  └──────────────────────────────────────────────────────────────────────────────────┘  │
│                                                                                        │
│  ┌──────────────────────────────────────────────────────────────────────────────────┐  │
│  │ Embedded React 19 Studio (/studio)                                               │  │
│  │ - Workflows, Templates, Delivery Logs, Subscriber Controls, System Metrics       │  │
│  └──────────────────────────────────────────────────────────────────────────────────┘  │
└────────────────────────────────────────────────────────────────────────────────────────┘
```

---

## 3. Technology Stack & Component Specifications

| Component | Technology | Rationale |
| :--- | :--- | :--- |
| **Language & Runtime** | Go 1.23+ | Memory footprint under 45MB, single static binary, goroutine concurrency |
| **HTTP Framework** | `go-chi/chi/v5` | Lightweight routing, zero memory allocations, middleware ecosystem |
| **Storage Engine** | Pure Go SQLite (`modernc.org/sqlite`) | Zero CGO requirement, cross-compiles without gcc, WAL journaling mode |
| **Real-Time Transport** | Server-Sent Events (SSE) | HTTP/1.1 and HTTP/2 compatible, native browser auto-reconnect, zero WebSocket overhead |
| **Web Push Protocol** | RFC 8292 (VAPID) / RFC 8291 | ECDSA P-256 signing, aes128gcm payload encryption |
| **Template Engine** | Tokenizer with Markdown support | Mustache compatible variable replacement, conditional blocks, HTML escaping |
| **Client Widget** | Vanilla Web Component + React 19 | Self-contained, under 3KB gzipped, served directly from the daemon |
| **Embedded Studio** | React 19, TypeScript, Vite, Tailwind | Zero external hosting needed, compiled directly into Go binary via `//go:embed` |

---

## 4. Database Schema Design (SQLite WAL Mode)

```sql
PRAGMA journal_mode = WAL;
PRAGMA foreign_keys = ON;
PRAGMA busy_timeout = 5000;
PRAGMA synchronous = NORMAL;

CREATE TABLE IF NOT EXISTS workflows (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    channels TEXT NOT NULL, -- JSON array of enabled channels: ["in_app", "email", "web_push"]
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS templates (
    id TEXT PRIMARY KEY,
    workflow_id TEXT NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    channel TEXT NOT NULL, -- in_app, email, web_push, telegram, slack, webhook
    subject TEXT,          -- Subject for email or push title
    body TEXT NOT NULL,     -- Mustache template content (Markdown / HTML / JSON)
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(workflow_id, channel)
);

CREATE TABLE IF NOT EXISTS subscribers (
    id TEXT PRIMARY KEY,
    email TEXT,
    telegram_chat_id TEXT,
    slack_webhook_url TEXT,
    web_push_subscription TEXT, -- JSON string of PushSubscription
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS preferences (
    subscriber_id TEXT NOT NULL REFERENCES subscribers(id) ON DELETE CASCADE,
    channel TEXT NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT 1,
    PRIMARY KEY (subscriber_id, channel)
);

CREATE TABLE IF NOT EXISTS notifications (
    id TEXT PRIMARY KEY,
    workflow_id TEXT NOT NULL REFERENCES workflows(id),
    subscriber_id TEXT NOT NULL REFERENCES subscribers(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    body TEXT NOT NULL,
    data TEXT,             -- JSON metadata
    action_url TEXT,
    is_read BOOLEAN DEFAULT 0,
    read_at DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS delivery_logs (
    id TEXT PRIMARY KEY,
    notification_id TEXT,
    workflow_id TEXT NOT NULL,
    subscriber_id TEXT NOT NULL,
    channel TEXT NOT NULL,
    status TEXT NOT NULL,  -- sent, failed, skipped
    error_message TEXT,
    payload TEXT,
    dispatched_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS idempotency_keys (
    key TEXT PRIMARY KEY,
    response_body TEXT NOT NULL,
    status_code INTEGER NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_notifications_subscriber ON notifications(subscriber_id, is_read, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_delivery_logs_subscriber ON delivery_logs(subscriber_id, dispatched_at DESC);
CREATE INDEX IF NOT EXISTS idx_delivery_logs_status ON delivery_logs(status, dispatched_at DESC);
```

---

## 5. API Endpoints Specification

### Authentication
All administrative and trigger endpoints require the HTTP header:
`Authorization: Bearer <API_KEY>` or `X-API-Key: <API_KEY>`.
The client widget and SSE stream use scoped public tokens or direct recipient queries.

### Ingestion API
- `POST /v1/trigger`: Trigger notification workflow. Supports `Idempotency-Key` header.
- `POST /v1/trigger/batch`: Batch ingest up to 500 trigger requests.

### Feed & In-App API (Client-facing)
- `GET /v1/feed?subscriber_id=<ID>&limit=20&unread_only=false`: List in-app notifications.
- `GET /v1/feed/stream?subscriber_id=<ID>`: Server-Sent Events stream for real-time updates.
- `PATCH /v1/feed/:id/read`: Mark notification as read.
- `PATCH /v1/feed/:id/unread`: Mark notification as unread.
- `POST /v1/feed/read-all`: Mark all notifications for a subscriber as read.
- `DELETE /v1/feed/:id`: Delete notification.

### Preferences API
- `GET /v1/preferences/:subscriber_id`: Get channel opt-in status.
- `PUT /v1/preferences/:subscriber_id`: Update channel opt-in status.

### Workflows & Templates API
- `GET /v1/workflows`: List registered workflows.
- `POST /v1/workflows`: Create workflow.
- `GET /v1/workflows/:id`: Retrieve workflow details.
- `PUT /v1/workflows/:id`: Update workflow.
- `DELETE /v1/workflows/:id`: Delete workflow.
- `POST /v1/templates/preview`: Render template against test variables.

### Admin & Operations API
- `GET /v1/logs`: Query delivery logs with filtering by status and channel.
- `GET /v1/subscribers`: List subscribers and subscriber delivery status.
- `GET /v1/health`: System health and memory usage statistics.
- `POST /v1/system/vapid`: Generate new RFC 8292 VAPID key pairs.
- `POST /v1/system/test-channel`: Test credentials for Email, Web Push, Telegram, Slack.

---

## 6. Implementation Tasks & Verification Roadmap

### Task 1: Project Setup, Tooling & Configuration
- [x] 1.1: Initialize Go module `github.com/alexandrmotologa/notiflux` with dependencies (`go-chi`, `modernc.org/sqlite`, `resend-go`, `webpush-go`).
- [x] 1.2: Implement configuration loader (`internal/config/config.go`) reading environment variables and YAML flags with strict validation.
- [x] 1.3: Create root CLI daemon commands (`cmd/notiflux/main.go`) supporting `serve`, `vapid`, and `version`.
- [x] 1.4: Configure Makefile with test, build, and static bundle targets.

### Task 2: Pure Go Storage Layer (SQLite WAL Mode)
- [x] 2.1: Implement SQLite storage engine (`internal/store/sqlite.go`) with WAL pragmas and connection pooling.
- [x] 2.2: Implement schema migrations (`internal/store/schema.go`).
- [x] 2.3: Implement repository models and query methods for workflows, templates, subscribers, preferences, notifications, delivery logs, and idempotency keys.
- [x] 2.4: Write unit tests verifying database operations, concurrent writes, and foreign key cascades.

### Task 3: Template Engine & Interpolation Processor
- [x] 3.1: Implement high-speed template compiler (`internal/engine/compiler.go`) parsing Mustache variables, nested maps, and conditional blocks.
- [x] 3.2: Implement Markdown-to-HTML parser with inline email styles for clean transactional email rendering.
- [x] 3.3: Write unit tests covering variable replacement, escaping, missing properties, and markdown conversions.

### Task 4: Workflow Router & Preferences Engine
- [x] 4.1: Implement recipient preference evaluator (`internal/engine/preferences.go`) checking channel opt-in, global opt-outs, and delivery overrides.
- [x] 4.2: Implement workflow orchestrator (`internal/engine/workflow.go`) coordinating channel dispatches and recording delivery logs.
- [x] 4.3: Implement idempotency evaluator (`internal/engine/idempotency.go`) caching responses and preventing duplicate sends.

### Task 5: Multi-Channel Dispatchers
- [x] 5.1: In-App Dispatcher (`internal/channels/inapp.go`): Persist messages and push to active SSE streams.
- [x] 5.2: Email Dispatcher (`internal/channels/email.go`): Dual-mode dispatcher supporting SMTP (STARTTLS/TLS) and Resend REST API.
- [x] 5.3: Web Push Dispatcher (`internal/channels/webpush.go`): RFC 8292 VAPID ECDSA P-256 signer and aes128gcm payload encryptor.
- [x] 5.4: Telegram Dispatcher (`internal/channels/telegram.go`): Telegram Bot API client with markdown parsing and action buttons.
- [x] 5.5: Slack Dispatcher (`internal/channels/slack.go`): Slack incoming webhook dispatcher with block layouts.
- [x] 5.6: Outbound Webhook Dispatcher (`internal/channels/webhook.go`): HTTPS POST with HMAC-SHA256 signature calculation.
- [x] 5.7: Write mock integration tests for all dispatchers verifying payload delivery and error handling.

### Task 6: Real-Time SSE Hub & Feed APIs
- [x] 6.1: Implement SSE broker (`internal/channels/sse_hub.go`) with subscriber channel subscriptions, 15s heartbeats, and client disconnect cleanup.
- [x] 6.2: Implement Feed API routes (`internal/api/feed.go`): fetch feed, mark read/unread, mark all read, delete.
- [x] 6.3: Implement Trigger API routes (`internal/api/triggers.go`): single trigger and batch trigger.
- [x] 6.4: Implement Preferences, Workflows, Templates, and Logs REST endpoints.

### Task 7: Client Drop-In Widget (< 3KB)
- [x] 7.1: Build Vanilla Web Component `<notiflux-bell>` in `widget/src/vanilla.ts` with zero dependencies, shadow DOM styling, and real-time SSE listener.
- [x] 7.2: Build React 19 `<NotificationBell />` wrapper in `widget/src/bell.tsx`.
- [x] 7.3: Implement dropdown popover with unread counter badge, relative timestamps, tabs, and mark-read interactions.
- [x] 7.4: Configure rollup/vite bundle generating `notiflux-bell.js` and `notiflux-bell.css` under 3KB gzipped.
- [x] 7.5: Embed widget bundle directly into Go server at `/widget/notiflux-bell.js`.

### Task 8: Embedded React 19 Administration Studio
- [x] 8.1: Scaffold React 19 + TypeScript + Vite + Tailwind studio in `web/`.
- [x] 8.2: Build Dashboard view with live memory meter, connection counts, and delivery metrics.
- [x] 8.3: Build Workflow Editor for configuring channels and execution rules.
- [x] 8.4: Build Template Designer with live preview for Email, Push, In-App, Telegram, and Slack.
- [x] 8.5: Build Delivery Logs viewer with live stream filtering and payload inspection.
- [x] 8.6: Build Subscribers and Channel Settings inspectors with credential test runners.
- [x] 8.7: Build static assets into `web/dist` and embed them into the Go binary with `//go:embed`.

### Task 9: System Integration, Benchmarks & Documentation
- [x] 9.1: Write end-to-end integration test suite exercising the entire trigger-to-widget pipeline.
- [x] 9.2: Verify RAM usage remains below 45MB during idle and under active load.
- [x] 9.3: Create comprehensive README.md following humanizer guidelines (no em dashes, factual, no AI clichés).
- [x] 9.4: Provide Dockerfile and Docker Compose configurations for production deployment.

### Task 10: Advanced Engine Features & Production Hardening
- [x] 10.1: Discord Webhooks Dispatcher with rich embed schemas, colors, and timestamps.
- [x] 10.2: Background Scheduler Loop (`internal/engine/scheduler.go`) evaluating step delays, quiet hours, retries, and digest flushing.
- [x] 10.3: Step Delays with Unread Fallback (`condition: "if_unread"`, `delay_seconds: 900`).
- [x] 10.4: Digest Aggregation buffering high-frequency events and delivering consolidated batch updates.
- [x] 10.5: Quiet Hours and Timezone evaluation postponing disruptive dispatches to morning release while allowing urgent bypass.
- [x] 10.6: Open and Click Engagement Tracking with 1x1 transparent PNG pixel injection and HTTP 302 click redirects.
- [x] 10.7: Online Hot SQLite Backups via SQLite VACUUM INTO and CLI subcommand `notiflux backup`.
- [x] 10.8: Scoped Workspace API Keys management with multi-key database validation and header authentication.
- [x] 10.9: Client Widget synthetic Web Audio chime and native desktop Web Push permission prompts.
- [x] 10.10: Integration Code Snippets generator in Admin Studio for cURL, Node.js, Python, Go, and PHP.

