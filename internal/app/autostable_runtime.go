package app

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/y08lin4/proxy-cat/internal/autostable"
	"github.com/y08lin4/proxy-cat/internal/core"
	"github.com/y08lin4/proxy-cat/internal/profile"
)

const autoStableDelayURL = "https://www.gstatic.com/generate_204"

type mihomoAutoStableRunner struct {
	manager    *autostable.Manager
	profile    profile.Profile
	httpClient *http.Client
	controller string
	secret     string
	enabled    bool
}

func newMihomoAutoStableRunner(p profile.Profile, httpClient *http.Client) (*mihomoAutoStableRunner, error) {
	manager, err := autostable.NewManager(autostable.Config{})
	if err != nil {
		return nil, err
	}
	runner := &mihomoAutoStableRunner{
		manager:    manager,
		profile:    p,
		httpClient: httpClient,
		controller: p.Settings.ExternalController,
		secret:     p.Settings.Secret,
	}
	for _, node := range p.Proxies {
		if err := manager.Register(node.Name); err != nil {
			return nil, err
		}
	}
	return runner, nil
}

func (r *mihomoAutoStableRunner) Status(ctx context.Context) (AutoStableStatus, error) {
	return AutoStableStatus{
		Enabled:   r.enabled,
		Available: true,
		Running:   r.enabled,
		Health:    r.health(time.Now()),
	}, nil
}

func (r *mihomoAutoStableRunner) SetEnabled(ctx context.Context, enabled bool) error {
	r.enabled = enabled
	return nil
}

func (r *mihomoAutoStableRunner) Tick(ctx context.Context) (AutoStableActionResult, error) {
	return r.Select(ctx, "AUTO-STABLE")
}

func (r *mihomoAutoStableRunner) Select(ctx context.Context, groupName string) (AutoStableActionResult, error) {
	if groupName == "" {
		groupName = "AUTO-STABLE"
	}
	now := time.Now()
	client, err := core.NewMihomoClient(r.controller, r.secret, r.httpClient)
	if err != nil {
		return AutoStableActionResult{}, err
	}

	for _, node := range r.nodesForGroup(groupName) {
		delay, err := client.TestProxyDelay(ctx, node, autoStableDelayURL, 5000)
		if err != nil {
			_ = r.manager.Record(autostable.Sample{
				NodeID:    node,
				Success:   false,
				CheckedAt: now,
			})
			continue
		}
		_ = r.manager.Record(autostable.Sample{
			NodeID:    node,
			Latency:   time.Duration(delay.Delay) * time.Millisecond,
			Success:   true,
			CheckedAt: now,
		})
	}

	decision := r.manager.Select(now)
	result := AutoStableActionResult{
		Action:      "select",
		GroupName:   groupName,
		Selected:    decision.SelectedID,
		Changed:     decision.Switched,
		Message:     fmt.Sprintf("Auto-stable kept %s", decision.SelectedID),
		CompletedAt: now,
		Health:      r.health(now),
	}
	if decision.Switched {
		result.Message = fmt.Sprintf("Auto-stable selected %s", decision.SelectedID)
		if decision.SelectedID != "" {
			if err := client.SelectProxy(ctx, groupName, decision.SelectedID); err != nil {
				return result, err
			}
		}
	}
	return result, nil
}

func (r *mihomoAutoStableRunner) nodesForGroup(groupName string) []string {
	nodeSet := make(map[string]struct{}, len(r.profile.Proxies))
	for _, node := range r.profile.Proxies {
		nodeSet[node.Name] = struct{}{}
	}
	for _, group := range r.profile.ProxyGroups {
		if group.Name != groupName {
			continue
		}
		var nodes []string
		for _, name := range group.Proxies {
			if _, ok := nodeSet[name]; ok {
				nodes = append(nodes, name)
			}
		}
		return nodes
	}
	return nil
}

func (r *mihomoAutoStableRunner) health(now time.Time) []AutoStableGroupHealth {
	snapshots := make(map[string]autostable.NodeSnapshot)
	for _, snapshot := range r.manager.Snapshots(now) {
		snapshots[snapshot.NodeID] = snapshot
	}
	nodeByName := make(map[string]profile.ProxyNode, len(r.profile.Proxies))
	for _, node := range r.profile.Proxies {
		nodeByName[node.Name] = node
	}

	var groups []AutoStableGroupHealth
	for _, group := range r.profile.ProxyGroups {
		if group.Type != "auto-stable" {
			continue
		}
		view := AutoStableGroupHealth{
			Name:     group.Name,
			Type:     group.Type,
			Selected: group.SelectedProxy,
			Proxies:  make([]AutoStableNodeHealth, 0, len(group.Proxies)),
		}
		for _, name := range group.Proxies {
			node, isNode := nodeByName[name]
			if !isNode {
				continue
			}
			snapshot := snapshots[name]
			view.Proxies = append(view.Proxies, AutoStableNodeHealth{
				Name:          name,
				Type:          node.Type,
				LatencyMS:     int(snapshot.LatencyMS),
				Alive:         snapshot.Available,
				Score:         snapshot.Score,
				SuccessCount:  snapshot.Successes,
				FailureCount:  snapshot.Failures,
				TotalChecks:   snapshot.Samples,
				FailureRate:   snapshot.FailureRate,
				CooldownUntil: snapshot.CooldownUntil,
			})
		}
		groups = append(groups, view)
	}
	return groups
}
