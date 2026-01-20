package tempo

import (
	"net/url"
	"strconv"
	"strings"
	"time"
)

// BuildQueryParams constructs query parameters to search for traces within
// the last N seconds. It returns `start` and `end` in Unix seconds, as
// expected by Tempo search APIs.
func BuildQueryParams(lookbackSeconds int) url.Values {
	now := time.Now().Unix()
	start := now - int64(lookbackSeconds)

	q := url.Values{}
	q.Set("start", strconv.FormatInt(start, 10))
	q.Set("end", strconv.FormatInt(now, 10))
	return q
}

// BuildRangeParams constructs query parameters for a fixed time range.
// It expects Unix epoch seconds for start and end.
func BuildRangeParams(start, end int64) url.Values {
	q := url.Values{}
	q.Set("start", strconv.FormatInt(start, 10))
	q.Set("end", strconv.FormatInt(end, 10))
	return q
}

// BuildSearchTags creates a logfmt tag filter for Tempo search.
func BuildSearchTags(service, endpoint string) string {
	return "service.name=" + logfmtValue(service) + " name=" + logfmtValue(endpoint)
}

func logfmtValue(value string) string {
	if strings.ContainsAny(value, " \t\"=") {
		escaped := strings.ReplaceAll(value, "\"", "\\\"")
		return "\"" + escaped + "\""
	}
	return value
}
