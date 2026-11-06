package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/alexandrmotologa/notiflux/internal/store"
)

func main() {
	dbPath := "demo.db"
	_ = os.Remove(dbPath)

	st, err := store.NewStore(dbPath)
	if err != nil {
		log.Fatalf("failed creating demo store: %v", err)
	}
	defer st.Close()

	now := time.Now().UTC()

	// 1. Workflows
	workflows := []*store.Workflow{
		{
			ID:          "order.shipped",
			Name:        "Order Shipped",
			Description: "Triggered when logistics marks an order in transit",
			Channels:    []string{store.ChannelInApp, store.ChannelEmail, store.ChannelTelegram},
			CreatedAt:   now.Add(-48 * time.Hour),
			UpdatedAt:   now.Add(-2 * time.Hour),
		},
		{
			ID:          "security.alert",
			Name:        "Security & Auth Alert",
			Description: "High-priority alerts for new device logins and password resets",
			Channels:    []string{store.ChannelInApp, store.ChannelDiscord, store.ChannelWebPush},
			CreatedAt:   now.Add(-36 * time.Hour),
			UpdatedAt:   now.Add(-1 * time.Hour),
		},
		{
			ID:          "billing.invoice",
			Name:        "Invoice Receipt",
			Description: "Automated monthly recurring billing statements",
			Channels:    []string{store.ChannelInApp, store.ChannelEmail, store.ChannelSlack},
			CreatedAt:   now.Add(-24 * time.Hour),
			UpdatedAt:   now.Add(-30 * time.Minute),
		},
		{
			ID:          "infra.incident",
			Name:        "Infrastructure Telemetry",
			Description: "Production threshold breaches and node degradations",
			Channels:    []string{store.ChannelInApp, store.ChannelDiscord, store.ChannelWebhook},
			CreatedAt:   now.Add(-12 * time.Hour),
			UpdatedAt:   now.Add(-10 * time.Minute),
		},
	}

	for _, wf := range workflows {
		if err := st.CreateWorkflow(wf); err != nil {
			log.Fatalf("failed creating workflow %s: %v", wf.ID, err)
		}
	}

	// 2. Templates
	templates := []*store.Template{
		{
			WorkflowID: "order.shipped",
			Channel:    store.ChannelInApp,
			Subject:    "Order #{{order_id}} has shipped",
			Body:       "Your package is with {{carrier}}, tracking: {{tracking_code}}.",
		},
		{
			WorkflowID: "order.shipped",
			Channel:    store.ChannelEmail,
			Subject:    "Your order #{{order_id}} is on its way",
			Body:       "## Good news!\n\nYour order **#{{order_id}}** has left the fulfillment center via {{carrier}}.\n\n[Track Your Package]({{action_url}})",
		},
		{
			WorkflowID: "security.alert",
			Channel:    store.ChannelInApp,
			Subject:    "New Login Detected",
			Body:       "A login occurred from {{location}} on {{device}}.",
		},
		{
			WorkflowID: "security.alert",
			Channel:    store.ChannelDiscord,
			Subject:    "Security Alert: {{account_id}}",
			Body:       "A new session was authenticated from IP {{ip_address}} ({{location}}).",
		},
		{
			WorkflowID: "billing.invoice",
			Channel:    store.ChannelEmail,
			Subject:    "Invoice #{{inv_num}} Statement",
			Body:       "# Payment Cleared\n\nYour payment of **${{amount}}** for {{plan_name}} has succeeded.\n\n[Download PDF]({{action_url}})",
		},
	}

	for _, tpl := range templates {
		_ = st.UpsertTemplate(tpl)
	}

	// 3. Subscribers
	subscribers := []*store.Subscriber{
		{
			ID:                "usr_demo",
			Email:             "alexandr@mtlg.site",
			TelegramChatID:    "108429104",
			SlackWebhookURL:   "https://hooks.slack.com/services/demo/ops",
			DiscordWebhookURL: "https://discord.com/api/webhooks/demo/alerts",
			Timezone:          "Europe/Bucharest",
			QuietHoursStart:   "23:00",
			QuietHoursEnd:     "07:30",
			CreatedAt:         now.Add(-72 * time.Hour),
		},
		{
			ID:                "usr_sarah",
			Email:             "sarah@acme.corp",
			SlackWebhookURL:   "https://hooks.slack.com/services/demo/sarah",
			Timezone:          "America/New_York",
			QuietHoursStart:   "22:00",
			QuietHoursEnd:     "08:00",
			CreatedAt:         now.Add(-48 * time.Hour),
		},
		{
			ID:                "usr_devops",
			Email:             "oncall@infra.net",
			DiscordWebhookURL: "https://discord.com/api/webhooks/demo/oncall",
			Timezone:          "UTC",
			CreatedAt:         now.Add(-24 * time.Hour),
		},
	}

	for _, sub := range subscribers {
		_ = st.UpsertSubscriber(sub)
	}

	// 4. In-App Notifications for usr_demo
	notifications := []*store.Notification{
		{
			ID:           "notif_demo_1",
			WorkflowID:   "security.alert",
			SubscriberID: "usr_demo",
			Title:        "Security Alert: New login from Frankfurt, Germany",
			Body:         "We detected a new session from Chrome on Linux (IP 194.26.29.112).",
			ActionURL:    "https://example.com/security/sessions",
			IsRead:       false,
			CreatedAt:    now.Add(-8 * time.Minute),
		},
		{
			ID:           "notif_demo_2",
			WorkflowID:   "billing.invoice",
			SubscriberID: "usr_demo",
			Title:        "Invoice #1042 successfully paid",
			Body:         "Monthly Pro plan renewal of $49.00 has cleared.",
			ActionURL:    "https://example.com/invoices/1042",
			IsRead:       false,
			CreatedAt:    now.Add(-42 * time.Minute),
		},
		{
			ID:           "notif_demo_3",
			WorkflowID:   "order.shipped",
			SubscriberID: "usr_demo",
			Title:        "Order #8942 has shipped via DHL Express",
			Body:         "Estimated delivery: Tomorrow by 18:00.",
			ActionURL:    "https://example.com/orders/8942",
			IsRead:       true,
			CreatedAt:    now.Add(-5 * time.Hour),
		},
	}

	for _, n := range notifications {
		_ = st.CreateNotification(n)
	}

	// 5. Delivery Logs with Engagement Tracking
	tOpened1 := now.Add(-6 * time.Minute)
	tClicked1 := now.Add(-5 * time.Minute)
	tOpened2 := now.Add(-25 * time.Minute)

	logs := []*store.DeliveryLog{
		{
			ID:             "dl_1",
			NotificationID: "notif_demo_1",
			WorkflowID:     "security.alert",
			SubscriberID:   "usr_demo",
			Channel:        store.ChannelInApp,
			Status:         store.StatusSent,
			Payload:        `{"ip":"194.26.29.112","device":"Linux / Chrome"}`,
			DispatchedAt:   now.Add(-8 * time.Minute),
		},
		{
			ID:             "dl_2",
			NotificationID: "notif_demo_1",
			WorkflowID:     "security.alert",
			SubscriberID:   "usr_demo",
			Channel:        store.ChannelDiscord,
			Status:         store.StatusSent,
			Payload:        `{"channel":"#security-ops","status":"alert"}`,
			DispatchedAt:   now.Add(-8 * time.Minute),
		},
		{
			ID:             "dl_3",
			NotificationID: "notif_demo_2",
			WorkflowID:     "billing.invoice",
			SubscriberID:   "usr_demo",
			Channel:        store.ChannelEmail,
			Status:         store.StatusSent,
			TrackingID:     "trk_inv_1042",
			TargetURL:      "https://example.com/invoices/1042",
			OpenedAt:       &tOpened1,
			ClickedAt:      &tClicked1,
			Payload:        `{"invoice":"#1042","amount":"$49.00"}`,
			DispatchedAt:   now.Add(-42 * time.Minute),
		},
		{
			ID:             "dl_4",
			NotificationID: "notif_demo_3",
			WorkflowID:     "order.shipped",
			SubscriberID:   "usr_demo",
			Channel:        store.ChannelEmail,
			Status:         store.StatusSent,
			TrackingID:     "trk_ord_8942",
			TargetURL:      "https://example.com/orders/8942",
			OpenedAt:       &tOpened2,
			Payload:        `{"carrier":"DHL Express","tracking":"DHL-98124"}`,
			DispatchedAt:   now.Add(-5 * time.Hour),
		},
		{
			ID:             "dl_5",
			WorkflowID:     "order.shipped",
			SubscriberID:   "usr_sarah",
			Channel:        store.ChannelTelegram,
			Status:         store.StatusSent,
			Payload:        `{"chat_id":"108429104","message":"Package in transit"}`,
			DispatchedAt:   now.Add(-6 * time.Hour),
		},
		{
			ID:             "dl_6",
			WorkflowID:     "infra.incident",
			SubscriberID:   "usr_devops",
			Channel:        store.ChannelDiscord,
			Status:         store.StatusSent,
			Payload:        `{"alert":"High memory threshold 92% on worker-03"}`,
			DispatchedAt:   now.Add(-10 * time.Minute),
		},
		{
			ID:             "dl_7",
			WorkflowID:     "infra.incident",
			SubscriberID:   "usr_devops",
			Channel:        store.ChannelWebhook,
			Status:         store.StatusFailed,
			ErrorMessage:   "connection timeout after 10000ms",
			Payload:        `{"endpoint":"https://pager.corp/hook"}`,
			DispatchedAt:   now.Add(-9 * time.Minute),
		},
	}

	for _, l := range logs {
		_ = st.RecordDeliveryLog(l)
	}

	// 6. Workspace API Keys
	_, _ = st.CreateAPIKey("Production Backend API")
	_, _ = st.CreateAPIKey("GitHub Actions CI/CD")

	fmt.Println("✓ Successfully seeded demo.db with realistic workflows, logs, tracking, and subscribers")
}
