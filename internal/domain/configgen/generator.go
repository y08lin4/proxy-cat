package configgen

import (
	"github.com/y08lin4/proxy-cat/internal/domain/proxy"
	"gopkg.in/yaml.v3"
)

// Options holds optional overrides for generated configuration.
type Options struct {
	MixedPort          int
	AllowLAN           *bool
	ExternalController string
	Secret             string
	DNS                DNSOptions
}

// DNSOptions holds DNS configuration overrides.
type DNSOptions struct {
	Enable       bool
	EnhancedMode string
	Nameservers  []string
}

// mihomoConfig mirrors the Mihomo YAML structure for marshaling.
type mihomoConfig struct {
	MixedPort          int              `yaml:"mixed-port"`
	AllowLAN           bool             `yaml:"allow-lan"`
	ExternalController string           `yaml:"external-controller"`
	Secret             string           `yaml:"secret,omitempty"`
	Proxies            []map[string]any `yaml:"proxies"`
	ProxyGroups        []mihomoGroup    `yaml:"proxy-groups"`
	Rules              []string         `yaml:"rules"`
	DNS                mihomoDNS        `yaml:"dns"`
}

type mihomoGroup struct {
	Name     string   `yaml:"name"`
	Type     string   `yaml:"type"`
	URL      string   `yaml:"url,omitempty"`
	Interval int      `yaml:"interval,omitempty"`
	Proxies  []string `yaml:"proxies"`
}

type mihomoDNS struct {
	Enable       bool     `yaml:"enable"`
	EnhancedMode string   `yaml:"enhanced-mode"`
	NameServer   []string `yaml:"nameserver"`
}

// GenerateMihomoYAML produces a Mihomo-compatible config YAML from a Proxy-Cat profile.
func GenerateMihomoYAML(p proxy.Profile, opts Options) ([]byte, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}

	settings := proxy.DefaultSettings()
	if p.Settings.MixedPort > 0 {
		settings.MixedPort = p.Settings.MixedPort
	}
	settings.AllowLAN = p.Settings.AllowLAN
	if p.Settings.ExternalController != "" {
		settings.ExternalController = p.Settings.ExternalController
	}
	if p.Settings.Secret != "" {
		settings.Secret = p.Settings.Secret
	}
	if opts.MixedPort > 0 {
		settings.MixedPort = opts.MixedPort
	}
	if opts.AllowLAN != nil {
		settings.AllowLAN = *opts.AllowLAN
	}
	if opts.ExternalController != "" {
		settings.ExternalController = opts.ExternalController
	}
	if opts.Secret != "" {
		settings.Secret = opts.Secret
	}

	groups := p.ProxyGroups
	if len(groups) == 0 {
		groups = proxy.DefaultGroups(p.ProxyNames())
	}
	rules := p.Rules
	if len(rules) == 0 {
		rules = proxy.DefaultRules("PROXY")
	}

	// Build proxies
	proxies := make([]map[string]any, 0, len(p.Proxies))
	for _, node := range p.Proxies {
		item := make(map[string]any, len(node.RawOptions)+4)
		for k, v := range node.RawOptions {
			item[k] = v
		}
		item["name"] = node.Name
		item["type"] = node.Type
		item["server"] = node.Server
		item["port"] = node.Port
		proxies = append(proxies, item)
	}

	// Build proxy groups
	mihomoGroups := make([]mihomoGroup, 0, len(groups))
	for _, g := range groups {
		groupType := g.Type
		if groupType == "auto-stable" {
			groupType = "select" // Mihomo doesn't know auto-stable
		}
		mg := mihomoGroup{
			Name:    g.Name,
			Type:    groupType,
			Proxies: g.Proxies,
		}
		if groupType == "url-test" {
			mg.URL = "https://www.gstatic.com/generate_204"
			mg.Interval = 300
			if g.TestURL != "" {
				mg.URL = g.TestURL
			}
		}
		mihomoGroups = append(mihomoGroups, mg)
	}

	// Build rules
	ruleStrings := make([]string, 0, len(rules))
	for _, rule := range rules {
		if rule.Type == "MATCH" {
			ruleStrings = append(ruleStrings, "MATCH,"+rule.TargetGroup)
		} else if rule.Value == "" {
			ruleStrings = append(ruleStrings, rule.Type+","+rule.TargetGroup)
		} else {
			ruleStrings = append(ruleStrings, rule.Type+","+rule.Value+","+rule.TargetGroup)
		}
	}

	// Build DNS config with optional overrides
	dnsConfig := mihomoDNS{
		Enable:       true,
		EnhancedMode: "fake-ip",
		NameServer:   []string{"223.5.5.5", "1.1.1.1"},
	}
	if opts.DNS.Enable {
		dnsConfig.Enable = opts.DNS.Enable
	}
	if opts.DNS.EnhancedMode != "" {
		dnsConfig.EnhancedMode = opts.DNS.EnhancedMode
	}
	if len(opts.DNS.Nameservers) > 0 {
		dnsConfig.NameServer = opts.DNS.Nameservers
	}

	cfg := mihomoConfig{
		MixedPort:          settings.MixedPort,
		AllowLAN:           settings.AllowLAN,
		ExternalController: settings.ExternalController,
		Secret:             settings.Secret,
		Proxies:            proxies,
		ProxyGroups:        mihomoGroups,
		Rules:              ruleStrings,
		DNS:                dnsConfig,
	}

	return yaml.Marshal(cfg)
}
