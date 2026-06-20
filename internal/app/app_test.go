package app

import (
	"net/http"
	"net/http/httptest"
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

func TestGetConnectionStatusFetchesMihomoConnections(t *testing.T) {
	var gotPath string
	var gotAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"uploadTotal": 1234,
			"downloadTotal": 5678,
			"connections": [{}, {}, {}]
		}`))
	}))
	defer server.Close()

	a := New()
	a.dataDir = t.TempDir()
	a.httpClient = server.Client()
	a.launcher = nil
	data := []byte(`
external-controller: ` + server.URL + `
secret: local-secret
proxies:
  - { name: "Node A", type: ss, server: example.com, port: 8388, cipher: aes-128-gcm, password: pass }
`)
	if err := a.LoadSubscriptionData("", data); err != nil {
		t.Fatalf("LoadSubscriptionData() error = %v", err)
	}
	a.status.CoreRunning = true

	status, err := a.GetConnectionStatus()
	if err != nil {
		t.Fatalf("GetConnectionStatus() error = %v", err)
	}

	if gotPath != "/connections" {
		t.Fatalf("path = %s, want /connections", gotPath)
	}
	if gotAuth != "Bearer local-secret" {
		t.Fatalf("Authorization = %q, want bearer token", gotAuth)
	}
	if !status.CoreRunning {
		t.Fatal("CoreRunning = false, want true")
	}
	if status.UploadTotal != 1234 || status.DownloadTotal != 5678 || status.ConnectionCount != 3 {
		t.Fatalf("unexpected connection status: %#v", status)
	}
}

func TestGetConnectionStatusStoppedDoesNotCallMihomo(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		http.Error(w, "unexpected call", http.StatusInternalServerError)
	}))
	defer server.Close()

	a := New()
	a.httpClient = server.Client()
	a.launcher = nil
	a.status.ControllerAddress = server.URL
	a.status.CoreRunning = false

	status, err := a.GetConnectionStatus()
	if err != nil {
		t.Fatalf("GetConnectionStatus() error = %v", err)
	}
	if called {
		t.Fatal("server was called while core was stopped")
	}
	if status.CoreRunning || status.UploadTotal != 0 || status.DownloadTotal != 0 || status.ConnectionCount != 0 {
		t.Fatalf("unexpected stopped status: %#v", status)
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
