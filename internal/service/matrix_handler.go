package service

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/y08lin4/proxy-cat/internal/domain/matrix"
)

// MatrixGenerateRequest is the request body for matrix generation.
type MatrixGenerateRequest struct {
	FrontNodes  []string          `json:"frontNodes"`
	ExitNodes   []matrix.ExitSpec `json:"exitNodes"`
	GroupPrefix string            `json:"groupPrefix,omitempty"`
}

// handleGenerateMatrix generates the N×M composite node matrix.
// POST /api/v1/matrix/generate
func (s *Service) handleGenerateMatrix(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	p := s.active
	s.mu.RUnlock()

	var req MatrixGenerateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	eng := matrix.NewEngine()
	result, err := eng.Generate(p, matrix.MatrixConfig{
		FrontNodes:  req.FrontNodes,
		ExitNodes:   req.ExitNodes,
		GroupPrefix: req.GroupPrefix,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// handleApplyMatrix generates the matrix AND applies it to the active profile.
// POST /api/v1/matrix/apply
func (s *Service) handleApplyMatrix(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var req MatrixGenerateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	eng := matrix.NewEngine()
	result, err := eng.Generate(s.active, matrix.MatrixConfig{
		FrontNodes:  req.FrontNodes,
		ExitNodes:   req.ExitNodes,
		GroupPrefix: req.GroupPrefix,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	result.MergeIntoProfile(&s.active)
	s.appendLogLocked("info", fmt.Sprintf("Matrix applied: %d composite nodes generated", len(result.CompositeNodes)))

	writeJSON(w, http.StatusOK, map[string]any{
		"matrix":      result,
		"profileName": s.active.Name,
		"proxyCount":  len(s.active.Proxies),
		"groupCount":  len(s.active.ProxyGroups),
	})
}
