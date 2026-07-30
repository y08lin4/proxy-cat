package groupconfig

import (
	"errors"

	"github.com/y08lin4/proxy-cat/internal/domain/proxy"
)

// FallbackConfig holds configuration for a fallback proxy group.
type FallbackConfig struct {
	TestURL   string `json:"testUrl"`
	Interval  int    `json:"interval"`
	Tolerance int    `json:"tolerance"`
	Lazy      bool   `json:"lazy"`
	TimeoutMS int    `json:"timeoutMs"`
}

// DefaultFallbackConfig returns a FallbackConfig with safe defaults.
func DefaultFallbackConfig() FallbackConfig {
	return FallbackConfig{
		TestURL:   "https://www.gstatic.com/generate_204",
		Interval:  300,
		Tolerance: 150,
		TimeoutMS: 5000,
	}
}

// Validate checks the config and applies defaults where needed.
func (c *FallbackConfig) Validate() error {
	if c.TestURL == "" {
		return errors.New("testUrl is required for fallback group")
	}
	if c.Interval < 0 {
		return errors.New("interval must be non-negative")
	}
	if c.Tolerance < 0 {
		return errors.New("tolerance must be non-negative")
	}
	if c.TimeoutMS <= 0 {
		c.TimeoutMS = 5000
	}
	return nil
}

// ApplyToGroup writes fallback configuration into a ProxyGroup.
func (c FallbackConfig) ApplyToGroup(g *proxy.ProxyGroup) {
	g.Type = "fallback"
	g.TestURL = c.TestURL
	g.Interval = c.Interval
	g.Tolerance = c.Tolerance
	g.Lazy = c.Lazy
}
