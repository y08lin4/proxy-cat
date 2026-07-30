package system

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

type recordedCommand struct {
	Name    string
	Args    []string
	IsQuery bool // true for RunOutput calls (reg query), false for Run calls (reg add)
}

type recordingRunner struct {
	commands  []recordedCommand
	queryOuts map[string]string // keyed by valueName, returns simulated reg query output
}

func (r *recordingRunner) Run(_ context.Context, name string, args ...string) error {
	r.commands = append(r.commands, recordedCommand{Name: name, Args: append([]string(nil), args...), IsQuery: false})
	return nil
}

func (r *recordingRunner) RunOutput(_ context.Context, name string, args ...string) (string, error) {
	r.commands = append(r.commands, recordedCommand{Name: name, Args: append([]string(nil), args...), IsQuery: true})

	// Extract the value name from args: "HKCU\...Internet Settings", "/v", "ProxyEnable"
	var valueName string
	for i, a := range args {
		if a == "/v" && i+1 < len(args) {
			valueName = args[i+1]
			break
		}
	}

	if out, ok := r.queryOuts[valueName]; ok {
		return out, nil
	}
	return "", fmt.Errorf("reg query %s: not found", valueName)
}

// regQueryOutput builds a realistic reg query output string.
func regQueryOutput(valueName, regType, data string) string {
	// Simulated reg query output format
	return fmt.Sprintf("\nHKEY_CURRENT_USER\\Software\\Microsoft\\Windows\\CurrentVersion\\Internet Settings\n    %s    %s    %s\n\n", valueName, regType, data)
}

func TestWindowsSystemProxyEnableWritesRegistryValues(t *testing.T) {
	runner := &recordingRunner{
		queryOuts: map[string]string{
			"ProxyEnable":   regQueryOutput("ProxyEnable", "REG_DWORD", "0x0"),
			"ProxyServer":   regQueryOutput("ProxyServer", "REG_SZ", ""),
			"ProxyOverride": regQueryOutput("ProxyOverride", "REG_SZ", ""),
		},
	}
	proxy := &WindowsSystemProxy{runner: runner}

	err := proxy.Enable(context.Background(), "127.0.0.1:7890", "<local>")
	if err != nil {
		t.Fatalf("Enable() error = %v", err)
	}

	// 3 reg query (backup) + 3 reg add = 6 commands
	if len(runner.commands) != 6 {
		t.Fatalf("commands = %d, want 6 (3 queries + 3 adds)", len(runner.commands))
	}

	// Verify first 3 commands are reg query (backup)
	queryNames := []string{"ProxyEnable", "ProxyServer", "ProxyOverride"}
	for i, vn := range queryNames {
		cmd := runner.commands[i]
		if !cmd.IsQuery {
			t.Fatalf("command %d should be a query, got %v", i, cmd)
		}
		if cmd.Name != "reg" {
			t.Fatalf("command %d name = %q, want reg", i, cmd.Name)
		}
		if cmd.Args[0] != "query" {
			t.Fatalf("command %d subcmd = %q, want query", i, cmd.Args[0])
		}
		if cmd.Args[3] != vn {
			t.Fatalf("command %d value name = %q, want %q", i, cmd.Args[3], vn)
		}
	}

	// Verify last 3 commands are reg add (enable)
	wantValues := []string{"ProxyServer", "ProxyOverride", "ProxyEnable"}
	for i, valueName := range wantValues {
		idx := 3 + i
		cmd := runner.commands[idx]
		if cmd.Name != "reg" {
			t.Fatalf("command %d name = %q, want reg", idx, cmd.Name)
		}
		if !reflect.DeepEqual(cmd.Args[:6], []string{"add", internetSettingsKey, "/v", valueName, "/t", cmd.Args[5]}) {
			t.Fatalf("command %d args = %#v", idx, cmd.Args)
		}
	}
	if got := runner.commands[3].Args[7]; got != "127.0.0.1:7890" {
		t.Fatalf("ProxyServer value = %q, want 127.0.0.1:7890", got)
	}
	if got := runner.commands[5].Args[7]; got != "1" {
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

// TestBackupBeforeEnable verifies that Enable() performs a backup (reg query)
// before writing registry values (reg add).
func TestBackupBeforeEnable(t *testing.T) {
	runner := &recordingRunner{
		queryOuts: map[string]string{
			"ProxyEnable":   regQueryOutput("ProxyEnable", "REG_DWORD", "0x0"),
			"ProxyServer":   regQueryOutput("ProxyServer", "REG_SZ", ""),
			"ProxyOverride": regQueryOutput("ProxyOverride", "REG_SZ", ""),
		},
	}
	proxy := &WindowsSystemProxy{runner: runner}

	err := proxy.Enable(context.Background(), "127.0.0.1:7890", "bypass")
	if err != nil {
		t.Fatalf("Enable() error = %v", err)
	}

	// Separate queries and writes
	var queries, writes []recordedCommand
	for _, cmd := range runner.commands {
		if cmd.IsQuery {
			queries = append(queries, cmd)
		} else {
			writes = append(writes, cmd)
		}
	}

	if len(queries) != 3 {
		t.Fatalf("expected 3 reg query commands, got %d", len(queries))
	}
	if len(writes) != 3 {
		t.Fatalf("expected 3 reg add commands, got %d", len(writes))
	}

	// All queries must come before all writes
	lastQueryIndex := -1
	firstWriteIndex := len(runner.commands)
	for i, cmd := range runner.commands {
		if cmd.IsQuery && i > lastQueryIndex {
			lastQueryIndex = i
		}
		if !cmd.IsQuery && i < firstWriteIndex {
			firstWriteIndex = i
		}
	}
	if firstWriteIndex < lastQueryIndex {
		t.Fatal("reg add (write) occurred before reg query (backup) completed")
	}

	t.Logf("Ordered commands: %d queries then %d writes", len(queries), len(writes))
}

// TestRestoreOnDisable verifies that after Enable+Disable, the previous proxy
// values are restored (when the backup showed previous proxy was enabled).
func TestRestoreOnDisable(t *testing.T) {
	prevServer := "old.proxy:8080"
	prevOverride := "old-bypass"
	prevEnabled := "0x1"

	runner := &recordingRunner{
		queryOuts: map[string]string{
			"ProxyEnable":   regQueryOutput("ProxyEnable", "REG_DWORD", prevEnabled),
			"ProxyServer":   regQueryOutput("ProxyServer", "REG_SZ", prevServer),
			"ProxyOverride": regQueryOutput("ProxyOverride", "REG_SZ", prevOverride),
		},
	}
	proxy := &WindowsSystemProxy{runner: runner}

	// Step 1: Enable
	err := proxy.Enable(context.Background(), "127.0.0.1:7890", "<local>")
	if err != nil {
		t.Fatalf("Enable() error = %v", err)
	}
	// Clear commands after enable so Disable starts fresh
	runner.commands = nil

	// Step 2: Disable — should restore previous values since prevEnabled=true
	err = proxy.Disable(context.Background())
	if err != nil {
		t.Fatalf("Disable() error = %v", err)
	}

	if len(runner.commands) != 3 {
		t.Fatalf("Disable commands = %d, want 3 (restore Server, Override, Enable)", len(runner.commands))
	}

	// Verify each command is reg add with restored values
	expectedRestores := map[string]string{
		"ProxyServer":   prevServer,
		"ProxyOverride": prevOverride,
	}
	for _, cmd := range runner.commands {
		if cmd.Name != "reg" || cmd.Args[0] != "add" {
			t.Fatalf("unexpected command: %v", cmd)
		}
		valueName := cmd.Args[3]

		if expected, ok := expectedRestores[valueName]; ok {
			if got := cmd.Args[7]; got != expected {
				t.Fatalf("valueName %s = %q, want %q (previous value)", valueName, got, expected)
			}
		} else if valueName == "ProxyEnable" {
			if got := cmd.Args[7]; got != "1" {
				t.Fatalf("ProxyEnable restore value = %q, want 1", got)
			}
		}
	}

	t.Log("Restore verified: previous server, override, and ProxyEnable=1 were restored on disable")
}

// TestRestoreOnDisableWasNotEnabled verifies that Disable only sets
// ProxyEnable=0 when the backup shows proxy was previously disabled.
func TestRestoreOnDisableWasNotEnabled(t *testing.T) {
	runner := &recordingRunner{
		queryOuts: map[string]string{
			"ProxyEnable":   regQueryOutput("ProxyEnable", "REG_DWORD", "0x0"),
			"ProxyServer":   regQueryOutput("ProxyServer", "REG_SZ", ""),
			"ProxyOverride": regQueryOutput("ProxyOverride", "REG_SZ", ""),
		},
	}
	proxy := &WindowsSystemProxy{runner: runner}

	// Enable first (does backup showing prevEnabled=false)
	err := proxy.Enable(context.Background(), "127.0.0.1:7890", "<local>")
	if err != nil {
		t.Fatalf("Enable() error = %v", err)
	}
	runner.commands = nil

	// Now disable — should NOT restore server/override, just set ProxyEnable=0
	err = proxy.Disable(context.Background())
	if err != nil {
		t.Fatalf("Disable() error = %v", err)
	}

	if len(runner.commands) != 1 {
		t.Fatalf("Disable commands = %d, want 1 (just ProxyEnable=0)", len(runner.commands))
	}
	if got := runner.commands[0].Args[7]; got != "0" {
		t.Fatalf("ProxyEnable value = %q, want 0", got)
	}

	t.Log("Disable with prevEnabled=false only set ProxyEnable=0, no restore of server/override")
}

// TestEnableWithEmptyServerReturnsError verifies that Enable rejects an empty
// server address.
func TestEnableWithEmptyServerReturnsError(t *testing.T) {
	proxy := &WindowsSystemProxy{runner: &recordingRunner{}}

	err := proxy.Enable(context.Background(), "", "")
	if err == nil {
		t.Fatal("Enable() with empty server should return error")
	}
	if !strings.Contains(err.Error(), "server") {
		t.Fatalf("error should mention server, got: %v", err)
	}

	t.Logf("Enable with empty server correctly returned error: %v", err)
}
