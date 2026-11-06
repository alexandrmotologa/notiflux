package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
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

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)

	if err := Migrate(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed running sqlite migrations: %w", err)
	}

	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

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

	var stepsJSON sql.NullString
	if len(wf.Steps) > 0 {
		b, _ := json.Marshal(wf.Steps)
		stepsJSON = sql.NullString{String: string(b), Valid: true}
	}

	query := `INSERT INTO workflows (id, name, description, channels, steps, digest_enabled, digest_window_seconds, digest_max_items, created_at, updated_at) 
	          VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	_, err = s.db.Exec(query, wf.ID, wf.Name, wf.Description, string(channelsJSON), stepsJSON, wf.DigestEnabled, wf.DigestWindowSeconds, wf.DigestMaxItems, wf.CreatedAt, wf.UpdatedAt)
	return err
}

func (s *Store) GetWorkflow(id string) (*Workflow, error) {
	query := `SELECT id, name, description, channels, steps, digest_enabled, digest_window_seconds, digest_max_items, created_at, updated_at FROM workflows WHERE id = ?`
	row := s.db.QueryRow(query, id)

	var wf Workflow
	var channelsRaw, stepsRaw sql.NullString
	err := row.Scan(&wf.ID, &wf.Name, &wf.Description, &channelsRaw, &stepsRaw, &wf.DigestEnabled, &wf.DigestWindowSeconds, &wf.DigestMaxItems, &wf.CreatedAt, &wf.UpdatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}

	if channelsRaw.Valid && channelsRaw.String != "" {
		_ = json.Unmarshal([]byte(channelsRaw.String), &wf.Channels)
	} else {
		wf.Channels = []string{}
	}

	if stepsRaw.Valid && stepsRaw.String != "" {
		_ = json.Unmarshal([]byte(stepsRaw.String), &wf.Steps)
	}

	return &wf, nil
}

func (s *Store) ListWorkflows() ([]*Workflow, error) {
	query := `SELECT id, name, description, channels, steps, digest_enabled, digest_window_seconds, digest_max_items, created_at, updated_at FROM workflows ORDER BY created_at ASC`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*Workflow
	for rows.Next() {
		var wf Workflow
		var channelsRaw, stepsRaw sql.NullString
		if err := rows.Scan(&wf.ID, &wf.Name, &wf.Description, &channelsRaw, &stepsRaw, &wf.DigestEnabled, &wf.DigestWindowSeconds, &wf.DigestMaxItems, &wf.CreatedAt, &wf.UpdatedAt); err != nil {
			return nil, err
		}
		if channelsRaw.Valid && channelsRaw.String != "" {
			_ = json.Unmarshal([]byte(channelsRaw.String), &wf.Channels)
		}
		if stepsRaw.Valid && stepsRaw.String != "" {
			_ = json.Unmarshal([]byte(stepsRaw.String), &wf.Steps)
		}
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

	var stepsJSON sql.NullString
	if len(wf.Steps) > 0 {
		b, _ := json.Marshal(wf.Steps)
		stepsJSON = sql.NullString{String: string(b), Valid: true}
	}

	query := `UPDATE workflows SET name = ?, description = ?, channels = ?, steps = ?, digest_enabled = ?, digest_window_seconds = ?, digest_max_items = ?, updated_at = ? WHERE id = ?`
	res, err := s.db.Exec(query, wf.Name, wf.Description, string(channelsJSON), stepsJSON, wf.DigestEnabled, wf.DigestWindowSeconds, wf.DigestMaxItems, wf.UpdatedAt, wf.ID)
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

	query := `INSERT INTO subscribers (id, email, telegram_chat_id, slack_webhook_url, discord_webhook_url, web_push_subscription, timezone, quiet_hours_start, quiet_hours_end, created_at, updated_at)
	          VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	          ON CONFLICT(id) DO UPDATE SET
	              email = COALESCE(excluded.email, subscribers.email),
	              telegram_chat_id = COALESCE(excluded.telegram_chat_id, subscribers.telegram_chat_id),
	              slack_webhook_url = COALESCE(excluded.slack_webhook_url, subscribers.slack_webhook_url),
	              discord_webhook_url = COALESCE(excluded.discord_webhook_url, subscribers.discord_webhook_url),
	              web_push_subscription = COALESCE(excluded.web_push_subscription, subscribers.web_push_subscription),
	              timezone = COALESCE(excluded.timezone, subscribers.timezone),
	              quiet_hours_start = COALESCE(excluded.quiet_hours_start, subscribers.quiet_hours_start),
	              quiet_hours_end = COALESCE(excluded.quiet_hours_end, subscribers.quiet_hours_end),
	              updated_at = excluded.updated_at`

	_, err := s.db.Exec(query, sub.ID, sub.Email, sub.TelegramChatID, sub.SlackWebhookURL, sub.DiscordWebhookURL, webPushJSON, sub.Timezone, sub.QuietHoursStart, sub.QuietHoursEnd, now, now)
	return err
}

func (s *Store) GetSubscriber(id string) (*Subscriber, error) {
	query := `SELECT id, email, telegram_chat_id, slack_webhook_url, discord_webhook_url, web_push_subscription, timezone, quiet_hours_start, quiet_hours_end, created_at, updated_at 
	          FROM subscribers WHERE id = ?`
	row := s.db.QueryRow(query, id)

	var sub Subscriber
	var email, telegram, slack, discord, pushSub, tz, qStart, qEnd sql.NullString
	err := row.Scan(&sub.ID, &email, &telegram, &slack, &discord, &pushSub, &tz, &qStart, &qEnd, &sub.CreatedAt, &sub.UpdatedAt)
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
	if discord.Valid {
		sub.DiscordWebhookURL = discord.String
	}
	if tz.Valid {
		sub.Timezone = tz.String
	}
	if qStart.Valid {
		sub.QuietHoursStart = qStart.String
	}
	if qEnd.Valid {
		sub.QuietHoursEnd = qEnd.String
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
	query := `SELECT id, email, telegram_chat_id, slack_webhook_url, discord_webhook_url, web_push_subscription, timezone, quiet_hours_start, quiet_hours_end, created_at, updated_at 
	          FROM subscribers ORDER BY updated_at DESC LIMIT ? OFFSET ?`
	rows, err := s.db.Query(query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*Subscriber
	for rows.Next() {
		var sub Subscriber
		var email, telegram, slack, discord, pushSub, tz, qStart, qEnd sql.NullString
		if err := rows.Scan(&sub.ID, &email, &telegram, &slack, &discord, &pushSub, &tz, &qStart, &qEnd, &sub.CreatedAt, &sub.UpdatedAt); err != nil {
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
		if discord.Valid {
			sub.DiscordWebhookURL = discord.String
		}
		if tz.Valid {
			sub.Timezone = tz.String
		}
		if qStart.Valid {
			sub.QuietHoursStart = qStart.String
		}
		if qEnd.Valid {
			sub.QuietHoursEnd = qEnd.String
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
			return true, nil
		}
		return false, err
	}
	return enabled, nil
}

// ==================== NOTIFICATIONS ====================

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

// ==================== DELIVERY LOGS & TRACKING ====================

func (s *Store) RecordDeliveryLog(log *DeliveryLog) error {
	if log.ID == "" {
		log.ID = uuid.NewString()
	}
	log.DispatchedAt = time.Now().UTC()

	var trackingID, targetURL sql.NullString
	if log.TrackingID != "" {
		trackingID = sql.NullString{String: log.TrackingID, Valid: true}
	}
	if log.TargetURL != "" {
		targetURL = sql.NullString{String: log.TargetURL, Valid: true}
	}

	query := `INSERT INTO delivery_logs (id, notification_id, workflow_id, subscriber_id, channel, status, error_message, payload, tracking_id, target_url, dispatched_at)
	          VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := s.db.Exec(query, log.ID, log.NotificationID, log.WorkflowID, log.SubscriberID, log.Channel, log.Status, log.ErrorMessage, log.Payload, trackingID, targetURL, log.DispatchedAt)
	return err
}

func (s *Store) RecordTrackingClick(trackingID string) (string, error) {
	now := time.Now().UTC()
	query := `UPDATE delivery_logs SET clicked_at = COALESCE(clicked_at, ?) WHERE tracking_id = ?`
	_, err := s.db.Exec(query, now, trackingID)
	if err != nil {
		return "", err
	}

	var targetURL sql.NullString
	querySelect := `SELECT target_url FROM delivery_logs WHERE tracking_id = ?`
	err = s.db.QueryRow(querySelect, trackingID).Scan(&targetURL)
	if err != nil {
		return "", err
	}

	if targetURL.Valid {
		return targetURL.String, nil
	}
	return "/", nil
}

func (s *Store) RecordTrackingOpen(trackingID string) error {
	now := time.Now().UTC()
	query := `UPDATE delivery_logs SET opened_at = COALESCE(opened_at, ?) WHERE tracking_id = ?`
	_, err := s.db.Exec(query, now, trackingID)
	return err
}

func (s *Store) ListDeliveryLogs(limit, offset int, filterChannel, filterStatus string) ([]*DeliveryLog, error) {
	if limit <= 0 {
		limit = 50
	}

	query := `SELECT id, notification_id, workflow_id, subscriber_id, channel, status, error_message, payload, tracking_id, target_url, opened_at, clicked_at, dispatched_at 
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
		var notifID, errMsg, payload, trackingID, targetURL sql.NullString
		var openedAt, clickedAt sql.NullTime

		if err := rows.Scan(&l.ID, &notifID, &l.WorkflowID, &l.SubscriberID, &l.Channel, &l.Status, &errMsg, &payload, &trackingID, &targetURL, &openedAt, &clickedAt, &l.DispatchedAt); err != nil {
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
		if trackingID.Valid {
			l.TrackingID = trackingID.String
		}
		if targetURL.Valid {
			l.TargetURL = targetURL.String
		}
		if openedAt.Valid {
			l.OpenedAt = &openedAt.Time
		}
		if clickedAt.Valid {
			l.ClickedAt = &clickedAt.Time
		}
		list = append(list, &l)
	}
	return list, rows.Err()
}

// ==================== SCHEDULED DELIVERIES & RETRIES ====================

func (s *Store) CreateScheduledDelivery(sd *ScheduledDelivery) error {
	if sd.ID == "" {
		sd.ID = uuid.NewString()
	}
	sd.CreatedAt = time.Now().UTC()
	if sd.Status == "" {
		sd.Status = "pending"
	}
	if sd.MaxAttempts <= 0 {
		sd.MaxAttempts = 3
	}

	query := `INSERT INTO scheduled_deliveries (id, notification_id, workflow_id, subscriber_id, channel, subject, body, html_body, action_url, payload, scheduled_for, attempts, max_attempts, condition, status, created_at)
	          VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	_, err := s.db.Exec(query, sd.ID, sd.NotificationID, sd.WorkflowID, sd.SubscriberID, sd.Channel, sd.Subject, sd.Body, sd.HTMLBody, sd.ActionURL, sd.Payload, sd.ScheduledFor, sd.Attempts, sd.MaxAttempts, sd.Condition, sd.Status, sd.CreatedAt)
	return err
}

func (s *Store) ListPendingScheduledDeliveries(before time.Time, limit int) ([]*ScheduledDelivery, error) {
	if limit <= 0 {
		limit = 50
	}
	query := `SELECT id, notification_id, workflow_id, subscriber_id, channel, subject, body, html_body, action_url, payload, scheduled_for, attempts, max_attempts, condition, status, created_at 
	          FROM scheduled_deliveries 
	          WHERE status = 'pending' AND scheduled_for <= ? 
	          ORDER BY scheduled_for ASC LIMIT ?`
	rows, err := s.db.Query(query, before, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*ScheduledDelivery
	for rows.Next() {
		var sd ScheduledDelivery
		var notifID, htmlBody, actionURL, payload, condition sql.NullString
		if err := rows.Scan(&sd.ID, &notifID, &sd.WorkflowID, &sd.SubscriberID, &sd.Channel, &sd.Subject, &sd.Body, &htmlBody, &actionURL, &payload, &sd.ScheduledFor, &sd.Attempts, &sd.MaxAttempts, &condition, &sd.Status, &sd.CreatedAt); err != nil {
			return nil, err
		}
		if notifID.Valid {
			sd.NotificationID = notifID.String
		}
		if htmlBody.Valid {
			sd.HTMLBody = htmlBody.String
		}
		if actionURL.Valid {
			sd.ActionURL = actionURL.String
		}
		if payload.Valid {
			sd.Payload = payload.String
		}
		if condition.Valid {
			sd.Condition = condition.String
		}
		list = append(list, &sd)
	}
	return list, rows.Err()
}

func (s *Store) UpdateScheduledDeliveryStatus(id, status, errorMsg string, attempts int) error {
	query := `UPDATE scheduled_deliveries SET status = ?, attempts = ? WHERE id = ?`
	_, err := s.db.Exec(query, status, attempts, id)
	return err
}

func (s *Store) DeleteScheduledDelivery(id string) error {
	query := `DELETE FROM scheduled_deliveries WHERE id = ?`
	_, err := s.db.Exec(query, id)
	return err
}

// ==================== DIGEST ENTRIES ====================

func (s *Store) AddDigestEntry(entry *DigestEntry) error {
	if entry.ID == "" {
		entry.ID = uuid.NewString()
	}
	entry.CreatedAt = time.Now().UTC()

	query := `INSERT INTO digest_entries (id, workflow_id, subscriber_id, payload, created_at)
	          VALUES (?, ?, ?, ?, ?)`
	_, err := s.db.Exec(query, entry.ID, entry.WorkflowID, entry.SubscriberID, entry.Payload, entry.CreatedAt)
	return err
}

func (s *Store) GetDigestEntries(workflowID, subscriberID string) ([]*DigestEntry, error) {
	query := `SELECT id, workflow_id, subscriber_id, payload, created_at 
	          FROM digest_entries WHERE workflow_id = ? AND subscriber_id = ? ORDER BY created_at ASC`
	rows, err := s.db.Query(query, workflowID, subscriberID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*DigestEntry
	for rows.Next() {
		var de DigestEntry
		if err := rows.Scan(&de.ID, &de.WorkflowID, &de.SubscriberID, &de.Payload, &de.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, &de)
	}
	return list, rows.Err()
}

func (s *Store) ClearDigestEntries(workflowID, subscriberID string) error {
	query := `DELETE FROM digest_entries WHERE workflow_id = ? AND subscriber_id = ?`
	_, err := s.db.Exec(query, workflowID, subscriberID)
	return err
}

func (s *Store) CountDigestEntries(workflowID, subscriberID string) (int, error) {
	query := `SELECT COUNT(*) FROM digest_entries WHERE workflow_id = ? AND subscriber_id = ?`
	var count int
	err := s.db.QueryRow(query, workflowID, subscriberID).Scan(&count)
	return count, err
}

func (s *Store) ListPendingDigestSubscribers() ([]struct{ WorkflowID, SubscriberID string }, error) {
	query := `SELECT DISTINCT workflow_id, subscriber_id FROM digest_entries`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []struct{ WorkflowID, SubscriberID string }
	for rows.Next() {
		var item struct{ WorkflowID, SubscriberID string }
		if err := rows.Scan(&item.WorkflowID, &item.SubscriberID); err != nil {
			return nil, err
		}
		list = append(list, item)
	}
	return list, rows.Err()
}

// ==================== API KEYS (WORKSPACES) ====================

func (s *Store) CreateAPIKey(name string) (string, error) {
	key := "nf_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	now := time.Now().UTC()
	query := `INSERT INTO api_keys (key, name, created_at) VALUES (?, ?, ?)`
	_, err := s.db.Exec(query, key, name, now)
	return key, err
}

func (s *Store) ListAPIKeys() ([]*APIKeyRecord, error) {
	query := `SELECT key, name, created_at FROM api_keys ORDER BY created_at DESC`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []*APIKeyRecord
	for rows.Next() {
		var rec APIKeyRecord
		if err := rows.Scan(&rec.Key, &rec.Name, &rec.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, &rec)
	}
	return list, rows.Err()
}

func (s *Store) DeleteAPIKey(key string) error {
	query := `DELETE FROM api_keys WHERE key = ?`
	_, err := s.db.Exec(query, key)
	return err
}

func (s *Store) ValidateAPIKey(key string) (bool, error) {
	query := `SELECT COUNT(*) FROM api_keys WHERE key = ?`
	var count int
	err := s.db.QueryRow(query, key).Scan(&count)
	return count > 0, err
}

// ==================== ONLINE HOT BACKUP ====================

func (s *Store) HotBackup(destPath string) error {
	query := fmt.Sprintf("VACUUM INTO '%s'", destPath)
	_, err := s.db.Exec(query)
	return err
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
