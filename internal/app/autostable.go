package app

import (
	"context"
	"errors"
	"fmt"
	"time"
)

var ErrAutoStableUnavailable = errors.New("auto-stable service is not available")

const autoStableTickCooldown = 5 * time.Second

// AutoStableRunner is the narrow boundary for the Phase 2 autostable package.
// The app layer owns Wails-facing state; a future runner owns health checks and
// selection policy.
type AutoStableRunner interface {
	Status(ctx context.Context) (AutoStableStatus, error)
	SetEnabled(ctx context.Context, enabled bool) error
	Tick(ctx context.Context) (AutoStableActionResult, error)
	Select(ctx context.Context, groupName string) (AutoStableActionResult, error)
}

// GetAutoStableStatus returns the current Phase 2 health/control snapshot.
func (a *App) GetAutoStableStatus() AutoStableStatus {
	ctx := a.context()
	a.mu.RLock()
	runner := a.autoStable
	a.mu.RUnlock()

	if runner != nil {
		status, err := runner.Status(ctx)
		if err != nil {
			_ = a.fail(fmt.Errorf("auto-stable status: %w", err))
			return a.autoStableSnapshot()
		}
		a.mu.Lock()
		a.mergeAutoStableStatusLocked(status)
		status = a.autoStableSnapshotLocked()
		a.mu.Unlock()
		return status
	}

	return a.autoStableSnapshot()
}

// SetAutoStableEnabled toggles manual/interval auto-stable behavior.
func (a *App) SetAutoStableEnabled(enabled bool) error {
	ctx := a.context()
	a.mu.RLock()
	runner := a.autoStable
	a.mu.RUnlock()

	if runner != nil {
		if err := runner.SetEnabled(ctx, enabled); err != nil {
			return a.fail(fmt.Errorf("set auto-stable enabled: %w", err))
		}
	}

	a.mu.Lock()
	a.status.AutoStableEnabled = enabled
	a.autoStableStatus.Enabled = enabled
	a.autoStableStatus.Available = runner != nil
	a.autoStableStatus.LastError = ""
	if enabled {
		a.appendLogLocked("info", "Auto-stable enabled")
	} else {
		a.appendLogLocked("info", "Auto-stable disabled")
	}
	a.mu.Unlock()
	return nil
}

// RunAutoStableTick runs one controllable auto-stable tick.
func (a *App) RunAutoStableTick() (AutoStableActionResult, error) {
	ctx := a.context()
	a.mu.RLock()
	runner := a.autoStable
	lastTickAt := a.autoStableTickAt
	a.mu.RUnlock()

	if runner == nil {
		result := a.unavailableAutoStableResult("tick", "")
		a.mu.Lock()
		a.autoStableStatus.LastTickAt = result.CompletedAt
		a.autoStableStatus.LastAction = result.Action
		a.autoStableStatus.LastError = result.Message
		a.appendLogLocked("warn", result.Message)
		a.mu.Unlock()
		return result, ErrAutoStableUnavailable
	}

	if time.Since(lastTickAt) < autoStableTickCooldown {
		result := AutoStableActionResult{
			Action:      "tick",
			Changed:     false,
			Message:     "Auto-stable tick skipped by cooldown",
			CompletedAt: time.Now(),
			Health:      a.autoStableSnapshot().Health,
		}
		a.mu.Lock()
		a.autoStableStatus.LastAction = result.Action
		a.autoStableStatus.LastError = result.Message
		a.appendLogLocked("info", result.Message)
		a.mu.Unlock()
		return result, nil
	}

	result, err := runner.Tick(ctx)
	if err != nil {
		return result, a.fail(fmt.Errorf("auto-stable tick: %w", err))
	}
	if result.CompletedAt.IsZero() {
		result.CompletedAt = time.Now()
	}
	a.recordAutoStableAction(result)
	a.mu.Lock()
	a.autoStableTickAt = result.CompletedAt
	a.mu.Unlock()
	return result, nil
}

// SelectAutoStableProxy asks Phase 2 to choose one proxy for a group now.
func (a *App) SelectAutoStableProxy(groupName string) (AutoStableActionResult, error) {
	if groupName == "" {
		return AutoStableActionResult{}, errors.New("proxy group name is required")
	}

	ctx := a.context()
	a.mu.RLock()
	runner := a.autoStable
	a.mu.RUnlock()

	if runner == nil {
		result := a.unavailableAutoStableResult("select", groupName)
		a.mu.Lock()
		a.autoStableStatus.LastAction = result.Action
		a.autoStableStatus.LastError = result.Message
		a.appendLogLocked("warn", result.Message)
		a.mu.Unlock()
		return result, ErrAutoStableUnavailable
	}

	result, err := runner.Select(ctx, groupName)
	if err != nil {
		return result, a.fail(fmt.Errorf("auto-stable select: %w", err))
	}
	a.recordAutoStableAction(result)
	return result, nil
}

func (a *App) autoStableSnapshot() AutoStableStatus {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.autoStableSnapshotLocked()
}

func (a *App) autoStableSnapshotLocked() AutoStableStatus {
	status := a.autoStableStatus
	status.Enabled = a.status.AutoStableEnabled
	if len(status.Health) == 0 {
		status.Health = a.buildAutoStableHealthLocked()
	}
	return status
}

func (a *App) mergeAutoStableStatusLocked(status AutoStableStatus) {
	lastTickAt := a.autoStableStatus.LastTickAt
	lastAction := a.autoStableStatus.LastAction
	lastSelected := a.autoStableStatus.LastSelected
	lastError := a.autoStableStatus.LastError
	a.autoStableStatus = status
	if a.autoStableStatus.LastTickAt.IsZero() {
		a.autoStableStatus.LastTickAt = lastTickAt
	}
	if a.autoStableStatus.LastAction == "" {
		a.autoStableStatus.LastAction = lastAction
	}
	if a.autoStableStatus.LastSelected == "" {
		a.autoStableStatus.LastSelected = lastSelected
	}
	if a.autoStableStatus.LastError == "" {
		a.autoStableStatus.LastError = lastError
	}
	a.status.AutoStableEnabled = status.Enabled
}

func (a *App) recordAutoStableAction(result AutoStableActionResult) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if result.CompletedAt.IsZero() {
		result.CompletedAt = time.Now()
	}
	a.autoStableStatus.LastTickAt = result.CompletedAt
	a.autoStableStatus.LastAction = result.Action
	a.autoStableStatus.LastSelected = result.Selected
	a.autoStableStatus.LastError = ""
	if len(result.Health) > 0 {
		a.autoStableStatus.Health = append([]AutoStableGroupHealth(nil), result.Health...)
	}
	if result.GroupName != "" && result.Selected != "" {
		for i := range a.active.ProxyGroups {
			if a.active.ProxyGroups[i].Name == result.GroupName {
				a.active.ProxyGroups[i].SelectedProxy = result.Selected
				break
			}
		}
	}
	message := result.Message
	if message == "" {
		message = fmt.Sprintf("Auto-stable %s completed", result.Action)
	}
	a.appendLogLocked("info", message)
}

func (a *App) unavailableAutoStableResult(action string, groupName string) AutoStableActionResult {
	message := "Auto-stable service is not available yet"
	return AutoStableActionResult{
		Action:      action,
		GroupName:   groupName,
		Changed:     false,
		Message:     message,
		CompletedAt: time.Now(),
		Health:      a.autoStableSnapshot().Health,
	}
}

func (a *App) buildAutoStableHealthLocked() []AutoStableGroupHealth {
	nodeByName := make(map[string]string, len(a.active.Proxies))
	for _, node := range a.active.Proxies {
		nodeByName[node.Name] = node.Type
	}

	views := make([]AutoStableGroupHealth, 0, len(a.active.ProxyGroups))
	for _, group := range a.active.ProxyGroups {
		view := AutoStableGroupHealth{
			Name:     group.Name,
			Type:     group.Type,
			Selected: group.SelectedProxy,
			Proxies:  make([]AutoStableNodeHealth, 0, len(group.Proxies)),
		}
		for _, name := range group.Proxies {
			proxyType := nodeByName[name]
			view.Proxies = append(view.Proxies, AutoStableNodeHealth{
				Name:  name,
				Type:  proxyType,
				Alive: proxyType != "" || name == "DIRECT" || name == "AUTO",
			})
		}
		views = append(views, view)
	}
	return views
}
