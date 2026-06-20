package group

import "testing"

func TestNodeHealthComputeScore(t *testing.T) {
	health := NodeHealth{
		ProxyID:       "node-a",
		LastLatencyMS: 80,
		SuccessCount:  7,
		FailureCount:  3,
	}

	if err := health.ComputeScore(); err != nil {
		t.Fatalf("ComputeScore() unexpected error: %v", err)
	}
	if health.Score != 380 {
		t.Fatalf("health.Score = %v, want 380", health.Score)
	}
}
