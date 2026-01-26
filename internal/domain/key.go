package domain

import (
	"fmt"
	"strconv"
	"time"
)

const (
	defaultTimezone = "Asia/Taipei"
	dayTypeWeekday  = "weekday"
	dayTypeWeekend  = "weekend"
)

// ParseTimeBucket converts a unix nano timestamp string into a TimeBucket using the given timezone.
// If timezone is empty, it defaults to Asia/Taipei.
func ParseTimeBucket(unixNano string, timezone string) (TimeBucket, error) {
	if timezone == "" {
		timezone = defaultTimezone
	}

	ns, err := strconv.ParseInt(unixNano, 10, 64)
	if err != nil {
		return TimeBucket{}, fmt.Errorf("invalid unix nano: %w", err)
	}

	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return TimeBucket{}, fmt.Errorf("load location '%s': %w", timezone, err)
	}

	t := time.Unix(0, ns).In(loc)
	hour := t.Hour() // 0-23

	// Determine weekend: Saturday or Sunday in given timezone.
	wd := t.Weekday()
	dayType := dayTypeWeekday
	if wd == time.Saturday || wd == time.Sunday {
		dayType = dayTypeWeekend
	}

	return TimeBucket{Hour: hour, DayType: dayType}, nil
}

// MakeBaselineKey generates the baseline cache key for the given service/endpoint and time bucket.
// Format: base:{service}|{endpoint}|{hour}|{dayType}
func MakeBaselineKey(service, endpoint string, bucket TimeBucket) string {
	return fmt.Sprintf("base:%s|%s|%d|%s", service, endpoint, bucket.Hour, bucket.DayType)
}

// MakeSpanBaselineKey generates the baseline cache key for span-level baselines.
// Format: spanbase:{service}|{spanName}|{hour}|{dayType}
func MakeSpanBaselineKey(service, spanName string, bucket TimeBucket) string {
	return fmt.Sprintf("spanbase:%s|%s|%d|%s", service, spanName, bucket.Hour, bucket.DayType)
}

// MakeDurationKey generates the rolling duration list key for the given service/endpoint and time bucket.
// Format: dur:{service}|{endpoint}|{hour}|{dayType}
func MakeDurationKey(service, endpoint string, bucket TimeBucket) string {
	return fmt.Sprintf("dur:%s|%s|%d|%s", service, endpoint, bucket.Hour, bucket.DayType)
}

// MakeSpanDurationKey generates the rolling duration list key for span-level baselines.
// Format: spandur:{service}|{spanName}|{hour}|{dayType}
func MakeSpanDurationKey(service, spanName string, bucket TimeBucket) string {
	return fmt.Sprintf("spandur:%s|%s|%d|%s", service, spanName, bucket.Hour, bucket.DayType)
}
