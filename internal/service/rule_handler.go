package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/y08lin4/proxy-cat/internal/domain/proxy"
	"github.com/y08lin4/proxy-cat/internal/domain/ruleengine"
)

// --- Rule CRUD ---

func (s *Service) GetRules() []proxy.Rule {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]proxy.Rule(nil), s.active.Rules...)
}

func (s *Service) CreateRule(rule proxy.Rule) (proxy.Rule, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if rule.ID == "" {
		rule.ID = fmt.Sprintf("rule_%d", time.Now().UnixNano())
	}
	if rule.Priority == 0 {
		// Auto-assign lowest priority (highest number)
		maxPri := 0
		for _, r := range s.active.Rules {
			if r.Priority > maxPri {
				maxPri = r.Priority
			}
		}
		rule.Priority = maxPri + 10
	}
	if !rule.Enabled {
		rule.Enabled = true // default enabled
	}
	rule.CreatedAt = time.Now()
	rule.UpdatedAt = rule.CreatedAt
	s.active.Rules = append(s.active.Rules, rule)
	s.appendLogLocked("info", fmt.Sprintf("Rule created: %s", rule.ID))
	return rule, nil
}

func (s *Service) UpdateRule(ruleID string, updates proxy.Rule) (proxy.Rule, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, r := range s.active.Rules {
		if r.ID == ruleID {
			updates.ID = ruleID
			updates.CreatedAt = r.CreatedAt
			updates.UpdatedAt = time.Now()
			s.active.Rules[i] = updates
			s.appendLogLocked("info", fmt.Sprintf("Rule updated: %s", ruleID))
			return updates, nil
		}
	}
	return proxy.Rule{}, errors.New("rule not found")
}

func (s *Service) DeleteRule(ruleID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, r := range s.active.Rules {
		if r.ID == ruleID {
			s.active.Rules = append(s.active.Rules[:i], s.active.Rules[i+1:]...)
			s.appendLogLocked("info", fmt.Sprintf("Rule deleted: %s", ruleID))
			return nil
		}
	}
	return errors.New("rule not found")
}

func (s *Service) ReorderRules(ruleIDs []string) ([]proxy.Rule, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ruleMap := make(map[string]proxy.Rule, len(s.active.Rules))
	for _, r := range s.active.Rules {
		ruleMap[r.ID] = r
	}
	reordered := make([]proxy.Rule, 0, len(ruleIDs))
	for i, id := range ruleIDs {
		r, ok := ruleMap[id]
		if !ok {
			return nil, fmt.Errorf("rule %q not found", id)
		}
		r.Priority = i * 10
		reordered = append(reordered, r)
	}
	s.active.Rules = reordered
	s.appendLogLocked("info", "Rules reordered")
	return reordered, nil
}

// --- Rule validation ---

func (s *Service) ValidateRules() []ruleengine.ValidationError {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var allErrs []ruleengine.ValidationError
	for _, r := range s.active.Rules {
		if errs := s.ruleEngine.Validate(r); len(errs) > 0 {
			allErrs = append(allErrs, errs...)
		}
	}
	return allErrs
}

// --- HTTP Handlers ---

func (s *Service) handleGetRules(w http.ResponseWriter, r *http.Request) {
	rules := s.GetRules()
	writeJSON(w, http.StatusOK, rules)
}

func (s *Service) handleCreateRule(w http.ResponseWriter, r *http.Request) {
	var rule proxy.Rule
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	created, err := s.CreateRule(rule)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

func (s *Service) handleUpdateRule(w http.ResponseWriter, r *http.Request) {
	ruleID := r.PathValue("id")
	var rule proxy.Rule
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	updated, err := s.UpdateRule(ruleID, rule)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (s *Service) handleDeleteRule(w http.ResponseWriter, r *http.Request) {
	ruleID := r.PathValue("id")
	if err := s.DeleteRule(ruleID); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Service) handleReorderRules(w http.ResponseWriter, r *http.Request) {
	var req struct {
		RuleIDs []string `json:"ruleIds"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	reordered, err := s.ReorderRules(req.RuleIDs)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, reordered)
}

func (s *Service) handleValidateRules(w http.ResponseWriter, r *http.Request) {
	errs := s.ValidateRules()
	writeJSON(w, http.StatusOK, errs)
}
