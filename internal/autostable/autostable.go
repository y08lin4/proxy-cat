package autostable

import (
	"errors"
	"math"
	"sort"
	"time"
)

const (
	defaultSampleLimit             = 10
	defaultSwitchThreshold         = 100
	defaultMinHoldTime             = time.Minute
	defaultCooldownAfterFailure    = time.Minute
	defaultConsecutiveFailureLimit = 2
	failureRatePenaltyMultiplier   = 1000
)

var (
	ErrEmptyNodeID                 = errors.New("node id cannot be empty")
	ErrNegativeLatency             = errors.New("latency must be >= 0")
	ErrInvalidSampleLimit          = errors.New("sample limit must be >= 0")
	ErrNegativeDuration            = errors.New("duration must be >= 0")
	ErrNegativeSwitchThreshold     = errors.New("switch threshold must be >= 0")
	ErrInvalidConsecutiveFailLimit = errors.New("consecutive failure limit must be >= 0")
)

// Config controls the pure auto-stable decision policy. A zero-value Config is
// valid and uses the Phase 2 frozen defaults.
type Config struct {
	SampleLimit          int
	MinHoldTime          time.Duration
	SwitchThreshold      float64
	CooldownAfterFailure time.Duration
	ConsecutiveFailLimit int
}

// DefaultConfig returns the defaults used when Config leaves optional fields as
// zero.
func DefaultConfig() Config {
	return Config{
		SampleLimit:          defaultSampleLimit,
		MinHoldTime:          defaultMinHoldTime,
		SwitchThreshold:      defaultSwitchThreshold,
		CooldownAfterFailure: defaultCooldownAfterFailure,
		ConsecutiveFailLimit: defaultConsecutiveFailureLimit,
	}
}

// Sample is one health check result for a proxy node.
type Sample struct {
	NodeID    string
	Latency   time.Duration
	Success   bool
	CheckedAt time.Time
}

// NodeSnapshot is the current cached health view for one node.
type NodeSnapshot struct {
	NodeID              string
	Samples             int
	Successes           int
	Failures            int
	ConsecutiveFailures int
	LatencyMS           float64
	FailureRate         float64
	Score               float64
	CooldownUntil       time.Time
	Available           bool
}

// Reason explains why a decision selected or retained a node.
type Reason string

const (
	ReasonInitial                 Reason = "initial"
	ReasonCurrentHeld             Reason = "current_held"
	ReasonMinHoldTime             Reason = "min_hold_time"
	ReasonSwitchThreshold         Reason = "switch_threshold"
	ReasonBetterCandidate         Reason = "better_candidate"
	ReasonCurrentUnavailable      Reason = "current_unavailable"
	ReasonConsecutiveFailureLimit Reason = "consecutive_failure_limit"
	ReasonNoAvailableNode         Reason = "no_available_node"
)

// Decision is the result of one Select call.
type Decision struct {
	PreviousID string
	SelectedID string
	Switched   bool
	Reason     Reason
	Selected   NodeSnapshot
}

// Manager owns the health sample cache and auto-stable selection state.
type Manager struct {
	cfg          Config
	nodes        map[string]*nodeState
	currentID    string
	lastSwitchAt time.Time
}

type nodeState struct {
	samples             []Sample
	consecutiveFailures int
	cooldownUntil       time.Time
}

// NewManager creates an auto-stable manager with validated configuration.
func NewManager(cfg Config) (*Manager, error) {
	normalized, err := normalizeConfig(cfg)
	if err != nil {
		return nil, err
	}
	return &Manager{
		cfg:   normalized,
		nodes: make(map[string]*nodeState),
	}, nil
}

// Config returns the normalized manager configuration.
func (m *Manager) Config() Config {
	return m.cfg
}

// Register ensures a node exists in the cache.
func (m *Manager) Register(nodeID string) error {
	if nodeID == "" {
		return ErrEmptyNodeID
	}
	m.node(nodeID)
	return nil
}

// SetCurrent pins the current node until a later Select call changes it.
func (m *Manager) SetCurrent(nodeID string, switchedAt time.Time) error {
	if nodeID == "" {
		return ErrEmptyNodeID
	}
	m.node(nodeID)
	m.currentID = nodeID
	m.lastSwitchAt = switchedAt
	return nil
}

// Current returns the currently selected node id.
func (m *Manager) Current() string {
	return m.currentID
}

// Record stores one health sample and updates failure cooldown state.
func (m *Manager) Record(sample Sample) error {
	if sample.NodeID == "" {
		return ErrEmptyNodeID
	}
	if sample.Latency < 0 {
		return ErrNegativeLatency
	}

	node := m.node(sample.NodeID)
	node.samples = append(node.samples, sample)
	if len(node.samples) > m.cfg.SampleLimit {
		node.samples = node.samples[len(node.samples)-m.cfg.SampleLimit:]
	}

	if sample.Success {
		node.consecutiveFailures = 0
		return nil
	}

	node.consecutiveFailures++
	if m.cfg.CooldownAfterFailure > 0 {
		node.cooldownUntil = sample.CheckedAt.Add(m.cfg.CooldownAfterFailure)
	}
	return nil
}

// Snapshot returns one node's current cached health view.
func (m *Manager) Snapshot(nodeID string, now time.Time) (NodeSnapshot, bool) {
	node, ok := m.nodes[nodeID]
	if !ok {
		return NodeSnapshot{}, false
	}
	return snapshot(nodeID, node, now), true
}

// Snapshots returns all cached node snapshots ordered by node id.
func (m *Manager) Snapshots(now time.Time) []NodeSnapshot {
	ids := make([]string, 0, len(m.nodes))
	for id := range m.nodes {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	snapshots := make([]NodeSnapshot, 0, len(ids))
	for _, id := range ids {
		snapshots = append(snapshots, snapshot(id, m.nodes[id], now))
	}
	return snapshots
}

// Select updates and returns the current best node according to the auto-stable
// policy. It performs no scheduling or I/O; callers decide when to invoke it.
func (m *Manager) Select(now time.Time) Decision {
	previousID := m.currentID
	best, hasBest := m.bestAvailable(now, "")
	if !hasBest {
		return Decision{
			PreviousID: previousID,
			SelectedID: previousID,
			Reason:     ReasonNoAvailableNode,
		}
	}

	if previousID == "" {
		m.switchTo(best.NodeID, now)
		return Decision{
			PreviousID: previousID,
			SelectedID: best.NodeID,
			Switched:   true,
			Reason:     ReasonInitial,
			Selected:   best,
		}
	}

	current, currentOK := m.Snapshot(previousID, now)
	if !currentOK || !current.Available {
		m.switchTo(best.NodeID, now)
		return Decision{
			PreviousID: previousID,
			SelectedID: best.NodeID,
			Switched:   best.NodeID != previousID,
			Reason:     ReasonCurrentUnavailable,
			Selected:   best,
		}
	}

	if m.fallbackDue(current) {
		if fallback, ok := m.bestAvailable(now, previousID); ok {
			m.switchTo(fallback.NodeID, now)
			return Decision{
				PreviousID: previousID,
				SelectedID: fallback.NodeID,
				Switched:   true,
				Reason:     ReasonConsecutiveFailureLimit,
				Selected:   fallback,
			}
		}
	}

	if best.NodeID == previousID {
		return Decision{
			PreviousID: previousID,
			SelectedID: previousID,
			Reason:     ReasonCurrentHeld,
			Selected:   current,
		}
	}

	if m.cfg.MinHoldTime > 0 && now.Sub(m.lastSwitchAt) < m.cfg.MinHoldTime {
		return Decision{
			PreviousID: previousID,
			SelectedID: previousID,
			Reason:     ReasonMinHoldTime,
			Selected:   current,
		}
	}

	improvement := current.Score - best.Score
	if best.Score >= current.Score || improvement < m.cfg.SwitchThreshold {
		return Decision{
			PreviousID: previousID,
			SelectedID: previousID,
			Reason:     ReasonSwitchThreshold,
			Selected:   current,
		}
	}

	m.switchTo(best.NodeID, now)
	return Decision{
		PreviousID: previousID,
		SelectedID: best.NodeID,
		Switched:   true,
		Reason:     ReasonBetterCandidate,
		Selected:   best,
	}
}

func (m *Manager) node(nodeID string) *nodeState {
	node := m.nodes[nodeID]
	if node == nil {
		node = &nodeState{}
		m.nodes[nodeID] = node
	}
	return node
}

func (m *Manager) switchTo(nodeID string, now time.Time) {
	m.currentID = nodeID
	m.lastSwitchAt = now
}

func (m *Manager) bestAvailable(now time.Time, excludeID string) (NodeSnapshot, bool) {
	var best NodeSnapshot
	found := false
	for id, node := range m.nodes {
		if id == excludeID {
			continue
		}
		candidate := snapshot(id, node, now)
		if !candidate.Available {
			continue
		}
		if !found || candidate.Score < best.Score {
			best = candidate
			found = true
		}
	}
	return best, found
}

func (m *Manager) fallbackDue(current NodeSnapshot) bool {
	return m.cfg.ConsecutiveFailLimit > 0 &&
		current.ConsecutiveFailures >= m.cfg.ConsecutiveFailLimit
}

func snapshot(nodeID string, node *nodeState, now time.Time) NodeSnapshot {
	out := NodeSnapshot{
		NodeID:              nodeID,
		Samples:             len(node.samples),
		ConsecutiveFailures: node.consecutiveFailures,
		CooldownUntil:       node.cooldownUntil,
		Score:               math.Inf(1),
	}

	var latencyTotal time.Duration
	for _, sample := range node.samples {
		if sample.Success {
			out.Successes++
			latencyTotal += sample.Latency
			continue
		}
		out.Failures++
	}

	if out.Samples > 0 {
		out.FailureRate = float64(out.Failures) / float64(out.Samples)
	}
	if out.Successes > 0 {
		out.LatencyMS = float64(latencyTotal) / float64(out.Successes) / float64(time.Millisecond)
		out.Score = out.LatencyMS + out.FailureRate*failureRatePenaltyMultiplier
	}

	cooling := !out.CooldownUntil.IsZero() && now.Before(out.CooldownUntil)
	out.Available = out.Successes > 0 && !cooling
	return out
}

func normalizeConfig(cfg Config) (Config, error) {
	if cfg.SampleLimit < 0 {
		return Config{}, ErrInvalidSampleLimit
	}
	if cfg.MinHoldTime < 0 || cfg.CooldownAfterFailure < 0 {
		return Config{}, ErrNegativeDuration
	}
	if cfg.SwitchThreshold < 0 {
		return Config{}, ErrNegativeSwitchThreshold
	}
	if cfg.ConsecutiveFailLimit < 0 {
		return Config{}, ErrInvalidConsecutiveFailLimit
	}

	defaults := DefaultConfig()
	if cfg.SampleLimit == 0 {
		cfg.SampleLimit = defaults.SampleLimit
	}
	if cfg.MinHoldTime == 0 {
		cfg.MinHoldTime = defaults.MinHoldTime
	}
	if cfg.SwitchThreshold == 0 {
		cfg.SwitchThreshold = defaults.SwitchThreshold
	}
	if cfg.CooldownAfterFailure == 0 {
		cfg.CooldownAfterFailure = defaults.CooldownAfterFailure
	}
	if cfg.ConsecutiveFailLimit == 0 {
		cfg.ConsecutiveFailLimit = defaults.ConsecutiveFailLimit
	}
	return cfg, nil
}
