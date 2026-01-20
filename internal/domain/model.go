package domain

import "time"

// TraceEvent represents a single trace summary used for anomaly detection.
// startTimeUnixNano is a string of Unix time in nanoseconds.
type TraceEvent struct {
	TraceID           string `json:"traceID" example:"abc123def456"`
	RootServiceName   string `json:"rootServiceName" example:"my-service"`
	RootTraceName     string `json:"rootTraceName" example:"GET /api/users"`
	StartTimeUnixNano string `json:"startTimeUnixNano" example:"1673000000000000000"`
	DurationMs        int64  `json:"durationMs" example:"250"`
}

// TraceLookupResponse returns traces matching a Tempo search.
type TraceLookupResponse struct {
	Service  string       `json:"service" example:"api-gateway"`
	Endpoint string       `json:"endpoint" example:"GET /api/users"`
	Start    int64        `json:"start" example:"1736928000"`
	End      int64        `json:"end" example:"1736931600"`
	Count    int          `json:"count" example:"2"`
	Traces   []TraceEvent `json:"traces"`
}

// SpanSummary represents a single span in a trace.
type SpanSummary struct {
	SpanID       string    `json:"spanId" example:"b7ad6b7169203331"`
	Name         string    `json:"name" example:"db.query"`
	Service      string    `json:"service" example:"orders-db"`
	DurationMs   int64     `json:"durationMs" example:"842"`
	StartTime    time.Time `json:"startTime" example:"2026-01-20T10:12:33.123Z"`
	EndTime      time.Time `json:"endTime" example:"2026-01-20T10:12:33.965Z"`
	ParentSpanID string    `json:"parentSpanId,omitempty" example:"a1b2c3d4e5f6a7b8"`
}

// LongestSpanResponse returns the longest span within a trace.
type LongestSpanResponse struct {
	TraceID     string      `json:"traceID" example:"abc123def456"`
	LongestSpan SpanSummary `json:"longestSpan"`
	Source      string      `json:"source" example:"tempo"`
	ComputedAt  time.Time   `json:"computedAt" example:"2026-01-20T10:12:35.001Z"`
}

// ErrorDetail describes an API error.
type ErrorDetail struct {
	Code    string         `json:"code" example:"trace_not_found"`
	Message string         `json:"message" example:"Trace not found in Tempo"`
	Details map[string]any `json:"details,omitempty"`
}

// ErrorResponse is a standard API error wrapper.
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// TimeBucket captures hour-of-day and day type (weekday/weekend).
type TimeBucket struct {
	Hour    int    `json:"hour" example:"9"`          // 0-23
	DayType string `json:"dayType" example:"weekday"` // "weekday" or "weekend"
}

// BaselineStats contains precomputed statistics for a given key/bucket.
type BaselineStats struct {
	P50         float64   `json:"p50" example:"1.0"`
	P95         float64   `json:"p95" example:"2.0"`
	MAD         float64   `json:"mad" example:"0.0"`
	SampleCount int       `json:"sampleCount" example:"188"`
	UpdatedAt   time.Time `json:"updatedAt" example:"2026-01-16T02:00:00Z"`
}

// AnomalyCheckRequest is the input for anomaly checking.
type AnomalyCheckRequest struct {
	Service       string `json:"service" example:"twdiw-customer-service-prod"`
	Endpoint      string `json:"endpoint" example:"AiPromptSyncScheduler.syncAiPromptsToDify"`
	TimestampNano int64  `json:"timestampNano" example:"1737000000000000000"`
	DurationMs    int64  `json:"durationMs" example:"5"`
}

// BaselineSource indicates which fallback level was used to obtain the baseline.
type BaselineSource string

const (
	SourceExact       BaselineSource = "exact"       // Level 1: Exact hour|dayType match
	SourceNearby      BaselineSource = "nearby"      // Level 2: Nearby hours (±1, ±2)
	SourceDayType     BaselineSource = "daytype"     // Level 3: All hours of same day type
	SourceGlobal      BaselineSource = "global"      // Level 4: All data (any hour, any day type)
	SourceUnavailable BaselineSource = "unavailable" // Level 5: No data available
)

// AnomalyCheckResponse is the output with decision and explanation.
type AnomalyCheckResponse struct {
	IsAnomaly       bool           `json:"isAnomaly" example:"false"`
	CannotDetermine bool           `json:"cannotDetermine,omitempty" example:"false"`
	Bucket          TimeBucket     `json:"bucket"`
	Baseline        *BaselineStats `json:"baseline,omitempty"`
	BaselineSource  BaselineSource `json:"baselineSource" example:"exact"`
	FallbackLevel   int            `json:"fallbackLevel,omitempty" example:"1"`
	SourceDetails   string         `json:"sourceDetails,omitempty" example:"exact match: 9|weekday"`
	Explanation     string         `json:"explanation" example:"duration 5ms within threshold 2.00ms"`
}

// ServiceEndpoint represents a service and endpoint pair with available baselines.
type ServiceEndpoint struct {
	Service  string   `json:"service" example:"twdiw-customer-service-prod"`
	Endpoint string   `json:"endpoint" example:"AiPromptSyncScheduler.syncAiPromptsToDify"`
	Buckets  []string `json:"buckets" example:"6|weekday,9|weekday,10|weekday,12|weekday,13|weekend,17|weekday,20|weekday"`
}

// AvailableServicesResponse is the output for listing available services and endpoints.
type AvailableServicesResponse struct {
	TotalServices  int               `json:"totalServices" example:"4"`
	TotalEndpoints int               `json:"totalEndpoints" example:"17"`
	Services       []ServiceEndpoint `json:"services"`
}
