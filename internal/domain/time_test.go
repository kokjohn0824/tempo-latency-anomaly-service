package domain

import (
    "fmt"
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
)

func tsNanoAt(t time.Time) string {
    return fmt.Sprintf("%d", t.UnixNano())
}

func TestParseTimeBucket_TimezoneAndDayType(t *testing.T) {
    loc, err := time.LoadLocation("Asia/Taipei")
    if !assert.NoError(t, err) { return }

    // Monday
    mon := time.Date(2024, 1, 8, 16, 30, 0, 0, loc) // Mon
    bMon, err := ParseTimeBucket(tsNanoAt(mon), "Asia/Taipei")
    assert.NoError(t, err)
    assert.Equal(t, 16, bMon.Hour)
    assert.Equal(t, "weekday", bMon.DayType)

    // Saturday
    sat := time.Date(2024, 1, 6, 3, 15, 0, 0, loc)
    bSat, err := ParseTimeBucket(tsNanoAt(sat), "Asia/Taipei")
    assert.NoError(t, err)
    assert.Equal(t, 3, bSat.Hour)
    assert.Equal(t, "weekend", bSat.DayType)

    // Sunday
    sun := time.Date(2024, 1, 7, 23, 59, 0, 0, loc)
    bSun, err := ParseTimeBucket(tsNanoAt(sun), "Asia/Taipei")
    assert.NoError(t, err)
    assert.Equal(t, 23, bSun.Hour)
    assert.Equal(t, "weekend", bSun.DayType)
}

func TestParseTimeBucket_DefaultTimezone(t *testing.T) {
    // Use a known instant and rely on default timezone (Asia/Taipei)
    loc, _ := time.LoadLocation("Asia/Taipei")
    ts := time.Date(2024, 1, 8, 0, 0, 0, 0, loc) // 00:00 Tue local
    b, err := ParseTimeBucket(tsNanoAt(ts), "")
    assert.NoError(t, err)
    assert.Equal(t, 0, b.Hour)
    assert.Equal(t, "weekday", b.DayType)
}

func TestParseTimeBucket_InvalidInputs(t *testing.T) {
    // Invalid unix nano string
    _, err := ParseTimeBucket("not-a-number", "Asia/Taipei")
    assert.Error(t, err)

    // Invalid timezone
    loc, _ := time.LoadLocation("Asia/Taipei")
    ts := time.Date(2024, 1, 8, 12, 0, 0, 0, loc)
    _, err = ParseTimeBucket(tsNanoAt(ts), "Mars/Phobos")
    assert.Error(t, err)
}

func TestParseTimeBucket_BoundaryHours(t *testing.T) {
    loc, _ := time.LoadLocation("Asia/Taipei")
    // 23:59
    late := time.Date(2024, 1, 8, 23, 59, 59, 0, loc)
    b1, err := ParseTimeBucket(tsNanoAt(late), "Asia/Taipei")
    assert.NoError(t, err)
    assert.Equal(t, 23, b1.Hour)

    // 00:00
    midnight := time.Date(2024, 1, 9, 0, 0, 0, 0, loc)
    b2, err := ParseTimeBucket(tsNanoAt(midnight), "Asia/Taipei")
    assert.NoError(t, err)
    assert.Equal(t, 0, b2.Hour)
}

