package api

import (
	"net/http"
	"strconv"

	"github.com/alexandrmotologa/notiflux/internal/store"
	"github.com/go-chi/chi/v5"
)

// LogsHandler serves delivery logs and subscriber lists.
type LogsHandler struct {
	store *store.Store
}

func NewLogsHandler(s *store.Store) *LogsHandler {
	return &LogsHandler{store: s}
}

func (h *LogsHandler) ListLogs(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil && val > 0 && val <= 200 {
			limit = val
		}
	}

	offset := 0
	if o := r.URL.Query().Get("offset"); o != "" {
		if val, err := strconv.Atoi(o); err == nil && val >= 0 {
			offset = val
		}
	}

	filterChannel := r.URL.Query().Get("channel")
	filterStatus := r.URL.Query().Get("status")

	logs, err := h.store.ListDeliveryLogs(limit, offset, filterChannel, filterStatus)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if logs == nil {
		logs = []*store.DeliveryLog{}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"logs":  logs,
		"total": len(logs),
	})
}

func (h *LogsHandler) ListSubscribers(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil && val > 0 && val <= 200 {
			limit = val
		}
	}

	offset := 0
	if o := r.URL.Query().Get("offset"); o != "" {
		if val, err := strconv.Atoi(o); err == nil && val >= 0 {
			offset = val
		}
	}

	subscribers, err := h.store.ListSubscribers(limit, offset)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if subscribers == nil {
		subscribers = []*store.Subscriber{}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"subscribers": subscribers,
		"total":       len(subscribers),
	})
}

func (h *LogsHandler) GetSubscriber(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	sub, err := h.store.GetSubscriber(id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if sub == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "subscriber not found"})
		return
	}

	prefs, _ := h.store.GetPreferences(id)
	unreadCount, _ := h.store.GetUnreadCount(id)

	writeJSON(w, http.StatusOK, map[string]any{
		"subscriber":   sub,
		"preferences":  prefs,
		"unread_count": unreadCount,
	})
}
