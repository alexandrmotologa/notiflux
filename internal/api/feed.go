package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/alexandrmotologa/notiflux/internal/channels"
	"github.com/alexandrmotologa/notiflux/internal/store"
	"github.com/go-chi/chi/v5"
)

// FeedHandler handles client-facing notification feeds and SSE streaming.
type FeedHandler struct {
	store  *store.Store
	sseHub *channels.SSEHub
}

func NewFeedHandler(s *store.Store, hub *channels.SSEHub) *FeedHandler {
	return &FeedHandler{store: s, sseHub: hub}
}

// GetFeed lists notifications for a subscriber.
func (h *FeedHandler) GetFeed(w http.ResponseWriter, r *http.Request) {
	subscriberID := r.URL.Query().Get("subscriber_id")
	if subscriberID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "subscriber_id query parameter is required"})
		return
	}

	limit := 20
	if l := r.URL.Query().Get("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil && val > 0 && val <= 100 {
			limit = val
		}
	}

	offset := 0
	if o := r.URL.Query().Get("offset"); o != "" {
		if val, err := strconv.Atoi(o); err == nil && val >= 0 {
			offset = val
		}
	}

	unreadOnly := r.URL.Query().Get("unread_only") == "true"

	notifications, err := h.store.ListNotifications(subscriberID, limit, offset, unreadOnly)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if notifications == nil {
		notifications = []*store.Notification{}
	}

	unreadCount, _ := h.store.GetUnreadCount(subscriberID)

	writeJSON(w, http.StatusOK, map[string]any{
		"notifications": notifications,
		"unread_count":  unreadCount,
		"total":         len(notifications),
	})
}

// StreamSSE establishes a persistent text/event-stream connection for real-time updates.
func (h *FeedHandler) StreamSSE(w http.ResponseWriter, r *http.Request) {
	subscriberID := r.URL.Query().Get("subscriber_id")
	if subscriberID == "" {
		http.Error(w, "subscriber_id query parameter is required", http.StatusBadRequest)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache, no-transform")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Subscribe to hub
	eventCh, cleanup := h.sseHub.Subscribe(subscriberID)
	defer cleanup()

	// Send initial connection event with current unread count
	unreadCount, _ := h.store.GetUnreadCount(subscriberID)
	initData, _ := json.Marshal(map[string]any{
		"subscriber_id": subscriberID,
		"unread_count":  unreadCount,
		"timestamp":     time.Now().UTC().Format(time.RFC3339),
	})
	fmt.Fprintf(w, "event: connected\ndata: %s\n\n", string(initData))
	flusher.Flush()

	// 15-second heartbeat ticker to keep proxies open
	heartbeatTicker := time.NewTicker(15 * time.Second)
	defer heartbeatTicker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-heartbeatTicker.C:
			fmt.Fprintf(w, ": ping\n\n")
			flusher.Flush()
		case msg, ok := <-eventCh:
			if !ok {
				return
			}
			w.Write(msg)
			flusher.Flush()
		}
	}
}

// MarkAsRead marks a single notification as read.
func (h *FeedHandler) MarkAsRead(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "notification id is required"})
		return
	}

	notif, err := h.store.GetNotification(id)
	if err != nil || notif == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "notification not found"})
		return
	}

	if err := h.store.MarkAsRead(id); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	unreadCount, _ := h.store.GetUnreadCount(notif.SubscriberID)
	h.sseHub.Broadcast(notif.SubscriberID, "notification.read", map[string]any{
		"id":           id,
		"unread_count": unreadCount,
	})

	writeJSON(w, http.StatusOK, map[string]any{
		"success":      true,
		"id":           id,
		"unread_count": unreadCount,
	})
}

// MarkAsUnread marks a single notification as unread.
func (h *FeedHandler) MarkAsUnread(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	notif, err := h.store.GetNotification(id)
	if err != nil || notif == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "notification not found"})
		return
	}

	if err := h.store.MarkAsUnread(id); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	unreadCount, _ := h.store.GetUnreadCount(notif.SubscriberID)
	h.sseHub.Broadcast(notif.SubscriberID, "notification.unread", map[string]any{
		"id":           id,
		"unread_count": unreadCount,
	})

	writeJSON(w, http.StatusOK, map[string]any{
		"success":      true,
		"id":           id,
		"unread_count": unreadCount,
	})
}

// MarkAllAsRead marks all notifications as read for a given subscriber.
func (h *FeedHandler) MarkAllAsRead(w http.ResponseWriter, r *http.Request) {
	var body struct {
		SubscriberID string `json:"subscriber_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.SubscriberID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "subscriber_id is required"})
		return
	}

	if err := h.store.MarkAllAsRead(body.SubscriberID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	h.sseHub.Broadcast(body.SubscriberID, "notification.read_all", map[string]any{
		"subscriber_id": body.SubscriberID,
		"unread_count":  0,
	})

	writeJSON(w, http.StatusOK, map[string]any{
		"success":      true,
		"unread_count": 0,
	})
}

// DeleteNotification removes a notification item.
func (h *FeedHandler) DeleteNotification(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	notif, err := h.store.GetNotification(id)
	if err != nil || notif == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "notification not found"})
		return
	}

	if err := h.store.DeleteNotification(id); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	unreadCount, _ := h.store.GetUnreadCount(notif.SubscriberID)
	h.sseHub.Broadcast(notif.SubscriberID, "notification.deleted", map[string]any{
		"id":           id,
		"unread_count": unreadCount,
	})

	writeJSON(w, http.StatusOK, map[string]any{
		"success":      true,
		"id":           id,
		"unread_count": unreadCount,
	})
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}
