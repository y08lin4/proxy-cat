package group

import "time"

// Candidate is a proxy node with a computed health score.
type Candidate struct {
	ProxyID string
	Score   float64
}

// CandidateFromHealth converts a cached health snapshot into a selectable
// candidate. Nodes in cooldown are skipped by returning false.
func CandidateFromHealth(health NodeHealth, now time.Time) (Candidate, bool) {
	if health.ProxyID == "" {
		return Candidate{}, false
	}
	if !health.CooldownUntil.IsZero() && now.Before(health.CooldownUntil) {
		return Candidate{}, false
	}

	score, err := Score(health.LastLatencyMS, health.FailureCount, health.TotalChecks())
	if err != nil {
		return Candidate{}, false
	}

	return Candidate{ProxyID: health.ProxyID, Score: score}, true
}

// BestCandidate returns the lowest-scoring candidate from the current snapshot.
func BestCandidate(candidates []Candidate) (Candidate, bool) {
	if len(candidates) == 0 {
		return Candidate{}, false
	}

	best := candidates[0]
	for _, candidate := range candidates[1:] {
		if candidate.Score < best.Score {
			best = candidate
		}
	}
	return best, true
}
