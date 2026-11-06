package api

import (
	"net/http"
	"strings"

	"github.com/alexandrmotologa/notiflux/internal/channels"
	"github.com/alexandrmotologa/notiflux/internal/config"
	"github.com/alexandrmotologa/notiflux/internal/engine"
	"github.com/alexandrmotologa/notiflux/internal/store"
	"github.com/alexandrmotologa/notiflux/internal/studio"
	"github.com/alexandrmotologa/notiflux/internal/widget"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

// RouterConfig gathers dependencies required to mount all HTTP handlers.
type RouterConfig struct {
	Config      *config.Config
	Store       *store.Store
	SSEHub      *channels.SSEHub
	Pool        *channels.DispatcherPool
	Engine      *engine.WorkflowEngine
	Idempotency *engine.IdempotencyEngine
}

// NewRouter constructs and configures the top-level Chi HTTP router.
func NewRouter(rc *RouterConfig) *chi.Mux {
	r := chi.NewRouter()

	// Base middleware stack
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// CORS middleware
	allowedOrigins := rc.Config.AllowedOrigins
	if len(allowedOrigins) == 0 {
		allowedOrigins = []string{"*"}
	}
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   allowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-API-Key", "Idempotency-Key"},
		ExposedHeaders:   []string{"Link", "X-Cache", "Content-Length"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Handlers
	triggerHandler := NewTriggerHandler(rc.Engine, rc.Pool, rc.Idempotency)
	feedHandler := NewFeedHandler(rc.Store, rc.SSEHub)
	prefsHandler := NewPreferencesHandler(rc.Store)
	wfHandler := NewWorkflowHandler(rc.Store)
	logsHandler := NewLogsHandler(rc.Store)
	systemHandler := NewSystemHandler(rc.Config, rc.SSEHub, rc.Pool)

	// Mount public widget routes
	widget.MountWidgetRoutes(r)

	// Mount embedded management studio routes
	studio.MountStudioRoutes(r)

	// API v1 routes
	r.Route("/v1", func(r chi.Router) {
		// Public health check
		r.Get("/health", systemHandler.Health)

		// Public client feed & SSE routes
		r.Route("/feed", func(r chi.Router) {
			r.Get("/", feedHandler.GetFeed)
			r.Get("/stream", feedHandler.StreamSSE)
			r.Patch("/{id}/read", feedHandler.MarkAsRead)
			r.Patch("/{id}/unread", feedHandler.MarkAsUnread)
			r.Post("/read-all", feedHandler.MarkAllAsRead)
			r.Delete("/{id}", feedHandler.DeleteNotification)
		})

		// Protected administrative & ingestion routes
		r.Group(func(r chi.Router) {
			r.Use(AuthMiddleware(rc.Config.APIKey))

			// Triggers
			r.Post("/trigger", triggerHandler.Trigger)
			r.Post("/trigger/batch", triggerHandler.TriggerBatch)

			// Preferences
			r.Get("/preferences/{subscriber_id}", prefsHandler.GetPreferences)
			r.Put("/preferences/{subscriber_id}", prefsHandler.UpdatePreferences)

			// Workflows & Templates
			r.Get("/workflows", wfHandler.ListWorkflows)
			r.Post("/workflows", wfHandler.CreateWorkflow)
			r.Get("/workflows/{id}", wfHandler.GetWorkflow)
			r.Put("/workflows/{id}", wfHandler.UpdateWorkflow)
			r.Delete("/workflows/{id}", wfHandler.DeleteWorkflow)
			r.Get("/workflows/{id}/templates", wfHandler.GetTemplates)
			r.Put("/workflows/{id}/templates", wfHandler.UpdateTemplates)
			r.Post("/templates/preview", wfHandler.PreviewTemplate)

			// Delivery Logs & Subscribers
			r.Get("/logs", logsHandler.ListLogs)
			r.Get("/subscribers", logsHandler.ListSubscribers)
			r.Get("/subscribers/{id}", logsHandler.GetSubscriber)

			// System Utilities
			r.Post("/system/vapid", systemHandler.GenerateVAPID)
			r.Post("/system/test-channel", systemHandler.TestChannel)
		})
	})

	return r
}

// AuthMiddleware validates API keys when configured.
func AuthMiddleware(apiKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if apiKey == "" {
				next.ServeHTTP(w, r)
				return
			}

			// Check Bearer token
			authHeader := r.Header.Get("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				token := strings.TrimPrefix(authHeader, "Bearer ")
				if token == apiKey {
					next.ServeHTTP(w, r)
					return
				}
			}

			// Check X-API-Key header
			if r.Header.Get("X-API-Key") == apiKey {
				next.ServeHTTP(w, r)
				return
			}

			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized: valid API key required"})
		})
	}
}
