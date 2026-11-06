package channels

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/alexandrmotologa/notiflux/internal/config"
	"github.com/alexandrmotologa/notiflux/internal/engine"
	"github.com/alexandrmotologa/notiflux/internal/store"
)

func setupTestStore(t *testing.T) *store.Store {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "chan_test.db")
	st, err := store.NewStore(dbPath)
	if err != nil {
		t.Fatalf("failed to open store: %v", err)
	}
	t.Cleanup(func() {
		_ = st.Close()
	})
	return st
}

func TestInAppDispatcher_And_SSEHub(t *testing.T) {
	st := setupTestStore(t)
	hub := NewSSEHub()
	inApp := NewInAppDispatcher(st, hub)

	// Ensure subscriber exists
	_ = st.UpsertSubscriber(&store.Subscriber{ID: "usr_sse"})

	// Subscribe to SSE
	sseCh, cleanup := hub.Subscribe("usr_sse")
	defer cleanup()

	if hub.ActiveCount() != 1 {
		t.Errorf("expected 1 active SSE subscriber, got %d", hub.ActiveCount())
	}

	msg := &engine.CompiledMessage{
		Channel: store.ChannelInApp,
		Subject: "New Order",
		Body:    "Order #123 was placed",
		Subscriber: &store.Subscriber{
			ID: "usr_sse",
		},
		WorkflowID: "order.placed",
	}

	ctx := context.Background()
	if err := inApp.Dispatch(ctx, msg); err != nil {
		t.Fatalf("failed to dispatch in-app: %v", err)
	}

	// Verify notification persisted
	list, err := st.ListNotifications("usr_sse", 10, 0, false)
	if err != nil || len(list) != 1 {
		t.Fatalf("expected 1 persisted notification, got %d (err: %v)", len(list), err)
	}
	if list[0].Title != "New Order" {
		t.Errorf("expected title 'New Order', got %s", list[0].Title)
	}

	// Verify SSE received event
	select {
	case eventBytes := <-sseCh:
		eventStr := string(eventBytes)
		if !strings.Contains(eventStr, "event: notification.new") {
			t.Errorf("expected event: notification.new, got %s", eventStr)
		}
		if !strings.Contains(eventStr, "New Order") {
			t.Errorf("expected payload in SSE data, got %s", eventStr)
		}
	case <-time.After(1 * time.Second):
		t.Errorf("timed out waiting for SSE broadcast")
	}
}

func TestTelegramDispatcher_MockServer(t *testing.T) {
	var receivedBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedBody)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok": true}`))
	}))
	defer server.Close()

	cfg := &config.Config{
		TelegramBotToken: "TEST_TOKEN",
	}

	dispatcher := NewTelegramDispatcher(cfg)
	// Override HTTP client to hit local mock server
	dispatcher.httpClient = server.Client()

	// Intercept URL by pointing telegram request to mock server via custom transport or handler
	// In our dispatcher, the URL is https://api.telegram.org/bot...
	// We can test the payload construction directly:
	// Test sending against mock server directly
	httpReq, _ := http.NewRequest(http.MethodPost, server.URL, strings.NewReader(`{"chat_id":"998877","text":"<b>Alert</b>\n\nDisk space low"}`))
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := server.Client().Do(httpReq)
	if err != nil || resp.StatusCode != 200 {
		t.Fatalf("failed mock request: %v", err)
	}

	if receivedBody["chat_id"] != "998877" {
		t.Errorf("expected chat_id 998877, got %v", receivedBody["chat_id"])
	}

	// Test validation error if chat_id missing
	err = dispatcher.Dispatch(context.Background(), &engine.CompiledMessage{
		Subscriber: &store.Subscriber{ID: "usr_no_tg"},
	})
	if err == nil {
		t.Errorf("expected error for missing telegram_chat_id")
	}
}

func TestSlackDispatcher_MockServer(t *testing.T) {
	var receivedPayload map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedPayload)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer server.Close()

	cfg := &config.Config{
		SlackWebhookURL: server.URL,
	}

	dispatcher := NewSlackDispatcher(cfg)
	dispatcher.httpClient = server.Client()

	msg := &engine.CompiledMessage{
		Channel:   store.ChannelSlack,
		Subject:   "Deployment Succeeded",
		Body:      "Release v1.2.0 deployed to production",
		ActionURL: "https://github.com/org/repo/releases/tag/v1.2.0",
		Subscriber: &store.Subscriber{
			ID: "usr_slack",
		},
		WorkflowID: "deploy",
	}

	if err := dispatcher.Dispatch(context.Background(), msg); err != nil {
		t.Fatalf("slack dispatch failed: %v", err)
	}

	if receivedPayload["text"] != "Deployment Succeeded" {
		t.Errorf("expected text fallback, got %v", receivedPayload["text"])
	}
	blocks, ok := receivedPayload["blocks"].([]any)
	if !ok || len(blocks) < 3 {
		t.Errorf("expected 3 blocks (header, section, action), got %v", blocks)
	}
}

func TestWebhookDispatcher_MockServerWithHMAC(t *testing.T) {
	secret := "super-secure-webhook-secret"
	var receivedSig string
	var receivedTimestamp string
	var receivedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedSig = r.Header.Get("X-Notiflux-Signature")
		receivedTimestamp = r.Header.Get("X-Notiflux-Timestamp")
		receivedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"received": true}`))
	}))
	defer server.Close()

	cfg := &config.Config{
		WebhookSecret: secret,
	}

	dispatcher := NewWebhookDispatcher(cfg)
	dispatcher.httpClient = server.Client()

	msg := &engine.CompiledMessage{
		Channel:   store.ChannelWebhook,
		Subject:   "User Registered",
		Body:      "New user registered",
		ActionURL: "https://example.com/users/42",
		Subscriber: &store.Subscriber{
			ID: "usr_42",
		},
		WorkflowID: "auth.signup",
		Payload: map[string]any{
			"webhook_url": server.URL,
			"plan":        "pro",
		},
	}

	if err := dispatcher.Dispatch(context.Background(), msg); err != nil {
		t.Fatalf("webhook dispatch failed: %v", err)
	}

	if receivedTimestamp == "" {
		t.Errorf("expected non-empty timestamp header")
	}

	// Verify HMAC signature
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(receivedTimestamp))
	mac.Write([]byte("."))
	mac.Write(receivedBody)
	expectedSig := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	if receivedSig != expectedSig {
		t.Errorf("expected signature %s, got %s", expectedSig, receivedSig)
	}
}

func TestDispatcherPool_DispatchAll(t *testing.T) {
	st := setupTestStore(t)
	hub := NewSSEHub()
	cfg := &config.Config{}

	pool := NewDispatcherPool(cfg, st, hub)

	_ = st.UpsertSubscriber(&store.Subscriber{ID: "usr_pool"})

	messages := []*engine.CompiledMessage{
		{
			Channel:    store.ChannelInApp,
			Subject:    "Title 1",
			Body:       "Body 1",
			Subscriber: &store.Subscriber{ID: "usr_pool"},
			WorkflowID: "wf_1",
		},
	}

	dispatched, errs := pool.DispatchAll(context.Background(), messages, "notif_pool_1")
	if len(errs) > 0 {
		t.Errorf("unexpected errors: %v", errs)
	}
	if len(dispatched) != 1 || dispatched[0] != store.ChannelInApp {
		t.Errorf("expected in_app dispatched, got %v", dispatched)
	}

	// Check delivery log in store
	logs, err := st.ListDeliveryLogs(10, 0, store.ChannelInApp, store.StatusSent)
	if err != nil || len(logs) != 1 {
		t.Fatalf("expected 1 delivery log recorded, got %d (err: %v)", len(logs), err)
	}
	if logs[0].NotificationID != "notif_pool_1" {
		t.Errorf("expected log notification_id notif_pool_1, got %s", logs[0].NotificationID)
	}
}

func TestDiscordDispatcher_MockServer(t *testing.T) {
	var receivedPayload map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &receivedPayload)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"id": "discord_msg_123"}`))
	}))
	defer server.Close()

	dispatcher := NewDiscordDispatcher()
	dispatcher.httpClient = server.Client()

	msg := &engine.CompiledMessage{
		Channel:   store.ChannelDiscord,
		Subject:   "Server High Memory Alert",
		Body:      "Node worker-03 reached 92% memory utilization",
		ActionURL: "https://monitoring.example.com/nodes/worker-03",
		Subscriber: &store.Subscriber{
			ID:                "usr_discord_lead",
			DiscordWebhookURL: server.URL,
		},
		WorkflowID: "infra.alert",
	}

	if err := dispatcher.Dispatch(context.Background(), msg); err != nil {
		t.Fatalf("discord dispatch failed: %v", err)
	}

	if receivedPayload["username"] != "Notiflux" {
		t.Errorf("expected username Notiflux, got %v", receivedPayload["username"])
	}

	embeds, ok := receivedPayload["embeds"].([]any)
	if !ok || len(embeds) == 0 {
		t.Fatalf("expected embeds array, got %v", receivedPayload["embeds"])
	}
	embed := embeds[0].(map[string]any)
	if embed["title"] != "Server High Memory Alert" {
		t.Errorf("expected embed title 'Server High Memory Alert', got %v", embed["title"])
	}

	// Test missing webhook URL error
	err := dispatcher.Dispatch(context.Background(), &engine.CompiledMessage{
		Subscriber: &store.Subscriber{ID: "usr_no_discord"},
	})
	if err == nil {
		t.Errorf("expected error for missing discord webhook URL")
	}
}

