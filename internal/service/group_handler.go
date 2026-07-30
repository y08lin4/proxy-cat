package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/y08lin4/proxy-cat/internal/domain/groupconfig"
	"github.com/y08lin4/proxy-cat/internal/domain/proxy"
)

// CreateGroup creates a new proxy group of any type.
func (s *Service) CreateGroup(req GroupCreateRequest) (proxy.ProxyGroup, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check for duplicate name
	for _, g := range s.active.ProxyGroups {
		if g.Name == req.Name {
			return proxy.ProxyGroup{}, errors.New("group name already exists")
		}
	}

	group := proxy.ProxyGroup{
		Name:    req.Name,
		Type:    req.Type,
		Proxies: req.Proxies,
	}

	// Apply type-specific config
	switch req.Type {
	case "load-balance":
		lbCfg := groupconfig.LoadBalanceConfig{
			Strategy:     groupconfig.LBStrategy(req.Strategy),
			StickyMaxAge: req.StickyMaxAge,
		}
		if err := lbCfg.Validate(); err != nil {
			return proxy.ProxyGroup{}, err
		}
		lbCfg.ApplyToGroup(&group)
	case "fallback":
		fbCfg := groupconfig.FallbackConfig{
			TestURL:   req.TestURL,
			Interval:  req.Interval,
			Tolerance: req.Tolerance,
			Lazy:      req.Lazy,
			TimeoutMS: 5000,
		}
		if err := fbCfg.Validate(); err != nil {
			return proxy.ProxyGroup{}, err
		}
		fbCfg.ApplyToGroup(&group)
	case "relay":
		if len(req.ChainNodes) < 2 {
			return proxy.ProxyGroup{}, errors.New("relay group requires at least 2 chain nodes")
		}
		group.ChainNodes = req.ChainNodes
	case "url-test":
		group.TestURL = req.TestURL
		if group.TestURL == "" {
			group.TestURL = "https://www.gstatic.com/generate_204"
		}
		group.Interval = req.Interval
		if group.Interval <= 0 {
			group.Interval = 300
		}
	}

	s.active.ProxyGroups = append(s.active.ProxyGroups, group)
	s.appendLogLocked("info", fmt.Sprintf("Group created: %s (type=%s)", group.Name, group.Type))
	return group, nil
}

// UpdateGroup updates an existing proxy group.
func (s *Service) UpdateGroup(name string, req GroupUpdateRequest) (proxy.ProxyGroup, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, g := range s.active.ProxyGroups {
		if g.Name == name {
			if req.Proxies != nil {
				s.active.ProxyGroups[i].Proxies = *req.Proxies
			}
			if req.TestURL != nil {
				s.active.ProxyGroups[i].TestURL = *req.TestURL
			}
			if req.Interval != nil {
				s.active.ProxyGroups[i].Interval = *req.Interval
			}
			if req.Tolerance != nil {
				s.active.ProxyGroups[i].Tolerance = *req.Tolerance
			}
			if req.Lazy != nil {
				s.active.ProxyGroups[i].Lazy = *req.Lazy
			}
			if req.Strategy != nil {
				s.active.ProxyGroups[i].Strategy = *req.Strategy
			}
			if req.StickyMaxAge != nil {
				s.active.ProxyGroups[i].StickyMaxAge = *req.StickyMaxAge
			}
			s.appendLogLocked("info", fmt.Sprintf("Group updated: %s", name))
			return s.active.ProxyGroups[i], nil
		}
	}
	return proxy.ProxyGroup{}, errors.New("group not found")
}

// DeleteGroup removes a proxy group by name.
func (s *Service) DeleteGroup(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, g := range s.active.ProxyGroups {
		if g.Name == name {
			s.active.ProxyGroups = append(s.active.ProxyGroups[:i], s.active.ProxyGroups[i+1:]...)
			s.appendLogLocked("info", fmt.Sprintf("Group deleted: %s", name))
			return nil
		}
	}
	return errors.New("group not found")
}

// ---------------------------------------------------------------------------
// Request / response types
// ---------------------------------------------------------------------------

// GroupCreateRequest is the JSON body for POST /api/v1/proxy-groups.
type GroupCreateRequest struct {
	Name         string               `json:"name"`
	Type         string               `json:"type"`
	Proxies      []string             `json:"proxies"`
	Strategy     string               `json:"strategy,omitempty"`
	StickyMaxAge int                  `json:"stickyMaxAge,omitempty"`
	TestURL      string               `json:"testUrl,omitempty"`
	Interval     int                  `json:"interval,omitempty"`
	Tolerance    int                  `json:"tolerance,omitempty"`
	Lazy         bool                 `json:"lazy,omitempty"`
	ChainNodes   []proxy.ChainProxyHop `json:"chainNodes,omitempty"`
}

// GroupUpdateRequest is the JSON body for PUT /api/v1/proxy-groups/{name}.
type GroupUpdateRequest struct {
	Proxies      *[]string             `json:"proxies,omitempty"`
	Strategy     *string               `json:"strategy,omitempty"`
	StickyMaxAge *int                  `json:"stickyMaxAge,omitempty"`
	TestURL      *string               `json:"testUrl,omitempty"`
	Interval     *int                  `json:"interval,omitempty"`
	Tolerance    *int                  `json:"tolerance,omitempty"`
	Lazy         *bool                 `json:"lazy,omitempty"`
	ChainNodes   *[]proxy.ChainProxyHop `json:"chainNodes,omitempty"`
}

// ---------------------------------------------------------------------------
// HTTP handlers
// ---------------------------------------------------------------------------

func (s *Service) handleCreateGroup(w http.ResponseWriter, r *http.Request) {
	var req GroupCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	group, err := s.CreateGroup(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, group)
}

func (s *Service) handleUpdateGroup(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	var req GroupUpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	group, err := s.UpdateGroup(name, req)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, group)
}

func (s *Service) handleDeleteGroup(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	if err := s.DeleteGroup(name); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
