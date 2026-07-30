package configgen

import (
	"testing"

	"github.com/y08lin4/proxy-cat/internal/domain/proxy"
	"gopkg.in/yaml.v3"
)

// parsedConfig mirrors the top-level YAML structure for test assertions.
type parsedConfig struct {
	MixedPort          int           `yaml:"mixed-port"`
	AllowLAN           bool          `yaml:"allow-lan"`
	ExternalController string        `yaml:"external-controller"`
	Secret             string        `yaml:"secret"`
	Proxies            []proxyEntry  `yaml:"proxies"`
	ProxyGroups        []mihomoGroup `yaml:"proxy-groups"`
	Rules              []string      `yaml:"rules"`
	DNS                mihomoDNS     `yaml:"dns"`
}

// proxyEntry maps a single proxy item; RawOptions fields are collected as extras.
type proxyEntry struct {
	Name   string
	Type   string
	Server string
	Port   int
	Extra  map[string]any `yaml:",inline"`
}

func (p *proxyEntry) UnmarshalYAML(value *yaml.Node) error {
	// First pass: decode into a raw map so we can collect extra fields.
	raw := make(map[string]any)
	if err := value.Decode(&raw); err != nil {
		return err
	}
	if v, ok := raw["name"]; ok {
		p.Name, _ = v.(string)
	}
	if v, ok := raw["type"]; ok {
		p.Type, _ = v.(string)
	}
	if v, ok := raw["server"]; ok {
		p.Server, _ = v.(string)
	}
	if v, ok := raw["port"]; ok {
		switch n := v.(type) {
		case int:
			p.Port = n
		case float64:
			p.Port = int(n)
		}
	}
	delete(raw, "name")
	delete(raw, "type")
	delete(raw, "server")
	delete(raw, "port")
	p.Extra = raw
	return nil
}

func TestGenerateMihomoYAML(t *testing.T) {
	p := proxy.Profile{
		Name: "Main",
		Proxies: []proxy.ProxyNode{
			{
				Name:   "HK 01",
				Type:   "ss",
				Server: "hk.example.com",
				Port:   8388,
				RawOptions: map[string]any{
					"name":     "HK 01",
					"type":     "ss",
					"server":   "hk.example.com",
					"port":     8388,
					"cipher":   "aes-128-gcm",
					"password": "secret",
				},
			},
			{
				Name:   "US",
				Type:   "vmess",
				Server: "us.example.com",
				Port:   443,
				RawOptions: map[string]any{
					"uuid": "abc",
					"tls":  true,
				},
			},
		},
		Settings: proxy.Settings{Secret: "local-secret"},
	}
	got, err := GenerateMihomoYAML(p, Options{})
	if err != nil {
		t.Fatalf("GenerateMihomoYAML() error = %v", err)
	}

	var cfg parsedConfig
	if err := yaml.Unmarshal(got, &cfg); err != nil {
		t.Fatalf("failed to unmarshal generated YAML: %v\nfull YAML:\n%s", err, string(got))
	}

	// Top-level settings
	if cfg.MixedPort != 7890 {
		t.Errorf("expected mixed-port 7890, got %d", cfg.MixedPort)
	}
	if cfg.AllowLAN != false {
		t.Errorf("expected allow-lan false, got %v", cfg.AllowLAN)
	}
	if cfg.ExternalController != "127.0.0.1:9090" {
		t.Errorf("expected external-controller 127.0.0.1:9090, got %s", cfg.ExternalController)
	}
	if cfg.Secret != "local-secret" {
		t.Errorf("expected secret local-secret, got %s", cfg.Secret)
	}

	// Proxies
	if len(cfg.Proxies) != 2 {
		t.Fatalf("expected 2 proxies, got %d", len(cfg.Proxies))
	}
	p1 := cfg.Proxies[0]
	if p1.Name != "HK 01" {
		t.Errorf("expected proxy[0].name 'HK 01', got %q", p1.Name)
	}
	if p1.Type != "ss" {
		t.Errorf("expected proxy[0].type 'ss', got %q", p1.Type)
	}
	if p1.Server != "hk.example.com" {
		t.Errorf("expected proxy[0].server 'hk.example.com', got %q", p1.Server)
	}
	if p1.Port != 8388 {
		t.Errorf("expected proxy[0].port 8388, got %d", p1.Port)
	}
	if p1.Extra["cipher"] != "aes-128-gcm" {
		t.Errorf("expected proxy[0] cipher aes-128-gcm, got %v", p1.Extra["cipher"])
	}
	if p1.Extra["password"] != "secret" {
		t.Errorf("expected proxy[0] password secret, got %v", p1.Extra["password"])
	}

	p2 := cfg.Proxies[1]
	if p2.Name != "US" {
		t.Errorf("expected proxy[1].name 'US', got %q", p2.Name)
	}
	if p2.Type != "vmess" {
		t.Errorf("expected proxy[1].type 'vmess', got %q", p2.Type)
	}
	if p2.Extra["uuid"] != "abc" {
		t.Errorf("expected proxy[1] uuid abc, got %v", p2.Extra["uuid"])
	}
	if p2.Extra["tls"] != true {
		t.Errorf("expected proxy[1] tls true, got %v", p2.Extra["tls"])
	}

	// Proxy groups
	if len(cfg.ProxyGroups) != 3 {
		t.Fatalf("expected 3 proxy groups, got %d", len(cfg.ProxyGroups))
	}
	g0 := cfg.ProxyGroups[0]
	if g0.Name != "PROXY" {
		t.Errorf("expected proxy-group[0].name 'PROXY', got %q", g0.Name)
	}
	if g0.Type != "select" {
		t.Errorf("expected proxy-group[0].type 'select', got %q", g0.Type)
	}
	// PROXY group should contain AUTO-STABLE, AUTO, DIRECT and the proxy names
	foundNames := make(map[string]bool)
	for _, name := range g0.Proxies {
		foundNames[name] = true
	}
	for _, want := range []string{"AUTO-STABLE", "AUTO", "DIRECT", "HK 01", "US"} {
		if !foundNames[want] {
			t.Errorf("expected proxy-group[0].proxies to contain %q", want)
		}
	}

	g1 := cfg.ProxyGroups[1]
	if g1.Name != "AUTO" {
		t.Errorf("expected proxy-group[1].name 'AUTO', got %q", g1.Name)
	}
	if g1.Type != "url-test" {
		t.Errorf("expected proxy-group[1].type 'url-test', got %q", g1.Type)
	}

	g2 := cfg.ProxyGroups[2]
	if g2.Name != "AUTO-STABLE" {
		t.Errorf("expected proxy-group[2].name 'AUTO-STABLE', got %q", g2.Name)
	}
	if g2.Type != "select" {
		t.Errorf("expected proxy-group[2].type 'select' (mapped from auto-stable), got %q", g2.Type)
	}

	// Rules
	if len(cfg.Rules) != 2 {
		t.Fatalf("expected 2 rules, got %d", len(cfg.Rules))
	}
	if cfg.Rules[0] != "GEOIP,CN,DIRECT" {
		t.Errorf("expected rule[0] 'GEOIP,CN,DIRECT', got %q", cfg.Rules[0])
	}
	if cfg.Rules[1] != "MATCH,PROXY" {
		t.Errorf("expected rule[1] 'MATCH,PROXY', got %q", cfg.Rules[1])
	}

	// DNS
	if cfg.DNS.Enable != true {
		t.Errorf("expected dns.enable true, got %v", cfg.DNS.Enable)
	}
	if cfg.DNS.EnhancedMode != "fake-ip" {
		t.Errorf("expected dns.enhanced-mode 'fake-ip', got %q", cfg.DNS.EnhancedMode)
	}
	if len(cfg.DNS.NameServer) != 2 || cfg.DNS.NameServer[0] != "223.5.5.5" || cfg.DNS.NameServer[1] != "1.1.1.1" {
		t.Errorf("expected dns.nameserver [223.5.5.5, 1.1.1.1], got %v", cfg.DNS.NameServer)
	}
}

func TestGenerateMihomoYAMLMapsAutoStableToSelect(t *testing.T) {
	p := proxy.Profile{
		Name: "Main",
		Proxies: []proxy.ProxyNode{{
			Name:   "HK",
			Type:   "ss",
			Server: "hk.example.com",
			Port:   8388,
		}},
		ProxyGroups: []proxy.ProxyGroup{{
			Name:    "AUTO-STABLE",
			Type:    "auto-stable",
			Proxies: []string{"HK"},
		}},
	}
	got, err := GenerateMihomoYAML(p, Options{})
	if err != nil {
		t.Fatalf("GenerateMihomoYAML() error = %v", err)
	}

	var cfg parsedConfig
	if err := yaml.Unmarshal(got, &cfg); err != nil {
		t.Fatalf("failed to unmarshal generated YAML: %v\nfull YAML:\n%s", err, string(got))
	}

	if len(cfg.ProxyGroups) != 1 {
		t.Fatalf("expected 1 proxy group, got %d", len(cfg.ProxyGroups))
	}
	g := cfg.ProxyGroups[0]
	if g.Name != "AUTO-STABLE" {
		t.Errorf("expected name 'AUTO-STABLE', got %q", g.Name)
	}
	if g.Type != "select" {
		t.Errorf("expected type 'select' (mapped from auto-stable), got %q", g.Type)
	}
	if len(g.Proxies) != 1 || g.Proxies[0] != "HK" {
		t.Errorf("expected proxies [HK], got %v", g.Proxies)
	}
}

func TestGenerateMihomoYAMLCustomDNS(t *testing.T) {
	p := proxy.Profile{
		Name: "Main",
		Proxies: []proxy.ProxyNode{{
			Name:   "HK",
			Type:   "ss",
			Server: "hk.example.com",
			Port:   8388,
		}},
	}
	got, err := GenerateMihomoYAML(p, Options{
		DNS: DNSOptions{
			Enable:       true,
			EnhancedMode: "redir-host",
			Nameservers:  []string{"8.8.8.8", "8.8.4.4"},
		},
	})
	if err != nil {
		t.Fatalf("GenerateMihomoYAML() error = %v", err)
	}

	var cfg parsedConfig
	if err := yaml.Unmarshal(got, &cfg); err != nil {
		t.Fatalf("failed to unmarshal generated YAML: %v\nfull YAML:\n%s", err, string(got))
	}

	if cfg.DNS.Enable != true {
		t.Errorf("expected dns.enable true, got %v", cfg.DNS.Enable)
	}
	if cfg.DNS.EnhancedMode != "redir-host" {
		t.Errorf("expected dns.enhanced-mode 'redir-host', got %q", cfg.DNS.EnhancedMode)
	}
	if len(cfg.DNS.NameServer) != 2 || cfg.DNS.NameServer[0] != "8.8.8.8" || cfg.DNS.NameServer[1] != "8.8.4.4" {
		t.Errorf("expected dns.nameserver [8.8.8.8, 8.8.4.4], got %v", cfg.DNS.NameServer)
	}
}

func TestGenerateMihomoYAMLFallbackGroup(t *testing.T) {
	p := proxy.Profile{
		Name: "Main",
		Proxies: []proxy.ProxyNode{{
			Name:   "HK",
			Type:   "ss",
			Server: "hk.example.com",
			Port:   8388,
		}},
		ProxyGroups: []proxy.ProxyGroup{{
			Name:      "FALLBACK-GROUP",
			Type:      "fallback",
			Proxies:   []string{"HK"},
			TestURL:   "https://cp.cloudflare.com/generate_204",
			Interval:  600,
			Tolerance: 150,
			Lazy:      true,
		}},
	}
	got, err := GenerateMihomoYAML(p, Options{})
	if err != nil {
		t.Fatalf("GenerateMihomoYAML() error = %v", err)
	}

	var cfg parsedConfig
	if err := yaml.Unmarshal(got, &cfg); err != nil {
		t.Fatalf("failed to unmarshal generated YAML: %v\nfull YAML:\n%s", err, string(got))
	}

	if len(cfg.ProxyGroups) != 1 {
		t.Fatalf("expected 1 proxy group, got %d", len(cfg.ProxyGroups))
	}
	g := cfg.ProxyGroups[0]
	if g.Name != "FALLBACK-GROUP" {
		t.Errorf("expected name 'FALLBACK-GROUP', got %q", g.Name)
	}
	if g.Type != "fallback" {
		t.Errorf("expected type 'fallback', got %q", g.Type)
	}
	if g.URL != "https://cp.cloudflare.com/generate_204" {
		t.Errorf("expected URL 'https://cp.cloudflare.com/generate_204', got %q", g.URL)
	}
	if g.Interval != 600 {
		t.Errorf("expected interval 600, got %d", g.Interval)
	}
	if g.Tolerance != 150 {
		t.Errorf("expected tolerance 150, got %d", g.Tolerance)
	}
	if g.Lazy != true {
		t.Errorf("expected lazy true, got %v", g.Lazy)
	}
}

func TestGenerateMihomoYAMLLoadBalanceGroup(t *testing.T) {
	p := proxy.Profile{
		Name: "Main",
		Proxies: []proxy.ProxyNode{
			{
				Name:   "HK",
				Type:   "ss",
				Server: "hk.example.com",
				Port:   8388,
			},
			{
				Name:   "US",
				Type:   "vmess",
				Server: "us.example.com",
				Port:   443,
			},
		},
		ProxyGroups: []proxy.ProxyGroup{{
			Name:         "LB-GROUP",
			Type:         "load-balance",
			Proxies:      []string{"HK", "US"},
			Strategy:     "round-robin",
			StickyMaxAge: 1800,
		}},
	}
	got, err := GenerateMihomoYAML(p, Options{})
	if err != nil {
		t.Fatalf("GenerateMihomoYAML() error = %v", err)
	}

	var cfg parsedConfig
	if err := yaml.Unmarshal(got, &cfg); err != nil {
		t.Fatalf("failed to unmarshal generated YAML: %v\nfull YAML:\n%s", err, string(got))
	}

	if len(cfg.ProxyGroups) != 1 {
		t.Fatalf("expected 1 proxy group, got %d", len(cfg.ProxyGroups))
	}
	g := cfg.ProxyGroups[0]
	if g.Name != "LB-GROUP" {
		t.Errorf("expected name 'LB-GROUP', got %q", g.Name)
	}
	if g.Type != "load-balance" {
		t.Errorf("expected type 'load-balance', got %q", g.Type)
	}
	if g.Strategy != "round-robin" {
		t.Errorf("expected strategy 'round-robin', got %q", g.Strategy)
	}
	if g.StickyMaxAge != 1800 {
		t.Errorf("expected sticky-max-age 1800, got %d", g.StickyMaxAge)
	}
	if g.URL != "" {
		t.Errorf("expected no URL for load-balance, got %q", g.URL)
	}
	if g.Interval != 0 {
		t.Errorf("expected no interval for load-balance, got %d", g.Interval)
	}
}
