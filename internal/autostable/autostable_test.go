package autostable

import (
	"math"
	"testing"
	"time"
)

func TestRecordMaintainsSampleCacheAndScore(t *testing.T) {
	now := time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC)
	manager := newTestManager(t, Config{SampleLimit: 3})

	records := []Sample{
		{NodeID: "node-a", Latency: 400 * time.Millisecond, Success: true, CheckedAt: now},
		{NodeID: "node-a", Latency: 100 * time.Millisecond, Success: true, CheckedAt: now.Add(time.Second)},
		{NodeID: "node-a", Latency: 0, Success: false, CheckedAt: now.Add(2 * time.Second)},
		{NodeID: "node-a", Latency: 200 * time.Millisecond, Success: true, CheckedAt: now.Add(3 * time.Second)},
	}
	for _, record := range records {
		if err := manager.Record(record); err != nil {
			t.Fatalf("Record() unexpected error: %v", err)
		}
	}

	snapshot, ok := manager.Snapshot("node-a", now.Add(2*time.Minute))
	if !ok {
		t.Fatal("Snapshot() ok = false, want true")
	}
	if snapshot.Samples != 3 {
		t.Fatalf("snapshot.Samples = %d, want 3", snapshot.Samples)
	}
	if snapshot.Successes != 2 || snapshot.Failures != 1 {
		t.Fatalf("success/failure = %d/%d, want 2/1", snapshot.Successes, snapshot.Failures)
	}
	assertFloat(t, snapshot.LatencyMS, 150)
	assertFloat(t, snapshot.FailureRate, 1.0/3.0)
	assertFloat(t, snapshot.Score, 150+1000.0/3.0)
	if !snapshot.Available {
		t.Fatal("snapshot.Available = false, want true")
	}
}

func TestFailedOnlyNodeIsNotAvailable(t *testing.T) {
	now := time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC)
	manager := newTestManager(t, Config{})

	if err := manager.Record(Sample{NodeID: "node-a", Success: false, CheckedAt: now}); err != nil {
		t.Fatalf("Record() unexpected error: %v", err)
	}

	snapshot, ok := manager.Snapshot("node-a", now)
	if !ok {
		t.Fatal("Snapshot() ok = false, want true")
	}
	if snapshot.Available {
		t.Fatal("snapshot.Available = true, want false")
	}
	if !math.IsInf(snapshot.Score, 1) {
		t.Fatalf("snapshot.Score = %v, want +Inf", snapshot.Score)
	}
}

func TestSelectInitialChoosesLowestScore(t *testing.T) {
	now := time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC)
	manager := newTestManager(t, Config{})
	recordSuccess(t, manager, "node-a", 120*time.Millisecond, now)
	recordSuccess(t, manager, "node-b", 80*time.Millisecond, now)

	decision := manager.Select(now)
	if decision.SelectedID != "node-b" || !decision.Switched || decision.Reason != ReasonInitial {
		t.Fatalf("Select() = %+v, want initial switch to node-b", decision)
	}
	if manager.Current() != "node-b" {
		t.Fatalf("Current() = %q, want node-b", manager.Current())
	}
}

func TestSelectHonorsMinHoldTime(t *testing.T) {
	start := time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC)
	manager := newTestManager(t, Config{MinHoldTime: time.Minute, SwitchThreshold: 1})
	recordSuccess(t, manager, "node-a", 100*time.Millisecond, start)
	recordSuccess(t, manager, "node-b", 10*time.Millisecond, start)

	if err := manager.SetCurrent("node-a", start); err != nil {
		t.Fatalf("SetCurrent() unexpected error: %v", err)
	}
	held := manager.Select(start.Add(30 * time.Second))
	if held.SelectedID != "node-a" || held.Switched || held.Reason != ReasonMinHoldTime {
		t.Fatalf("Select() during hold = %+v, want held node-a", held)
	}

	switched := manager.Select(start.Add(time.Minute))
	if switched.SelectedID != "node-b" || !switched.Switched || switched.Reason != ReasonBetterCandidate {
		t.Fatalf("Select() after hold = %+v, want switch to node-b", switched)
	}
}

func TestSelectHonorsSwitchThreshold(t *testing.T) {
	now := time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC)
	manager := newTestManager(t, Config{SwitchThreshold: 30})
	recordSuccess(t, manager, "node-a", 100*time.Millisecond, now)
	recordSuccess(t, manager, "node-b", 80*time.Millisecond, now)

	if err := manager.SetCurrent("node-a", now); err != nil {
		t.Fatalf("SetCurrent() unexpected error: %v", err)
	}
	held := manager.Select(now.Add(time.Minute))
	if held.SelectedID != "node-a" || held.Switched || held.Reason != ReasonSwitchThreshold {
		t.Fatalf("Select() below threshold = %+v, want held node-a", held)
	}

	recordSuccess(t, manager, "node-b", 20*time.Millisecond, now.Add(2*time.Minute))
	switched := manager.Select(now.Add(2 * time.Minute))
	if switched.SelectedID != "node-b" || !switched.Switched || switched.Reason != ReasonBetterCandidate {
		t.Fatalf("Select() above threshold = %+v, want switch to node-b", switched)
	}
}

func TestCooldownAfterFailureExcludesCurrentThenAllowsReturn(t *testing.T) {
	start := time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC)
	manager := newTestManager(t, Config{CooldownAfterFailure: time.Minute, SwitchThreshold: 1, MinHoldTime: time.Second, SampleLimit: 2})
	recordSuccess(t, manager, "node-a", 10*time.Millisecond, start)
	recordSuccess(t, manager, "node-b", 100*time.Millisecond, start)
	if err := manager.SetCurrent("node-a", start); err != nil {
		t.Fatalf("SetCurrent() unexpected error: %v", err)
	}

	if err := manager.Record(Sample{NodeID: "node-a", Success: false, CheckedAt: start.Add(10 * time.Second)}); err != nil {
		t.Fatalf("Record() unexpected error: %v", err)
	}
	fallback := manager.Select(start.Add(20 * time.Second))
	if fallback.SelectedID != "node-b" || !fallback.Switched || fallback.Reason != ReasonCurrentUnavailable {
		t.Fatalf("Select() during cooldown = %+v, want fallback to node-b", fallback)
	}

	recordSuccess(t, manager, "node-a", 10*time.Millisecond, start.Add(90*time.Second))
	recordSuccess(t, manager, "node-a", 10*time.Millisecond, start.Add(91*time.Second))
	afterCooldown := manager.Select(start.Add(2 * time.Minute))
	if afterCooldown.SelectedID != "node-a" || !afterCooldown.Switched || afterCooldown.Reason != ReasonBetterCandidate {
		t.Fatalf("Select() after cooldown = %+v, want return to node-a", afterCooldown)
	}
}

func TestConsecutiveFailureFallbackBypassesHoldAndThreshold(t *testing.T) {
	start := time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC)
	manager := newTestManager(t, Config{
		MinHoldTime:          time.Hour,
		SwitchThreshold:      1000,
		CooldownAfterFailure: time.Nanosecond,
		ConsecutiveFailLimit: 2,
	})
	recordSuccess(t, manager, "node-a", 10*time.Millisecond, start)
	recordSuccess(t, manager, "node-b", 100*time.Millisecond, start)
	if err := manager.SetCurrent("node-a", start); err != nil {
		t.Fatalf("SetCurrent() unexpected error: %v", err)
	}

	if err := manager.Record(Sample{NodeID: "node-a", Success: false, CheckedAt: start.Add(time.Second)}); err != nil {
		t.Fatalf("Record() unexpected error: %v", err)
	}
	if err := manager.Record(Sample{NodeID: "node-a", Success: false, CheckedAt: start.Add(2 * time.Second)}); err != nil {
		t.Fatalf("Record() unexpected error: %v", err)
	}

	decision := manager.Select(start.Add(3 * time.Second))
	if decision.SelectedID != "node-b" || !decision.Switched || decision.Reason != ReasonConsecutiveFailureLimit {
		t.Fatalf("Select() = %+v, want consecutive failure fallback to node-b", decision)
	}
}

func TestSelectNoAvailableNodeKeepsPrevious(t *testing.T) {
	now := time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC)
	manager := newTestManager(t, Config{})
	if err := manager.SetCurrent("node-a", now); err != nil {
		t.Fatalf("SetCurrent() unexpected error: %v", err)
	}

	decision := manager.Select(now)
	if decision.SelectedID != "node-a" || decision.Switched || decision.Reason != ReasonNoAvailableNode {
		t.Fatalf("Select() = %+v, want no available keep node-a", decision)
	}
}

func TestRecordAndConfigValidation(t *testing.T) {
	if _, err := NewManager(Config{SampleLimit: -1}); err != ErrInvalidSampleLimit {
		t.Fatalf("NewManager(sample limit) error = %v, want %v", err, ErrInvalidSampleLimit)
	}
	if _, err := NewManager(Config{MinHoldTime: -time.Second}); err != ErrNegativeDuration {
		t.Fatalf("NewManager(duration) error = %v, want %v", err, ErrNegativeDuration)
	}
	if _, err := NewManager(Config{SwitchThreshold: -1}); err != ErrNegativeSwitchThreshold {
		t.Fatalf("NewManager(threshold) error = %v, want %v", err, ErrNegativeSwitchThreshold)
	}
	if _, err := NewManager(Config{ConsecutiveFailLimit: -1}); err != ErrInvalidConsecutiveFailLimit {
		t.Fatalf("NewManager(consecutive failures) error = %v, want %v", err, ErrInvalidConsecutiveFailLimit)
	}

	manager := newTestManager(t, Config{})
	if err := manager.Record(Sample{}); err != ErrEmptyNodeID {
		t.Fatalf("Record(empty id) error = %v, want %v", err, ErrEmptyNodeID)
	}
	if err := manager.Record(Sample{NodeID: "node-a", Latency: -time.Millisecond}); err != ErrNegativeLatency {
		t.Fatalf("Record(negative latency) error = %v, want %v", err, ErrNegativeLatency)
	}
	if err := manager.Register(""); err != ErrEmptyNodeID {
		t.Fatalf("Register(empty id) error = %v, want %v", err, ErrEmptyNodeID)
	}
	if err := manager.SetCurrent("", time.Time{}); err != ErrEmptyNodeID {
		t.Fatalf("SetCurrent(empty id) error = %v, want %v", err, ErrEmptyNodeID)
	}
}

func newTestManager(t *testing.T, cfg Config) *Manager {
	t.Helper()
	manager, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager() unexpected error: %v", err)
	}
	return manager
}

func recordSuccess(t *testing.T, manager *Manager, nodeID string, latency time.Duration, checkedAt time.Time) {
	t.Helper()
	if err := manager.Record(Sample{NodeID: nodeID, Latency: latency, Success: true, CheckedAt: checkedAt}); err != nil {
		t.Fatalf("Record(%s) unexpected error: %v", nodeID, err)
	}
}

func assertFloat(t *testing.T, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 0.000001 {
		t.Fatalf("got %v, want %v", got, want)
	}
}
