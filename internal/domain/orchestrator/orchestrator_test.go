package orchestrator

import (
	"testing"
	"time"

	"github.com/y08lin4/proxy-cat/internal/autostable"
	"github.com/y08lin4/proxy-cat/internal/domain/proxy"
)

func newTestManager(t *testing.T) *autostable.Manager {
	t.Helper()
	m, err := autostable.NewManager(autostable.Config{})
	if err != nil {
		t.Fatal(err)
	}
	return m
}

func TestResolveChain(t *testing.T) {
	p := proxy.Profile{
		Proxies: []proxy.ProxyNode{
			{Name: "front-01", Type: "socks5", Server: "1.1.1.1", Port: 1080},
			{Name: "us-via-front-01", Type: "ss", Server: "us.exit.com", Port: 443, RawOptions: map[string]any{"dialer-proxy": "front-01"}},
			{Name: "us-via-front-02", Type: "ss", Server: "us.exit.com", Port: 443, RawOptions: map[string]any{"dialer-proxy": "front-01"}},
		},
	}

	o := New(newTestManager(t))
	trace, err := o.ResolveChain(p, "us-via-front-01")
	if err != nil {
		t.Fatalf("ResolveChain() error = %v", err)
	}
	if trace.TotalHops != 2 {
		t.Errorf("expected 2 hops, got %d", trace.TotalHops)
	}
	if trace.Hops[0].Name != "us-via-front-01" {
		t.Errorf("first hop should be us-via-front-01, got %s", trace.Hops[0].Name)
	}
	if trace.Hops[1].Name != "front-01" {
		t.Errorf("second hop should be front-01, got %s", trace.Hops[1].Name)
	}
}

func TestValidateDialerChains(t *testing.T) {
	p := proxy.Profile{
		Proxies: []proxy.ProxyNode{
			{Name: "front-01", Type: "socks5", Server: "1.1.1.1", Port: 1080},
			{Name: "us-via-front-01", Type: "ss", Server: "us.exit.com", Port: 443, RawOptions: map[string]any{"dialer-proxy": "front-01"}},
			{Name: "bad-chain", Type: "ss", Server: "bad.com", Port: 443, RawOptions: map[string]any{"dialer-proxy": "missing-front"}},
		},
	}

	o := New(newTestManager(t))
	results := o.ValidateDialerChains(p)

	if len(results) != 2 {
		t.Fatalf("expected 2 validations, got %d", len(results))
	}

	// us-via-front-01 should be valid (front-01 exists)
	if !results[0].Valid {
		t.Errorf("us-via-front-01 chain should be valid")
	}
	// bad-chain should be invalid (missing-front doesn't exist)
	if results[1].Valid {
		t.Errorf("bad-chain should be invalid")
	}
}

func TestGetDialerStatus(t *testing.T) {
	p := proxy.Profile{
		Proxies: []proxy.ProxyNode{
			{Name: "front-01", Type: "socks5", Server: "1.1.1.1", Port: 1080},
			{Name: "us-via-front-01", Type: "ss", Server: "us.exit.com", Port: 443, RawOptions: map[string]any{"dialer-proxy": "front-01"}},
		},
	}

	m := newTestManager(t)
	m.Register("front-01")
	m.Register("us-via-front-01")
	m.Record(autostable.Sample{NodeID: "front-01", Latency: 50 * time.Millisecond, Success: true, CheckedAt: time.Now()})
	m.Record(autostable.Sample{NodeID: "us-via-front-01", Latency: 100 * time.Millisecond, Success: true, CheckedAt: time.Now()})

	o := New(m)
	statuses := o.GetDialerStatus(p)

	if len(statuses) < 3 {
		t.Fatalf("expected at least 3 status entries, got %d", len(statuses))
	}
}
