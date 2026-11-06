package channels

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/SherClockHolmes/webpush-go"
	"github.com/alexandrmotologa/notiflux/internal/config"
	"github.com/alexandrmotologa/notiflux/internal/engine"
	"github.com/alexandrmotologa/notiflux/internal/store"
)

// WebPushDispatcher sends RFC 8292 VAPID-signed push notifications to browser push services.
type WebPushDispatcher struct {
	cfg *config.Config
}

func NewWebPushDispatcher(cfg *config.Config) *WebPushDispatcher {
	return &WebPushDispatcher{cfg: cfg}
}

func (d *WebPushDispatcher) Channel() string {
	return store.ChannelWebPush
}

func (d *WebPushDispatcher) Dispatch(ctx context.Context, msg *engine.CompiledMessage) error {
	if msg.Subscriber == nil || msg.Subscriber.WebPushSubscription == nil {
		return fmt.Errorf("web push subscription is required")
	}

	sub := msg.Subscriber.WebPushSubscription
	if sub.Endpoint == "" || sub.Keys.P256dh == "" || sub.Keys.Auth == "" {
		return fmt.Errorf("invalid push subscription: endpoint or keys missing")
	}

	if d.cfg.VAPIDPublicKey == "" || d.cfg.VAPIDPrivateKey == "" {
		return fmt.Errorf("vapid keys not configured: set NOTIFLUX_VAPID_PUBLIC_KEY and NOTIFLUX_VAPID_PRIVATE_KEY")
	}

	payloadMap := map[string]any{
		"title":      msg.Subject,
		"body":       msg.Body,
		"action_url": msg.ActionURL,
		"data":       msg.Payload,
	}

	payloadBytes, err := json.Marshal(payloadMap)
	if err != nil {
		return fmt.Errorf("failed encoding push payload: %w", err)
	}

	subscription := &webpush.Subscription{
		Endpoint: sub.Endpoint,
		Keys: webpush.Keys{
			P256dh: sub.Keys.P256dh,
			Auth:   sub.Keys.Auth,
		},
	}

	subject := d.cfg.VAPIDSubject
	if subject == "" {
		subject = "mailto:admin@notiflux.dev"
	}

	resp, err := webpush.SendNotificationWithContext(ctx, payloadBytes, subscription, &webpush.Options{
		Subscriber:      subject,
		VAPIDPublicKey:  d.cfg.VAPIDPublicKey,
		VAPIDPrivateKey: d.cfg.VAPIDPrivateKey,
		TTL:             86400,
		Urgency:         webpush.UrgencyNormal,
	})
	if err != nil {
		return fmt.Errorf("web push delivery failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusGone {
		return fmt.Errorf("subscription expired (410 Gone)")
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("web push service returned status %d", resp.StatusCode)
	}

	return nil
}
