package group

import "errors"

var (
	ErrNegativeLatency      = errors.New("latency must be >= 0")
	ErrNegativeCheckCount   = errors.New("check counts must be >= 0")
	ErrFailedChecksOverflow = errors.New("failed checks cannot exceed total checks")
	ErrUnsupportedGroupType = errors.New("unsupported group type")
)

// Score returns the stable-first score used by the early group selector:
//
//	score = latency_ms + failed_checks / total_checks * 1000
//
// Lower scores are better. When there are no checks yet, the failure penalty is
// zero so callers can decide how to rank unknown nodes at a higher layer.
func Score(latencyMS, failedChecks, totalChecks int) (float64, error) {
	if latencyMS < 0 {
		return 0, ErrNegativeLatency
	}
	if failedChecks < 0 || totalChecks < 0 {
		return 0, ErrNegativeCheckCount
	}
	if failedChecks > totalChecks {
		return 0, ErrFailedChecksOverflow
	}
	if totalChecks == 0 {
		return float64(latencyMS), nil
	}

	failurePenalty := float64(failedChecks) / float64(totalChecks) * 1000
	return float64(latencyMS) + failurePenalty, nil
}
