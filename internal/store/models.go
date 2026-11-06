package store

import (
	"time"
)

// Channel constants
const (
	ChannelInApp   = "in_app"
	ChannelEmail   = "email"
	ChannelWebPush = "web_push"
	ChannelTelegram = "telegram"
	ChannelSlack   = "slack"
	ChannelWebhook = "webhook"
)

// Delivery status constants
const (
	StatusSent    = "sent"
	StatusFailed  = "failed"
	StatusSkipped = "skipped"
)

// Workflow represents a notification routing workflow.
type Workflow struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Channels    []string  `json:"channels"` // e.g. ["in_app", "email", "web_push"]
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Template represents the message definition for a specific channel in a workflow.
type Template struct {
	ID         string    `json:"id"`
	WorkflowID string    `json:"workflow_id"`
	Channel    string    `json:"channel"` // in_app, email, web_push, telegram, slack, webhook
	Subject    string    `json:"subject,omitempty"`
	Body       string    `json:"body"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// WebPushSubscription represents browser push subscription details.
type WebPushSubscription struct {
	Endpoint string            `json:"endpoint"`
	Keys     WebPushKeys       `json:"keys"`
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
	WebPushSubscription *WebPushSubscription `json:"web_push_subscription,omitempty"`
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

// DeliveryLog tracks the dispatch attempt of a channel.
type DeliveryLog struct {
	ID             string    `json:"id"`
	NotificationID string    `json:"notification_id,omitempty"`
	WorkflowID     string    `json:"workflow_id"`
	SubscriberID   string    `json:"subscriber_id"`
	Channel        string    `json:"channel"`
	Status         string    `json:"status"` // sent, failed, skipped
	ErrorMessage   string    `json:"error_message,omitempty"`
	Payload        string    `json:"payload,omitempty"`
	DispatchedAt   time.Time `json:"dispatched_at"`
}

// IdempotencyRecord stores cached responses for idempotency keys.
type IdempotencyRecord struct {
	Key          string    `json:"key"`
	StatusCode   int       `json:"status_code"`
	ResponseBody string    `json:"response_body"`
	CreatedAt    time.Time `json:"created_at"`
}
