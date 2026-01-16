package stats

import (
    "testing"

    "github.com/stretchr/testify/assert"
)

func TestP50_OddAndEvenSamples(t *testing.T) {
    // Odd sample count
    odd := []int64{5, 1, 3}
    // Keep a copy to ensure input is not modified
    origOdd := append([]int64(nil), odd...)
    mOdd := P50(odd)
    assert.Equal(t, 3.0, mOdd)
    assert.Equal(t, origOdd, odd, "P50 must not modify input slice")

    // Even sample count
    even := []int64{4, 1, 3, 2}
    origEven := append([]int64(nil), even...)
    mEven := P50(even)
    assert.Equal(t, 2.5, mEven)
    assert.Equal(t, origEven, even, "P50 must not modify input slice")
}

func TestP95_NearestRankAndBoundaries(t *testing.T) {
    // Sorted 1..20 → rank = ceil(0.95*20) = 19 → value 19
    var s []int64
    for i := 1; i <= 20; i++ {
        s = append(s, int64(i))
    }
    orig := append([]int64(nil), s...)
    p := P95(s)
    assert.Equal(t, 19.0, p)
    assert.Equal(t, orig, s, "P95 must not modify input slice")

    // Small n
    assert.Equal(t, 42.0, P95([]int64{42}))
    // n=2 → ceil(0.95*2)=2 → second element when sorted
    assert.Equal(t, 7.0, P95([]int64{1, 7}))
}

func TestMAD_Computation(t *testing.T) {
    // Simple set: median=2, deviations=[1,0,1] → MAD=1
    samples := []int64{1, 2, 3}
    med := P50(samples)
    mad := MAD(samples, med)
    assert.Equal(t, 1.0, mad)

    // Another set: [1,1,2,2] → median=1.5 → deviations all 0.5 → MAD=0.5
    s2 := []int64{1, 1, 2, 2}
    med2 := P50(s2)
    mad2 := MAD(s2, med2)
    assert.InDelta(t, 0.5, mad2, 1e-9)
}

func TestComputeBaseline_EmptyAndSingle(t *testing.T) {
    // Empty
    b0 := ComputeBaseline(nil)
    assert.Equal(t, 0, b0.SampleCount)
    assert.Equal(t, 0.0, b0.P50)
    assert.Equal(t, 0.0, b0.P95)
    assert.Equal(t, 0.0, b0.MAD)

    // Single sample
    b1 := ComputeBaseline([]int64{42})
    assert.Equal(t, 1, b1.SampleCount)
    assert.Equal(t, 42.0, b1.P50)
    assert.Equal(t, 42.0, b1.P95)
    assert.Equal(t, 0.0, b1.MAD)
}

func TestThresholdFormula_MaxOfRelativeAndAbsolute(t *testing.T) {
    // Construct a sample set with spread to exercise both branches
    samples := []int64{100, 120, 150, 200, 300, 400, 1000, 1100, 1200, 1300}
    b := ComputeBaseline(samples)

    // Case 1: relative dominates → threshold = p95 * factor
    factor := 2.0
    k := 3.0
    rel := b.P95 * factor
    abs := b.P50 + k*b.MAD
    // Expected from manual calculation: p95=1300, p50=350, MAD=240
    assert.InDelta(t, 1300.0, b.P95, 1e-9)
    assert.InDelta(t, 350.0, b.P50, 1e-9)
    assert.InDelta(t, 240.0, b.MAD, 1e-9)
    threshold := rel
    if abs > threshold {
        threshold = abs
    }
    assert.InDelta(t, 2600.0, threshold, 1e-9)

    // Case 2: absolute dominates → threshold = p50 + k*MAD
    factor2 := 1.1
    k2 := 10.0
    rel2 := b.P95 * factor2
    abs2 := b.P50 + k2*b.MAD
    thr2 := rel2
    if abs2 > thr2 {
        thr2 = abs2
    }
    assert.InDelta(t, 2750.0, thr2, 1e-9)
}
