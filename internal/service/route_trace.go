package service

import (
	"net/http"

	"github.com/y08lin4/proxy-cat/internal/domain/orchestrator"
)

// handleGetRouteTrace returns the dialer-proxy chain trace for a node.
// GET /api/v1/route-trace?node={nodeName}
func (s *Service) handleGetRouteTrace(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	p := s.active
	manager := s.autoStable
	s.mu.RUnlock()

	if manager == nil {
		writeError(w, http.StatusServiceUnavailable, "auto-stable not initialized; load a subscription first")
		return
	}

	nodeName := r.URL.Query().Get("node")
	if nodeName == "" {
		writeError(w, http.StatusBadRequest, "node query parameter is required")
		return
	}

	orch := orchestrator.New(manager)
	trace, err := orch.ResolveChain(p, nodeName)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, trace)
}

// handleGetDialerStatus returns dialer chain health for all nodes.
// GET /api/v1/dialer-status
func (s *Service) handleGetDialerStatus(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	p := s.active
	manager := s.autoStable
	s.mu.RUnlock()

	if manager == nil {
		writeError(w, http.StatusServiceUnavailable, "auto-stable not initialized")
		return
	}

	orch := orchestrator.New(manager)
	validations := orch.ValidateDialerChains(p)
	statuses := orch.GetDialerStatus(p)

	writeJSON(w, http.StatusOK, map[string]any{
		"validations": validations,
		"statuses":    statuses,
	})
}
