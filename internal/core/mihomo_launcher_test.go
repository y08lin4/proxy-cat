package core

import (
	"context"
	"os"
	"os/exec"
	"reflect"
	"testing"
	"time"
)

func TestMihomoArgs(t *testing.T) {
	conf := MihomoLaunchConfig{
		BinaryPath: `C:\Proxy-Cat\mihomo.exe`,
		ConfigPath: `C:\Proxy-Cat\profiles\active\config.yaml`,
		HomeDir:    `C:\Proxy-Cat\mihomo`,
	}

	got := mihomoArgs(conf)
	want := []string{"-f", conf.ConfigPath, "-d", conf.HomeDir}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("mihomoArgs() = %#v, want %#v", got, want)
	}
}

func TestMihomoLaunchConfigValidate(t *testing.T) {
	tests := []struct {
		name string
		conf MihomoLaunchConfig
	}{
		{name: "missing binary", conf: MihomoLaunchConfig{ConfigPath: "config.yaml", HomeDir: "mihomo"}},
		{name: "missing config", conf: MihomoLaunchConfig{BinaryPath: "mihomo.exe", HomeDir: "mihomo"}},
		{name: "missing home", conf: MihomoLaunchConfig{BinaryPath: "mihomo.exe", ConfigPath: "config.yaml"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.conf.validate(); err == nil {
				t.Fatal("validate() error = nil, want error")
			}
		})
	}
}

func TestMihomoLauncherRecordsUnexpectedExitAndRecovers(t *testing.T) {
	launcher := NewMihomoLauncher()
	starts := 0
	launcher.command = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		starts++
		if starts == 1 {
			cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestMihomoLauncherHelperProcess$")
			cmd.Env = append(os.Environ(), "PROXY_CAT_CORE_HELPER=unexpected")
			return cmd
		}
		cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestMihomoLauncherHelperProcess$")
		cmd.Env = append(os.Environ(), "PROXY_CAT_CORE_HELPER=sleep")
		return cmd
	}
	conf := MihomoLaunchConfig{BinaryPath: "mihomo.exe", ConfigPath: "config.yaml", HomeDir: "mihomo"}

	if err := launcher.Start(context.Background(), conf); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	waitFor(t, time.Second, func() bool {
		return !launcher.Status().Running
	})

	status := launcher.Status()
	if status.LastExit.Expected {
		t.Fatalf("LastExit.Expected = true, want false")
	}
	if status.LastExit.ExitCode != 7 {
		t.Fatalf("LastExit.ExitCode = %d, want 7", status.LastExit.ExitCode)
	}
	if !launcher.NeedsRecovery() {
		t.Fatal("NeedsRecovery() = false, want true")
	}

	recovered, err := launcher.RecoverIfNeeded(context.Background())
	if err != nil {
		t.Fatalf("RecoverIfNeeded() error = %v", err)
	}
	if !recovered {
		t.Fatal("RecoverIfNeeded() recovered = false, want true")
	}
	if starts != 2 {
		t.Fatalf("starts = %d, want 2", starts)
	}
	if !launcher.Status().Running {
		t.Fatal("Status().Running = false, want true")
	}

	if err := launcher.Stop(context.Background()); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	if launcher.NeedsRecovery() {
		t.Fatal("NeedsRecovery() = true after Stop, want false")
	}
}

func TestMihomoLauncherHelperProcess(t *testing.T) {
	switch os.Getenv("PROXY_CAT_CORE_HELPER") {
	case "unexpected":
		os.Exit(7)
	case "sleep":
		time.Sleep(30 * time.Second)
	default:
		t.Skip("helper process only")
	}
}

func waitFor(t *testing.T, timeout time.Duration, ok func() bool) {
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
