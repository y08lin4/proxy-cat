package core

import (
	"context"
	"reflect"
	"testing"
)

type recordedCommand struct {
	Name string
	Args []string
}

type recordingRunner struct {
	commands []recordedCommand
}

func (r *recordingRunner) Run(_ context.Context, name string, args ...string) error {
	r.commands = append(r.commands, recordedCommand{Name: name, Args: append([]string(nil), args...)})
	return nil
}

func TestWindowsSystemProxyEnableWritesRegistryValues(t *testing.T) {
	runner := &recordingRunner{}
	proxy := &WindowsSystemProxy{runner: runner}

	err := proxy.Enable(context.Background(), "127.0.0.1:7890", "<local>")
	if err != nil {
		t.Fatalf("Enable() error = %v", err)
	}

	wantValues := []string{"ProxyServer", "ProxyOverride", "ProxyEnable"}
	if len(runner.commands) != len(wantValues) {
		t.Fatalf("commands = %d, want %d", len(runner.commands), len(wantValues))
	}
	for i, valueName := range wantValues {
		if runner.commands[i].Name != "reg" {
			t.Fatalf("command %d name = %q, want reg", i, runner.commands[i].Name)
		}
		if !reflect.DeepEqual(runner.commands[i].Args[:6], []string{"add", internetSettingsKey, "/v", valueName, "/t", runner.commands[i].Args[5]}) {
			t.Fatalf("command %d args = %#v", i, runner.commands[i].Args)
		}
	}
	if got := runner.commands[0].Args[7]; got != "127.0.0.1:7890" {
		t.Fatalf("ProxyServer value = %q, want 127.0.0.1:7890", got)
	}
	if got := runner.commands[2].Args[7]; got != "1" {
		t.Fatalf("ProxyEnable value = %q, want 1", got)
	}
}

func TestWindowsSystemProxyDisable(t *testing.T) {
	runner := &recordingRunner{}
	proxy := &WindowsSystemProxy{runner: runner}

	err := proxy.Disable(context.Background())
	if err != nil {
		t.Fatalf("Disable() error = %v", err)
	}

	if len(runner.commands) != 1 {
		t.Fatalf("commands = %d, want 1", len(runner.commands))
	}
	if got := runner.commands[0].Args[7]; got != "0" {
		t.Fatalf("ProxyEnable value = %q, want 0", got)
	}
}
