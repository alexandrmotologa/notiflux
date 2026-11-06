package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/alexandrmotologa/notiflux/internal/channels"
	"github.com/alexandrmotologa/notiflux/internal/config"
	"github.com/alexandrmotologa/notiflux/internal/engine"
	"github.com/alexandrmotologa/notiflux/internal/store"
)

func setupTestRouter(t *testing.T, apiKey string) (*chiRouterTestContext, func()) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "api_test.db")
	st, err := store.NewStore(dbPath)
	if err != nil {
		t.Fatalf("failed opening test store: %v", err)
	}

	cfg := &config.Config{
		Port:           8085,
		DBPath:         dbPath,
		APIKey:         apiKey,
		AllowedOrigins: []string{"*"},
	}

	hub := channels.NewSSEHub()
	pool := channels.NewDispatcherPool(cfg, st, hub)
	eng := engine.NewWorkflowEngine(st)
	idem := engine.NewIdempotencyEngine(st)

	rc := &RouterConfig{
		Config:      cfg,
		Store:       st,
		SSEHub:      hub,
		Pool:        pool,
		Engine:      eng,
		Idempotency: idem,
	}

	router := NewRouter(rc)
	cleanup := func() {
		_ = st.Close()
	}

	return &chiRouterTestContext{
		router: router,
		store:  st,
		hub:    hub,
	}, cleanup
}

type chiRouterTestContext struct {
	router http.Handler
	store  *store.Store
	hub    *channels.SSEHub
}

func TestAPI_Health(t *testing.T) {
	ctx, cleanup := setupTestRouter(t, "")
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/v1/health", nil)
	rec := httptest.NewRecorder()
	ctx.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed decoding response: %v", err)
	}

	if resp["status"] != "healthy" {
		t.Errorf("expected status healthy, got %v", resp["status"])
	}
}

func TestAPI_AuthMiddleware(t *testing.T) {
	ctx, cleanup := setupTestRouter(t, "secret-key")
	defer cleanup()

	// 1. Without auth header -> 401
	req := httptest.NewRequest(http.MethodGet, "/v1/workflows", nil)
	rec := httptest.NewRecorder()
	ctx.router.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 Unauthorized, got %d", rec.Code)
	}

	// 2. With Bearer token -> 200
	reqValid := httptest.NewRequest(http.MethodGet, "/v1/workflows", nil)
	reqValid.Header.Set("Authorization", "Bearer secret-key")
	recValid := httptest.NewRecorder()
	ctx.router.ServeHTTP(recValid, reqValid)
	if recValid.Code != http.StatusOK {
		t.Errorf("expected 200 OK with valid bearer, got %d", recValid.Code)
	}

	// 3. With X-API-Key header -> 200
	reqHeader := httptest.NewRequest(http.MethodGet, "/v1/workflows", nil)
	reqHeader.Header.Set("X-API-Key", "secret-key")
	recHeader := httptest.NewRecorder()
	ctx.router.ServeHTTP(recHeader, reqHeader)
	if recHeader.Code != http.StatusOK {
		t.Errorf("expected 200 OK with valid X-API-Key, got %d", recHeader.Code)
	}
}

func TestAPI_WorkflowAndTriggerLifecycle(t *testing.T) {
	ctx, cleanup := setupTestRouter(t, "")
	defer cleanup()

	// 1. Create Workflow
	wfPayload := map[string]any{
		"id":          "order.created",
		"name":        "Order Created Notification",
		"description": "Triggered when order finishes",
		"channels":    []string{store.ChannelInApp},
	}
	bodyBytes, _ := json.Marshal(wfPayload)
	req := httptest.NewRequest(http.MethodPost, "/v1/workflows", bytes.NewReader(bodyBytes))
	rec := httptest.NewRecorder()
	ctx.router.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created for workflow, got %d (body: %s)", rec.Code, rec.Body.String())
	}

	// 2. Add Template
	tplPayload := map[string]any{
		"templates": []map[string]string{
			{
				"channel": store.ChannelInApp,
				"subject": "Order #{{order_id}} Confirmed",
				"body":    "Total amount: ${{amount}}",
			},
		},
	}
	tplBytes, _ := json.Marshal(tplPayload)
	reqTpl := httptest.NewRequest(http.MethodPut, "/v1/workflows/order.created/templates", bytes.NewReader(tplBytes))
	recTpl := httptest.NewRecorder()
	ctx.router.ServeHTTP(recTpl, reqTpl)
	if recTpl.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for template update, got %d", recTpl.Code)
	}

	// 3. Trigger Notification with Idempotency Key
	triggerPayload := map[string]any{
		"workflow": "order.created",
		"subscriber": map[string]any{
			"id":    "usr_api_test",
			"email": "buyer@example.com",
		},
		"payload": map[string]any{
			"order_id":   "8844",
			"amount":     "129.50",
			"action_url": "https://example.com/orders/8844",
		},
	}
	trigBytes, _ := json.Marshal(triggerPayload)
	reqTrig := httptest.NewRequest(http.MethodPost, "/v1/trigger", bytes.NewReader(trigBytes))
	reqTrig.Header.Set("Idempotency-Key", "idem_test_key_1")
	recTrig := httptest.NewRecorder()
	ctx.router.ServeHTTP(recTrig, reqTrig)

	if recTrig.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for trigger, got %d (body: %s)", recTrig.Code, recTrig.Body.String())
	}

	var trigResp engine.TriggerResponse
	_ = json.Unmarshal(recTrig.Body.Bytes(), &trigResp)
	if !trigResp.Success || len(trigResp.DispatchedChannels) != 1 {
		t.Errorf("unexpected trigger response: %+v", trigResp)
	}

	// Replay with same Idempotency-Key -> should hit cache
	reqTrig2 := httptest.NewRequest(http.MethodPost, "/v1/trigger", bytes.NewReader(trigBytes))
	reqTrig2.Header.Set("Idempotency-Key", "idem_test_key_1")
	recTrig2 := httptest.NewRecorder()
	ctx.router.ServeHTTP(recTrig2, reqTrig2)
	if recTrig2.Header().Get("X-Cache") != "HIT" {
		t.Errorf("expected X-Cache: HIT on replayed idempotency key")
	}

	// 4. Query In-App Feed
	reqFeed := httptest.NewRequest(http.MethodGet, "/v1/feed?subscriber_id=usr_api_test", nil)
	recFeed := httptest.NewRecorder()
	ctx.router.ServeHTTP(recFeed, reqFeed)

	if recFeed.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for feed, got %d", recFeed.Code)
	}

	var feedData struct {
		Notifications []*store.Notification `json:"notifications"`
		UnreadCount   int                   `json:"unread_count"`
	}
	_ = json.Unmarshal(recFeed.Body.Bytes(), &feedData)

	if feedData.UnreadCount != 1 || len(feedData.Notifications) != 1 {
		t.Fatalf("expected 1 unread notification, got %d (list len: %d)", feedData.UnreadCount, len(feedData.Notifications))
	}
	notifID := feedData.Notifications[0].ID
	if feedData.Notifications[0].Title != "Order #8844 Confirmed" {
		t.Errorf("unexpected notification title: %s", feedData.Notifications[0].Title)
	}

	// 5. Mark as Read
	reqRead := httptest.NewRequest(http.MethodPatch, "/v1/feed/"+notifID+"/read", nil)
	recRead := httptest.NewRecorder()
	ctx.router.ServeHTTP(recRead, reqRead)
	if recRead.Code != http.StatusOK {
		t.Errorf("expected 200 for mark as read, got %d", recRead.Code)
	}

	// 6. Verify unread count is 0
	reqFeed2 := httptest.NewRequest(http.MethodGet, "/v1/feed?subscriber_id=usr_api_test", nil)
	recFeed2 := httptest.NewRecorder()
	ctx.router.ServeHTTP(recFeed2, reqFeed2)
	_ = json.Unmarshal(recFeed2.Body.Bytes(), &feedData)
	if feedData.UnreadCount != 0 {
		t.Errorf("expected unread count 0 after mark read, got %d", feedData.UnreadCount)
	}
}

func TestAPI_PreferencesEndpoints(t *testing.T) {
	ctx, cleanup := setupTestRouter(t, "")
	defer cleanup()

	// Get initial preferences (defaults)
	req := httptest.NewRequest(http.MethodGet, "/v1/preferences/usr_pref_test", nil)
	rec := httptest.NewRecorder()
	ctx.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", rec.Code)
	}

	var prefsResp struct {
		Channels map[string]bool `json:"channels"`
	}
	_ = json.Unmarshal(rec.Body.Bytes(), &prefsResp)
	if !prefsResp.Channels[store.ChannelEmail] {
		t.Errorf("expected email to be enabled by default")
	}

	// Update preference (mute email)
	updatePayload := map[string]any{
		"channels": map[string]bool{
			store.ChannelEmail: false,
		},
	}
	b, _ := json.Marshal(updatePayload)
	reqUpdate := httptest.NewRequest(http.MethodPut, "/v1/preferences/usr_pref_test", bytes.NewReader(b))
	recUpdate := httptest.NewRecorder()
	ctx.router.ServeHTTP(recUpdate, reqUpdate)

	if recUpdate.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for update, got %d", recUpdate.Code)
	}

	// Re-verify
	reqCheck := httptest.NewRequest(http.MethodGet, "/v1/preferences/usr_pref_test", nil)
	recCheck := httptest.NewRecorder()
	ctx.router.ServeHTTP(recCheck, reqCheck)
	_ = json.Unmarshal(recCheck.Body.Bytes(), &prefsResp)

	if prefsResp.Channels[store.ChannelEmail] {
		t.Errorf("expected email to be muted (false), got true")
	}
}

func TestAPI_WidgetServing(t *testing.T) {
	ctx, cleanup := setupTestRouter(t, "")
	defer cleanup()

	req := httptest.NewRequest(http.MethodGet, "/widget/notiflux-bell.js", nil)
	rec := httptest.NewRecorder()
	ctx.router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for widget script, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "notiflux-bell") {
		t.Errorf("expected widget script content, got %s", rec.Body.String()[:100])
	}
}
