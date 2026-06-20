package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/y08lin4/proxy-cat/internal/core"
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

func TestRecoverCoreIfNeededRestartsUnexpectedExit(t *testing.T) {
	a := New()
	a.dataDir = t.TempDir()
	a.mihomoHome = filepath.Join(a.dataDir, "mihomo")
	starts := 0
	launcher := core.NewMihomoLauncherWithCommand(func(ctx context.Context, name string, args ...string) *exec.Cmd {
		starts++
		if starts == 1 {
			cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestAppHelperProcess$")
			cmd.Env = append(os.Environ(), "PROXY_CAT_APP_HELPER=unexpected")
			return cmd
		}
		cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestAppHelperProcess$")
		cmd.Env = append(os.Environ(), "PROXY_CAT_APP_HELPER=sleep")
		return cmd
	})
	a.launcher = launcher
	data := []byte(`
proxies:
  - { name: "Node A", type: ss, server: example.com, port: 8388, cipher: aes-128-gcm, password: pass }
`)
	if err := a.LoadSubscriptionData("", data); err != nil {
		t.Fatalf("LoadSubscriptionData() error = %v", err)
	}
	if err := a.StartCore(); err != nil {
		t.Fatalf("StartCore() error = %v", err)
	}
	waitForApp(t, time.Second, func() bool {
		return !launcher.Status().Running
	})
	if !launcher.NeedsRecovery() {
		t.Fatal("launcher.NeedsRecovery() = false, want true")
	}

	status := a.GetAppStatus()
	if !status.CoreRunning {
		t.Fatal("GetAppStatus().CoreRunning = false after automatic recovery, want true")
	}
	if status.LastError != "" {
		t.Fatalf("LastError = %q, want empty after recovery", status.LastError)
	}
	if starts != 2 {
		t.Fatalf("starts = %d, want 2", starts)
	}

	recovered, err := a.RecoverCoreIfNeeded()
	if err != nil {
		t.Fatalf("RecoverCoreIfNeeded() error = %v", err)
	}
	if recovered {
		t.Fatal("RecoverCoreIfNeeded() recovered = true after automatic recovery, want false")
	}

	if err := a.StopCore(); err != nil {
		t.Fatalf("StopCore() error = %v", err)
	}
	recovered, err = a.RecoverCoreIfNeeded()
	if err != nil {
		t.Fatalf("RecoverCoreIfNeeded() after stop error = %v", err)
	}
	if recovered {
		t.Fatal("RecoverCoreIfNeeded() recovered after explicit stop, want false")
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

func TestAppHelperProcess(t *testing.T) {
	switch os.Getenv("PROXY_CAT_APP_HELPER") {
	case "unexpected":
		os.Exit(9)
	case "sleep":
		time.Sleep(30 * time.Second)
	default:
		t.Skip("helper process only")
	}
}

func waitForApp(t *testing.T, timeout time.Duration, ok func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if ok() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("condition was not met before timeout")
}
