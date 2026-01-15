package stats

import (
    "math"
    "sort"
)

// P50 computes the median (50th percentile) of the given samples.
// It does not modify the input slice.
func P50(samples []int64) float64 {
    n := len(samples)
    if n == 0 {
        return 0
    }

    // Work on a copy to avoid mutating the input slice.
    arr := make([]int64, n)
    copy(arr, samples)
    sort.Slice(arr, func(i, j int) bool { return arr[i] < arr[j] })

    mid := n / 2
    if n%2 == 1 {
        // Odd: middle element
        return float64(arr[mid])
    }
    // Even: average of two middle elements
    return (float64(arr[mid-1]) + float64(arr[mid])) / 2.0
}

// P95 computes the 95th percentile using the nearest-rank method.
// It does not modify the input slice.
// Definition: rank = ceil(0.95 * n); index = rank - 1 (0-based).
func P95(samples []int64) float64 {
    n := len(samples)
    if n == 0 {
        return 0
    }

    arr := make([]int64, n)
    copy(arr, samples)
    sort.Slice(arr, func(i, j int) bool { return arr[i] < arr[j] })

    rank := int(math.Ceil(0.95 * float64(n)))
    if rank < 1 {
        rank = 1
    }
    if rank > n {
        rank = n
    }
    idx := rank - 1
    return float64(arr[idx])
}

