package group

import "time"

// NodeHealth is a cached health snapshot for one proxy node.
type NodeHealth struct {
	ProxyID       string
	LastLatencyMS int
	SuccessCount  int
	FailureCount  int
	LastCheckedAt time.Time
	Score         float64
	CooldownUntil time.Time
}

// TotalChecks returns the number of samples represented by this snapshot.
func (h NodeHealth) TotalChecks() int {
	return h.SuccessCount + h.FailureCount
}

// ComputeScore updates h.Score using the Phase 1 scoring rule.
func (h *NodeHealth) ComputeScore() error {
	score, err := Score(h.LastLatencyMS, h.FailureCount, h.TotalChecks())
	if err != nil {
		return err
	}
	h.Score = score
	return nil
}
