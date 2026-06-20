package group

import (
	"testing"
	"time"
)

func TestCandidateFromHealthComputesScore(t *testing.T) {
	now := time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC)
	health := NodeHealth{
		ProxyID:       "node-a",
		LastLatencyMS: 180,
		SuccessCount:  10,
		FailureCount:  0,
	}

	candidate, ok := CandidateFromHealth(health, now)
	if !ok {
		t.Fatal("CandidateFromHealth() ok = false, want true")
	}
	if candidate.ProxyID != "node-a" {
		t.Fatalf("candidate.ProxyID = %q, want node-a", candidate.ProxyID)
	}
	if candidate.Score != 180 {
		t.Fatalf("candidate.Score = %v, want 180", candidate.Score)
	}
}

func TestCandidateFromHealthSkipsCooldown(t *testing.T) {
	now := time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC)
	health := NodeHealth{
		ProxyID:       "node-a",
		LastLatencyMS: 80,
		CooldownUntil: now.Add(time.Minute),
	}

	if _, ok := CandidateFromHealth(health, now); ok {
		t.Fatal("CandidateFromHealth() ok = true, want false")
	}
}

func TestBestCandidateReturnsLowestScore(t *testing.T) {
	best, ok := BestCandidate([]Candidate{
		{ProxyID: "node-a", Score: 180},
		{ProxyID: "node-b", Score: 380},
		{ProxyID: "node-c", Score: 120},
	})
	if !ok {
		t.Fatal("BestCandidate() ok = false, want true")
	}
	if best.ProxyID != "node-c" {
		t.Fatalf("best.ProxyID = %q, want node-c", best.ProxyID)
	}
}

func TestBestCandidateEmpty(t *testing.T) {
	if _, ok := BestCandidate(nil); ok {
		t.Fatal("BestCandidate(nil) ok = true, want false")
	}
}
