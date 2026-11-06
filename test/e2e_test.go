package test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alexandrmotologa/notiflux/internal/api"
	"github.com/alexandrmotologa/notiflux/internal/channels"
	"github.com/alexandrmotologa/notiflux/internal/config"
	"github.com/alexandrmotologa/notiflux/internal/engine"
	"github.com/alexandrmotologa/notiflux/internal/store"
)

func TestEndToEndPipeline(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "e2e_test.db")
	st, err := store.NewStore(dbPath)
	if err != nil {
		t.Fatalf("failed opening store: %v", err)
	}
	defer st.Close()

	cfg := &config.Config{
		Port:           8085,
		DBPath:         dbPath,
		AllowedOrigins: []string{"*"},
	}

	hub := channels.NewSSEHub()
	pool := channels.NewDispatcherPool(cfg, st, hub)
	eng := engine.NewWorkflowEngine(st)
	idem := engine.NewIdempotencyEngine(st)

	router := api.NewRouter(&api.RouterConfig{
		Config:      cfg,
		Store:       st,
		SSEHub:      hub,
		Pool:        pool,
		Engine:      eng,
		Idempotency: idem,
	})

	server := httptest.NewServer(router)
	defer server.Close()

	// 1. Create a workflow
	wf := &store.Workflow{
		ID:       "checkout.success",
		Name:     "Checkout Success",
		Channels: []string{store.ChannelInApp},
	}
	_ = st.CreateWorkflow(wf)
	_ = st.UpsertTemplate(&store.Template{
		WorkflowID: "checkout.success",
		Channel:    store.ChannelInApp,
		Subject:    "Order #{{order_id}} Completed",
		Body:       "Total charged: {{total}}",
	})

	// 2. Open an SSE connection to simulate real-time client widget
	sseCtx, sseCancel := context.WithCancel(context.Background())
	defer sseCancel()

	sseReq, _ := http.NewRequestWithContext(sseCtx, http.MethodGet, server.URL+"/v1/feed/stream?subscriber_id=usr_e2e", nil)
	sseResp, err := server.Client().Do(sseReq)
	if err != nil {
		t.Fatalf("failed connecting to SSE: %v", err)
	}
	defer sseResp.Body.Close()

	reader := bufio.NewReader(sseResp.Body)
	// Read initial connected event
	initialLine, _ := reader.ReadString('\n')
	if !strings.Contains(initialLine, "event: connected") {
		t.Errorf("expected initial event: connected, got %s", initialLine)
	}

	// 3. Trigger Notification
	trigBody := map[string]any{
		"workflow": "checkout.success",
		"subscriber": map[string]any{
			"id": "usr_e2e",
		},
		"payload": map[string]any{
			"order_id": "ORD-5544",
			"total":    "$199.99",
		},
	}
	b, _ := json.Marshal(trigBody)
	resp, err := server.Client().Post(server.URL+"/v1/trigger", "application/json", bytes.NewReader(b))
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("trigger failed: code=%d err=%v", resp.StatusCode, err)
	}

	// 4. Read SSE event triggered by in-app dispatcher
	var receivedSSEEvent string
	for i := 0; i < 5; i++ {
		line, _ := reader.ReadString('\n')
		receivedSSEEvent += line
		if strings.Contains(receivedSSEEvent, "ORD-5544") {
			break
		}
	}

	if !strings.Contains(receivedSSEEvent, "notification.new") || !strings.Contains(receivedSSEEvent, "$199.99") {
		t.Errorf("expected real-time SSE event with $199.99, got: %s", receivedSSEEvent)
	}

	// 5. Query feed via REST API
	feedResp, err := server.Client().Get(server.URL + "/v1/feed?subscriber_id=usr_e2e")
	if err != nil || feedResp.StatusCode != http.StatusOK {
		t.Fatalf("feed request failed: %v", err)
	}
	feedBody, _ := io.ReadAll(feedResp.Body)

	var feedData struct {
		Notifications []*store.Notification `json:"notifications"`
		UnreadCount   int                   `json:"unread_count"`
	}
	_ = json.Unmarshal(feedBody, &feedData)

	if feedData.UnreadCount != 1 || len(feedData.Notifications) != 1 {
		t.Fatalf("expected 1 unread notification, got %d", feedData.UnreadCount)
	}
	if feedData.Notifications[0].Body != "Total charged: $199.99" {
		t.Errorf("expected body 'Total charged: $199.99', got %s", feedData.Notifications[0].Body)
	}

	// 6. Mark read and verify unread count is 0
	readReq, _ := http.NewRequest(http.MethodPatch, fmt.Sprintf("%s/v1/feed/%s/read", server.URL, feedData.Notifications[0].ID), nil)
	readResp, err := server.Client().Do(readReq)
	if err != nil || readResp.StatusCode != http.StatusOK {
		t.Fatalf("failed mark as read: %v", err)
	}

	unreadCount, _ := st.GetUnreadCount("usr_e2e")
	if unreadCount != 0 {
		t.Errorf("expected unread count 0, got %d", unreadCount)
	}

	// 7. Verify Health and Memory Telemetry < 45MB
	healthResp, err := server.Client().Get(server.URL + "/v1/health")
	if err != nil || healthResp.StatusCode != http.StatusOK {
		t.Fatalf("health check failed: %v", err)
	}
	var healthData map[string]any
	_ = json.NewDecoder(healthResp.Body).Decode(&healthData)

	sysMB := healthData["memory_sys_mb"].(float64)
	if sysMB > 45.0 {
		t.Errorf("memory exceeded 45MB budget: %.2f MB", sysMB)
	}
}
