package system

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

const internetSettingsKey = `HKCU\Software\Microsoft\Windows\CurrentVersion\Internet Settings`

type commandRunner interface {
	Run(ctx context.Context, name string, args ...string) error
	RunOutput(ctx context.Context, name string, args ...string) (string, error)
}

type execRunner struct{}

func (execRunner) Run(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%s %v: %w: %s", name, args, err, string(output))
	}
	return nil
}

func (execRunner) RunOutput(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%s %v: %w: %s", name, args, err, strings.TrimSpace(stderr.String()))
	}
	return stdout.String(), nil
}

type WindowsSystemProxy struct {
	runner       commandRunner
	prevServer   string
	prevOverride string
	prevEnabled  bool
	backedUp     bool
}

func NewWindows() *WindowsSystemProxy {
	return &WindowsSystemProxy{runner: execRunner{}}
}

func (p *WindowsSystemProxy) Enable(ctx context.Context, server string, bypass string) error {
	if server == "" {
		return errors.New("system proxy server is required")
	}

	if !p.backedUp {
		if err := p.backup(ctx); err != nil {
			return fmt.Errorf("backup existing proxy settings: %w", err)
		}
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
	if p.backedUp {
		if p.prevEnabled {
			if err := p.regAdd(ctx, "ProxyServer", "REG_SZ", p.prevServer); err != nil {
				return err
			}
			if p.prevOverride != "" {
				if err := p.regAdd(ctx, "ProxyOverride", "REG_SZ", p.prevOverride); err != nil {
					return err
				}
			}
			return p.regAdd(ctx, "ProxyEnable", "REG_DWORD", "1")
		}
		return p.regAdd(ctx, "ProxyEnable", "REG_DWORD", "0")
	}
	return p.regAdd(ctx, "ProxyEnable", "REG_DWORD", "0")
}

func (p *WindowsSystemProxy) backup(ctx context.Context) error {
	enabledStr, err := p.readRegValue(ctx, "ProxyEnable")
	if err != nil {
		p.prevEnabled = false
	} else {
		p.prevEnabled = strings.TrimSpace(enabledStr) == "0x1"
	}

	server, err := p.readRegValue(ctx, "ProxyServer")
	if err == nil {
		p.prevServer = server
	}

	override, err := p.readRegValue(ctx, "ProxyOverride")
	if err == nil {
		p.prevOverride = override
	}

	p.backedUp = true
	return nil
}

func (p *WindowsSystemProxy) readRegValue(ctx context.Context, valueName string) (string, error) {
	output, err := p.runner.RunOutput(ctx, "reg", "query", internetSettingsKey, "/v", valueName)
	if err != nil {
		return "", fmt.Errorf("reg query %s: %w", valueName, err)
	}
	return parseRegQueryOutput(output, valueName)
}

func parseRegQueryOutput(output string, valueName string) (string, error) {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, valueName) {
			parts := strings.Fields(trimmed)
			if len(parts) >= 3 {
				return strings.Join(parts[2:], " "), nil
			}
			return "", fmt.Errorf("unexpected reg query output format: %q", trimmed)
		}
	}
	return "", fmt.Errorf("value %q not found in reg query output", valueName)
}

func (p *WindowsSystemProxy) regAdd(ctx context.Context, valueName string, valueType string, value string) error {
	if p.runner == nil {
		p.runner = execRunner{}
	}
	return p.runner.Run(ctx, "reg", "add", internetSettingsKey, "/v", valueName, "/t", valueType, "/d", value, "/f")
}
