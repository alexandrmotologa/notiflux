package store

import (
	"path/filepath"
	"testing"
)

func setupTestStore(t *testing.T) *Store {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_notiflux.db")
	st, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("failed to create test store: %v", err)
	}
	t.Cleanup(func() {
		_ = st.Close()
	})
	return st
}

func TestStore_WorkflowsAndTemplates(t *testing.T) {
	st := setupTestStore(t)

	// Create workflow
	wf := &Workflow{
		ID:          "order.created",
		Name:        "Order Created",
		Description: "Dispatched upon checkout completion",
		Channels:    []string{ChannelInApp, ChannelEmail},
	}
	if err := st.CreateWorkflow(wf); err != nil {
		t.Fatalf("failed to create workflow: %v", err)
	}

	// Fetch workflow
	fetched, err := st.GetWorkflow("order.created")
	if err != nil {
		t.Fatalf("failed to get workflow: %v", err)
	}
	if fetched == nil || fetched.Name != "Order Created" {
		t.Errorf("expected workflow 'Order Created', got %v", fetched)
	}
	if len(fetched.Channels) != 2 || fetched.Channels[0] != ChannelInApp {
		t.Errorf("unexpected channels: %v", fetched.Channels)
	}

	// Upsert template
	tpl := &Template{
		WorkflowID: "order.created",
		Channel:    ChannelInApp,
		Subject:    "Order #{{order_id}} Confirmed",
		Body:       "Thanks {{customer_name}}, your order of {{total}} was received.",
	}
	if err := st.UpsertTemplate(tpl); err != nil {
		t.Fatalf("failed to upsert template: %v", err)
	}

	// Fetch template
	fetchedTpl, err := st.GetTemplateByWorkflowAndChannel("order.created", ChannelInApp)
	if err != nil {
		t.Fatalf("failed to get template: %v", err)
	}
	if fetchedTpl == nil || fetchedTpl.Subject != "Order #{{order_id}} Confirmed" {
		t.Errorf("unexpected template: %v", fetchedTpl)
	}

	// List templates
	tpls, err := st.GetTemplatesByWorkflow("order.created")
	if err != nil || len(tpls) != 1 {
		t.Errorf("expected 1 template, got %d (err: %v)", len(tpls), err)
	}
}

func TestStore_SubscribersAndPreferences(t *testing.T) {
	st := setupTestStore(t)

	sub := &Subscriber{
		ID:             "usr_42",
		Email:          "dev@example.com",
		TelegramChatID: "123456",
		WebPushSubscription: &WebPushSubscription{
			Endpoint: "https://push.example.com/test",
			Keys: WebPushKeys{
				P256dh: "key-p256",
				Auth:   "key-auth",
			},
		},
	}

	if err := st.UpsertSubscriber(sub); err != nil {
		t.Fatalf("failed to upsert subscriber: %v", err)
	}

	fetched, err := st.GetSubscriber("usr_42")
	if err != nil {
		t.Fatalf("failed to get subscriber: %v", err)
	}
	if fetched == nil || fetched.Email != "dev@example.com" {
		t.Errorf("expected subscriber email dev@example.com, got %v", fetched)
	}
	if fetched.WebPushSubscription == nil || fetched.WebPushSubscription.Endpoint != "https://push.example.com/test" {
		t.Errorf("web push subscription not properly deserialized: %v", fetched.WebPushSubscription)
	}

	// Default preference check (should default to true)
	enabled, err := st.IsChannelEnabled("usr_42", ChannelEmail)
	if err != nil || !enabled {
		t.Errorf("expected default email enabled=true, got %v (err: %v)", enabled, err)
	}

	// Opt-out from email
	if err := st.SetPreference("usr_42", ChannelEmail, false); err != nil {
		t.Fatalf("failed to set preference: %v", err)
	}

	enabledAfter, err := st.IsChannelEnabled("usr_42", ChannelEmail)
	if err != nil || enabledAfter {
		t.Errorf("expected email enabled=false after opt-out, got %v", enabledAfter)
	}

	prefs, err := st.GetPreferences("usr_42")
	if err != nil || prefs[ChannelEmail] != false {
		t.Errorf("expected prefs[email]==false, got %v", prefs)
	}
}

func TestStore_NotificationsFeed(t *testing.T) {
	st := setupTestStore(t)

	// Ensure subscriber exists for FK constraint
	_ = st.UpsertSubscriber(&Subscriber{ID: "usr_feed"})

	// Create 3 notifications
	for i := 1; i <= 3; i++ {
		n := &Notification{
			ID:           "",
			WorkflowID:   "order.status",
			SubscriberID: "usr_feed",
			Title:        "Notification Title",
			Body:         "Notification Body Content",
			ActionURL:    "https://example.com",
			Data:         map[string]any{"index": i},
		}
		if err := st.CreateNotification(n); err != nil {
			t.Fatalf("failed to create notification %d: %v", i, err)
		}
	}

	unreadCount, err := st.GetUnreadCount("usr_feed")
	if err != nil || unreadCount != 3 {
		t.Errorf("expected unread count 3, got %d (err: %v)", unreadCount, err)
	}

	list, err := st.ListNotifications("usr_feed", 10, 0, false)
	if err != nil || len(list) != 3 {
		t.Fatalf("expected 3 notifications, got %d (err: %v)", len(list), err)
	}

	// Mark first as read
	if err := st.MarkAsRead(list[0].ID); err != nil {
		t.Fatalf("failed to mark notification as read: %v", err)
	}

	unreadCountAfter, _ := st.GetUnreadCount("usr_feed")
	if unreadCountAfter != 2 {
		t.Errorf("expected unread count 2, got %d", unreadCountAfter)
	}

	// Mark all as read
	if err := st.MarkAllAsRead("usr_feed"); err != nil {
		t.Fatalf("failed to mark all as read: %v", err)
	}

	unreadCountFinal, _ := st.GetUnreadCount("usr_feed")
	if unreadCountFinal != 0 {
		t.Errorf("expected unread count 0, got %d", unreadCountFinal)
	}
}

func TestStore_IdempotencyAndDeliveryLogs(t *testing.T) {
	st := setupTestStore(t)

	// Test Idempotency
	rec := &IdempotencyRecord{
		Key:          "key_12345",
		StatusCode:   200,
		ResponseBody: `{"status":"ok"}`,
	}
	if err := st.SaveIdempotencyKey(rec); err != nil {
		t.Fatalf("failed to save idempotency key: %v", err)
	}

	fetchedRec, err := st.GetIdempotencyKey("key_12345")
	if err != nil || fetchedRec == nil {
		t.Fatalf("failed to get idempotency key: %v", err)
	}
	if fetchedRec.StatusCode != 200 || fetchedRec.ResponseBody != `{"status":"ok"}` {
		t.Errorf("unexpected idempotency record: %v", fetchedRec)
	}

	// Non-existent key
	missing, err := st.GetIdempotencyKey("non_existent")
	if err != nil || missing != nil {
		t.Errorf("expected nil for missing key, got %v (err: %v)", missing, err)
	}

	// Test Delivery Logs
	log := &DeliveryLog{
		WorkflowID:   "welcome",
		SubscriberID: "usr_99",
		Channel:      ChannelEmail,
		Status:       StatusSent,
		Payload:      `{"to":"usr_99@example.com"}`,
	}
	if err := st.RecordDeliveryLog(log); err != nil {
		t.Fatalf("failed to record delivery log: %v", err)
	}

	logs, err := st.ListDeliveryLogs(10, 0, ChannelEmail, StatusSent)
	if err != nil || len(logs) != 1 {
		t.Fatalf("expected 1 delivery log, got %d (err: %v)", len(logs), err)
	}
	if logs[0].SubscriberID != "usr_99" {
		t.Errorf("expected subscriber_id usr_99, got %s", logs[0].SubscriberID)
	}
}
