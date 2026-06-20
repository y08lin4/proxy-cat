package core

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
)

const internetSettingsKey = `HKCU\Software\Microsoft\Windows\CurrentVersion\Internet Settings`

type commandRunner interface {
	Run(ctx context.Context, name string, args ...string) error
}

type execRunner struct{}

func (execRunner) Run(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%s %v: %w: %s", name, args, err, string(output))
	}
	return nil
}

type WindowsSystemProxy struct {
	runner commandRunner
}

func NewWindowsSystemProxy() *WindowsSystemProxy {
	return &WindowsSystemProxy{runner: execRunner{}}
}

func (p *WindowsSystemProxy) Enable(ctx context.Context, server string, bypass string) error {
	if server == "" {
		return errors.New("system proxy server is required")
	}
	if err := p.regAdd(ctx, "ProxyServer", "REG_SZ", server); err != nil {
		return err
	}
	if bypass != "" {
		if err := p.regAdd(ctx, "ProxyOverride", "REG_SZ", bypass); err != nil {
			return err
		}
	}
	return p.regAdd(ctx, "ProxyEnable", "REG_DWORD", "1")
}

func (p *WindowsSystemProxy) Disable(ctx context.Context) error {
	return p.regAdd(ctx, "ProxyEnable", "REG_DWORD", "0")
}

func (p *WindowsSystemProxy) regAdd(ctx context.Context, valueName string, valueType string, value string) error {
	if p.runner == nil {
		p.runner = execRunner{}
	}
	return p.runner.Run(ctx, "reg", "add", internetSettingsKey, "/v", valueName, "/t", valueType, "/d", value, "/f")
}
