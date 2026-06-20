package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadSubscriptionDataGeneratesConfigAndGroups(t *testing.T) {
	a := New()
	a.dataDir = t.TempDir()
	a.mihomoHome = filepath.Join(a.dataDir, "mihomo")

	data := []byte(`
proxies:
  - { name: "Node A", type: ss, server: example.com, port: 8388, cipher: aes-128-gcm, password: pass }
rules:
  - MATCH,PROXY
`)

	if err := a.LoadSubscriptionData("https://example.test/sub", data); err != nil {
		t.Fatalf("LoadSubscriptionData() error = %v", err)
	}

	status := a.GetAppStatus()
	if status.ActiveProfileName != "Default" {
		t.Fatalf("ActiveProfileName = %q, want Default", status.ActiveProfileName)
	}
	groups := a.GetProxyGroups()
	if !hasGroupWithProxy(groups, "PROXY", "Node A") {
		t.Fatalf("groups missing PROXY/Node A: %+v", groups)
	}
	config, err := os.ReadFile(filepath.Join(a.dataDir, "profiles", "active", "config.yaml"))
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if !strings.Contains(string(config), "mixed-port: 7890") || !strings.Contains(string(config), "Node A") {
		t.Fatalf("generated config missing expected content:\n%s", string(config))
	}
}

func TestSelectProxyUpdatesInMemoryGroupWhenCoreStopped(t *testing.T) {
	a := New()
	a.dataDir = t.TempDir()
	data := []byte(`
proxies:
  - { name: "Node A", type: ss, server: example.com, port: 8388, cipher: aes-128-gcm, password: pass }
  - { name: "Node B", type: ss, server: example.org, port: 8389, cipher: aes-128-gcm, password: pass }
proxy-groups:
  - name: PROXY
    type: select
    proxies:
      - Node A
      - Node B
`)
	if err := a.LoadSubscriptionData("", data); err != nil {
		t.Fatalf("LoadSubscriptionData() error = %v", err)
	}
	if err := a.SelectProxy("PROXY", "Node B"); err != nil {
		t.Fatalf("SelectProxy() error = %v", err)
	}

	groups := a.GetProxyGroups()
	if len(groups) != 1 {
		t.Fatalf("len(groups) = %d, want 1", len(groups))
	}
	if groups[0].Selected != "Node B" {
		t.Fatalf("Selected = %q, want Node B", groups[0].Selected)
	}
}

func hasGroupWithProxy(groups []ProxyGroupView, groupName string, proxyName string) bool {
	for _, group := range groups {
		if group.Name != groupName {
			continue
		}
		for _, proxy := range group.Proxies {
			if proxy.Name == proxyName {
				return true
			}
		}
	}
	return false
}
