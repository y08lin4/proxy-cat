package system

import "context"

// SystemProxy abstracts platform-specific system proxy configuration.
type SystemProxy interface {
	Enable(ctx context.Context, server string, bypass string) error
	Disable(ctx context.Context) error
}
