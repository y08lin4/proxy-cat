package matrix

import (
	"testing"

	"github.com/y08lin4/proxy-cat/internal/domain/proxy"
)

func TestEngineGenerateBasic(t *testing.T) {
	p := proxy.Profile{
		Name: "test",
		Proxies: []proxy.ProxyNode{
			{Name: "front-01", Type: "socks5", Server: "1.1.1.1", Port: 1080},
			{Name: "front-02", Type: "socks5", Server: "2.2.2.2", Port: 1080},
		},
	}

	eng := NewEngine()
	result, err := eng.Generate(p, MatrixConfig{
		FrontNodes: []string{"front-01", "front-02"},
		ExitNodes: []ExitSpec{
			{Name: "us-01", Type: "ss", Server: "us.example.com", Port: 443, Region: "US", RawOptions: map[string]any{"cipher": "aes-128-gcm", "password": "test"}},
			{Name: "jp-01", Type: "ss", Server: "jp.example.com", Port: 443, Region: "JP", RawOptions: map[string]any{"cipher": "aes-128-gcm", "password": "test"}},
		},
	})

	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}

	// 2 fronts × 2 exits = 4 composite nodes total
	if len(result.CompositeNodes) != 4 {
		t.Errorf("expected 4 composite nodes, got %d", len(result.CompositeNodes))
	}

	// 2 regions = 2 groups
	if len(result.Groups) != 2 {
		t.Errorf("expected 2 groups, got %d", len(result.Groups))
	}

	// Each group should have 2 proxies (one per front)
	for _, g := range result.Groups {
		if len(g.Proxies) != 2 {
			t.Errorf("group %s: expected 2 proxies, got %d", g.Name, len(g.Proxies))
		}
		if g.Type != "url-test" {
			t.Errorf("group %s: expected type url-test, got %s", g.Name, g.Type)
		}
	}

	// Check dialer-proxy is set
	for _, node := range result.CompositeNodes {
		dp, ok := node.RawOptions["dialer-proxy"]
		if !ok {
			t.Errorf("node %s: missing dialer-proxy", node.Name)
		}
		if dpStr := dp.(string); dpStr != "front-01" && dpStr != "front-02" {
			t.Errorf("node %s: unexpected dialer-proxy value %q", node.Name, dpStr)
		}
	}
}

func TestEngineValidateMissingFront(t *testing.T) {
	p := proxy.Profile{Name: "test"}
	eng := NewEngine()
	_, err := eng.Generate(p, MatrixConfig{
		FrontNodes: []string{"nonexistent"},
		ExitNodes:  []ExitSpec{{Name: "exit", Type: "ss", Server: "x.com", Port: 443, Region: "US"}},
	})
	if err == nil {
		t.Error("expected error for missing front node, got nil")
	}
}

func TestEngineEmptyInputs(t *testing.T) {
	eng := NewEngine()

	_, err := eng.Generate(proxy.Profile{}, MatrixConfig{})
	if err == nil {
		t.Error("expected error for empty config")
	}

	_, err = eng.Generate(proxy.Profile{}, MatrixConfig{
		FrontNodes: []string{"f1"},
		ExitNodes:  []ExitSpec{},
	})
	if err == nil {
		t.Error("expected error for empty exits")
	}
}

func TestMergeIntoProfile(t *testing.T) {
	p := &proxy.Profile{
		Name: "test",
		Proxies: []proxy.ProxyNode{
			{Name: "original", Type: "socks5", Server: "1.1.1.1", Port: 1080},
		},
		ProxyGroups: []proxy.ProxyGroup{
			{Name: "EXIT-US", Type: "url-test", Proxies: []string{"old"}},
		},
	}

	result := &GenerateResult{
		CompositeNodes: []proxy.ProxyNode{
			{Name: "us-via-f1-01", Type: "ss", Server: "us.com", Port: 443, RawOptions: map[string]any{"dialer-proxy": "f1"}},
		},
		Groups: []proxy.ProxyGroup{
			{Name: "EXIT-US", Type: "url-test", Proxies: []string{"us-via-f1-01"}},
		},
	}

	result.MergeIntoProfile(p)

	// Should have original + composite nodes
	if len(p.Proxies) != 2 {
		t.Errorf("expected 2 proxies (1 original + 1 composite), got %d", len(p.Proxies))
	}

	// Old EXIT-US group should be replaced
	if len(p.ProxyGroups) != 1 {
		t.Errorf("expected 1 group, got %d", len(p.ProxyGroups))
	}
	if p.ProxyGroups[0].Proxies[0] != "us-via-f1-01" {
		t.Errorf("group not updated correctly")
	}
}
