package app

import "context"

func (a *App) Startup(ctx context.Context) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.ctx = ctx
}

func (a *App) DomReady(ctx context.Context) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.ctx = ctx
}

func (a *App) BeforeClose(ctx context.Context) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.ctx = ctx
	return false
}

func (a *App) Shutdown(ctx context.Context) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.ctx = ctx
}
