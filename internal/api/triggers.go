package api

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"

	"github.com/alexandrmotologa/notiflux/internal/channels"
	"github.com/alexandrmotologa/notiflux/internal/engine"
)

// TriggerHandler coordinates ingestion requests, idempotency checks, and channel execution.
type TriggerHandler struct {
	engine      *engine.WorkflowEngine
	pool        *channels.DispatcherPool
	idempotency *engine.IdempotencyEngine
}

func NewTriggerHandler(
	eng *engine.WorkflowEngine,
	p *channels.DispatcherPool,
	idem *engine.IdempotencyEngine,
) *TriggerHandler {
	return &TriggerHandler{
		engine:      eng,
		pool:        p,
		idempotency: idem,
	}
}

// Trigger processes a single workflow notification dispatch.
func (h *TriggerHandler) Trigger(w http.ResponseWriter, r *http.Request) {
	idempotencyKey := r.Header.Get("Idempotency-Key")
	if idempotencyKey != "" {
		if cached, err := h.idempotency.Check(idempotencyKey); err == nil && cached != nil {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("X-Cache", "HIT")
			w.WriteHeader(cached.StatusCode)
			w.Write([]byte(cached.ResponseBody))
			return
		}
	}

	var req engine.TriggerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json payload: " + err.Error()})
		return
	}

	compiledMessages, notifID, skipped, err := h.engine.PrepareDispatch(&req)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	dispatched, dispatchErrors := h.pool.DispatchAll(r.Context(), compiledMessages, notifID)

	resp := engine.TriggerResponse{
		Success:            len(dispatchErrors) == 0,
		NotificationID:     notifID,
		DispatchedChannels: dispatched,
		SkippedChannels:    skipped,
	}

	statusCode := http.StatusOK
	if len(dispatchErrors) > 0 && len(dispatched) == 0 {
		statusCode = http.StatusBadGateway
		resp.ErrorMessage = "all channel dispatches failed"
	}

	if idempotencyKey != "" {
		_ = h.idempotency.Save(idempotencyKey, statusCode, resp)
	}

	writeJSON(w, statusCode, resp)
}

// TriggerBatch processes an array of up to 500 notifications concurrently.
func (h *TriggerHandler) TriggerBatch(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Notifications []engine.TriggerRequest `json:"notifications"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid batch json: " + err.Error()})
		return
	}

	if len(body.Notifications) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "notifications array is empty"})
		return
	}
	if len(body.Notifications) > 500 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "batch size exceeds maximum of 500"})
		return
	}

	results := make([]engine.TriggerResponse, len(body.Notifications))
	var wg sync.WaitGroup

	// Concurrency limiter of 10 workers
	sem := make(chan struct{}, 10)

	for i, req := range body.Notifications {
		wg.Add(1)
		sem <- struct{}{}

		go func(idx int, r engine.TriggerRequest) {
			defer wg.Done()
			defer func() { <-sem }()

			compiled, notifID, skipped, err := h.engine.PrepareDispatch(&r)
			if err != nil {
				results[idx] = engine.TriggerResponse{
					Success:      false,
					ErrorMessage: err.Error(),
				}
				return
			}

			dispatched, _ := h.pool.DispatchAll(context.Background(), compiled, notifID)
			results[idx] = engine.TriggerResponse{
				Success:            true,
				NotificationID:     notifID,
				DispatchedChannels: dispatched,
				SkippedChannels:    skipped,
			}
		}(i, req)
	}

	wg.Wait()

	writeJSON(w, http.StatusOK, map[string]any{
		"total":   len(results),
		"results": results,
	})
}
