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

// TelegramDispatcher delivers notifications via Telegram Bot API.
type TelegramDispatcher struct {
	cfg        *config.Config
	httpClient *http.Client
}

func NewTelegramDispatcher(cfg *config.Config) *TelegramDispatcher {
	return &TelegramDispatcher{
		cfg: cfg,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (d *TelegramDispatcher) Channel() string {
	return store.ChannelTelegram
}

func (d *TelegramDispatcher) Dispatch(ctx context.Context, msg *engine.CompiledMessage) error {
	if msg.Subscriber == nil || msg.Subscriber.TelegramChatID == "" {
		return fmt.Errorf("subscriber telegram_chat_id is required")
	}

	botToken := d.cfg.TelegramBotToken
	if botToken == "" {
		return fmt.Errorf("telegram bot token not configured: set NOTIFLUX_TELEGRAM_BOT_TOKEN")
	}

	text := msg.Body
	if msg.Subject != "" && msg.Subject != msg.Body {
		text = fmt.Sprintf("<b>%s</b>\n\n%s", msg.Subject, msg.Body)
	}

	reqBody := map[string]any{
		"chat_id":    msg.Subscriber.TelegramChatID,
		"text":       text,
		"parse_mode": "HTML",
	}

	// Add inline keyboard button if ActionURL is present
	if msg.ActionURL != "" {
		reqBody["reply_markup"] = map[string]any{
			"inline_keyboard": [][]map[string]string{
				{
					{
						"text": "Open",
						"url":  msg.ActionURL,
					},
				},
			},
		}
	}

	b, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", botToken)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := d.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("telegram request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		respBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("telegram API returned status %d: %s", resp.StatusCode, string(respBytes))
	}

	return nil
}
