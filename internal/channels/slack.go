package channels

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/alexandrmotologa/notiflux/internal/config"
	"github.com/alexandrmotologa/notiflux/internal/engine"
	"github.com/alexandrmotologa/notiflux/internal/store"
)

// SlackDispatcher delivers notifications to Slack Incoming Webhooks using Block Kit layouts.
type SlackDispatcher struct {
	cfg        *config.Config
	httpClient *http.Client
}

func NewSlackDispatcher(cfg *config.Config) *SlackDispatcher {
	return &SlackDispatcher{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (d *SlackDispatcher) Channel() string {
	return store.ChannelSlack
}

func (d *SlackDispatcher) Dispatch(ctx context.Context, msg *engine.CompiledMessage) error {
	webhookURL := ""
	if msg.Subscriber != nil && msg.Subscriber.SlackWebhookURL != "" {
		webhookURL = msg.Subscriber.SlackWebhookURL
	} else if d.cfg.SlackWebhookURL != "" {
		webhookURL = d.cfg.SlackWebhookURL
	}

	if webhookURL == "" {
		return fmt.Errorf("slack webhook URL is required: neither subscriber URL nor default URL configured")
	}

	blocks := []map[string]any{}

	// Header block
	if msg.Subject != "" {
		blocks = append(blocks, map[string]any{
			"type": "header",
			"text": map[string]string{
				"type":  "plain_text",
				"text":  msg.Subject,
				"emoji": "true",
			},
		})
	}

	// Section body block
	if msg.Body != "" {
		blocks = append(blocks, map[string]any{
			"type": "section",
			"text": map[string]string{
				"type": "mrkdwn",
				"text": msg.Body,
			},
		})
	}

	// Action button block if ActionURL is provided
	if msg.ActionURL != "" {
		blocks = append(blocks, map[string]any{
			"type": "actions",
			"elements": []map[string]any{
				{
					"type": "button",
					"text": map[string]string{
						"type": "plain_text",
						"text": "View Details",
					},
					"url": msg.ActionURL,
				},
			},
		})
	}

	payload := map[string]any{
		"text":   msg.Subject, // Fallback for notifications
		"blocks": blocks,
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
		return fmt.Errorf("slack request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("slack webhook returned status %d: %s", resp.StatusCode, string(respBytes))
	}

	return nil
}
