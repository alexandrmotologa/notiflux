package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

// Store encapsulates the SQLite database connection and operations.
type Store struct {
	db *sql.DB
}

// NewStore opens or creates a SQLite database at dbPath and executes migrations.
func NewStore(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed opening sqlite database: %w", err)
	}

	// SQLite handles concurrent reads with WAL, but single writer is safest
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)

	if err := Migrate(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed running sqlite migrations: %w", err)
	}

	return &Store{db: db}, nil
}

// Close closes the database handle.
func (s *Store) Close() error {
	return s.db.Close()
}

// DB returns the underlying database handle for advanced usage or testing.
func (s *Store) DB() *sql.DB {
	return s.db
}

// ==================== WORKFLOWS ====================

func (s *Store) CreateWorkflow(wf *Workflow) error {
	if wf.ID == "" {
		wf.ID = uuid.NewString()
	}
	wf.CreatedAt = time.Now().UTC()
	wf.UpdatedAt = wf.CreatedAt

	channelsJSON, err := json.Marshal(wf.Channels)
	if err != nil {
		return err
	}

	query := `INSERT INTO workflows (id, name, description, channels, created_at, updated_at) 
	          VALUES (?, ?, ?, ?, ?, ?)`
	_, err = s.db.Exec(query, wf.ID, wf.Name, wf.Description, string(channelsJSON), wf.CreatedAt, wf.UpdatedAt)
	return err
}

func (s *Store) GetWorkflow(id string) (*Workflow, error) {
	query := `SELECT id, name, description, channels, created_at, updated_at FROM workflows WHERE id = ?`
	row := s.db.QueryRow(query, id)

	var wf Workflow
	var channelsRaw string
	err := row.Scan(&wf.ID, &wf.Name, &wf.Description, &channelsRaw, &wf.CreatedAt, &wf.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	if err := json.Unmarshal([]byte(channelsRaw), &wf.Channels); err != nil {
		wf.Channels = []string{}
	}
	return &wf, nil
}

func (s *Store) ListWorkflows() ([]*Workflow, error) {
	query := `SELECT id, name, description, channels, created_at, updated_at FROM workflows ORDER BY created_at ASC`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*Workflow
	for rows.Next() {
		var wf Workflow
		var channelsRaw string
		if err := rows.Scan(&wf.ID, &wf.Name, &wf.Description, &channelsRaw, &wf.CreatedAt, &wf.UpdatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(channelsRaw), &wf.Channels)
		list = append(list, &wf)
	}
	return list, rows.Err()
}

func (s *Store) UpdateWorkflow(wf *Workflow) error {
	wf.UpdatedAt = time.Now().UTC()
	channelsJSON, err := json.Marshal(wf.Channels)
	if err != nil {
		return err
	}

	query := `UPDATE workflows SET name = ?, description = ?, channels = ?, updated_at = ? WHERE id = ?`
	res, err := s.db.Exec(query, wf.Name, wf.Description, string(channelsJSON), wf.UpdatedAt, wf.ID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("workflow %q not found", wf.ID)
	}
	return nil
}

func (s *Store) DeleteWorkflow(id string) error {
	query := `DELETE FROM workflows WHERE id = ?`
	res, err := s.db.Exec(query, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("workflow %q not found", id)
	}
	return nil
}

// ==================== TEMPLATES ====================

func (s *Store) UpsertTemplate(tpl *Template) error {
	if tpl.ID == "" {
		tpl.ID = uuid.NewString()
	}
	now := time.Now().UTC()
	tpl.UpdatedAt = now

	query := `INSERT INTO templates (id, workflow_id, channel, subject, body, created_at, updated_at)
	          VALUES (?, ?, ?, ?, ?, ?, ?)
	          ON CONFLICT(workflow_id, channel) DO UPDATE SET
	              subject = excluded.subject,
	              body = excluded.body,
	              updated_at = excluded.updated_at`

	_, err := s.db.Exec(query, tpl.ID, tpl.WorkflowID, tpl.Channel, tpl.Subject, tpl.Body, now, now)
	return err
}

func (s *Store) GetTemplatesByWorkflow(workflowID string) ([]*Template, error) {
	query := `SELECT id, workflow_id, channel, subject, body, created_at, updated_at 
	          FROM templates WHERE workflow_id = ? ORDER BY channel ASC`
	rows, err := s.db.Query(query, workflowID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*Template
	for rows.Next() {
		var tpl Template
		var subject sql.NullString
		if err := rows.Scan(&tpl.ID, &tpl.WorkflowID, &tpl.Channel, &subject, &tpl.Body, &tpl.CreatedAt, &tpl.UpdatedAt); err != nil {
			return nil, err
		}
		if subject.Valid {
			tpl.Subject = subject.String
		}
		list = append(list, &tpl)
	}
	return list, rows.Err()
}

func (s *Store) GetTemplateByWorkflowAndChannel(workflowID, channel string) (*Template, error) {
	query := `SELECT id, workflow_id, channel, subject, body, created_at, updated_at 
	          FROM templates WHERE workflow_id = ? AND channel = ?`
	row := s.db.QueryRow(query, workflowID, channel)

	var tpl Template
	var subject sql.NullString
	err := row.Scan(&tpl.ID, &tpl.WorkflowID, &tpl.Channel, &subject, &tpl.Body, &tpl.CreatedAt, &tpl.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	if subject.Valid {
		tpl.Subject = subject.String
	}
	return &tpl, nil
}

func (s *Store) DeleteTemplate(workflowID, channel string) error {
	query := `DELETE FROM templates WHERE workflow_id = ? AND channel = ?`
	_, err := s.db.Exec(query, workflowID, channel)
	return err
}

// ==================== SUBSCRIBERS ====================

func (s *Store) UpsertSubscriber(sub *Subscriber) error {
	now := time.Now().UTC()
	sub.UpdatedAt = now

	var webPushJSON sql.NullString
	if sub.WebPushSubscription != nil {
		b, err := json.Marshal(sub.WebPushSubscription)
		if err != nil {
			return err
		}
		webPushJSON = sql.NullString{String: string(b), Valid: true}
	}

	query := `INSERT INTO subscribers (id, email, telegram_chat_id, slack_webhook_url, web_push_subscription, created_at, updated_at)
	          VALUES (?, ?, ?, ?, ?, ?, ?)
	          ON CONFLICT(id) DO UPDATE SET
	              email = COALESCE(excluded.email, subscribers.email),
	              telegram_chat_id = COALESCE(excluded.telegram_chat_id, subscribers.telegram_chat_id),
	              slack_webhook_url = COALESCE(excluded.slack_webhook_url, subscribers.slack_webhook_url),
	              web_push_subscription = COALESCE(excluded.web_push_subscription, subscribers.web_push_subscription),
	              updated_at = excluded.updated_at`

	_, err := s.db.Exec(query, sub.ID, sub.Email, sub.TelegramChatID, sub.SlackWebhookURL, webPushJSON, now, now)
	return err
}

func (s *Store) GetSubscriber(id string) (*Subscriber, error) {
	query := `SELECT id, email, telegram_chat_id, slack_webhook_url, web_push_subscription, created_at, updated_at 
	          FROM subscribers WHERE id = ?`
	row := s.db.QueryRow(query, id)

	var sub Subscriber
	var email, telegram, slack, pushSub sql.NullString
	err := row.Scan(&sub.ID, &email, &telegram, &slack, &pushSub, &sub.CreatedAt, &sub.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	if email.Valid {
		sub.Email = email.String
	}
	if telegram.Valid {
		sub.TelegramChatID = telegram.String
	}
	if slack.Valid {
		sub.SlackWebhookURL = slack.String
	}
	if pushSub.Valid && pushSub.String != "" {
		var wps WebPushSubscription
		if err := json.Unmarshal([]byte(pushSub.String), &wps); err == nil {
			sub.WebPushSubscription = &wps
		}
	}
	return &sub, nil
}

func (s *Store) ListSubscribers(limit, offset int) ([]*Subscriber, error) {
	if limit <= 0 {
		limit = 50
	}
	query := `SELECT id, email, telegram_chat_id, slack_webhook_url, web_push_subscription, created_at, updated_at 
	          FROM subscribers ORDER BY updated_at DESC LIMIT ? OFFSET ?`
	rows, err := s.db.Query(query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*Subscriber
	for rows.Next() {
		var sub Subscriber
		var email, telegram, slack, pushSub sql.NullString
		if err := rows.Scan(&sub.ID, &email, &telegram, &slack, &pushSub, &sub.CreatedAt, &sub.UpdatedAt); err != nil {
			return nil, err
		}
		if email.Valid {
			sub.Email = email.String
		}
		if telegram.Valid {
			sub.TelegramChatID = telegram.String
		}
		if slack.Valid {
			sub.SlackWebhookURL = slack.String
		}
		if pushSub.Valid && pushSub.String != "" {
			var wps WebPushSubscription
			if err := json.Unmarshal([]byte(pushSub.String), &wps); err == nil {
				sub.WebPushSubscription = &wps
			}
		}
		list = append(list, &sub)
	}
	return list, rows.Err()
}

func (s *Store) DeleteSubscriber(id string) error {
	query := `DELETE FROM subscribers WHERE id = ?`
	_, err := s.db.Exec(query, id)
	return err
}

// ==================== PREFERENCES ====================

func (s *Store) SetPreference(subscriberID, channel string, enabled bool) error {
	query := `INSERT INTO preferences (subscriber_id, channel, enabled)
	          VALUES (?, ?, ?)
	          ON CONFLICT(subscriber_id, channel) DO UPDATE SET enabled = excluded.enabled`
	_, err := s.db.Exec(query, subscriberID, channel, enabled)
	return err
}

func (s *Store) GetPreferences(subscriberID string) (map[string]bool, error) {
	query := `SELECT channel, enabled FROM preferences WHERE subscriber_id = ?`
	rows, err := s.db.Query(query, subscriberID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	prefs := make(map[string]bool)
	for rows.Next() {
		var ch string
		var enabled bool
		if err := rows.Scan(&ch, &enabled); err != nil {
			return nil, err
		}
		prefs[ch] = enabled
	}
	return prefs, rows.Err()
}

func (s *Store) IsChannelEnabled(subscriberID, channel string) (bool, error) {
	query := `SELECT enabled FROM preferences WHERE subscriber_id = ? AND channel = ?`
	row := s.db.QueryRow(query, subscriberID, channel)
	var enabled bool
	err := row.Scan(&enabled)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// By default, if no preference is explicitly set to false, it is enabled
			return true, nil
		}
		return false, err
	}
	return enabled, nil
}

// ==================== NOTIFICATIONS (IN-APP FEED) ====================

func (s *Store) CreateNotification(n *Notification) error {
	if n.ID == "" {
		n.ID = uuid.NewString()
	}
	n.CreatedAt = time.Now().UTC()

	var dataJSON sql.NullString
	if n.Data != nil {
		b, err := json.Marshal(n.Data)
		if err != nil {
			return err
		}
		dataJSON = sql.NullString{String: string(b), Valid: true}
	}

	query := `INSERT INTO notifications (id, workflow_id, subscriber_id, title, body, data, action_url, is_read, read_at, created_at)
	          VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := s.db.Exec(query, n.ID, n.WorkflowID, n.SubscriberID, n.Title, n.Body, dataJSON, n.ActionURL, n.IsRead, n.ReadAt, n.CreatedAt)
	return err
}

func (s *Store) GetNotification(id string) (*Notification, error) {
	query := `SELECT id, workflow_id, subscriber_id, title, body, data, action_url, is_read, read_at, created_at 
	          FROM notifications WHERE id = ?`
	row := s.db.QueryRow(query, id)

	var n Notification
	var dataRaw, actionURL sql.NullString
	var readAt sql.NullTime

	err := row.Scan(&n.ID, &n.WorkflowID, &n.SubscriberID, &n.Title, &n.Body, &dataRaw, &actionURL, &n.IsRead, &readAt, &n.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	if dataRaw.Valid && dataRaw.String != "" {
		_ = json.Unmarshal([]byte(dataRaw.String), &n.Data)
	}
	if actionURL.Valid {
		n.ActionURL = actionURL.String
	}
	if readAt.Valid {
		n.ReadAt = &readAt.Time
	}
	return &n, nil
}

func (s *Store) ListNotifications(subscriberID string, limit, offset int, unreadOnly bool) ([]*Notification, error) {
	if limit <= 0 {
		limit = 20
	}

	var query string
	var args []any

	if unreadOnly {
		query = `SELECT id, workflow_id, subscriber_id, title, body, data, action_url, is_read, read_at, created_at 
		         FROM notifications 
		         WHERE subscriber_id = ? AND is_read = 0 
		         ORDER BY created_at DESC LIMIT ? OFFSET ?`
		args = []any{subscriberID, limit, offset}
	} else {
		query = `SELECT id, workflow_id, subscriber_id, title, body, data, action_url, is_read, read_at, created_at 
		         FROM notifications 
		         WHERE subscriber_id = ? 
		         ORDER BY created_at DESC LIMIT ? OFFSET ?`
		args = []any{subscriberID, limit, offset}
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*Notification
	for rows.Next() {
		var n Notification
		var dataRaw, actionURL sql.NullString
		var readAt sql.NullTime

		if err := rows.Scan(&n.ID, &n.WorkflowID, &n.SubscriberID, &n.Title, &n.Body, &dataRaw, &actionURL, &n.IsRead, &readAt, &n.CreatedAt); err != nil {
			return nil, err
		}

		if dataRaw.Valid && dataRaw.String != "" {
			_ = json.Unmarshal([]byte(dataRaw.String), &n.Data)
		}
		if actionURL.Valid {
			n.ActionURL = actionURL.String
		}
		if readAt.Valid {
			n.ReadAt = &readAt.Time
		}
		list = append(list, &n)
	}
	return list, rows.Err()
}

func (s *Store) GetUnreadCount(subscriberID string) (int, error) {
	query := `SELECT COUNT(*) FROM notifications WHERE subscriber_id = ? AND is_read = 0`
	var count int
	err := s.db.QueryRow(query, subscriberID).Scan(&count)
	return count, err
}

func (s *Store) MarkAsRead(id string) error {
	now := time.Now().UTC()
	query := `UPDATE notifications SET is_read = 1, read_at = ? WHERE id = ?`
	res, err := s.db.Exec(query, now, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("notification %q not found", id)
	}
	return nil
}

func (s *Store) MarkAsUnread(id string) error {
	query := `UPDATE notifications SET is_read = 0, read_at = NULL WHERE id = ?`
	res, err := s.db.Exec(query, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("notification %q not found", id)
	}
	return nil
}

func (s *Store) MarkAllAsRead(subscriberID string) error {
	now := time.Now().UTC()
	query := `UPDATE notifications SET is_read = 1, read_at = ? WHERE subscriber_id = ? AND is_read = 0`
	_, err := s.db.Exec(query, now, subscriberID)
	return err
}

func (s *Store) DeleteNotification(id string) error {
	query := `DELETE FROM notifications WHERE id = ?`
	res, err := s.db.Exec(query, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("notification %q not found", id)
	}
	return nil
}

// ==================== DELIVERY LOGS ====================

func (s *Store) RecordDeliveryLog(log *DeliveryLog) error {
	if log.ID == "" {
		log.ID = uuid.NewString()
	}
	log.DispatchedAt = time.Now().UTC()

	query := `INSERT INTO delivery_logs (id, notification_id, workflow_id, subscriber_id, channel, status, error_message, payload, dispatched_at)
	          VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := s.db.Exec(query, log.ID, log.NotificationID, log.WorkflowID, log.SubscriberID, log.Channel, log.Status, log.ErrorMessage, log.Payload, log.DispatchedAt)
	return err
}

func (s *Store) ListDeliveryLogs(limit, offset int, filterChannel, filterStatus string) ([]*DeliveryLog, error) {
	if limit <= 0 {
		limit = 50
	}

	query := `SELECT id, notification_id, workflow_id, subscriber_id, channel, status, error_message, payload, dispatched_at 
	          FROM delivery_logs WHERE 1=1`
	var args []any

	if filterChannel != "" {
		query += " AND channel = ?"
		args = append(args, filterChannel)
	}
	if filterStatus != "" {
		query += " AND status = ?"
		args = append(args, filterStatus)
	}

	query += " ORDER BY dispatched_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*DeliveryLog
	for rows.Next() {
		var l DeliveryLog
		var notifID, errMsg, payload sql.NullString
		if err := rows.Scan(&l.ID, &notifID, &l.WorkflowID, &l.SubscriberID, &l.Channel, &l.Status, &errMsg, &payload, &l.DispatchedAt); err != nil {
			return nil, err
		}
		if notifID.Valid {
			l.NotificationID = notifID.String
		}
		if errMsg.Valid {
			l.ErrorMessage = errMsg.String
		}
		if payload.Valid {
			l.Payload = payload.String
		}
		list = append(list, &l)
	}
	return list, rows.Err()
}

// ==================== IDEMPOTENCY KEYS ====================

func (s *Store) GetIdempotencyKey(key string) (*IdempotencyRecord, error) {
	query := `SELECT key, response_body, status_code, created_at FROM idempotency_keys WHERE key = ?`
	row := s.db.QueryRow(query, key)

	var rec IdempotencyRecord
	err := row.Scan(&rec.Key, &rec.ResponseBody, &rec.StatusCode, &rec.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &rec, nil
}

func (s *Store) SaveIdempotencyKey(rec *IdempotencyRecord) error {
	rec.CreatedAt = time.Now().UTC()
	query := `INSERT INTO idempotency_keys (key, response_body, status_code, created_at)
	          VALUES (?, ?, ?, ?)
	          ON CONFLICT(key) DO UPDATE SET response_body = excluded.response_body, status_code = excluded.status_code`
	_, err := s.db.Exec(query, rec.Key, rec.ResponseBody, rec.StatusCode, rec.CreatedAt)
	return err
}
