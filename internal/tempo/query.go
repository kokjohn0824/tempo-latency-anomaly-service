package tempo

import (
    "net/url"
    "strconv"
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

