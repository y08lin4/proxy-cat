package service

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
)

var frontendDir string

func SetFrontendDir(dir string) {
	frontendDir = dir
}

// Router returns a configured HTTP handler for the service API.
func (s *Service) Router() http.Handler {
	mux := http.NewServeMux()

	// Status
	mux.HandleFunc("GET /api/v1/status", s.handleGetStatus)
	mux.HandleFunc("GET /api/v1/status/connection", s.handleGetConnectionStatus)

	// Core management
	mux.HandleFunc("POST /api/v1/core/start", s.handleStartCore)
	mux.HandleFunc("POST /api/v1/core/stop", s.handleStopCore)
	mux.HandleFunc("POST /api/v1/core/restart", s.handleRestartCore)
	mux.HandleFunc("POST /api/v1/core/recover", s.handleRecoverCore)

	// System proxy
	mux.HandleFunc("GET /api/v1/system-proxy", s.handleGetSystemProxy)
	mux.HandleFunc("POST /api/v1/system-proxy", s.handleSetSystemProxy)

	// Subscription
	mux.HandleFunc("POST /api/v1/subscription", s.handleLoadSubscription)

	// Proxy groups
	mux.HandleFunc("GET /api/v1/proxy-groups", s.handleGetProxyGroups)
	mux.HandleFunc("PUT /api/v1/proxy-groups/", s.handleSelectProxy)
	mux.HandleFunc("GET /api/v1/proxy-groups/{name}/test-url", s.handleGetTestURL)
	mux.HandleFunc("PUT /api/v1/proxy-groups/{name}/test-url", s.handleSetTestURL)

	// Auto-stable
	mux.HandleFunc("GET /api/v1/autostable/status", s.handleGetAutoStableStatus)
	mux.HandleFunc("PUT /api/v1/autostable/enabled", s.handleSetAutoStableEnabled)
	mux.HandleFunc("POST /api/v1/autostable/tick", s.handleRunAutoStableTick)
	mux.HandleFunc("POST /api/v1/autostable/select", s.handleSelectAutoStableProxy)

	// Route trace
	mux.HandleFunc("GET /api/v1/route-trace", s.handleGetRouteTrace)
	mux.HandleFunc("GET /api/v1/dialer-status", s.handleGetDialerStatus)

	// Matrix
	mux.HandleFunc("POST /api/v1/matrix/generate", s.handleGenerateMatrix)
	mux.HandleFunc("POST /api/v1/matrix/apply", s.handleApplyMatrix)

	// Logs
	mux.HandleFunc("GET /api/v1/logs", s.handleGetLogs)

	// Frontend SPA
	mux.HandleFunc("/", s.handleFrontend)

	return withMiddleware(mux)
}

// withMiddleware adds common middleware to the handler.
func withMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// CORS headers for development
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// writeJSON writes v as JSON to the response.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// writeError writes an error response.
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]any{
		"code":    status,
		"message": message,
	})
}

// handler implementations below - each extracts params from the request,
// calls the corresponding Service method, and writes JSON responses.

func (s *Service) handleGetStatus(w http.ResponseWriter, r *http.Request) {
	status := s.GetAppStatus()
	writeJSON(w, http.StatusOK, status)
}

func (s *Service) handleGetConnectionStatus(w http.ResponseWriter, r *http.Request) {
	status, err := s.GetConnectionStatus()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, status)
}

func (s *Service) handleStartCore(w http.ResponseWriter, r *http.Request) {
	if err := s.StartCore(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "started"})
}

func (s *Service) handleStopCore(w http.ResponseWriter, r *http.Request) {
	if err := s.StopCore(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "stopped"})
}

func (s *Service) handleRestartCore(w http.ResponseWriter, r *http.Request) {
	if err := s.RestartCore(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "restarted"})
}

func (s *Service) handleRecoverCore(w http.ResponseWriter, r *http.Request) {
	recovered, err := s.RecoverCoreIfNeeded()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"recovered": recovered})
}

func (s *Service) handleGetSystemProxy(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	enabled := s.status.SystemProxyEnabled
	s.mu.RUnlock()
	writeJSON(w, http.StatusOK, map[string]bool{"enabled": enabled})
}

func (s *Service) handleSetSystemProxy(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := s.SetSystemProxy(req.Enabled); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Service) handleLoadSubscription(w http.ResponseWriter, r *http.Request) {
	var req struct {
		URL  string `json:"url"`
		Name string `json:"name,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := s.LoadSubscription(req.URL); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "loaded"})
}

func (s *Service) handleGetProxyGroups(w http.ResponseWriter, r *http.Request) {
	groups := s.GetProxyGroups()
	writeJSON(w, http.StatusOK, groups)
}

func (s *Service) handleSelectProxy(w http.ResponseWriter, r *http.Request) {
	// Path: /api/v1/proxy-groups/{groupName}/select
	groupName := strings.TrimPrefix(r.URL.Path, "/api/v1/proxy-groups/")
	groupName = strings.TrimSuffix(groupName, "/select")

	var req struct {
		ProxyName string `json:"proxyName"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := s.SelectProxy(groupName, req.ProxyName); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "selected"})
}

// handleGetTestURL returns the test URL for a proxy group.
// GET /api/v1/proxy-groups/{name}/test-url
func (s *Service) handleGetTestURL(w http.ResponseWriter, r *http.Request) {
	groupName := r.PathValue("name")

	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, g := range s.active.ProxyGroups {
		if g.Name == groupName {
			writeJSON(w, http.StatusOK, map[string]string{"testUrl": g.TestURL})
			return
		}
	}
	writeError(w, http.StatusNotFound, "group not found")
}

// handleSetTestURL sets the test URL for a proxy group.
// PUT /api/v1/proxy-groups/{name}/test-url
func (s *Service) handleSetTestURL(w http.ResponseWriter, r *http.Request) {
	groupName := r.PathValue("name")

	var req struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.active.ProxyGroups {
		if s.active.ProxyGroups[i].Name == groupName {
			s.active.ProxyGroups[i].TestURL = req.URL
			s.appendLogLocked("info", fmt.Sprintf("Group %s test URL set to %s", groupName, req.URL))
			writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
			return
		}
	}
	writeError(w, http.StatusNotFound, "group not found")
}

func (s *Service) handleGetAutoStableStatus(w http.ResponseWriter, r *http.Request) {
	status := s.GetAutoStableStatus()
	writeJSON(w, http.StatusOK, status)
}

func (s *Service) handleSetAutoStableEnabled(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := s.SetAutoStableEnabled(req.Enabled); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Service) handleRunAutoStableTick(w http.ResponseWriter, r *http.Request) {
	result, err := s.RunAutoStableTick()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Service) handleSelectAutoStableProxy(w http.ResponseWriter, r *http.Request) {
	var req struct {
		GroupName string `json:"groupName"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	result, err := s.SelectAutoStableProxy(req.GroupName)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Service) handleGetLogs(w http.ResponseWriter, r *http.Request) {
	limit := 100
	// Simple limit extraction from query
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := fmt.Sscanf(l, "%d", &limit); err != nil || parsed != 1 {
			limit = 100
		}
	}
	logs := s.GetLogs(limit)
	writeJSON(w, http.StatusOK, logs)
}

func (s *Service) handleFrontend(w http.ResponseWriter, r *http.Request) {
	if frontendDir == "" {
		writeJSON(w, http.StatusOK, map[string]string{
			"message": "Proxy-Cat API server is running.",
		})
		return
	}
	// SPA fallback: non-dotted paths serve index.html
	path := r.URL.Path
	if !strings.Contains(path, ".") {
		path = "/"
	}
	http.ServeFile(w, r, filepath.Join(frontendDir, filepath.Clean(path)))
}
