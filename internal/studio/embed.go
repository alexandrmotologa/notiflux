package studio

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

//go:embed dist/*
var studioFS embed.FS

// MountStudioRoutes registers endpoints serving the embedded administration studio SPA.
func MountStudioRoutes(r chi.Router) {
	subFS, err := fs.Sub(studioFS, "dist")
	if err != nil {
		return
	}

	fileServer := http.FileServer(http.FS(subFS))

	// Handler for /studio and /
	handler := func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/studio")
		if path == "" || path == "/" {
			path = "/index.html"
		}

		// Check if file exists in embedded assets
		f, err := subFS.Open(strings.TrimPrefix(path, "/"))
		if err == nil {
			_ = f.Close()
			// Serve static asset with appropriate caching
			if strings.HasPrefix(path, "/assets/") {
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			}
			http.StripPrefix("/studio", fileServer).ServeHTTP(w, r)
			return
		}

		// Fallback to index.html for SPA client-side routing
		indexData, err := studioFS.ReadFile("dist/index.html")
		if err != nil {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write(indexData)
	}

	r.Get("/studio", handler)
	r.Get("/studio/*", handler)

	// Also serve at root / if not hitting /v1 or /widget
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		indexData, err := studioFS.ReadFile("dist/index.html")
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write(indexData)
	})

	r.Get("/assets/*", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		fileServer.ServeHTTP(w, r)
	})
}
