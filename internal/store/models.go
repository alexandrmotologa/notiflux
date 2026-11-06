package store

import (
	"time"
)

// Channel constants
const (
	ChannelInApp    = "in_app"
	ChannelEmail    = "email"
	ChannelWebPush  = "web_push"
	ChannelTelegram = "telegram"
	ChannelSlack    = "slack"
	ChannelDiscord  = "discord"
	ChannelWebhook  = "webhook"
)

// Delivery status constants
const (
	StatusSent      = "sent"
	StatusFailed    = "failed"
	StatusSkipped   = "skipped"
	StatusScheduled = "scheduled"
)

// WorkflowStep defines a sequential delivery step with optional delay and conditional checks.
type WorkflowStep struct {
	Channel      string `json:"channel"`
	DelaySeconds int    `json:"delay_seconds,omitempty"` // e.g. 900 for 15 minutes
	Condition    string `json:"condition,omitempty"`     // "always", "if_unread"
}

// Workflow represents a notification routing workflow.
type Workflow struct {
	ID                  string         `json:"id"`
	Name                string         `json:"name"`
	Description         string         `json:"description,omitempty"`
	Channels            []string       `json:"channels"` // e.g. ["in_app", "email", "web_push"]
	Steps               []WorkflowStep `json:"steps,omitempty"`
	DigestEnabled       bool           `json:"digest_enabled,omitempty"`
	DigestWindowSeconds int            `json:"digest_window_seconds,omitempty"` // e.g. 300 for 5 minutes
	DigestMaxItems      int            `json:"digest_max_items,omitempty"`
	CreatedAt           time.Time      `json:"created_at"`
	UpdatedAt           time.Time      `json:"updated_at"`
}

// Template represents the message definition for a specific channel in a workflow.
type Template struct {
	ID         string    `json:"id"`
	WorkflowID string    `json:"workflow_id"`
	Channel    string    `json:"channel"` // in_app, email, web_push, telegram, slack, discord, webhook
	Subject    string    `json:"subject,omitempty"`
	Body       string    `json:"body"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// WebPushSubscription represents browser push subscription details.
type WebPushSubscription struct {
	Endpoint string      `json:"endpoint"`
	Keys     WebPushKeys `json:"keys"`
}

type WebPushKeys struct {
	P256dh string `json:"p256dh"`
	Auth   string `json:"auth"`
}

// Subscriber represents a notification recipient.
type Subscriber struct {
	ID                   string                `json:"id"`
	Email                string                `json:"email,omitempty"`
	TelegramChatID       string                `json:"telegram_chat_id,omitempty"`
	SlackWebhookURL      string                `json:"slack_webhook_url,omitempty"`
	DiscordWebhookURL    string                `json:"discord_webhook_url,omitempty"`
	WebPushSubscription *WebPushSubscription `json:"web_push_subscription,omitempty"`
	Timezone             string                `json:"timezone,omitempty"`          // e.g. "Europe/Bucharest"
	QuietHoursStart      string                `json:"quiet_hours_start,omitempty"` // e.g. "22:00"
	QuietHoursEnd        string                `json:"quiet_hours_end,omitempty"`   // e.g. "08:00"
	CreatedAt            time.Time             `json:"created_at"`
	UpdatedAt            time.Time             `json:"updated_at"`
}

// Preference represents channel opt-in/opt-out status for a subscriber.
type Preference struct {
	SubscriberID string `json:"subscriber_id"`
	Channel      string `json:"channel"`
	Enabled      bool   `json:"enabled"`
}

// Notification represents an in-app feed item.
type Notification struct {
	ID           string         `json:"id"`
	WorkflowID   string         `json:"workflow_id"`
	SubscriberID string         `json:"subscriber_id"`
	Title        string         `json:"title"`
	Body         string         `json:"body"`
	Data         map[string]any `json:"data,omitempty"`
	ActionURL    string         `json:"action_url,omitempty"`
	IsRead       bool           `json:"is_read"`
	ReadAt       *time.Time     `json:"read_at,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
}

// DeliveryLog tracks the dispatch attempt of a channel, including click and open tracking.
type DeliveryLog struct {
	ID             string     `json:"id"`
	NotificationID string     `json:"notification_id,omitempty"`
	WorkflowID     string     `json:"workflow_id"`
	SubscriberID   string     `json:"subscriber_id"`
	Channel        string     `json:"channel"`
	Status         string     `json:"status"` // sent, failed, skipped, scheduled
	ErrorMessage   string     `json:"error_message,omitempty"`
	Payload        string     `json:"payload,omitempty"`
	TrackingID     string     `json:"tracking_id,omitempty"`
	TargetURL      string     `json:"target_url,omitempty"`
	OpenedAt       *time.Time `json:"opened_at,omitempty"`
	ClickedAt      *time.Time `json:"clicked_at,omitempty"`
	DispatchedAt   time.Time  `json:"dispatched_at"`
}

// IdempotencyRecord stores cached responses for idempotency keys.
type IdempotencyRecord struct {
	Key          string    `json:"key"`
	StatusCode   int       `json:"status_code"`
	ResponseBody string    `json:"response_body"`
	CreatedAt    time.Time `json:"created_at"`
}

// ScheduledDelivery represents a delayed or retry delivery pending in SQLite.
type ScheduledDelivery struct {
	ID             string    `json:"id"`
	NotificationID string    `json:"notification_id,omitempty"`
	WorkflowID     string    `json:"workflow_id"`
	SubscriberID   string    `json:"subscriber_id"`
	Channel        string    `json:"channel"`
	Subject        string    `json:"subject"`
	Body           string    `json:"body"`
	HTMLBody       string    `json:"html_body,omitempty"`
	ActionURL      string    `json:"action_url,omitempty"`
	Payload        string    `json:"payload,omitempty"`
	ScheduledFor   time.Time `json:"scheduled_for"`
	Attempts       int       `json:"attempts"`
	MaxAttempts    int       `json:"max_attempts"`
	Condition      string    `json:"condition,omitempty"` // "if_unread"
	Status         string    `json:"status"`              // "pending", "executing", "completed", "failed"
	CreatedAt      time.Time `json:"created_at"`
}

// DigestEntry buffers incoming events for a subscriber until the digest window flushes.
type DigestEntry struct {
	ID           string    `json:"id"`
	WorkflowID   string    `json:"workflow_id"`
	SubscriberID string    `json:"subscriber_id"`
	Payload      string    `json:"payload"`
	CreatedAt    time.Time `json:"created_at"`
}

// APIKeyRecord holds multiple workspace/project API keys.
type APIKeyRecord struct {
	Key       string    `json:"key"`
	Name      string    `json:"name"` // e.g. "Development", "Production"
	CreatedAt time.Time `json:"created_at"`
}
