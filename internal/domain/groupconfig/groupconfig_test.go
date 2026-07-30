package groupconfig

import (
	"testing"

	"github.com/y08lin4/proxy-cat/internal/domain/proxy"
)

// ---------------------------------------------------------------------------
// LoadBalanceConfig validation
// ---------------------------------------------------------------------------

func TestLoadBalanceConfig_Validate_ValidStrategies(t *testing.T) {
	strategies := []LBStrategy{
		StrategyRoundRobin,
		StrategyConsistentHashing,
		StrategyStickySessions,
	}
	for _, s := range strategies {
		c := LoadBalanceConfig{Strategy: s}
		if err := c.Validate(); err != nil {
			t.Errorf("expected valid strategy %q, got error: %v", s, err)
		}
	}
}

func TestLoadBalanceConfig_Validate_InvalidStrategy(t *testing.T) {
	c := LoadBalanceConfig{Strategy: "bogus"}
	err := c.Validate()
	if err == nil {
		t.Fatal("expected error for invalid strategy, got nil")
	}
}

func TestLoadBalanceConfig_Validate_NegativeStickyMaxAge(t *testing.T) {
	c := LoadBalanceConfig{Strategy: StrategyStickySessions, StickyMaxAge: -1}
	err := c.Validate()
	if err == nil {
		t.Fatal("expected error for negative StickyMaxAge, got nil")
	}
}

func TestLoadBalanceConfig_Validate_StickySessionsDefaultAge(t *testing.T) {
	c := LoadBalanceConfig{Strategy: StrategyStickySessions, StickyMaxAge: 0}
	if err := c.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.StickyMaxAge != 3600 {
		t.Errorf("expected StickyMaxAge to default to 3600, got %d", c.StickyMaxAge)
	}
}

// ---------------------------------------------------------------------------
// FallbackConfig validation
// ---------------------------------------------------------------------------

func TestFallbackConfig_Validate_Valid(t *testing.T) {
	c := DefaultFallbackConfig()
	if err := c.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFallbackConfig_Validate_MissingTestURL(t *testing.T) {
	c := FallbackConfig{TestURL: "", Interval: 300, Tolerance: 150, TimeoutMS: 5000}
	err := c.Validate()
	if err == nil {
		t.Fatal("expected error for missing testUrl, got nil")
	}
}

func TestFallbackConfig_Validate_NegativeInterval(t *testing.T) {
	c := FallbackConfig{TestURL: "http://example.com", Interval: -1, Tolerance: 150, TimeoutMS: 5000}
	err := c.Validate()
	if err == nil {
		t.Fatal("expected error for negative interval, got nil")
	}
}

func TestFallbackConfig_Validate_NegativeTolerance(t *testing.T) {
	c := FallbackConfig{TestURL: "http://example.com", Interval: 300, Tolerance: -1, TimeoutMS: 5000}
	err := c.Validate()
	if err == nil {
		t.Fatal("expected error for negative tolerance, got nil")
	}
}

func TestFallbackConfig_Validate_ZeroTimeoutDefaults(t *testing.T) {
	c := FallbackConfig{TestURL: "http://example.com", Interval: 300, Tolerance: 150, TimeoutMS: 0}
	if err := c.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.TimeoutMS != 5000 {
		t.Errorf("expected TimeoutMS to default to 5000, got %d", c.TimeoutMS)
	}
}

// ---------------------------------------------------------------------------
// ApplyToGroup
// ---------------------------------------------------------------------------

func TestLoadBalanceConfig_ApplyToGroup(t *testing.T) {
	c := LoadBalanceConfig{
		Strategy:     StrategyStickySessions,
		StickyMaxAge: 7200,
	}

	g := &proxy.ProxyGroup{}
	c.ApplyToGroup(g)

	if g.Type != "load-balance" {
		t.Errorf("expected type load-balance, got %q", g.Type)
	}
	if g.Strategy != string(StrategyStickySessions) {
		t.Errorf("expected strategy sticky-sessions, got %q", g.Strategy)
	}
	if g.StickyMaxAge != 7200 {
		t.Errorf("expected StickyMaxAge 7200, got %d", g.StickyMaxAge)
	}
}

func TestFallbackConfig_ApplyToGroup(t *testing.T) {
	c := FallbackConfig{
		TestURL:   "https://cp.cloudflare.com",
		Interval:  600,
		Tolerance: 200,
		Lazy:      true,
		TimeoutMS: 3000,
	}

	g := &proxy.ProxyGroup{}
	c.ApplyToGroup(g)

	if g.Type != "fallback" {
		t.Errorf("expected type fallback, got %q", g.Type)
	}
	if g.TestURL != "https://cp.cloudflare.com" {
		t.Errorf("expected test URL cp.cloudflare.com, got %q", g.TestURL)
	}
	if g.Interval != 600 {
		t.Errorf("expected interval 600, got %d", g.Interval)
	}
	if g.Tolerance != 200 {
		t.Errorf("expected tolerance 200, got %d", g.Tolerance)
	}
	if !g.Lazy {
		t.Error("expected lazy to be true")
	}
}
