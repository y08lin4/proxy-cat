package core

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"sync"
)

var ErrMihomoRunning = errors.New("mihomo is already running")

type MihomoLaunchConfig struct {
	BinaryPath string
	ConfigPath string
	HomeDir    string
}

type MihomoStatus struct {
	Running    bool
	PID        int
	BinaryPath string
	ConfigPath string
	HomeDir    string
}

type MihomoLauncher struct {
	mu       sync.Mutex
	command  commandContextFunc
	cmd      *exec.Cmd
	done     chan error
	lastConf MihomoLaunchConfig
}

type commandContextFunc func(context.Context, string, ...string) *exec.Cmd

func NewMihomoLauncher() *MihomoLauncher {
	return &MihomoLauncher{command: exec.CommandContext}
}

func (l *MihomoLauncher) Start(ctx context.Context, conf MihomoLaunchConfig) error {
	if err := conf.validate(); err != nil {
		return err
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if l.isRunningLocked() {
		return ErrMihomoRunning
	}

	cmd := l.command(ctx, conf.BinaryPath, mihomoArgs(conf)...)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start mihomo: %w", err)
	}

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	l.cmd = cmd
	l.done = done
	l.lastConf = conf
	return nil
}

func (l *MihomoLauncher) Stop(ctx context.Context) error {
	l.mu.Lock()
	cmd := l.cmd
	done := l.done
	l.mu.Unlock()

	if cmd == nil || cmd.Process == nil {
		return nil
	}

	if err := cmd.Process.Kill(); err != nil {
		return fmt.Errorf("stop mihomo: %w", err)
	}

	select {
	case err := <-done:
		l.clearIfCurrent(cmd)
		if err != nil && !isExpectedProcessExit(err) {
			return fmt.Errorf("wait mihomo exit: %w", err)
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (l *MihomoLauncher) Restart(ctx context.Context, conf MihomoLaunchConfig) error {
	if err := l.Stop(ctx); err != nil {
		return err
	}
	return l.Start(ctx, conf)
}

func (l *MihomoLauncher) Status() MihomoStatus {
	l.mu.Lock()
	defer l.mu.Unlock()

	status := MihomoStatus{
		Running:    l.isRunningLocked(),
		BinaryPath: l.lastConf.BinaryPath,
		ConfigPath: l.lastConf.ConfigPath,
		HomeDir:    l.lastConf.HomeDir,
	}
	if status.Running && l.cmd != nil && l.cmd.Process != nil {
		status.PID = l.cmd.Process.Pid
	}
	return status
}

func (l *MihomoLauncher) clearIfCurrent(cmd *exec.Cmd) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.cmd == cmd {
		l.cmd = nil
		l.done = nil
	}
}

func (l *MihomoLauncher) isRunningLocked() bool {
	if l.cmd == nil || l.done == nil {
		return false
	}

	select {
	case <-l.done:
		l.cmd = nil
		l.done = nil
		return false
	default:
		return true
	}
}

func (conf MihomoLaunchConfig) validate() error {
	if conf.BinaryPath == "" {
		return errors.New("mihomo binary path is required")
	}
	if conf.ConfigPath == "" {
		return errors.New("mihomo config path is required")
	}
	if conf.HomeDir == "" {
		return errors.New("mihomo home directory is required")
	}
	return nil
}

func mihomoArgs(conf MihomoLaunchConfig) []string {
	return []string{"-f", conf.ConfigPath, "-d", conf.HomeDir}
}

func isExpectedProcessExit(err error) bool {
	var exitErr *exec.ExitError
	return errors.As(err, &exitErr)
}
