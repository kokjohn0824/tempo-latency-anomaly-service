package domain

import "time"

// TraceEvent represents a single trace summary used for anomaly detection.
// startTimeUnixNano is a string of Unix time in nanoseconds.
type TraceEvent struct {
    TraceID            string `json:"traceID" example:"abc123def456"`
    RootServiceName    string `json:"rootServiceName" example:"my-service"`
    RootTraceName      string `json:"rootTraceName" example:"GET /api/users"`
    StartTimeUnixNano  string `json:"startTimeUnixNano" example:"1673000000000000000"`
    DurationMs         int64  `json:"durationMs" example:"250"`
}

// TimeBucket captures hour-of-day and day type (weekday/weekend).
type TimeBucket struct {
    Hour    int    `json:"hour" example:"16"`              // 0-23
    DayType string `json:"dayType" example:"weekday"`      // "weekday" or "weekend"
}

// BaselineStats contains precomputed statistics for a given key/bucket.
type BaselineStats struct {
    P50         float64   `json:"p50" example:"233.5"`
    P95         float64   `json:"p95" example:"562.0"`
    MAD         float64   `json:"mad" example:"43.0"`
    SampleCount int       `json:"sampleCount" example:"50"`
    UpdatedAt   time.Time `json:"updatedAt" example:"2026-01-15T08:00:00Z"`
}

// AnomalyCheckRequest is the input for anomaly checking.
type AnomalyCheckRequest struct {
    Service       string `json:"service" example:"twdiw-customer-service-prod"`
    Endpoint      string `json:"endpoint" example:"GET /actuator/health"`
    TimestampNano int64  `json:"timestampNano" example:"1673000000000000000"`
    DurationMs    int64  `json:"durationMs" example:"250"`
}

// AnomalyCheckResponse is the output with decision and explanation.
type AnomalyCheckResponse struct {
    IsAnomaly   bool            `json:"isAnomaly" example:"false"`
    Bucket      TimeBucket      `json:"bucket"`
    Baseline    *BaselineStats  `json:"baseline,omitempty"`
    Explanation string          `json:"explanation" example:"duration 250ms within threshold 1124.00ms"`
}

// ServiceEndpoint represents a service and endpoint pair with available baselines.
type ServiceEndpoint struct {
    Service   string   `json:"service" example:"twdiw-customer-service-prod"`
    Endpoint  string   `json:"endpoint" example:"GET /actuator/health"`
    Buckets   []string `json:"buckets" example:"16|weekday,17|weekday"`
}

// AvailableServicesResponse is the output for listing available services and endpoints.
type AvailableServicesResponse struct {
    TotalServices  int               `json:"totalServices" example:"3"`
    TotalEndpoints int               `json:"totalEndpoints" example:"15"`
    Services       []ServiceEndpoint `json:"services"`
}

