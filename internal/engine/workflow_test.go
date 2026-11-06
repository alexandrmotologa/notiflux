package engine

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/alexandrmotologa/notiflux/internal/store"
)

func setupTestEngine(t *testing.T) (*WorkflowEngine, *store.Store) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "engine_test.db")
	st, err := store.NewStore(dbPath)
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	t.Cleanup(func() {
		_ = st.Close()
	})
	return NewWorkflowEngine(st), st
}

func TestWorkflowEngine_PrepareDispatch(t *testing.T) {
	engine, st := setupTestEngine(t)

	// Create workflow
	wf := &store.Workflow{
		ID:       "invoice.paid",
		Name:     "Invoice Paid",
		Channels: []string{store.ChannelInApp, store.ChannelEmail, store.ChannelTelegram},
	}
	if err := st.CreateWorkflow(wf); err != nil {
		t.Fatalf("failed to create workflow: %v", err)
	}

	// Create templates
	tplInApp := &store.Template{
		WorkflowID: "invoice.paid",
		Channel:    store.ChannelInApp,
		Subject:    "Invoice #{{inv_num}} Received",
		Body:       "Amount of ${{amount}} was paid successfully.",
	}
	_ = st.UpsertTemplate(tplInApp)

	tplEmail := &store.Template{
		WorkflowID: "invoice.paid",
		Channel:    store.ChannelEmail,
		Subject:    "Receipt for Invoice #{{inv_num}}",
		Body:       "# Payment Confirmation\n\nYour payment of **${{amount}}** has been cleared.",
	}
	_ = st.UpsertTemplate(tplEmail)

	// Trigger request with email but missing telegram_chat_id
	req := &TriggerRequest{
		Workflow: "invoice.paid",
		Subscriber: store.Subscriber{
			ID:    "usr_100",
			Email: "alex@example.com",
			// No Telegram ID
		},
		Payload: map[string]any{
			"inv_num": "INV-9901",
			"amount":  "49.99",
		},
	}

	messages, notifID, skipped, err := engine.PrepareDispatch(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if notifID == "" {
		t.Errorf("expected non-empty notification ID")
	}

	// Telegram should be skipped due to missing credentials
	if skipped[store.ChannelTelegram] != "missing_telegram_chat_id" {
		t.Errorf("expected telegram to be skipped for missing credentials, got %v", skipped)
	}

	// Should have 2 compiled messages: in_app and email
	if len(messages) != 2 {
		t.Fatalf("expected 2 compiled messages, got %d", len(messages))
	}

	var inAppMsg, emailMsg *CompiledMessage
	for _, m := range messages {
		if m.Channel == store.ChannelInApp {
			inAppMsg = m
		} else if m.Channel == store.ChannelEmail {
			emailMsg = m
		}
	}

	if inAppMsg == nil || inAppMsg.Subject != "Invoice #INV-9901 Received" {
		t.Errorf("unexpected in-app message: %v", inAppMsg)
	}
	if inAppMsg.Body != "Amount of $49.99 was paid successfully." {
		t.Errorf("unexpected in-app body: %s", inAppMsg.Body)
	}

	if emailMsg == nil || emailMsg.Subject != "Receipt for Invoice #INV-9901" {
		t.Errorf("unexpected email subject: %v", emailMsg)
	}
	if emailMsg.HTMLBody == "" {
		t.Errorf("expected non-empty HTML body for email")
	}
}

func TestWorkflowEngine_UserOptOut(t *testing.T) {
	engine, st := setupTestEngine(t)

	wf := &store.Workflow{
		ID:       "newsletter",
		Name:     "Weekly Newsletter",
		Channels: []string{store.ChannelEmail},
	}
	_ = st.CreateWorkflow(wf)

	// Create subscriber and set preference email = false
	_ = st.UpsertSubscriber(&store.Subscriber{ID: "usr_optout", Email: "optout@example.com"})
	_ = st.SetPreference("usr_optout", store.ChannelEmail, false)

	req := &TriggerRequest{
		Workflow: "newsletter",
		Subscriber: store.Subscriber{
			ID:    "usr_optout",
			Email: "optout@example.com",
		},
		Payload: map[string]any{"edition": "42"},
	}

	messages, _, skipped, err := engine.PrepareDispatch(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(messages) != 0 {
		t.Errorf("expected 0 messages due to opt out, got %d", len(messages))
	}
	if skipped[store.ChannelEmail] != "user_opt_out" {
		t.Errorf("expected user_opt_out reason, got %v", skipped)
	}
}

func TestWorkflowEngine_QuietHours(t *testing.T) {
	sub := &store.Subscriber{
		ID:              "usr_night_owl",
		Timezone:        "UTC",
		QuietHoursStart: "22:00",
		QuietHoursEnd:   "08:00",
	}

	// 23:30 UTC -> inside quiet hours
	t1 := time.Date(2026, 11, 7, 23, 30, 0, 0, time.UTC)
	inQuiet1, release1 := IsInQuietHours(sub, t1)
	if !inQuiet1 {
		t.Errorf("expected 23:30 to be within quiet hours")
	}
	if release1.Before(t1) {
		t.Errorf("expected release time to be in future, got %v", release1)
	}

	// 04:15 UTC -> inside quiet hours
	t2 := time.Date(2026, 11, 7, 4, 15, 0, 0, time.UTC)
	inQuiet2, _ := IsInQuietHours(sub, t2)
	if !inQuiet2 {
		t.Errorf("expected 04:15 to be within quiet hours")
	}

	// 14:00 UTC -> daytime, outside quiet hours
	t3 := time.Date(2026, 11, 7, 14, 0, 0, 0, time.UTC)
	inQuiet3, _ := IsInQuietHours(sub, t3)
	if inQuiet3 {
		t.Errorf("expected 14:00 to be outside quiet hours")
	}
}

func TestWorkflowEngine_StepDelays(t *testing.T) {
	engine, st := setupTestEngine(t)

	wf := &store.Workflow{
		ID:       "security.verify",
		Name:     "Security Verification",
		Channels: []string{store.ChannelInApp, store.ChannelEmail},
		Steps: []store.WorkflowStep{
			{Channel: store.ChannelInApp, DelaySeconds: 0},
			{Channel: store.ChannelEmail, DelaySeconds: 600, Condition: "if_unread"},
		},
	}
	_ = st.CreateWorkflow(wf)

	req := &TriggerRequest{
		Workflow: "security.verify",
		Subscriber: store.Subscriber{
			ID:    "usr_step_test",
			Email: "step@example.com",
		},
		Payload: map[string]any{
			"title": "Suspicious login attempt",
			"body":  "Please verify this login from your dashboard.",
		},
	}

	messages, notifID, skipped, err := engine.PrepareDispatch(req)
	if err != nil {
		t.Fatalf("prepare dispatch failed: %v", err)
	}

	if notifID == "" {
		t.Errorf("expected non-empty notification ID")
	}

	// Immediate step should only be in_app
	if len(messages) != 1 || messages[0].Channel != store.ChannelInApp {
		t.Fatalf("expected 1 immediate in_app message, got %d", len(messages))
	}

	if !strings.HasPrefix(skipped[store.ChannelEmail], "scheduled_for_") {
		t.Errorf("expected email to be scheduled as delayed step, got %s", skipped[store.ChannelEmail])
	}

	// Check pending scheduled deliveries
	pending, err := st.ListPendingScheduledDeliveries(time.Now().Add(1*time.Hour), 10)
	if err != nil || len(pending) != 1 {
		t.Fatalf("expected 1 pending scheduled delivery for email step, got %d (err: %v)", len(pending), err)
	}
	if pending[0].Channel != store.ChannelEmail || pending[0].Condition != "if_unread" {
		t.Errorf("unexpected scheduled delivery: %+v", pending[0])
	}
}

func TestWorkflowEngine_DigestBuffering(t *testing.T) {
	engine, st := setupTestEngine(t)

	wf := &store.Workflow{
		ID:                  "activity.digest",
		Name:                "Activity Digest",
		Channels:            []string{store.ChannelEmail},
		DigestEnabled:       true,
		DigestWindowSeconds: 300,
	}
	_ = st.CreateWorkflow(wf)

	req := &TriggerRequest{
		Workflow: "activity.digest",
		Subscriber: store.Subscriber{
			ID:    "usr_digest_test",
			Email: "digest@example.com",
		},
		Payload: map[string]any{
			"title": "Comment on task #10",
			"body":  "Alexandr commented on your issue",
		},
	}

	messages, _, skipped, err := engine.PrepareDispatch(req)
	if err != nil {
		t.Fatalf("prepare dispatch failed: %v", err)
	}

	if len(messages) != 0 {
		t.Errorf("expected 0 immediate messages for digest workflow, got %d", len(messages))
	}
	if skipped["digest"] != "buffered_in_window" {
		t.Errorf("expected buffered_in_window, got %v", skipped)
	}

	entries, err := st.GetDigestEntries("activity.digest", "usr_digest_test")
	if err != nil || len(entries) != 1 {
		t.Fatalf("expected 1 buffered digest entry, got %d (err: %v)", len(entries), err)
	}
}

