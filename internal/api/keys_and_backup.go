package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"time"

	"github.com/alexandrmotologa/notiflux/internal/store"
	"github.com/go-chi/chi/v5"
)

// KeyAndBackupHandler manages project API keys and hot database backups.
type KeyAndBackupHandler struct {
	store *store.Store
}

func NewKeyAndBackupHandler(s *store.Store) *KeyAndBackupHandler {
	return &KeyAndBackupHandler{store: s}
}

func (h *KeyAndBackupHandler) ListKeys(w http.ResponseWriter, r *http.Request) {
	keys, err := h.store.ListAPIKeys()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if keys == nil {
		keys = []*store.APIKeyRecord{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"keys": keys})
}

func (h *KeyAndBackupHandler) CreateKey(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Name == "" {
		body.Name = "Default"
	}

	key, err := h.store.CreateAPIKey(body.Name)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"key":        key,
		"name":       body.Name,
		"created_at": time.Now().UTC(),
	})
}

func (h *KeyAndBackupHandler) DeleteKey(w http.ResponseWriter, r *http.Request) {
	key := chi.URLParam(r, "key")
	if err := h.store.DeleteAPIKey(key); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"success": true, "deleted_key": key})
}

func (h *KeyAndBackupHandler) TriggerBackup(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Destination string `json:"destination"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)

	dest := body.Destination
	if dest == "" {
		dest = fmt.Sprintf("notiflux_backup_%d.db", time.Now().Unix())
	}

	absPath, err := filepath.Abs(dest)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	if err := h.store.HotBackup(absPath); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "backup failed: " + err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success":     true,
		"destination": absPath,
		"backup_file": absPath,
		"timestamp":   time.Now().UTC(),
	})
}
