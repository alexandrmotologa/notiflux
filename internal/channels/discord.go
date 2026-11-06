package channels

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/alexandrmotologa/notiflux/internal/engine"
	"github.com/alexandrmotologa/notiflux/internal/store"
)

// DiscordDispatcher delivers notifications to Discord via Webhooks with rich embeds.
type DiscordDispatcher struct {
	httpClient *http.Client
}

func NewDiscordDispatcher() *DiscordDispatcher {
	return &DiscordDispatcher{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (d *DiscordDispatcher) Channel() string {
	return store.ChannelDiscord
}

func (d *DiscordDispatcher) Dispatch(ctx context.Context, msg *engine.CompiledMessage) error {
	webhookURL := ""
	if msg.Subscriber != nil && msg.Subscriber.DiscordWebhookURL != "" {
		webhookURL = msg.Subscriber.DiscordWebhookURL
	} else if urlVal, ok := msg.Payload["discord_webhook_url"]; ok && urlVal != nil {
		webhookURL = fmt.Sprintf("%v", urlVal)
	}

	if webhookURL == "" {
		return fmt.Errorf("discord webhook URL is required")
	}

	description := msg.Body
	if msg.ActionURL != "" {
		description = fmt.Sprintf("%s\n\n[Open Action Link](%s)", description, msg.ActionURL)
	}

	embed := map[string]any{
		"title":       msg.Subject,
		"description": description,
		"color":       0x6366f1, // Notiflux Indigo
		"timestamp":   time.Now().UTC().Format(time.RFC3339),
		"footer": map[string]string{
			"text": "Notiflux Notification Engine",
		},
	}

	if msg.ActionURL != "" {
		embed["url"] = msg.ActionURL
	}

	payload := map[string]any{
		"username": "Notiflux",
		"embeds":   []map[string]any{embed},
	}

	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(b))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := d.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("discord request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("discord API returned status %d: %s", resp.StatusCode, string(respBytes))
	}

	return nil
}
