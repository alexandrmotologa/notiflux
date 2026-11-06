package api

import (
	"context"
	"encoding/json"
	"net/http"
	"runtime"
	"time"

	"github.com/SherClockHolmes/webpush-go"
	"github.com/alexandrmotologa/notiflux/internal/channels"
	"github.com/alexandrmotologa/notiflux/internal/config"
	"github.com/alexandrmotologa/notiflux/internal/engine"
	"github.com/alexandrmotologa/notiflux/internal/store"
)

var startTime = time.Now()

// SystemHandler provides diagnostic, health check, and utility endpoints.
type SystemHandler struct {
	cfg    *config.Config
	sseHub *channels.SSEHub
	pool   *channels.DispatcherPool
}

func NewSystemHandler(cfg *config.Config, hub *channels.SSEHub, pool *channels.DispatcherPool) *SystemHandler {
	return &SystemHandler{
		cfg:    cfg,
		sseHub: hub,
		pool:   pool,
	}
}

func (h *SystemHandler) Health(w http.ResponseWriter, r *http.Request) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	allocMB := float64(m.Alloc) / 1024 / 1024
	sysMB := float64(m.Sys) / 1024 / 1024

	activeSSE := 0
	if h.sseHub != nil {
		activeSSE = h.sseHub.ActiveCount()
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":                 "healthy",
		"version":                "1.0.0",
		"uptime_seconds":         int(time.Since(startTime).Seconds()),
		"memory_alloc_mb":        round(allocMB, 2),
		"memory_sys_mb":          round(sysMB, 2),
		"goroutines":             runtime.NumGoroutine(),
		"active_sse_connections": activeSSE,
	})
}

func (h *SystemHandler) GenerateVAPID(w http.ResponseWriter, r *http.Request) {
	privateKey, publicKey, err := webpush.GenerateVAPIDKeys()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed generating VAPID keys: " + err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"public_key":  publicKey,
		"private_key": privateKey,
	})
}

func (h *SystemHandler) TestChannel(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Channel    string           `json:"channel"`
		Subscriber store.Subscriber `json:"subscriber"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json: " + err.Error()})
		return
	}

	if body.Subscriber.ID == "" {
		body.Subscriber.ID = "test_user"
	}

	testMsg := &engine.CompiledMessage{
		Channel:    body.Channel,
		Subject:    "Notiflux Connection Test",
		Body:       "This is a test notification confirming your channel configuration is operational.",
		HTMLBody:   engine.WrapEmailHTML("<p>This is a test notification confirming your email channel configuration is operational.</p>", "Notiflux Connection Test"),
		Subscriber: &body.Subscriber,
		WorkflowID: "system.test",
	}

	dispatched, errs := h.pool.DispatchAll(context.Background(), []*engine.CompiledMessage{testMsg}, "test_ping")
	if len(errs) > 0 {
		writeJSON(w, http.StatusBadGateway, map[string]any{
			"success": false,
			"error":   errs[body.Channel],
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success":    true,
		"dispatched": dispatched,
	})
}

func round(val float64, decimals int) float64 {
	pow := 1.0
	for i := 0; i < decimals; i++ {
		pow *= 10
	}
	return float64(int(val*pow+0.5)) / pow
}
