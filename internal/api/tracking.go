package api

import (
	"net/http"
	"strings"

	"github.com/alexandrmotologa/notiflux/internal/store"
	"github.com/go-chi/chi/v5"
)

// Transparent 1x1 PNG bytes
var transparentPixel = []byte{
	0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a,
	0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
	0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
	0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0x15, 0x63,
	0x34, 0x00, 0x00, 0x00, 0x0a, 0x49, 0x44, 0x41,
	0x54, 0x78, 0x9c, 0x63, 0x00, 0x01, 0x00, 0x00,
	0x05, 0x00, 0x01, 0x0d, 0x0a, 0x2d, 0xb4, 0x00,
	0x00, 0x00, 0x00, 0x49, 0x45, 0x4e, 0x44, 0xae,
	0x42, 0x60, 0x82,
}

// TrackingHandler handles click redirections and email open pixels.
type TrackingHandler struct {
	store *store.Store
}

func NewTrackingHandler(s *store.Store) *TrackingHandler {
	return &TrackingHandler{store: s}
}

// TrackClick redirects the user to the destination URL while recording the click.
func (h *TrackingHandler) TrackClick(w http.ResponseWriter, r *http.Request) {
	trackingID := chi.URLParam(r, "tracking_id")
	if trackingID == "" {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}

	targetURL, err := h.store.RecordTrackingClick(trackingID)
	if err != nil || targetURL == "" {
		targetURL = "/"
	}

	http.Redirect(w, r, targetURL, http.StatusFound)
}

// TrackOpen serves a 1x1 transparent PNG while recording the email open event.
func (h *TrackingHandler) TrackOpen(w http.ResponseWriter, r *http.Request) {
	trackingID := chi.URLParam(r, "tracking_id")
	trackingID = strings.TrimSuffix(trackingID, ".png")

	if trackingID != "" {
		_ = h.store.RecordTrackingOpen(trackingID)
	}

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate, private")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
	w.WriteHeader(http.StatusOK)
	w.Write(transparentPixel)
}
