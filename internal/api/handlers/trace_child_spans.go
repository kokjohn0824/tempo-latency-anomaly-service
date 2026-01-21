package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/alexchang/tempo-latency-anomaly-service/internal/domain"
	"github.com/alexchang/tempo-latency-anomaly-service/internal/tempo"
)

// TraceChildSpans godoc
// @Summary Get child spans of a specific span
// @Description Retrieve all child spans for a given span ID within a trace
// @Tags Traces
// @Accept json
// @Produce json
// @Param traceId path string true "Trace ID" example("abc123def456")
// @Param spanId path string true "Parent Span ID" example("xyz789")
// @Success 200 {object} domain.ChildSpansResponse
// @Failure 400 {object} domain.ErrorResponse "Invalid trace ID or span ID"
// @Failure 404 {object} domain.ErrorResponse "Trace or span not found"
// @Failure 422 {object} domain.ErrorResponse "Trace has no spans"
// @Failure 502 {object} domain.ErrorResponse "Tempo error"
// @Failure 504 {object} domain.ErrorResponse "Tempo timeout"
// @Failure 503 {object} domain.ErrorResponse "Tempo not available"
// @Router /v1/traces/{traceId}/spans/{spanId}/children [get]
func TraceChildSpans(client *tempo.Client) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if client == nil {
			writeError(w, http.StatusServiceUnavailable, "tempo_unavailable", "tempo client not available", nil)
			return
		}

		traceID, spanID, ok := parseTraceAndSpanID(r.URL.Path)
		if !ok {
			// Debug: log the path that failed to parse
			writeError(w, http.StatusBadRequest, "invalid_parameters", "traceId and spanID must be provided", map[string]any{"traceId": traceID, "spanId": spanID, "path": r.URL.Path})
			return
		}

		spans, err := client.GetTraceSpans(r.Context(), traceID)
		if err != nil {
			if tempo.IsTimeout(err) {
				writeError(w, http.StatusGatewayTimeout, "tempo_timeout", "tempo request timed out", map[string]any{"traceId": traceID})
				return
			}
			var respErr tempo.ResponseError
			if errors.As(err, &respErr) {
				switch respErr.StatusCode {
				case http.StatusNotFound:
					writeError(w, http.StatusNotFound, "trace_not_found", "trace not found in Tempo", map[string]any{"traceId": traceID})
					return
				case http.StatusTooManyRequests:
					writeError(w, http.StatusBadGateway, "tempo_error", "tempo rate limited", map[string]any{"traceId": traceID, "tempoStatus": respErr.StatusCode})
					return
				default:
					if respErr.StatusCode >= 500 {
						writeError(w, http.StatusBadGateway, "tempo_error", "tempo request failed", map[string]any{"traceId": traceID, "tempoStatus": respErr.StatusCode})
						return
					}
					writeError(w, http.StatusBadGateway, "tempo_error", "tempo request failed", map[string]any{"traceId": traceID, "tempoStatus": respErr.StatusCode})
					return
				}
			}
			writeError(w, http.StatusBadGateway, "tempo_error", "tempo request failed", map[string]any{"traceId": traceID})
			return
		}

		if len(spans) == 0 {
			writeError(w, http.StatusUnprocessableEntity, "trace_empty", "trace has no spans", map[string]any{"traceId": traceID})
			return
		}

		// Find parent span
		parentSpan, found := findSpanByID(spans, spanID)
		if !found {
			writeError(w, http.StatusNotFound, "span_not_found", "parent span not found in trace", map[string]any{"traceId": traceID, "spanId": spanID})
			return
		}

		// Find child spans
		childSpans := findChildSpans(spans, spanID)

		resp := domain.ChildSpansResponse{
			TraceID: traceID,
			ParentSpan: domain.SpanSummary{
				SpanID:       parentSpan.SpanID,
				Name:         parentSpan.Name,
				Service:      parentSpan.ServiceName,
				DurationMs:   calculateDuration(parentSpan.StartTimeUnixNano, parentSpan.EndTimeUnixNano),
				StartTime:    parseTime(parentSpan.StartTimeUnixNano),
				EndTime:      parseTime(parentSpan.EndTimeUnixNano),
				ParentSpanID: parentSpan.ParentSpanID,
			},
			Children:   childSpans,
			ChildCount: len(childSpans),
			ComputedAt: time.Now().UTC(),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
}

// parseTraceAndSpanID extracts trace ID and span ID from URL path
// Expected format: /v1/traces/{traceId}/spans/{spanId}/children
func parseTraceAndSpanID(path string) (traceID string, spanID string, ok bool) {
	const prefix = "/v1/traces/"
	if !strings.HasPrefix(path, prefix) {
		return "", "", false
	}
	trimmed := strings.TrimPrefix(path, prefix)
	parts := strings.Split(trimmed, "/")
	// Expected: [traceId, "spans", spanId, "children"]
	if len(parts) != 4 || parts[0] == "" || parts[1] != "spans" || parts[2] == "" || parts[3] != "children" {
		return "", "", false
	}
	
	// The span ID is already URL decoded by the HTTP server
	return parts[0], parts[2], true
}

// findSpanByID finds a span by its ID
func findSpanByID(spans []tempo.SpanData, spanID string) (tempo.SpanData, bool) {
	for _, span := range spans {
		if span.SpanID == spanID {
			return span, true
		}
	}
	return tempo.SpanData{}, false
}

// findChildSpans finds all child spans of a given parent span ID
func findChildSpans(spans []tempo.SpanData, parentSpanID string) []domain.SpanSummary {
	var children []domain.SpanSummary
	for _, span := range spans {
		if span.ParentSpanID == parentSpanID {
			children = append(children, domain.SpanSummary{
				SpanID:       span.SpanID,
				Name:         span.Name,
				Service:      span.ServiceName,
				DurationMs:   calculateDuration(span.StartTimeUnixNano, span.EndTimeUnixNano),
				StartTime:    parseTime(span.StartTimeUnixNano),
				EndTime:      parseTime(span.EndTimeUnixNano),
				ParentSpanID: span.ParentSpanID,
			})
		}
	}
	return children
}

// calculateDuration calculates duration in milliseconds from nano timestamps
func calculateDuration(startNano, endNano string) int64 {
	start, err := strconv.ParseInt(startNano, 10, 64)
	if err != nil {
		return 0
	}
	end, err := strconv.ParseInt(endNano, 10, 64)
	if err != nil || end <= start {
		return 0
	}
	return (end - start) / int64(time.Millisecond)
}

// parseTime converts nano timestamp string to time.Time
func parseTime(nanoStr string) time.Time {
	nano, err := strconv.ParseInt(nanoStr, 10, 64)
	if err != nil {
		return time.Time{}
	}
	return time.Unix(0, nano).UTC()
}
