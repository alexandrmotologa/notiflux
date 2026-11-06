package channels

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/alexandrmotologa/notiflux/internal/config"
	"github.com/alexandrmotologa/notiflux/internal/engine"
	"github.com/alexandrmotologa/notiflux/internal/store"
)

// WebhookDispatcher delivers notifications via outbound HTTPS webhooks with cryptographic HMAC signatures.
type WebhookDispatcher struct {
	cfg        *config.Config
	httpClient *http.Client
}

func NewWebhookDispatcher(cfg *config.Config) *WebhookDispatcher {
	return &WebhookDispatcher{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (d *WebhookDispatcher) Channel() string {
	return store.ChannelWebhook
}

func (d *WebhookDispatcher) Dispatch(ctx context.Context, msg *engine.CompiledMessage) error {
	targetURL := ""
	if urlVal, ok := msg.Payload["webhook_url"]; ok && urlVal != nil {
		targetURL = fmt.Sprintf("%v", urlVal)
	}

	if targetURL == "" {
		return fmt.Errorf("no webhook_url specified in payload")
	}

	now := time.Now().UTC()
	timestampStr := strconv.FormatInt(now.Unix(), 10)

	payloadMap := map[string]any{
		"event":         "notification.dispatched",
		"workflow_id":   msg.WorkflowID,
		"subscriber_id": msg.Subscriber.ID,
		"subject":       msg.Subject,
		"body":          msg.Body,
		"action_url":    msg.ActionURL,
		"payload":       msg.Payload,
		"timestamp":     now.Format(time.RFC3339),
	}

	bodyBytes, err := json.Marshal(payloadMap)
	if err != nil {
		return err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return err
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", "Notiflux-Webhook-Engine/1.0")
	httpReq.Header.Set("X-Notiflux-Timestamp", timestampStr)
	httpReq.Header.Set("X-Notiflux-Event", "notification.dispatched")

	// Calculate HMAC signature if secret is present
	if d.cfg.WebhookSecret != "" {
		mac := hmac.New(sha256.New, []byte(d.cfg.WebhookSecret))
		mac.Write([]byte(timestampStr))
		mac.Write([]byte("."))
		mac.Write(bodyBytes)
		sig := hex.EncodeToString(mac.Sum(nil))
		httpReq.Header.Set("X-Notiflux-Signature", "sha256="+sig)
	}

	resp, err := d.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("webhook delivery failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("webhook endpoint returned status %d: %s", resp.StatusCode, string(respBytes))
	}

	return nil
}
