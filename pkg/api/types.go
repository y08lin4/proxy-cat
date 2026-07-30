// Package api defines shared request/response types for the Proxy-Cat HTTP API.
// These types are the single source of truth for both Go handlers and TypeScript client code.
package api

import "time"

// ---------------------------------- Request types ----------------------------------

// LoadSubscriptionRequest is the body for POST /api/v1/subscription.
type LoadSubscriptionRequest struct {
	URL  string `json:"url"`
	Name string `json:"name,omitempty"`
}

// SelectProxyRequest is the body for PUT /api/v1/proxy-groups/{groupName}/select.
type SelectProxyRequest struct {
	ProxyName string `json:"proxyName"`
}

// SetSystemProxyRequest is the body for POST /api/v1/system-proxy.
type SetSystemProxyRequest struct {
	Enabled bool `json:"enabled"`
}

// SetAutoStableEnabledRequest is the body for PUT /api/v1/autostable/enabled.
type SetAutoStableEnabledRequest struct {
	Enabled bool `json:"enabled"`
}

// SelectAutoStableRequest is the body for POST /api/v1/autostable/select.
type SelectAutoStableRequest struct {
	GroupName string `json:"groupName"`
}

// ---------------------------------- Response types ----------------------------------

// ErrorResponse is returned for non-2xx responses.
type ErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// AppStatusResponse is the main status surface.
type AppStatusResponse struct {
	CoreRunning        bool   `json:"coreRunning"`
	SystemProxyEnabled bool   `json:"systemProxyEnabled"`
	AutoStableEnabled  bool   `json:"autoStableEnabled"`
	ActiveProfileName  string `json:"activeProfileName"`
	ControllerAddress  string `json:"controllerAddress"`
	LastError          string `json:"lastError,omitempty"`
}

// ConnectionStatusResponse holds connection statistics.
type ConnectionStatusResponse struct {
	CoreRunning     bool  `json:"coreRunning"`
	UploadTotal     int64 `json:"uploadTotal"`
	DownloadTotal   int64 `json:"downloadTotal"`
	ConnectionCount int   `json:"connectionCount"`
}

// ProxyGroupView is the frontend-facing proxy group snapshot.
type ProxyGroupView struct {
	Name     string      `json:"name"`
	Type     string      `json:"type"`
	Selected string      `json:"selected"`
	Proxies  []ProxyView `json:"proxies"`
}

// ProxyView is the minimal node shape for group selection.
type ProxyView struct {
	Name      string `json:"name"`
	Type      string `json:"type,omitempty"`
	LatencyMS int    `json:"latencyMs,omitempty"`
	Alive     bool   `json:"alive"`
}

// LogLine is an application or core log entry.
type LogLine struct {
	Time    time.Time `json:"time"`
	Level   string    `json:"level"`
	Message string    `json:"message"`
}

// AutoStableStatusResponse is the health and control surface.
type AutoStableStatusResponse struct {
	Enabled      bool                       `json:"enabled"`
	Available    bool                       `json:"available"`
	Running      bool                       `json:"running"`
	LastTickAt   time.Time                  `json:"lastTickAt,omitempty"`
	LastAction   string                     `json:"lastAction,omitempty"`
	LastSelected string                     `json:"lastSelected,omitempty"`
	LastError    string                     `json:"lastError,omitempty"`
	Health       []AutoStableGroupHealth    `json:"health"`
}

// AutoStableGroupHealth is a health snapshot for one proxy group.
type AutoStableGroupHealth struct {
	Name     string                 `json:"name"`
	Type     string                 `json:"type"`
	Selected string                 `json:"selected,omitempty"`
	Proxies  []AutoStableNodeHealth `json:"proxies"`
}

// AutoStableNodeHealth is a health snapshot for one proxy node.
type AutoStableNodeHealth struct {
	Name          string    `json:"name"`
	Type          string    `json:"type,omitempty"`
	LatencyMS     int       `json:"latencyMs,omitempty"`
	Alive         bool      `json:"alive"`
	Score         float64   `json:"score,omitempty"`
	SuccessCount  int       `json:"successCount,omitempty"`
	FailureCount  int       `json:"failureCount,omitempty"`
	TotalChecks   int       `json:"totalChecks,omitempty"`
	FailureRate   float64   `json:"failureRate,omitempty"`
	LastCheckedAt time.Time `json:"lastCheckedAt,omitempty"`
	CooldownUntil time.Time `json:"cooldownUntil,omitempty"`
}

// AutoStableActionResult is the result of one auto-stable action.
type AutoStableActionResult struct {
	Action      string                  `json:"action"`
	GroupName   string                  `json:"groupName,omitempty"`
	Selected    string                  `json:"selected,omitempty"`
	Changed     bool                    `json:"changed"`
	Message     string                  `json:"message,omitempty"`
	CompletedAt time.Time               `json:"completedAt"`
	Health      []AutoStableGroupHealth `json:"health,omitempty"`
}
