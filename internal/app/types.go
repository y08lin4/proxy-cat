package app

import "time"

// AppStatus is the small status surface consumed by the Wails frontend.
type AppStatus struct {
	CoreRunning        bool   `json:"coreRunning"`
	SystemProxyEnabled bool   `json:"systemProxyEnabled"`
	AutoStableEnabled  bool   `json:"autoStableEnabled"`
	ActiveProfileName  string `json:"activeProfileName"`
	ControllerAddress  string `json:"controllerAddress"`
	LastError          string `json:"lastError,omitempty"`
}

// ProxyGroupView is the frontend-facing proxy group snapshot.
type ProxyGroupView struct {
	Name     string      `json:"name"`
	Type     string      `json:"type"`
	Selected string      `json:"selected"`
	Proxies  []ProxyView `json:"proxies"`
}

// ProxyView is the minimal node shape needed for Phase 1 group selection.
type ProxyView struct {
	Name      string `json:"name"`
	Type      string `json:"type,omitempty"`
	LatencyMS int    `json:"latencyMs,omitempty"`
	Alive     bool   `json:"alive"`
}

// LogLine is an application or core log line returned to the frontend.
type LogLine struct {
	Time    time.Time `json:"time"`
	Level   string    `json:"level"`
	Message string    `json:"message"`
}

// AutoStableStatus is the app-layer health and control surface for Phase 2.
type AutoStableStatus struct {
	Enabled      bool                    `json:"enabled"`
	Available    bool                    `json:"available"`
	Running      bool                    `json:"running"`
	LastTickAt   time.Time               `json:"lastTickAt,omitempty"`
	LastAction   string                  `json:"lastAction,omitempty"`
	LastSelected string                  `json:"lastSelected,omitempty"`
	LastError    string                  `json:"lastError,omitempty"`
	Health       []AutoStableGroupHealth `json:"health"`
}

// AutoStableGroupHealth is a frontend-facing health snapshot for one group.
type AutoStableGroupHealth struct {
	Name     string                 `json:"name"`
	Type     string                 `json:"type"`
	Selected string                 `json:"selected,omitempty"`
	Proxies  []AutoStableNodeHealth `json:"proxies"`
}

// AutoStableNodeHealth is a frontend-facing health snapshot for one proxy.
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

// AutoStableActionResult reports one manual Phase 2 action.
type AutoStableActionResult struct {
	Action      string                  `json:"action"`
	GroupName   string                  `json:"groupName,omitempty"`
	Selected    string                  `json:"selected,omitempty"`
	Changed     bool                    `json:"changed"`
	Message     string                  `json:"message,omitempty"`
	CompletedAt time.Time               `json:"completedAt"`
	Health      []AutoStableGroupHealth `json:"health,omitempty"`
}
