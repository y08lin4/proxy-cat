package mihomo

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"sync"
	"time"
)

var ErrMihomoRunning = errors.New("mihomo is already running")

type LaunchConfig struct {
	BinaryPath string
	ConfigPath string
	HomeDir    string
}

type Status struct {
	Running    bool
	PID        int
	BinaryPath string
	ConfigPath string
	HomeDir    string
	LastExit   ExitStatus
}

type ExitStatus struct {
	Exited   bool
	PID      int
	ExitCode int
	Error    string
	Expected bool
	ExitedAt time.Time
}

type Launcher struct {
	mu       sync.Mutex
	command  CommandContextFunc
	cmd      *exec.Cmd
	done     chan mihomoProcessExit
	lastConf LaunchConfig
	lastExit ExitStatus
}

type CommandContextFunc func(context.Context, string, ...string) *exec.Cmd

type mihomoProcessExit struct {
	pid int
	err error
	at  time.Time
}

func NewLauncher() *Launcher {
	return &Launcher{command: exec.CommandContext}
}

func NewLauncherWithCommand(command CommandContextFunc) *Launcher {
	if command == nil {
		command = exec.CommandContext
	}
	return &Launcher{command: command}
}

func (l *Launcher) Start(ctx context.Context, conf LaunchConfig) error {
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

	done := make(chan mihomoProcessExit, 1)
	pid := cmd.Process.Pid
	go func() {
		done <- mihomoProcessExit{pid: pid, err: cmd.Wait(), at: time.Now()}
	}()

	l.cmd = cmd
	l.done = done
	l.lastConf = conf
	l.lastExit = ExitStatus{}
	return nil
}

func (l *Launcher) Stop(ctx context.Context) error {
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
	case exit := <-done:
		l.clearIfCurrent(cmd, exit, true)
		if exit.err != nil && !isExpectedProcessExit(exit.err) {
			return fmt.Errorf("wait mihomo exit: %w", exit.err)
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (l *Launcher) Restart(ctx context.Context, conf LaunchConfig) error {
	if err := l.Stop(ctx); err != nil {
		return err
	}
	return l.Start(ctx, conf)
}

func (l *Launcher) Status() Status {
	l.mu.Lock()
	defer l.mu.Unlock()

	status := Status{
		Running:    l.isRunningLocked(),
		BinaryPath: l.lastConf.BinaryPath,
		ConfigPath: l.lastConf.ConfigPath,
		HomeDir:    l.lastConf.HomeDir,
		LastExit:   l.lastExit,
	}
	if status.Running && l.cmd != nil && l.cmd.Process != nil {
		status.PID = l.cmd.Process.Pid
	}
	return status
}

func (l *Launcher) NeedsRecovery() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.isRunningLocked()
	return l.needsRecoveryLocked()
}

func (l *Launcher) RecoverIfNeeded(ctx context.Context) (bool, error) {
	l.mu.Lock()
	if l.isRunningLocked() || !l.needsRecoveryLocked() {
		l.mu.Unlock()
		return false, nil
	}
	conf := l.lastConf
	l.mu.Unlock()

	if err := l.Start(ctx, conf); err != nil {
		return false, err
	}
	return true, nil
}

func (l *Launcher) clearIfCurrent(cmd *exec.Cmd, exit mihomoProcessExit, expected bool) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.cmd == cmd {
		l.recordExitLocked(exit, expected)
		l.cmd = nil
		l.done = nil
	}
}

func (l *Launcher) isRunningLocked() bool {
	if l.cmd == nil || l.done == nil {
		return false
	}

	select {
	case exit := <-l.done:
		l.recordExitLocked(exit, false)
		l.cmd = nil
		l.done = nil
		return false
	default:
		return true
	}
}

func (l *Launcher) needsRecoveryLocked() bool {
	if l.cmd != nil && l.done != nil {
		return false
	}
	return l.lastExit.Exited && !l.lastExit.Expected && l.lastConf.validate() == nil
}

func (l *Launcher) recordExitLocked(exit mihomoProcessExit, expected bool) {
	status := ExitStatus{
		Exited:   true,
		PID:      exit.pid,
		ExitCode: exitCode(exit.err),
		Expected: expected,
		ExitedAt: exit.at,
	}
	if exit.err != nil {
		status.Error = exit.err.Error()
	}
	l.lastExit = status
}

func (conf LaunchConfig) validate() error {
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

func mihomoArgs(conf LaunchConfig) []string {
	return []string{"-f", conf.ConfigPath, "-d", conf.HomeDir}
}

func isExpectedProcessExit(err error) bool {
	var exitErr *exec.ExitError
	return errors.As(err, &exitErr)
}

func exitCode(err error) int {
	if err == nil {
		return 0
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode()
	}
	return -1
}
