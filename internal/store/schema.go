package store

import (
	"database/sql"
	"fmt"
)

const schemaSQL = `
CREATE TABLE IF NOT EXISTS workflows (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    channels TEXT NOT NULL, -- JSON array of enabled channels
    steps TEXT,             -- JSON array of WorkflowStep
    digest_enabled BOOLEAN DEFAULT 0,
    digest_window_seconds INTEGER DEFAULT 0,
    digest_max_items INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS templates (
    id TEXT PRIMARY KEY,
    workflow_id TEXT NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    channel TEXT NOT NULL,
    subject TEXT,
    body TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(workflow_id, channel)
);

CREATE TABLE IF NOT EXISTS subscribers (
    id TEXT PRIMARY KEY,
    email TEXT,
    telegram_chat_id TEXT,
    slack_webhook_url TEXT,
    discord_webhook_url TEXT,
    web_push_subscription TEXT,
    timezone TEXT,
    quiet_hours_start TEXT,
    quiet_hours_end TEXT,
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
    workflow_id TEXT NOT NULL,
    subscriber_id TEXT NOT NULL REFERENCES subscribers(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    body TEXT NOT NULL,
    data TEXT,
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
    status TEXT NOT NULL,
    error_message TEXT,
    payload TEXT,
    tracking_id TEXT,
    target_url TEXT,
    opened_at DATETIME,
    clicked_at DATETIME,
    dispatched_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS idempotency_keys (
    key TEXT PRIMARY KEY,
    response_body TEXT NOT NULL,
    status_code INTEGER NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS scheduled_deliveries (
    id TEXT PRIMARY KEY,
    notification_id TEXT,
    workflow_id TEXT NOT NULL,
    subscriber_id TEXT NOT NULL,
    channel TEXT NOT NULL,
    subject TEXT NOT NULL,
    body TEXT NOT NULL,
    html_body TEXT,
    action_url TEXT,
    payload TEXT,
    scheduled_for DATETIME NOT NULL,
    attempts INTEGER DEFAULT 0,
    max_attempts INTEGER DEFAULT 3,
    condition TEXT,
    status TEXT DEFAULT 'pending',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS digest_entries (
    id TEXT PRIMARY KEY,
    workflow_id TEXT NOT NULL,
    subscriber_id TEXT NOT NULL,
    payload TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS api_keys (
    key TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_notifications_subscriber ON notifications(subscriber_id, is_read, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_delivery_logs_subscriber ON delivery_logs(subscriber_id, dispatched_at DESC);
CREATE INDEX IF NOT EXISTS idx_delivery_logs_status ON delivery_logs(status, dispatched_at DESC);
CREATE INDEX IF NOT EXISTS idx_delivery_logs_tracking ON delivery_logs(tracking_id);
CREATE INDEX IF NOT EXISTS idx_scheduled_deliveries_pending ON scheduled_deliveries(status, scheduled_for ASC);
CREATE INDEX IF NOT EXISTS idx_digest_entries_lookup ON digest_entries(workflow_id, subscriber_id, created_at ASC);
`

// Migrate runs SQLite pragmas and applies all schema tables and columns.
func Migrate(db *sql.DB) error {
	pragmas := []string{
		"PRAGMA journal_mode = WAL;",
		"PRAGMA foreign_keys = ON;",
		"PRAGMA busy_timeout = 5000;",
		"PRAGMA synchronous = NORMAL;",
	}

	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			return fmt.Errorf("failed executing pragma %q: %w", p, err)
		}
	}

	if _, err := db.Exec(schemaSQL); err != nil {
		return fmt.Errorf("failed creating schema: %w", err)
	}

	// Graceful column migrations for existing databases
	columnMigrations := []string{
		"ALTER TABLE workflows ADD COLUMN steps TEXT;",
		"ALTER TABLE workflows ADD COLUMN digest_enabled BOOLEAN DEFAULT 0;",
		"ALTER TABLE workflows ADD COLUMN digest_window_seconds INTEGER DEFAULT 0;",
		"ALTER TABLE workflows ADD COLUMN digest_max_items INTEGER DEFAULT 0;",
		"ALTER TABLE subscribers ADD COLUMN discord_webhook_url TEXT;",
		"ALTER TABLE subscribers ADD COLUMN timezone TEXT;",
		"ALTER TABLE subscribers ADD COLUMN quiet_hours_start TEXT;",
		"ALTER TABLE subscribers ADD COLUMN quiet_hours_end TEXT;",
		"ALTER TABLE delivery_logs ADD COLUMN tracking_id TEXT;",
		"ALTER TABLE delivery_logs ADD COLUMN target_url TEXT;",
		"ALTER TABLE delivery_logs ADD COLUMN opened_at DATETIME;",
		"ALTER TABLE delivery_logs ADD COLUMN clicked_at DATETIME;",
	}

	for _, colSQL := range columnMigrations {
		_, _ = db.Exec(colSQL) // ignore error if column already exists
	}

	return nil
}
