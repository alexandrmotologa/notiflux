package engine

import (
	"github.com/alexandrmotologa/notiflux/internal/store"
)

// PreferenceEvaluator determines which channels are eligible for dispatch.
type PreferenceEvaluator struct {
	store *store.Store
}

func NewPreferenceEvaluator(s *store.Store) *PreferenceEvaluator {
	return &PreferenceEvaluator{store: s}
}

// FilterActiveChannels checks workflow channels against user preferences, overrides, and subscriber credentials.
func (pe *PreferenceEvaluator) FilterActiveChannels(
	workflow *store.Workflow,
	sub *store.Subscriber,
	overrides []string,
) (activeChannels []string, skippedReasons map[string]string, err error) {
	skippedReasons = make(map[string]string)

	candidateChannels := workflow.Channels
	if len(overrides) > 0 {
		candidateChannels = overrides
	}

	for _, ch := range candidateChannels {
		// 1. Check user preferences
		enabled, err := pe.store.IsChannelEnabled(sub.ID, ch)
		if err != nil {
			return nil, nil, err
		}
		if !enabled {
			skippedReasons[ch] = "user_opt_out"
			continue
		}

		// 2. Check credentials availability
		switch ch {
		case store.ChannelInApp:
			if sub.ID == "" {
				skippedReasons[ch] = "missing_subscriber_id"
				continue
			}
		case store.ChannelEmail:
			if sub.Email == "" {
				skippedReasons[ch] = "missing_email_address"
				continue
			}
		case store.ChannelTelegram:
			if sub.TelegramChatID == "" {
				skippedReasons[ch] = "missing_telegram_chat_id"
				continue
			}
		case store.ChannelSlack:
			if sub.SlackWebhookURL == "" {
				// can fallback to global if configured, otherwise missing
				skippedReasons[ch] = "missing_slack_webhook_url"
				continue
			}
		case store.ChannelWebPush:
			if sub.WebPushSubscription == nil || sub.WebPushSubscription.Endpoint == "" {
				skippedReasons[ch] = "missing_web_push_subscription"
				continue
			}
		case store.ChannelWebhook:
			// Webhooks are usually driven by workflow configuration or payload
		}

		activeChannels = append(activeChannels, ch)
	}

	return activeChannels, skippedReasons, nil
}
