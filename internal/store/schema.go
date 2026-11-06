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
    web_push_subscription TEXT,
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
`

// Migrate runs SQLite pragmas and applies all schema tables and indexes.
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

	return nil
}
