package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/SherClockHolmes/webpush-go"
	"github.com/alexandrmotologa/notiflux/internal/api"
	"github.com/alexandrmotologa/notiflux/internal/channels"
	"github.com/alexandrmotologa/notiflux/internal/config"
	"github.com/alexandrmotologa/notiflux/internal/engine"
	"github.com/alexandrmotologa/notiflux/internal/store"
)

var version = "1.0.0"

func main() {
	args := os.Args[1:]

	if len(args) > 0 {
		switch args[0] {
		case "version", "--version", "-v":
			fmt.Printf("notiflux v%s (%s/%s)\n", version, runtime.GOOS, runtime.GOARCH)
			return
		case "vapid", "generate-vapid":
			runVAPID()
			return
		case "help", "--help", "-h":
			printUsage()
			return
		case "serve":
			args = args[1:]
		}
	}

	runServe(args)
}

func printUsage() {
	fmt.Printf(`notiflux v%s - Self-Hosted Multi-Channel Notification Engine (< 45MB RAM)

Usage:
  notiflux [command] [flags]

Commands:
  serve              Start the notification daemon and embedded studio (default)
  vapid              Generate RFC 8292 ECDSA P-256 VAPID keys for Web Push
  version            Print version information
  help               Show this help message

Flags for 'serve':
  --port             HTTP server port (default: 8085, env: NOTIFLUX_PORT)
  --db               Path to SQLite database (default: notiflux.db, env: NOTIFLUX_DB_PATH)
  --api-key          Master API key for administrative operations (env: NOTIFLUX_API_KEY)
  --allowed-origins  CORS allowed origins (default: *, env: NOTIFLUX_ALLOWED_ORIGINS)
  --smtp-host        Outbound SMTP server host (env: NOTIFLUX_SMTP_HOST)
  --smtp-port        Outbound SMTP server port (default: 587, env: NOTIFLUX_SMTP_PORT)
  --smtp-user        SMTP authentication username (env: NOTIFLUX_SMTP_USER)
  --smtp-pass        SMTP authentication password (env: NOTIFLUX_SMTP_PASS)
  --smtp-from        Default sender email (env: NOTIFLUX_SMTP_FROM)
  --resend-api-key   Resend REST API key (env: NOTIFLUX_RESEND_API_KEY)
  --vapid-public-key RFC 8292 VAPID public key (env: NOTIFLUX_VAPID_PUBLIC_KEY)
  --vapid-private-key RFC 8292 VAPID private key (env: NOTIFLUX_VAPID_PRIVATE_KEY)
  --vapid-subject    VAPID mailto subject (env: NOTIFLUX_VAPID_SUBJECT)
  --telegram-token   Telegram Bot API token (env: NOTIFLUX_TELEGRAM_BOT_TOKEN)
  --slack-webhook-url Default Slack webhook URL (env: NOTIFLUX_SLACK_WEBHOOK_URL)
  --webhook-secret   Shared secret for outbound HMAC signatures (env: NOTIFLUX_WEBHOOK_SECRET)
`, version)
}

func runVAPID() {
	privateKey, publicKey, err := webpush.GenerateVAPIDKeys()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating VAPID keys: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("=== Generated RFC 8292 VAPID Key Pair ===")
	fmt.Printf("Public Key:  %s\n", publicKey)
	fmt.Printf("Private Key: %s\n\n", privateKey)
	fmt.Println("Export environment variables:")
	fmt.Printf("export NOTIFLUX_VAPID_PUBLIC_KEY=\"%s\"\n", publicKey)
	fmt.Printf("export NOTIFLUX_VAPID_PRIVATE_KEY=\"%s\"\n", privateKey)
	fmt.Printf("export NOTIFLUX_VAPID_SUBJECT=\"mailto:admin@yourdomain.com\"\n")
}

func runServe(args []string) {
	cfg, err := config.LoadConfig(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Configuration error: %v\n", err)
		os.Exit(1)
	}

	st, err := store.NewStore(cfg.DBPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Database initialization error: %v\n", err)
		os.Exit(1)
	}
	defer st.Close()

	// Seed starter workflows if empty
	seedStarterWorkflows(st)

	hub := channels.NewSSEHub()
	pool := channels.NewDispatcherPool(cfg, st, hub)
	eng := engine.NewWorkflowEngine(st)
	idem := engine.NewIdempotencyEngine(st)

	routerConfig := &api.RouterConfig{
		Config:      cfg,
		Store:       st,
		SSEHub:      hub,
		Pool:        pool,
		Engine:      eng,
		Idempotency: idem,
	}

	router := api.NewRouter(routerConfig)

	addr := fmt.Sprintf(":%d", cfg.Port)
	server := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Memory telemetry
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	allocMB := float64(m.Alloc) / 1024 / 1024

	fmt.Println("-----------------------------------------------------------------")
	fmt.Printf("  Notiflux Notification Engine v%s\n", version)
	fmt.Println("  Single binary alternative to Novu, Knock, and Courier (< 45MB RAM)")
	fmt.Println("-----------------------------------------------------------------")
	fmt.Printf("  • Server listening on:   http://localhost:%d\n", cfg.Port)
	fmt.Printf("  • Management Studio:    http://localhost:%d/studio\n", cfg.Port)
	fmt.Printf("  • Drop-in Client Bell:  http://localhost:%d/widget/notiflux-bell.js\n", cfg.Port)
	fmt.Printf("  • SQLite Database:      %s (WAL mode)\n", cfg.DBPath)
	fmt.Printf("  • Initial Memory Footprint: %.2f MB RAM\n", allocMB)
	fmt.Println("-----------------------------------------------------------------")

	// Graceful shutdown handling
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "Server listen error: %v\n", err)
			os.Exit(1)
		}
	}()

	<-stop
	fmt.Println("\nShutting down Notiflux daemon gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Server shutdown error: %v\n", err)
	}

	fmt.Println("Notiflux daemon stopped.")
}

func seedStarterWorkflows(s *store.Store) {
	list, err := s.ListWorkflows()
	if err == nil && len(list) == 0 {
		// 1. Order Status Workflow
		orderWf := &store.Workflow{
			ID:          "order.status",
			Name:        "Order Status Updates",
			Description: "Transactional order updates across In-App, Email, and Push",
			Channels:    []string{store.ChannelInApp, store.ChannelEmail, store.ChannelWebPush},
		}
		_ = s.CreateWorkflow(orderWf)
		_ = s.UpsertTemplate(&store.Template{
			WorkflowID: "order.status",
			Channel:    store.ChannelInApp,
			Subject:    "Order {{order_id}} Update",
			Body:       "Your order of {{total}} has been updated.",
		})
		_ = s.UpsertTemplate(&store.Template{
			WorkflowID: "order.status",
			Channel:    store.ChannelEmail,
			Subject:    "Order {{order_id}} Status Confirmation",
			Body:       "## Order Update\n\nYour order **{{order_id}}** totaling {{total}} is now confirmed.\n\n[Track Order]({{action_url}})",
		})

		// 2. Security Alert Workflow
		secWf := &store.Workflow{
			ID:          "security.alert",
			Name:        "Security & Login Alerts",
			Description: "High-priority security alerts dispatched via Telegram and In-App",
			Channels:    []string{store.ChannelInApp, store.ChannelTelegram, store.ChannelEmail},
		}
		_ = s.CreateWorkflow(secWf)
		_ = s.UpsertTemplate(&store.Template{
			WorkflowID: "security.alert",
			Channel:    store.ChannelInApp,
			Subject:    "Security Alert: New Sign-In",
			Body:       "New sign-in detected on your account.",
		})
		_ = s.UpsertTemplate(&store.Template{
			WorkflowID: "security.alert",
			Channel:    store.ChannelTelegram,
			Subject:    "Security Alert",
			Body:       "<b>Security Alert</b>\n\nNew sign-in detected on your account.",
		})
	}
}
