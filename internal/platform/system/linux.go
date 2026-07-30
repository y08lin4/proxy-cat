//go:build linux

package system

import (
	"context"
	"errors"
)

type LinuxSystemProxy struct{}

func NewLinux() *LinuxSystemProxy {
	return &LinuxSystemProxy{}
}

func (p *LinuxSystemProxy) Enable(ctx context.Context, server string, bypass string) error {
	// gsettings-based implementation would go here for GNOME
	return errors.New("system proxy control is not implemented for Linux; please configure your system proxy manually")
}

func (p *LinuxSystemProxy) Disable(ctx context.Context) error {
	return errors.New("system proxy control is not implemented for Linux; please configure your system proxy manually")
}
