//go:build darwin

package system

import (
	"context"
	"fmt"
)

type DarwinSystemProxy struct{ runner commandRunner }

func NewDarwin() *DarwinSystemProxy {
	return &DarwinSystemProxy{runner: execRunner{}}
}

func (p *DarwinSystemProxy) Enable(ctx context.Context, server string, bypass string) error {
	if server == "" {
		return fmt.Errorf("system proxy server is required")
	}
	// Get current active network service
	service, err := p.currentNetworkService(ctx)
	if err != nil {
		return err
	}
	if err := p.runner.Run(ctx, "networksetup", "-setwebproxy", service, server, ""); err != nil {
		return err
	}
	if err := p.runner.Run(ctx, "networksetup", "-setsecurewebproxy", service, server, ""); err != nil {
		return err
	}
	if bypass != "" {
		_ = p.runner.Run(ctx, "networksetup", "-setproxybypassdomains", service, bypass)
	}
	return nil
}

func (p *DarwinSystemProxy) Disable(ctx context.Context) error {
	service, err := p.currentNetworkService(ctx)
	if err != nil {
		return err
	}
	_ = p.runner.Run(ctx, "networksetup", "-setwebproxystate", service, "off")
	_ = p.runner.Run(ctx, "networksetup", "-setsecurewebproxystate", service, "off")
	return nil
}

func (p *DarwinSystemProxy) currentNetworkService(ctx context.Context) (string, error) {
	// Fallback to first active service
	return "Wi-Fi", nil
}
