package service

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/y08lin4/proxy-cat/internal/domain/persistence"
	"github.com/y08lin4/proxy-cat/internal/domain/proxy"
)

// --- Template methods ---

func (s *Service) GetRuleTemplates() []proxy.RuleTemplate {
	return proxy.PredefinedTemplates()
}

func (s *Service) ApplyRuleTemplate(templateID string, overrideGroup string) ([]proxy.Rule, error) {
	templates := proxy.PredefinedTemplates()
	var tmpl *proxy.RuleTemplate
	for _, t := range templates {
		if t.ID == templateID {
			tmpl = &t
			break
		}
	}
	if tmpl == nil {
		return nil, fmt.Errorf("template %q not found", templateID)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	maxPri := 0
	for _, r := range s.active.Rules {
		if r.Priority > maxPri {
			maxPri = r.Priority
		}
	}

	var created []proxy.Rule
	for _, rule := range tmpl.Rules {
		rule.ID = fmt.Sprintf("rule_%d", time.Now().UnixNano())
		rule.CreatedAt = now
		rule.UpdatedAt = now
		rule.Priority = maxPri + rule.Priority + 10
		if overrideGroup != "" {
			rule.TargetGroup = overrideGroup
		}
		s.active.Rules = append(s.active.Rules, rule)
		created = append(created, rule)
	}

	s.appendLogLocked("info", fmt.Sprintf("Template %s applied: %d rules created", tmpl.Name, len(created)))
	return created, nil
}

// --- Persistence methods ---

func (s *Service) SaveProfile() error {
	s.mu.RLock()
	p := s.active
	s.mu.RUnlock()

	if s.store == nil {
		return fmt.Errorf("persistence store not initialized")
	}
	if p.ID == "" {
		p.ID = fmt.Sprintf("profile_%d", time.Now().Unix())
	}
	return s.store.SaveProfile(p)
}

func (s *Service) LoadProfile(id string) error {
	if s.store == nil {
		return fmt.Errorf("persistence store not initialized")
	}
	p, err := s.store.LoadProfile(id)
	if err != nil {
		return err
	}

	s.mu.Lock()
	s.active = p
	s.mu.Unlock()

	s.appendLogLocked("info", fmt.Sprintf("Profile %s loaded", p.Name))
	return nil
}

func (s *Service) ListProfiles() ([]persistence.ProfileMeta, error) {
	if s.store == nil {
		return nil, nil
	}
	return s.store.ListProfiles()
}

func (s *Service) DeleteSavedProfile(id string) error {
	if s.store == nil {
		return fmt.Errorf("persistence store not initialized")
	}
	return s.store.DeleteProfile(id)
}

// --- HTTP Handlers ---

func (s *Service) handleGetTemplates(w http.ResponseWriter, r *http.Request) {
	templates := s.GetRuleTemplates()
	writeJSON(w, http.StatusOK, templates)
}

func (s *Service) handleApplyTemplate(w http.ResponseWriter, r *http.Request) {
	templateID := r.PathValue("id")
	var req struct {
		TargetGroup string `json:"targetGroup"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	rules, err := s.ApplyRuleTemplate(templateID, req.TargetGroup)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, rules)
}

func (s *Service) handleListProfiles(w http.ResponseWriter, r *http.Request) {
	profiles, err := s.ListProfiles()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, profiles)
}

func (s *Service) handleSaveProfile(w http.ResponseWriter, r *http.Request) {
	if err := s.SaveProfile(); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "saved"})
}

func (s *Service) handleLoadProfile(w http.ResponseWriter, r *http.Request) {
	profileID := r.PathValue("id")
	if err := s.LoadProfile(profileID); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "loaded"})
}

func (s *Service) handleDeleteProfile(w http.ResponseWriter, r *http.Request) {
	profileID := r.PathValue("id")
	if err := s.DeleteSavedProfile(profileID); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
