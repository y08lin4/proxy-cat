package profile

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// Profile is Proxy-Cat's minimal Phase 1 intermediate model.
type Profile struct {
	ID            string
	Name          string
	Subscriptions []Subscription
	Proxies       []ProxyNode
	ProxyGroups   []ProxyGroup
	Rules         []Rule
	Settings      Settings
}

type Subscription struct {
	ID        string
	Name      string
	URL       string
	UpdatedAt time.Time
}

type ProxyNode struct {
	ID         string
	Name       string
	Type       string
	Server     string
	Port       int
	RawOptions map[string]any
}

type ProxyGroup struct {
	Name          string
	Type          string
	Proxies       []string
	SelectedProxy string
}

type Rule struct {
	Type        string
	Value       string
	TargetGroup string
}

type Settings struct {
	MixedPort          int
	AllowLAN           bool
	ExternalController string
	Secret             string
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
		{Type: "GEOIP", Value: "CN", TargetGroup: "DIRECT"},
		{Type: "MATCH", TargetGroup: targetGroup},
	}
}
