package config

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds runtime configuration options for the Notiflux daemon.
type Config struct {
	Port           int
	DBPath         string
	APIKey         string
	AllowedOrigins []string

	// Email (SMTP & Resend)
	SMTPHost       string
	SMTPPort       int
	SMTPUser       string
	SMTPPass       string
	SMTPFrom       string
	SMTPRequireTLS bool
	ResendAPIKey   string

	// Web Push (RFC 8292 VAPID)
	VAPIDPublicKey  string
	VAPIDPrivateKey string
	VAPIDSubject    string

	// Telegram
	TelegramBotToken string

	// Slack
	SlackWebhookURL string

	// Outbound Webhooks
	WebhookSecret string
}

// LoadConfig parses flags and environment variables into a Config instance.
// Environment variables take precedence if set, otherwise flag or default values are used.
func LoadConfig(args []string) (*Config, error) {
	fs := flag.NewFlagSet("notiflux", flag.ContinueOnError)

	portFlag := fs.Int("port", 8085, "HTTP server port (env: NOTIFLUX_PORT)")
	dbPathFlag := fs.String("db", "notiflux.db", "SQLite database path (env: NOTIFLUX_DB_PATH)")
	apiKeyFlag := fs.String("api-key", "", "Master API Key for admin endpoints (env: NOTIFLUX_API_KEY)")
	originsFlag := fs.String("allowed-origins", "*", "Comma-separated CORS allowed origins (env: NOTIFLUX_ALLOWED_ORIGINS)")

	smtpHostFlag := fs.String("smtp-host", "", "SMTP server host (env: NOTIFLUX_SMTP_HOST)")
	smtpPortFlag := fs.Int("smtp-port", 587, "SMTP server port (env: NOTIFLUX_SMTP_PORT)")
	smtpUserFlag := fs.String("smtp-user", "", "SMTP username (env: NOTIFLUX_SMTP_USER)")
	smtpPassFlag := fs.String("smtp-pass", "", "SMTP password (env: NOTIFLUX_SMTP_PASS)")
	smtpFromFlag := fs.String("smtp-from", "", "Default sender email address (env: NOTIFLUX_SMTP_FROM)")
	smtpTLSFlag := fs.Bool("smtp-require-tls", true, "Require TLS for SMTP (env: NOTIFLUX_SMTP_REQUIRE_TLS)")

	resendAPIKeyFlag := fs.String("resend-api-key", "", "Resend REST API key (env: NOTIFLUX_RESEND_API_KEY)")

	vapidPublicFlag := fs.String("vapid-public-key", "", "VAPID public key (env: NOTIFLUX_VAPID_PUBLIC_KEY)")
	vapidPrivateFlag := fs.String("vapid-private-key", "", "VAPID private key (env: NOTIFLUX_VAPID_PRIVATE_KEY)")
	vapidSubjectFlag := fs.String("vapid-subject", "", "VAPID mailto subject (env: NOTIFLUX_VAPID_SUBJECT)")

	telegramTokenFlag := fs.String("telegram-token", "", "Telegram Bot API token (env: NOTIFLUX_TELEGRAM_BOT_TOKEN)")
	slackWebhookFlag := fs.String("slack-webhook-url", "", "Default Slack webhook URL (env: NOTIFLUX_SLACK_WEBHOOK_URL)")
	webhookSecretFlag := fs.String("webhook-secret", "", "Shared secret for outbound webhook HMAC signatures (env: NOTIFLUX_WEBHOOK_SECRET)")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	cfg := &Config{
		Port:             getEnvInt("NOTIFLUX_PORT", *portFlag),
		DBPath:           getEnvString("NOTIFLUX_DB_PATH", *dbPathFlag),
		APIKey:           getEnvString("NOTIFLUX_API_KEY", *apiKeyFlag),
		SMTPHost:         getEnvString("NOTIFLUX_SMTP_HOST", *smtpHostFlag),
		SMTPPort:         getEnvInt("NOTIFLUX_SMTP_PORT", *smtpPortFlag),
		SMTPUser:         getEnvString("NOTIFLUX_SMTP_USER", *smtpUserFlag),
		SMTPPass:         getEnvString("NOTIFLUX_SMTP_PASS", *smtpPassFlag),
		SMTPFrom:         getEnvString("NOTIFLUX_SMTP_FROM", *smtpFromFlag),
		SMTPRequireTLS:   getEnvBool("NOTIFLUX_SMTP_REQUIRE_TLS", *smtpTLSFlag),
		ResendAPIKey:     getEnvString("NOTIFLUX_RESEND_API_KEY", *resendAPIKeyFlag),
		VAPIDPublicKey:  getEnvString("NOTIFLUX_VAPID_PUBLIC_KEY", *vapidPublicFlag),
		VAPIDPrivateKey: getEnvString("NOTIFLUX_VAPID_PRIVATE_KEY", *vapidPrivateFlag),
		VAPIDSubject:    getEnvString("NOTIFLUX_VAPID_SUBJECT", *vapidSubjectFlag),
		TelegramBotToken: getEnvString("NOTIFLUX_TELEGRAM_BOT_TOKEN", *telegramTokenFlag),
		SlackWebhookURL: getEnvString("NOTIFLUX_SLACK_WEBHOOK_URL", *slackWebhookFlag),
		WebhookSecret:   getEnvString("NOTIFLUX_WEBHOOK_SECRET", *webhookSecretFlag),
	}

	originsRaw := getEnvString("NOTIFLUX_ALLOWED_ORIGINS", *originsFlag)
	if originsRaw == "*" {
		cfg.AllowedOrigins = []string{"*"}
	} else {
		for _, o := range strings.Split(originsRaw, ",") {
			trimmed := strings.TrimSpace(o)
			if trimmed != "" {
				cfg.AllowedOrigins = append(cfg.AllowedOrigins, trimmed)
			}
		}
	}

	if cfg.Port <= 0 || cfg.Port > 65535 {
		return nil, fmt.Errorf("invalid port number: %d", cfg.Port)
	}

	return cfg, nil
}

func getEnvString(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok && strings.TrimSpace(val) != "" {
		return strings.TrimSpace(val)
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if val, ok := os.LookupEnv(key); ok {
		if i, err := strconv.Atoi(strings.TrimSpace(val)); err == nil {
			return i
		}
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	if val, ok := os.LookupEnv(key); ok {
		lower := strings.ToLower(strings.TrimSpace(val))
		return lower == "1" || lower == "true" || lower == "yes"
	}
	return fallback
}
