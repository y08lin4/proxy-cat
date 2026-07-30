package groupconfig

import (
	"errors"

	"github.com/y08lin4/proxy-cat/internal/domain/proxy"
)

// LBStrategy represents the load-balancing strategy type.
type LBStrategy string

const (
	StrategyRoundRobin        LBStrategy = "round-robin"
	StrategyConsistentHashing LBStrategy = "consistent-hashing"
	StrategyStickySessions    LBStrategy = "sticky-sessions"
)

// LoadBalanceConfig holds configuration for a load-balance proxy group.
type LoadBalanceConfig struct {
	Strategy     LBStrategy `json:"strategy"`
	StickyMaxAge int        `json:"stickyMaxAge,omitempty"`
}

// DefaultLoadBalanceConfig returns a LoadBalanceConfig with safe defaults.
func DefaultLoadBalanceConfig() LoadBalanceConfig {
	return LoadBalanceConfig{Strategy: StrategyRoundRobin}
}

// Validate checks the config and applies defaults where needed.
func (c *LoadBalanceConfig) Validate() error {
	switch c.Strategy {
	case StrategyRoundRobin, StrategyConsistentHashing, StrategyStickySessions:
	default:
		return errors.New("invalid load-balance strategy: " + string(c.Strategy))
	}
	if c.StickyMaxAge < 0 {
		return errors.New("stickyMaxAge must be non-negative")
	}
	if c.Strategy == StrategyStickySessions && c.StickyMaxAge == 0 {
		c.StickyMaxAge = 3600 // default 1 hour
	}
	return nil
}

// ApplyToGroup writes load-balance configuration into a ProxyGroup.
func (c LoadBalanceConfig) ApplyToGroup(g *proxy.ProxyGroup) {
	g.Type = "load-balance"
	g.Strategy = string(c.Strategy)
	g.StickyMaxAge = c.StickyMaxAge
}
