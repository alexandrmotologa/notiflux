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

func TestAPI_TrackingEndpoints(t *testing.T) {
	ctx, cleanup := setupTestRouter(t, "")
	defer cleanup()

	// Seed subscriber and delivery log with tracking ID
	_ = ctx.store.UpsertSubscriber(&store.Subscriber{ID: "usr_trk_test", Email: "trk@example.com"})
	logRecord := &store.DeliveryLog{
		ID:           "dl_trk_1",
		WorkflowID:   "order.shipped",
		SubscriberID: "usr_trk_test",
		Channel:      store.ChannelEmail,
		Status:       store.StatusSent,
		TrackingID:   "trk_test_token_88",
		TargetURL:    "https://example.com/orders/track-88",
	}
	_ = ctx.store.RecordDeliveryLog(logRecord)

	// 1. Test Open Pixel: GET /v1/o/trk_test_token_88.png
	reqPixel := httptest.NewRequest(http.MethodGet, "/v1/o/trk_test_token_88.png", nil)
	recPixel := httptest.NewRecorder()
	ctx.router.ServeHTTP(recPixel, reqPixel)

	if recPixel.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for tracking pixel, got %d", recPixel.Code)
	}
	if recPixel.Header().Get("Content-Type") != "image/png" {
		t.Errorf("expected image/png content type, got %s", recPixel.Header().Get("Content-Type"))
	}

	// Verify opened_at is recorded
	logs, err := ctx.store.ListDeliveryLogs(10, 0, "", "")
	if err != nil || len(logs) == 0 {
		t.Fatalf("failed listing logs: %v", err)
	}
	if logs[0].OpenedAt == nil {
		t.Errorf("expected OpenedAt to be populated in database")
	}

	// 2. Test Click Redirect: GET /v1/r/trk_test_token_88
	reqRedirect := httptest.NewRequest(http.MethodGet, "/v1/r/trk_test_token_88", nil)
	recRedirect := httptest.NewRecorder()
	ctx.router.ServeHTTP(recRedirect, reqRedirect)

	if recRedirect.Code != http.StatusFound {
		t.Fatalf("expected 302 Found for redirect, got %d", recRedirect.Code)
	}
	if recRedirect.Header().Get("Location") != "https://example.com/orders/track-88" {
		t.Errorf("expected redirect location https://example.com/orders/track-88, got %s", recRedirect.Header().Get("Location"))
	}

	// Verify clicked_at is recorded
	logs, _ = ctx.store.ListDeliveryLogs(10, 0, "", "")
	if logs[0].ClickedAt == nil {
		t.Errorf("expected ClickedAt to be populated in database")
	}
}

func TestAPI_MultiAPIKeysAndHotBackup(t *testing.T) {
	ctx, cleanup := setupTestRouter(t, "root-master-key")
	defer cleanup()

	// 1. Create a workspace key
	keyPayload := map[string]string{"name": "Billing Service Key"}
	b, _ := json.Marshal(keyPayload)
	reqCreate := httptest.NewRequest(http.MethodPost, "/v1/system/keys", bytes.NewReader(b))
	reqCreate.Header.Set("Authorization", "Bearer root-master-key")
	recCreate := httptest.NewRecorder()
	ctx.router.ServeHTTP(recCreate, reqCreate)

	if recCreate.Code != http.StatusCreated {
		t.Fatalf("expected 201 Created for API key, got %d (body: %s)", recCreate.Code, recCreate.Body.String())
	}

	var createdResp map[string]string
	_ = json.Unmarshal(recCreate.Body.Bytes(), &createdResp)
	newKey := createdResp["key"]
	if !strings.HasPrefix(newKey, "nf_") {
		t.Errorf("expected key to start with nf_, got %s", newKey)
	}

	// 2. List keys
	reqList := httptest.NewRequest(http.MethodGet, "/v1/system/keys", nil)
	reqList.Header.Set("Authorization", "Bearer root-master-key")
	recList := httptest.NewRecorder()
	ctx.router.ServeHTTP(recList, reqList)

	if recList.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for list keys, got %d", recList.Code)
	}

	// 3. Authenticate with newly generated workspace key
	reqAuth := httptest.NewRequest(http.MethodGet, "/v1/workflows", nil)
	reqAuth.Header.Set("Authorization", "Bearer "+newKey)
	recAuth := httptest.NewRecorder()
	ctx.router.ServeHTTP(recAuth, reqAuth)

	if recAuth.Code != http.StatusOK {
		t.Errorf("expected 200 OK using workspace key, got %d", recAuth.Code)
	}

	// 4. Test Hot Backup: POST /v1/system/backup
	reqBackup := httptest.NewRequest(http.MethodPost, "/v1/system/backup", nil)
	reqBackup.Header.Set("Authorization", "Bearer root-master-key")
	recBackup := httptest.NewRecorder()
	ctx.router.ServeHTTP(recBackup, reqBackup)

	if recBackup.Code != http.StatusOK {
		t.Fatalf("expected 200 OK for hot backup, got %d (body: %s)", recBackup.Code, recBackup.Body.String())
	}

	var backupResp map[string]any
	_ = json.Unmarshal(recBackup.Body.Bytes(), &backupResp)
	if backupResp["success"] != true || backupResp["backup_file"] == nil {
		t.Errorf("unexpected hot backup response: %v", backupResp)
	}

	// 5. Delete workspace key
	reqDel := httptest.NewRequest(http.MethodDelete, "/v1/system/keys/"+newKey, nil)
	reqDel.Header.Set("Authorization", "Bearer root-master-key")
	recDel := httptest.NewRecorder()
	ctx.router.ServeHTTP(recDel, reqDel)

	if recDel.Code != http.StatusOK {
		t.Errorf("expected 200 OK for key deletion, got %d", recDel.Code)
	}
}

