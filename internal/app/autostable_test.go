package app

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func TestAutoStableStatusBuildsHealthViewFromActiveProfile(t *testing.T) {
	a := New()
	a.dataDir = t.TempDir()
	a.mihomoHome = filepath.Join(a.dataDir, "mihomo")

	data := []byte(`
proxies:
  - { name: "Node A", type: ss, server: example.com, port: 8388, cipher: aes-128-gcm, password: pass }
proxy-groups:
  - name: AUTO
    type: select
    proxies:
      - Node A
`)
	if err := a.LoadSubscriptionData("", data); err != nil {
		t.Fatalf("LoadSubscriptionData() error = %v", err)
	}

	status := a.GetAutoStableStatus()
	if !status.Available {
		t.Fatal("Available = false, want true after loading subscription")
	}
	if len(status.Health) != 1 {
		t.Fatalf("len(Health) = %d, want 1", len(status.Health))
	}
	if status.Health[0].Name != "AUTO" || len(status.Health[0].Proxies) != 1 {
		t.Fatalf("unexpected health view: %#v", status.Health)
	}
	if status.Health[0].Proxies[0].Name != "Node A" || !status.Health[0].Proxies[0].Alive {
		t.Fatalf("unexpected node health: %#v", status.Health[0].Proxies[0])
	}
}

func TestSetAutoStableEnabledUpdatesStatusAndLogs(t *testing.T) {
	a := New()

	if err := a.SetAutoStableEnabled(true); err != nil {
		t.Fatalf("SetAutoStableEnabled(true) error = %v", err)
	}

	if !a.GetAppStatus().AutoStableEnabled {
		t.Fatal("AutoStableEnabled = false, want true")
	}
	status := a.GetAutoStableStatus()
	if !status.Enabled {
		t.Fatal("status.Enabled = false, want true")
	}
	logs := a.GetLogs(1)
	if len(logs) != 1 || logs[0].Message != "Auto-stable enabled" {
		t.Fatalf("unexpected logs: %#v", logs)
	}
}

func TestRunAutoStableTickUnavailableIsControllable(t *testing.T) {
	a := New()

	result, err := a.RunAutoStableTick()
	if !errors.Is(err, ErrAutoStableUnavailable) {
		t.Fatalf("RunAutoStableTick() error = %v, want ErrAutoStableUnavailable", err)
	}
	if result.Action != "tick" || result.Changed {
		t.Fatalf("unexpected result: %#v", result)
	}
	status := a.GetAutoStableStatus()
	if status.LastAction != "tick" || status.LastError == "" {
		t.Fatalf("unexpected status: %#v", status)
	}
}

func TestAutoStableRunnerMethodsAreWired(t *testing.T) {
	a := New()
	completedAt := time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC)
	runner := &fakeAutoStableRunner{
		status: AutoStableStatus{Available: true, Enabled: true, Running: true},
		tick: AutoStableActionResult{
			Action:      "tick",
			Selected:    "Node B",
			Changed:     true,
			Message:     "Auto-stable selected Node B",
			CompletedAt: completedAt,
		},
		selectResult: AutoStableActionResult{
			Action:      "select",
			GroupName:   "PROXY",
			Selected:    "Node A",
			Changed:     true,
			CompletedAt: completedAt.Add(time.Minute),
		},
	}
	a.autoStable = runner

	if err := a.SetAutoStableEnabled(true); err != nil {
		t.Fatalf("SetAutoStableEnabled(true) error = %v", err)
	}
	if !runner.enabled {
		t.Fatal("runner.enabled = false, want true")
	}

	status := a.GetAutoStableStatus()
	if !status.Available || !status.Enabled || !status.Running {
		t.Fatalf("unexpected status: %#v", status)
	}

	tick, err := a.RunAutoStableTick()
	if err != nil {
		t.Fatalf("RunAutoStableTick() error = %v", err)
	}
	if tick.Selected != "Node B" {
		t.Fatalf("tick.Selected = %q, want Node B", tick.Selected)
	}

	selected, err := a.SelectAutoStableProxy("PROXY")
	if err != nil {
		t.Fatalf("SelectAutoStableProxy() error = %v", err)
	}
	if selected.Selected != "Node A" || runner.selectedGroup != "PROXY" {
		t.Fatalf("unexpected select result=%#v selectedGroup=%q", selected, runner.selectedGroup)
	}
}

func TestRunAutoStableTickCooldownSkipsRunner(t *testing.T) {
	a := New()
	completedAt := time.Now()
	runner := &fakeAutoStableRunner{
		status: AutoStableStatus{Available: true, Enabled: true, Running: true},
		tick: AutoStableActionResult{
			Action:      "tick",
			Selected:    "Node A",
			CompletedAt: completedAt,
		},
	}
	a.autoStable = runner

	first, err := a.RunAutoStableTick()
	if err != nil {
		t.Fatalf("first RunAutoStableTick() error = %v", err)
	}
	if first.Selected != "Node A" {
		t.Fatalf("first.Selected = %q, want Node A", first.Selected)
	}

	second, err := a.RunAutoStableTick()
	if err != nil {
		t.Fatalf("second RunAutoStableTick() error = %v", err)
	}
	if second.Changed || second.Message != "Auto-stable tick skipped by cooldown" {
		t.Fatalf("unexpected cooldown result: %#v", second)
	}
	if runner.tickCalls != 1 {
		t.Fatalf("tickCalls = %d, want 1", runner.tickCalls)
	}
	status := a.GetAutoStableStatus()
	if status.LastError != "Auto-stable tick skipped by cooldown" {
		t.Fatalf("LastError = %q, want cooldown message", status.LastError)
	}
}

type fakeAutoStableRunner struct {
	status        AutoStableStatus
	tick          AutoStableActionResult
	selectResult  AutoStableActionResult
	enabled       bool
	selectedGroup string
	tickCalls     int
}

func (f *fakeAutoStableRunner) Status(context.Context) (AutoStableStatus, error) {
	return f.status, nil
}

func (f *fakeAutoStableRunner) SetEnabled(_ context.Context, enabled bool) error {
	f.enabled = enabled
	f.status.Enabled = enabled
	return nil
}

func (f *fakeAutoStableRunner) Tick(context.Context) (AutoStableActionResult, error) {
	f.tickCalls++
	return f.tick, nil
}

func (f *fakeAutoStableRunner) Select(_ context.Context, groupName string) (AutoStableActionResult, error) {
	f.selectedGroup = groupName
	return f.selectResult, nil
}
