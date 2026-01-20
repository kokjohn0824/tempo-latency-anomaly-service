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

// TraceByIDResponse maps Tempo's trace-by-id payload (OTLP JSON).
type TraceByIDResponse struct {
	ResourceSpans []ResourceSpan `json:"resourceSpans"`
	Batches       []ResourceSpan `json:"batches"`
}

// ResourceSpan represents a resource and its spans (OTLP JSON).
type ResourceSpan struct {
	Resource                    Resource    `json:"resource"`
	ScopeSpans                  []ScopeSpan `json:"scopeSpans"`
	InstrumentationLibrarySpans []ScopeSpan `json:"instrumentationLibrarySpans"`
}

// Resource contains resource attributes like service name.
type Resource struct {
	Attributes []KeyValue `json:"attributes"`
}

// KeyValue represents an OTLP attribute.
type KeyValue struct {
	Key   string         `json:"key"`
	Value AttributeValue `json:"value"`
}

// AttributeValue contains the typed value of an attribute.
type AttributeValue struct {
	StringValue string `json:"stringValue"`
}

// ScopeSpan represents a collection of spans.
type ScopeSpan struct {
	Spans []Span `json:"spans"`
}

// Span represents a single OTLP span.
type Span struct {
	TraceID           string `json:"traceId"`
	SpanID            string `json:"spanId"`
	ParentSpanID      string `json:"parentSpanId"`
	Name              string `json:"name"`
	StartTimeUnixNano string `json:"startTimeUnixNano"`
	EndTimeUnixNano   string `json:"endTimeUnixNano"`
}

// SpanData represents a normalized span with service context.
type SpanData struct {
	TraceID           string
	SpanID            string
	ParentSpanID      string
	Name              string
	ServiceName       string
	StartTimeUnixNano string
	EndTimeUnixNano   string
}
