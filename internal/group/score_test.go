package group

import "testing"

func TestScoreUsesLatencyAndFailurePenalty(t *testing.T) {
	tests := []struct {
		name      string
		latencyMS int
		failed    int
		total     int
		wantScore float64
	}{
		{
			name:      "stable slower node",
			latencyMS: 180,
			failed:    0,
			total:     10,
			wantScore: 180,
		},
		{
			name:      "fast flaky node",
			latencyMS: 80,
			failed:    3,
			total:     10,
			wantScore: 380,
		},
		{
			name:      "unknown failure rate",
			latencyMS: 120,
			failed:    0,
			total:     0,
			wantScore: 120,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Score(tt.latencyMS, tt.failed, tt.total)
			if err != nil {
				t.Fatalf("Score() unexpected error: %v", err)
			}
			if got != tt.wantScore {
				t.Fatalf("Score() = %v, want %v", got, tt.wantScore)
			}
		})
	}
}

func TestScoreRejectsInvalidInputs(t *testing.T) {
	tests := []struct {
		name      string
		latencyMS int
		failed    int
		total     int
		wantErr   error
	}{
		{name: "negative latency", latencyMS: -1, failed: 0, total: 1, wantErr: ErrNegativeLatency},
		{name: "negative failed checks", latencyMS: 1, failed: -1, total: 1, wantErr: ErrNegativeCheckCount},
		{name: "negative total checks", latencyMS: 1, failed: 0, total: -1, wantErr: ErrNegativeCheckCount},
		{name: "failed exceeds total", latencyMS: 1, failed: 2, total: 1, wantErr: ErrFailedChecksOverflow},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Score(tt.latencyMS, tt.failed, tt.total)
			if err != tt.wantErr {
				t.Fatalf("Score() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}
