package proxy

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// Profile is Proxy-Cat's minimal Phase 1 intermediate model.
type Profile struct {
	ID            string         `json:"id,omitempty"`
	Name          string         `json:"name"`
	Subscriptions []Subscription `json:"subscriptions,omitempty"`
	Proxies       []ProxyNode    `json:"proxies"`
	ProxyGroups   []ProxyGroup   `json:"proxyGroups"`
	Rules         []Rule         `json:"rules"`
	Settings      Settings       `json:"settings"`
}

type Subscription struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	URL       string    `json:"url"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type ProxyNode struct {
	ID         string         `json:"id"`
	Name       string         `json:"name"`
	Type       string         `json:"type"`
	Server     string         `json:"server"`
	Port       int            `json:"port"`
	RawOptions map[string]any `json:"rawOptions,omitempty"`
}

type ProxyGroup struct {
	Name          string          `json:"name"`
	Type          string          `json:"type"`
	Proxies       []string        `json:"proxies"`
	SelectedProxy string          `json:"selectedProxy"`
	TestURL       string          `json:"testUrl,omitempty"`
	Interval      int             `json:"interval,omitempty"`
	Tolerance     int             `json:"tolerance,omitempty"`
	Lazy          bool            `json:"lazy,omitempty"`
	Strategy      string          `json:"strategy,omitempty"`
	StickyMaxAge  int             `json:"stickyMaxAge,omitempty"`
	ChainNodes    []ChainProxyHop `json:"chainNodes,omitempty"`
}

type ChainProxyHop struct {
	ProxyName   string `json:"proxyName"`
	DialerProxy string `json:"dialerProxy,omitempty"`
}

type Rule struct {
	Type        string    `json:"type"`
	Value       string    `json:"value,omitempty"`
	TargetGroup string    `json:"targetGroup"`
	ID          string    `json:"id"`
	Priority    int       `json:"priority"`
	Enabled     bool      `json:"enabled"`
	Description string    `json:"description,omitempty"`
	Category    string    `json:"category,omitempty"`
	Tag         string    `json:"tag,omitempty"`
	TemplateID  string    `json:"templateId,omitempty"`
	SubRules    []Rule    `json:"subRules,omitempty"`
	InlineRule  string    `json:"inlineRule,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type Settings struct {
	MixedPort          int    `json:"mixedPort"`
	AllowLAN           bool   `json:"allowLan"`
	ExternalController string `json:"externalController,omitempty"`
	Secret             string `json:"secret,omitempty"`
}

func (p Profile) Validate() error {
	if strings.TrimSpace(p.Name) == "" {
		return errors.New("profile name is required")
	}
	if len(p.Proxies) == 0 {
		return errors.New("profile requires at least one proxy")
	}
	seen := make(map[string]struct{}, len(p.Proxies))
	for i, proxy := range p.Proxies {
		if err := proxy.Validate(); err != nil {
			return fmt.Errorf("proxy %d: %w", i, err)
		}
		if _, ok := seen[proxy.Name]; ok {
			return fmt.Errorf("duplicate proxy name %q", proxy.Name)
		}
		seen[proxy.Name] = struct{}{}
	}
	return nil
}

func (n ProxyNode) Validate() error {
	if strings.TrimSpace(n.Name) == "" {
		return errors.New("name is required")
	}
	if strings.TrimSpace(n.Type) == "" {
		return errors.New("type is required")
	}
	if strings.TrimSpace(n.Server) == "" {
		return errors.New("server is required")
	}
	if n.Port <= 0 || n.Port > 65535 {
		return fmt.Errorf("port %d is out of range", n.Port)
	}
	return nil
}

func (p Profile) ProxyNames() []string {
	names := make([]string, 0, len(p.Proxies))
	for _, proxy := range p.Proxies {
		names = append(names, proxy.Name)
	}
	return names
}

func DefaultSettings() Settings {
	return Settings{
		MixedPort:          7890,
		AllowLAN:           false,
		ExternalController: "127.0.0.1:9090",
	}
}

func DefaultGroups(proxyNames []string) []ProxyGroup {
	names := append([]string(nil), proxyNames...)
	proxyGroupMembers := make([]string, 0, len(names)+2)
	proxyGroupMembers = append(proxyGroupMembers, "AUTO-STABLE", "AUTO", "DIRECT")
	proxyGroupMembers = append(proxyGroupMembers, names...)

	return []ProxyGroup{
		{
			Name:    "PROXY",
			Type:    "select",
			Proxies: proxyGroupMembers,
		},
		{
			Name:    "AUTO",
			Type:    "url-test",
			Proxies: names,
		},
		{
			Name:    "AUTO-STABLE",
			Type:    "auto-stable",
			Proxies: names,
		},
	}
}

func DefaultRules(targetGroup string) []Rule {
	if strings.TrimSpace(targetGroup) == "" {
		targetGroup = "PROXY"
	}
	return []Rule{
		{Type: "GEOIP", Value: "CN", TargetGroup: "DIRECT", Enabled: true},
		{Type: "MATCH", TargetGroup: targetGroup, Enabled: true},
	}
}
