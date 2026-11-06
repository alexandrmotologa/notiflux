package engine

import (
	"path/filepath"
	"testing"

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
