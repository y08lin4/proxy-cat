package assets

import (
	"net/http"
	"strings"
)

// frontendFS holds the frontend build output directory.
// Set via SetFrontendAssets before starting the server.
var frontendFS http.FileSystem

// SetFrontendAssets configures the directory serving the frontend SPA.
func SetFrontendAssets(dir string) {
	frontendFS = http.Dir(dir)
}

func frontendHandler() http.Handler {
	if frontendFS == nil {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("Frontend assets not loaded."))
		})
	}
	return http.FileServer(frontendFS)
}

func spaHandler(apiPrefix string, h http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, apiPrefix) {
			http.NotFound(w, r)
			return
		}
		if !strings.Contains(r.URL.Path, ".") {
			r.URL.Path = "/"
		}
		h.ServeHTTP(w, r)
	}
}
