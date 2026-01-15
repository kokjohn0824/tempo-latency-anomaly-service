package stats

import (
    "github.com/alexchang/tempo-latency-anomaly-service/internal/domain"
)

// ComputeBaseline calculates baseline statistics (p50, p95, mad, sampleCount)
// from the provided samples. It does not modify the input slice.
// Edge cases:
// - Empty samples: returns zeros for all stats and SampleCount=0.
// - Any sample size: uses nearest-rank for p95 and standard median for p50/MAD.
func ComputeBaseline(samples []int64) domain.BaselineStats {
    n := len(samples)
    if n == 0 {
        return domain.BaselineStats{}
    }

    p50 := P50(samples)
    p95 := P95(samples)
    mad := MAD(samples, p50)

    return domain.BaselineStats{
        P50:         p50,
        P95:         p95,
        MAD:         mad,
        SampleCount: n,
        // UpdatedAt left as zero; set by persistence layer when stored.
    }
}

