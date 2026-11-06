package api

import (
	"encoding/json"
	"net/http"

	"github.com/alexandrmotologa/notiflux/internal/store"
	"github.com/go-chi/chi/v5"
)

// PreferencesHandler manages recipient channel preferences.
type PreferencesHandler struct {
	store *store.Store
}

func NewPreferencesHandler(s *store.Store) *PreferencesHandler {
	return &PreferencesHandler{store: s}
}

func (h *PreferencesHandler) GetPreferences(w http.ResponseWriter, r *http.Request) {
	subscriberID := chi.URLParam(r, "subscriber_id")
	if subscriberID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "subscriber_id parameter is required"})
		return
	}

	prefs, err := h.store.GetPreferences(subscriberID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// Fill in defaults for known channels if not explicitly set
	allChannels := []string{
		store.ChannelInApp,
		store.ChannelEmail,
		store.ChannelWebPush,
		store.ChannelTelegram,
		store.ChannelSlack,
	}
	channelMap := make(map[string]bool)
	for _, ch := range allChannels {
		if val, exists := prefs[ch]; exists {
			channelMap[ch] = val
		} else {
			channelMap[ch] = true // Default enabled
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"subscriber_id": subscriberID,
		"channels":      channelMap,
	})
}

func (h *PreferencesHandler) UpdatePreferences(w http.ResponseWriter, r *http.Request) {
	subscriberID := chi.URLParam(r, "subscriber_id")
	if subscriberID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "subscriber_id parameter is required"})
		return
	}

	// Ensure subscriber exists
	sub, err := h.store.GetSubscriber(subscriberID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if sub == nil {
		_ = h.store.UpsertSubscriber(&store.Subscriber{ID: subscriberID})
	}

	var body struct {
		Channels map[string]bool `json:"channels"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json: " + err.Error()})
		return
	}

	for ch, enabled := range body.Channels {
		if err := h.store.SetPreference(subscriberID, ch, enabled); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success":       true,
		"subscriber_id": subscriberID,
		"updated":       body.Channels,
	})
}
