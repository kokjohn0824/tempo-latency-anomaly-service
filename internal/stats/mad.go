package stats

import "sort"

// MAD computes the Median Absolute Deviation given samples and a precomputed median.
// Formula: median(|x - median|). It does not modify the input slice.
func MAD(samples []int64, median float64) float64 {
    n := len(samples)
    if n == 0 {
        return 0
    }

    // Compute absolute deviations (float64) without modifying the input slice.
    dev := make([]float64, 0, n)
    for _, v := range samples {
        d := float64(v) - median
        if d < 0 {
            d = -d
        }
        dev = append(dev, d)
    }

    // Median of deviations
    sort.Float64s(dev)
    mid := n / 2
    if n%2 == 1 {
        return dev[mid]
    }
    return (dev[mid-1] + dev[mid]) / 2.0
}

