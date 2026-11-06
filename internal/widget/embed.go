package widget

import (
	"embed"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

//go:embed dist/*
var widgetFS embed.FS

// MountWidgetRoutes registers endpoints serving the pre-bundled widget files.
func MountWidgetRoutes(r chi.Router) {
	r.Get("/widget/*", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/widget/")
		if path == "" || path == "/" {
			path = "notiflux-bell.js"
		}

		data, err := widgetFS.ReadFile("dist/" + path)
		if err != nil {
			http.NotFound(w, r)
			return
		}

		if strings.HasSuffix(path, ".js") {
			w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
		} else if strings.HasSuffix(path, ".css") {
			w.Header().Set("Content-Type", "text/css; charset=utf-8")
		}

		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Cache-Control", "public, max-age=3600")
		w.WriteHeader(http.StatusOK)
		w.Write(data)
	})
}
