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

// TraceLongestSpan godoc
// @Summary Get longest span in a trace
// @Description Retrieve the longest span detail for a given trace ID
// @Tags Traces
// @Accept json
// @Produce json
// @Param traceId path string true "Trace ID" example("abc123def456")
// @Success 200 {object} domain.LongestSpanResponse
// @Failure 400 {object} domain.ErrorResponse "Invalid trace ID"
// @Failure 404 {object} domain.ErrorResponse "Trace not found"
// @Failure 422 {object} domain.ErrorResponse "Trace has no spans"
// @Failure 502 {object} domain.ErrorResponse "Tempo error"
// @Failure 504 {object} domain.ErrorResponse "Tempo timeout"
// @Failure 503 {object} domain.ErrorResponse "Tempo not available"
// @Router /v1/traces/{traceId}/longest-span [get]
func TraceLongestSpan(client *tempo.Client) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if client == nil {
			writeError(w, http.StatusServiceUnavailable, "tempo_unavailable", "tempo client not available", nil)
			return
		}

		traceID, ok := parseTraceID(r.URL.Path)
		if !ok {
			writeError(w, http.StatusBadRequest, "invalid_trace_id", "traceId must be provided", map[string]any{"traceId": ""})
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

		longestSpan, ok := selectLongestSpan(spans)
		if !ok {
			writeError(w, http.StatusUnprocessableEntity, "trace_empty", "trace has no spans", map[string]any{"traceId": traceID})
			return
		}

		resp := domain.LongestSpanResponse{
			TraceID:     traceID,
			LongestSpan: longestSpan,
			Source:      "tempo",
			ComputedAt:  time.Now().UTC(),
		}
		json.NewEncoder(w).Encode(resp)
	})
}

func parseTraceID(path string) (string, bool) {
	const prefix = "/v1/traces/"
	if !strings.HasPrefix(path, prefix) {
		return "", false
	}
	trimmed := strings.TrimPrefix(path, prefix)
	parts := strings.Split(trimmed, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] != "longest-span" {
		return "", false
	}
	return parts[0], true
}

func selectLongestSpan(spans []tempo.SpanData) (domain.SpanSummary, bool) {
	var (
		longest domain.SpanSummary
		found   bool
	)
	for _, span := range spans {
		start, err := strconv.ParseInt(span.StartTimeUnixNano, 10, 64)
		if err != nil {
			continue
		}
		end, err := strconv.ParseInt(span.EndTimeUnixNano, 10, 64)
		if err != nil || end <= start {
			continue
		}
		durationMs := (end - start) / int64(time.Millisecond)
		if !found || durationMs > longest.DurationMs {
			longest = domain.SpanSummary{
				SpanID:       span.SpanID,
				Name:         span.Name,
				Service:      span.ServiceName,
				DurationMs:   durationMs,
				StartTime:    time.Unix(0, start).UTC(),
				EndTime:      time.Unix(0, end).UTC(),
				ParentSpanID: span.ParentSpanID,
			}
			found = true
		}
	}
	return longest, found
}

func writeError(w http.ResponseWriter, status int, code, message string, details map[string]any) {
	resp := domain.ErrorResponse{
		Error: domain.ErrorDetail{
			Code:    code,
			Message: message,
			Details: details,
		},
	}
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(resp)
}
