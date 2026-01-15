package tempo

// TempoResponse represents the JSON payload returned by Tempo search APIs.
// It contains a list of trace summaries.
type TempoResponse struct {
    Traces []TraceData `json:"traces"`
}

// TraceData maps a single trace summary from Tempo's response.
// JSON tags follow Tempo API field names. We intentionally name the
// field RootTraceName (domain language) while mapping to Tempo's
// root span name via the json tag.
type TraceData struct {
    TraceID           string `json:"traceID"`
    RootServiceName   string `json:"rootServiceName"`
    RootTraceName     string `json:"rootTraceName"`
    StartTimeUnixNano string `json:"startTimeUnixNano"`
    DurationMs        int64  `json:"durationMs"`
}

