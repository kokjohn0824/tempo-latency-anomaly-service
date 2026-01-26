package service

import (
	"fmt"

	"github.com/alexchang/tempo-latency-anomaly-service/internal/config"
	"github.com/alexchang/tempo-latency-anomaly-service/internal/store"
)

// DurationEvaluation captures anomaly decision and explanation.
type DurationEvaluation struct {
	IsAnomaly   bool
	ThresholdMs float64
	Explanation string
}

// EvaluateDuration applies the default threshold strategy and returns decision details.
func EvaluateDuration(cfg *config.Config, durationMs int64, baseline *store.Baseline) DurationEvaluation {
	if cfg == nil || baseline == nil {
		return DurationEvaluation{
			IsAnomaly:   false,
			ThresholdMs: 0,
			Explanation: "baseline unavailable",
		}
	}

	rel := baseline.P95 * cfg.Stats.Factor
	abs := baseline.P50 + float64(cfg.Stats.K)*baseline.MAD
	threshold := rel
	if abs > threshold {
		threshold = abs
	}

	dur := float64(durationMs)
	isAnomaly := dur > threshold
	explanation := fmt.Sprintf(
		"duration %.0fms %s threshold %.2fms (p50=%.2f, p95=%.2f, MAD=%.2f, factor=%.2f, k=%d)",
		dur,
		ternary(isAnomaly, "exceeds", "within"),
		threshold,
		baseline.P50, baseline.P95, baseline.MAD,
		cfg.Stats.Factor,
		cfg.Stats.K,
	)

	return DurationEvaluation{
		IsAnomaly:   isAnomaly,
		ThresholdMs: threshold,
		Explanation: explanation,
	}
}
