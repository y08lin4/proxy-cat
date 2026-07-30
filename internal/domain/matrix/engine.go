package matrix

import (
	"fmt"
	"strings"

	"github.com/y08lin4/proxy-cat/internal/domain/proxy"
)

// Engine generates composite proxy nodes using dialer-proxy chains.
type Engine struct{}

// NewEngine creates a new matrix engine.
func NewEngine() *Engine { return &Engine{} }

// MatrixConfig specifies how to generate the node matrix.
type MatrixConfig struct {
	// FrontNodes are proxy names that serve as entry/front proxies.
	FrontNodes []string
	// ExitNodes are exit/destination proxy specs.
	ExitNodes []ExitSpec
	// GroupPrefix is the prefix for generated url-test group names (e.g., "EXIT").
	GroupPrefix string
}

// ExitSpec describes an exit node with its target region/group.
type ExitSpec struct {
	Name       string         // e.g., "us-exit-01"
	Type       string         // proxy type: ss, vmess, trojan, etc.
	Server     string
	Port       int
	Region     string // e.g., "US", "JP", "SG"
	RawOptions map[string]any // additional Mihomo fields (cipher, password, uuid, etc.)
}

// GenerateResult contains the generated composite matrix.
type GenerateResult struct {
	// CompositeNodes are the N×M dialer-proxy composite nodes.
	CompositeNodes []proxy.ProxyNode
	// Groups are url-test groups, one per exit region.
	Groups []proxy.ProxyGroup
	// Rules are auto-generated routing rules.
	Rules []proxy.Rule
}

// Generate produces the full matrix from config and existing profile.
func (e *Engine) Generate(p proxy.Profile, cfg MatrixConfig) (*GenerateResult, error) {
	// Validate inputs
	if len(cfg.FrontNodes) == 0 {
		return nil, fmt.Errorf("at least one front node is required")
	}
	if len(cfg.ExitNodes) == 0 {
		return nil, fmt.Errorf("at least one exit node is required")
	}

	// Build front node lookup for validation
	frontSet := make(map[string]bool, len(cfg.FrontNodes))
	for _, name := range cfg.FrontNodes {
		frontSet[name] = true
	}

	// Validate all front nodes exist in profile
	existingNames := make(map[string]bool, len(p.Proxies))
	for _, node := range p.Proxies {
		existingNames[node.Name] = true
	}
	for _, name := range cfg.FrontNodes {
		if !existingNames[name] {
			return nil, fmt.Errorf("front node %q not found in profile", name)
		}
	}

	// Group exit nodes by region
	regionNodes := make(map[string][]ExitSpec)
	for _, exit := range cfg.ExitNodes {
		region := exit.Region
		if region == "" {
			region = "DEFAULT"
		}
		regionNodes[region] = append(regionNodes[region], exit)
	}

	var compositeNodes []proxy.ProxyNode
	var groups []proxy.ProxyGroup
	var rules []proxy.Rule

	// Generate N×M composites and url-test groups
	for region, exits := range regionNodes {
		groupProxies := make([]string, 0)

		for i, exit := range exits {
			for j, front := range cfg.FrontNodes {
				// Generate composite node name: {region}-via-{front}
				compositeName := fmt.Sprintf("%s-via-%s-%02d", strings.ToLower(region), sanitizeName(front), (i*len(cfg.FrontNodes))+j+1)

				// Build composite proxy node
				rawOpts := make(map[string]any)
				for k, v := range exit.RawOptions {
					rawOpts[k] = v
				}
				rawOpts["dialer-proxy"] = front

				composite := proxy.ProxyNode{
					ID:         sanitizeName(compositeName),
					Name:       compositeName,
					Type:       exit.Type,
					Server:     exit.Server,
					Port:       exit.Port,
					RawOptions: rawOpts,
				}
				compositeNodes = append(compositeNodes, composite)
				groupProxies = append(groupProxies, compositeName)
			}
		}

		// Create url-test group for this region
		if cfg.GroupPrefix == "" {
			cfg.GroupPrefix = "EXIT"
		}
		groupName := fmt.Sprintf("%s-%s", cfg.GroupPrefix, strings.ToUpper(region))
		groups = append(groups, proxy.ProxyGroup{
			Name:    groupName,
			Type:    "url-test",
			Proxies: groupProxies,
		})

		// Auto-generate a MATCH rule pointing to first region
		// (caller can override with domain-specific rules)
	}

	// Default rule: MATCH -> first group
	if len(groups) > 0 {
		rules = append(rules, proxy.Rule{
			Type:        "MATCH",
			TargetGroup: groups[0].Name,
		})
	}

	return &GenerateResult{
		CompositeNodes: compositeNodes,
		Groups:         groups,
		Rules:          rules,
	}, nil
}

// MergeIntoProfile merges generated results into an existing profile,
// replacing any existing auto-stable groups with the generated ones.
func (r *GenerateResult) MergeIntoProfile(p *proxy.Profile) {
	// Add composite nodes
	p.Proxies = append(p.Proxies, r.CompositeNodes...)
	// Replace or add groups
	for _, g := range r.Groups {
		// Remove old group with same name
		filtered := make([]proxy.ProxyGroup, 0, len(p.ProxyGroups))
		for _, existing := range p.ProxyGroups {
			if existing.Name != g.Name {
				filtered = append(filtered, existing)
			}
		}
		p.ProxyGroups = filtered
		p.ProxyGroups = append(p.ProxyGroups, g)
	}
}

func sanitizeName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	var b strings.Builder
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-' || r == '_':
			b.WriteRune(r)
		case r == ' ':
			b.WriteRune('-')
		}
	}
	if b.Len() == 0 {
		return "node"
	}
	return b.String()
}
