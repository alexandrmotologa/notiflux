package engine

import (
	"fmt"
	"time"

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
		case store.ChannelDiscord:
			if sub.DiscordWebhookURL == "" {
				skippedReasons[ch] = "missing_discord_webhook_url"
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

// IsInQuietHours checks if the subscriber has quiet hours active at the given time.
// If active, it returns true and the time when the quiet window ends.
func IsInQuietHours(sub *store.Subscriber, now time.Time) (bool, time.Time) {
	if sub == nil || sub.QuietHoursStart == "" || sub.QuietHoursEnd == "" {
		return false, time.Time{}
	}

	loc := time.UTC
	if sub.Timezone != "" {
		if l, err := time.LoadLocation(sub.Timezone); err == nil {
			loc = l
		}
	}

	localNow := now.In(loc)
	startH, startM, err1 := parseHourMinute(sub.QuietHoursStart)
	endH, endM, err2 := parseHourMinute(sub.QuietHoursEnd)
	if err1 != nil || err2 != nil {
		return false, time.Time{}
	}

	currentMinutes := localNow.Hour()*60 + localNow.Minute()
	startMinutes := startH*60 + startM
	endMinutes := endH*60 + endM

	inQuiet := false
	var endYear, endMonth, endDay int
	endYear, endMonth, endDay = localNow.Year(), int(localNow.Month()), localNow.Day()

	if startMinutes < endMinutes {
		// e.g. 01:00 to 06:00
		inQuiet = currentMinutes >= startMinutes && currentMinutes < endMinutes
	} else {
		// e.g. 22:00 to 08:00 (spans midnight)
		inQuiet = currentMinutes >= startMinutes || currentMinutes < endMinutes
		if currentMinutes >= startMinutes {
			// Quiet hours end next day
			nextDay := localNow.AddDate(0, 0, 1)
			endYear, endMonth, endDay = nextDay.Year(), int(nextDay.Month()), nextDay.Day()
		}
	}

	if inQuiet {
		endTime := time.Date(endYear, time.Month(endMonth), endDay, endH, endM, 0, 0, loc).UTC()
		return true, endTime
	}

	return false, time.Time{}
}

func parseHourMinute(s string) (int, int, error) {
	var h, m int
	n, err := fmt.Sscanf(s, "%d:%d", &h, &m)
	if err != nil || n != 2 || h < 0 || h > 23 || m < 0 || m > 59 {
		return 0, 0, fmt.Errorf("invalid time format %q, expected HH:MM", s)
	}
	return h, m, nil
}
