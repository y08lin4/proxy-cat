package app

import "time"

// AppStatus is the small status surface consumed by the Wails frontend.
type AppStatus struct {
	CoreRunning        bool   `json:"coreRunning"`
	SystemProxyEnabled bool   `json:"systemProxyEnabled"`
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
