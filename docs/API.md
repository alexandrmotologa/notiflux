# Notiflux REST API Reference

The Notiflux daemon exposes a REST API on port `8085` (default). All administrative endpoints require an authorization token.

## Authentication

Provide your API key using either of the following HTTP headers:
```http
Authorization: Bearer <API_KEY>
X-API-Key: <API_KEY>
```

Client feed endpoints (such as `GET /v1/feed`) can be called with `subscriber_id` query parameters directly or with scoped public tokens.

---

## Trigger Notification

### Single Trigger
Dispatches a workflow event to a specific recipient across configured channels.

```http
POST /v1/trigger
Content-Type: application/json
Idempotency-Key: req_98124801 (optional)

{
  "workflow": "order.shipped",
  "subscriber": {
    "id": "usr_1001",
    "email": "user@example.com",
    "telegram_chat_id": "123456789",
    "slack_webhook_url": "https://hooks.slack.com/services/...",
    "web_push_subscription": {
      "endpoint": "https://updates.push.services.mozilla.com/...",
      "keys": {
        "p256dh": "BNcRdreALRFXTkOOUHK1T9S...",
        "auth": "tBHItJI5svb..."
      }
    }
  },
  "payload": {
    "order_id": "#8942",
    "carrier": "DHL Express",
    "tracking_number": "DHL-98124-US",
    "action_url": "https://store.example.com/orders/8942"
  },
  "overrides": {
    "channels": ["in_app", "email", "telegram"]
  }
}
```

#### Response:
```json
{
  "success": true,
  "notification_id": "notif_01hw56kq...",
  "dispatched_channels": ["in_app", "email", "telegram"],
  "skipped_channels": ["web_push", "slack"]
}
```

---

### Batch Trigger
Triggers up to 500 notifications simultaneously.

```http
POST /v1/trigger/batch
Content-Type: application/json

{
  "notifications": [
    {
      "workflow": "system.maintenance",
      "subscriber": { "id": "usr_1", "email": "usr1@example.com" },
      "payload": { "time": "22:00 UTC" }
    },
    {
      "workflow": "system.maintenance",
      "subscriber": { "id": "usr_2", "email": "usr2@example.com" },
      "payload": { "time": "22:00 UTC" }
    }
  ]
}
```

---

## In-App Feed & Client API

### List In-App Notifications
Retrieves notifications stored for an in-app subscriber.

```http
GET /v1/feed?subscriber_id=usr_1001&limit=20&unread_only=false
```

#### Response:
```json
{
  "notifications": [
    {
      "id": "notif_01hw56kq...",
      "workflow_id": "order.shipped",
      "title": "Your order #8942 has shipped",
      "body": "DHL Express tracking number DHL-98124-US",
      "action_url": "https://store.example.com/orders/8942",
      "is_read": false,
      "read_at": null,
      "created_at": "2026-11-07T00:10:00Z"
    }
  ],
  "unread_count": 1,
  "total": 1
}
```

### Real-Time SSE Stream
Establishes a persistent Server-Sent Events stream for real-time notifications and unread badge synchronization.

```http
GET /v1/feed/stream?subscriber_id=usr_1001
Accept: text/event-stream
```

Events dispatched:
- `event: connected` - Emitted upon connection with current unread count.
- `event: notification.new` - Dispatched when a new in-app notification arrives.
- `event: notification.read` - Dispatched when a notification is marked read.
- `event: notification.read_all` - Dispatched when all notifications are marked read.
- `: ping` - Keep-alive comment dispatched every 15 seconds.

### Mark Notification as Read
```http
PATCH /v1/feed/:id/read
```

### Mark All as Read
```http
POST /v1/feed/read-all
Content-Type: application/json

{
  "subscriber_id": "usr_1001"
}
```

### Delete Notification
```http
DELETE /v1/feed/:id
```

---

## Subscriber Preferences API

### Get Preferences
```http
GET /v1/preferences/usr_1001
```

#### Response:
```json
{
  "subscriber_id": "usr_1001",
  "channels": {
    "in_app": true,
    "email": true,
    "web_push": false,
    "telegram": true,
    "slack": false
  }
}
```

### Update Preferences
```http
PUT /v1/preferences/usr_1001
Content-Type: application/json

{
  "channels": {
    "web_push": true,
    "email": false
  }
}
```

---

## Workflows & Templates API

### List Workflows
```http
GET /v1/workflows
```

### Create Workflow
```http
POST /v1/workflows
Content-Type: application/json

{
  "id": "order.shipped",
  "name": "Order Shipped Notification",
  "description": "Triggered when a warehouse marks an order as shipped",
  "channels": ["in_app", "email", "web_push"]
}
```

### Update Workflow Templates
```http
PUT /v1/workflows/:id/templates
Content-Type: application/json

{
  "templates": [
    {
      "channel": "in_app",
      "subject": "Order #{{order_id}} shipped",
      "body": "Your order is with {{carrier}}, tracking: {{tracking_number}}"
    },
    {
      "channel": "email",
      "subject": "Package in transit: Order #{{order_id}}",
      "body": "## Good news!\n\nYour order **#{{order_id}}** has shipped via {{carrier}}.\n\n[Track Package]({{action_url}})"
    }
  ]
}
```

---

## System & Diagnostics API

### Health and Memory Stats
```http
GET /v1/health
```

#### Response:
```json
{
  "status": "healthy",
  "version": "1.0.0",
  "uptime_seconds": 3600,
  "memory_alloc_mb": 14.8,
  "memory_sys_mb": 26.2,
  "goroutines": 18,
  "active_sse_connections": 42
}
```

### Generate VAPID Keys
```http
POST /v1/system/vapid
```

#### Response:
```json
{
  "public_key": "BNcRdreALRFXTkOOUHK1T9S...",
  "private_key": "MC4CAQAwBQYDK2VwBCIEI..."
}
```

---

## Workspace API Keys Management

### List Active Keys
```http
GET /v1/system/keys
Authorization: Bearer <ROOT_API_KEY>
```

#### Response:
```json
{
  "keys": [
    {
      "key": "nf_bf727c56605949d590c35b72768df236",
      "name": "Production Backend",
      "created_at": "2026-11-07T01:00:00Z"
    }
  ]
}
```

### Create Key
```http
POST /v1/system/keys
Authorization: Bearer <ROOT_API_KEY>
Content-Type: application/json

{
  "name": "CI Deployment Pipeline"
}
```

#### Response:
```json
{
  "key": "nf_88a44b1c20e24177b960d70f9b69b221",
  "name": "CI Deployment Pipeline",
  "created_at": "2026-11-07T01:05:00Z"
}
```

### Delete Key
```http
DELETE /v1/system/keys/:key
Authorization: Bearer <ROOT_API_KEY>
```

---

## Online Hot SQLite Backup

Atomic snapshot using SQLite VACUUM INTO, safe during concurrent WAL reads and writes with zero downtime.

### Create Online Backup
```http
POST /v1/system/backup
Authorization: Bearer <ROOT_API_KEY>
Content-Type: application/json

{
  "destination": "backups/notiflux_daily.db"
}
```

#### Response:
```json
{
  "success": true,
  "backup_file": "/absolute/path/to/backups/notiflux_daily.db",
  "destination": "/absolute/path/to/backups/notiflux_daily.db",
  "timestamp": "2026-11-07T01:10:00Z"
}
```

---

## Engagement Tracking

### 1x1 Open Pixel
Returns a transparent 1x1 PNG pixel and records the open timestamp for email analytics.
```http
GET /v1/o/:tracking_id.png
```

### Click Redirect
Records the click timestamp and immediately redirects (HTTP 302) the recipient to the original action URL.
```http
GET /v1/r/:tracking_id
```

