package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/alexchang/tempo-latency-anomaly-service/internal/domain"
	"github.com/alexchang/tempo-latency-anomaly-service/internal/service"
	"github.com/alexchang/tempo-latency-anomaly-service/internal/tempo"
)

// TraceChildSpanAnomaliesRequest represents the request body for child span anomalies.
type TraceChildSpanAnomaliesRequest struct {
	TraceID      string `json:"traceId" example:"abc123def456"`
	ParentSpanID string `json:"parentSpanId,omitempty" example:"xyz789"`
}

// TraceChildSpanAnomalies godoc
// @Summary Get child span anomalies for a parent span
// @Description Evaluate anomalies for direct child spans under a parent span in a trace
// @Tags Traces
// @Accept json
// @Produce json
// @Param request body TraceChildSpanAnomaliesRequest true "Trace and optional parent span ID"
// @Success 200 {object} domain.ChildSpanAnomaliesResponse
// @Failure 400 {object} domain.ErrorResponse "Invalid request"
// @Failure 404 {object} domain.ErrorResponse "Trace or span not found"
// @Failure 422 {object} domain.ErrorResponse "Trace has no spans"
// @Failure 502 {object} domain.ErrorResponse "Tempo error"
// @Failure 504 {object} domain.ErrorResponse "Tempo timeout"
// @Failure 503 {object} domain.ErrorResponse "Tempo not available"
// @Router /v1/traces/child-span-anomalies [post]
func TraceChildSpanAnomalies(client *tempo.Client, checker *service.SpanCheck) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if client == nil || checker == nil {
			writeError(w, http.StatusServiceUnavailable, "tempo_unavailable", "tempo client not available", nil)
			return
		}

		var req TraceChildSpanAnomaliesRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_request", "invalid JSON body", map[string]any{"error": err.Error()})
			return
		}

		if req.TraceID == "" {
			writeError(w, http.StatusBadRequest, "invalid_parameters", "traceId is required", map[string]any{"traceId": req.TraceID})
			return
		}

		spans, err := client.GetTraceSpans(r.Context(), req.TraceID)
		if err != nil {
			if tempo.IsTimeout(err) {
				writeError(w, http.StatusGatewayTimeout, "tempo_timeout", "tempo request timed out", map[string]any{"traceId": req.TraceID})
				return
			}
			var respErr tempo.ResponseError
			if errors.As(err, &respErr) {
				switch respErr.StatusCode {
				case http.StatusNotFound:
					writeError(w, http.StatusNotFound, "trace_not_found", "trace not found in Tempo", map[string]any{"traceId": req.TraceID})
					return
				case http.StatusTooManyRequests:
					writeError(w, http.StatusBadGateway, "tempo_error", "tempo rate limited", map[string]any{"traceId": req.TraceID, "tempoStatus": respErr.StatusCode})
					return
				default:
					if respErr.StatusCode >= 500 {
						writeError(w, http.StatusBadGateway, "tempo_error", "tempo request failed", map[string]any{"traceId": req.TraceID, "tempoStatus": respErr.StatusCode})
						return
					}
					writeError(w, http.StatusBadGateway, "tempo_error", "tempo request failed", map[string]any{"traceId": req.TraceID, "tempoStatus": respErr.StatusCode})
					return
				}
			}
			writeError(w, http.StatusBadGateway, "tempo_error", "tempo request failed", map[string]any{"traceId": req.TraceID})
			return
		}

		if len(spans) == 0 {
			writeError(w, http.StatusUnprocessableEntity, "trace_empty", "trace has no spans", map[string]any{"traceId": req.TraceID})
			return
		}

		var parentSpan tempo.SpanData
		var found bool
		if req.ParentSpanID == "" {
			parentSpan, found = findRootSpan(spans)
			if !found {
				writeError(w, http.StatusNotFound, "root_span_not_found", "root span not found in trace", map[string]any{"traceId": req.TraceID})
				return
			}
		} else {
			parentSpan, found = findSpanByID(spans, req.ParentSpanID)
			if !found {
				writeError(w, http.StatusNotFound, "span_not_found", "parent span not found in trace", map[string]any{"traceId": req.TraceID, "spanId": req.ParentSpanID})
				return
			}
		}

		childrenData := findChildSpanData(spans, parentSpan.SpanID)
		children := make([]domain.ChildSpanAnomaly, 0, len(childrenData))
		anomalyCount := 0
		for _, child := range childrenData {
			summary := buildSpanSummary(child)
			durationMs := summary.DurationMs
			startNano, err := strconv.ParseInt(child.StartTimeUnixNano, 10, 64)
			if err != nil || durationMs <= 0 {
				children = append(children, buildUnavailableChildAnomaly(summary, "invalid span timestamps"))
				continue
			}

			res, err := checker.Evaluate(r.Context(), domain.SpanAnomalyCheckRequest{
				Service:       child.ServiceName,
				SpanName:      child.Name,
				TimestampNano: startNano,
				DurationMs:    durationMs,
			})
			if err != nil {
				children = append(children, buildUnavailableChildAnomaly(summary, err.Error()))
				continue
			}

			if res.IsAnomaly {
				anomalyCount++
			}

			children = append(children, domain.ChildSpanAnomaly{
				Span:            summary,
				IsAnomaly:       res.IsAnomaly,
				CannotDetermine: res.CannotDetermine,
				Bucket:          res.Bucket,
				Baseline:        res.Baseline,
				BaselineSource:  res.BaselineSource,
				FallbackLevel:   res.FallbackLevel,
				SourceDetails:   res.SourceDetails,
				Explanation:     res.Explanation,
			})
		}

		resp := domain.ChildSpanAnomaliesResponse{
			TraceID:      req.TraceID,
			ParentSpan:   buildSpanSummary(parentSpan),
			Children:     children,
			ChildCount:   len(children),
			AnomalyCount: anomalyCount,
			ComputedAt:   time.Now().UTC(),
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
}

func buildSpanSummary(span tempo.SpanData) domain.SpanSummary {
	return domain.SpanSummary{
		SpanID:       span.SpanID,
		Name:         span.Name,
		Service:      span.ServiceName,
		DurationMs:   calculateDuration(span.StartTimeUnixNano, span.EndTimeUnixNano),
		StartTime:    parseTime(span.StartTimeUnixNano),
		EndTime:      parseTime(span.EndTimeUnixNano),
		ParentSpanID: span.ParentSpanID,
	}
}

func buildUnavailableChildAnomaly(summary domain.SpanSummary, reason string) domain.ChildSpanAnomaly {
	return domain.ChildSpanAnomaly{
		Span:            summary,
		IsAnomaly:       false,
		CannotDetermine: true,
		Bucket:          domain.TimeBucket{},
		Baseline:        nil,
		BaselineSource:  domain.SourceUnavailable,
		Explanation:     reason,
	}
}

func findRootSpan(spans []tempo.SpanData) (tempo.SpanData, bool) {
	var root tempo.SpanData
	var earliest int64
	found := false
	for _, span := range spans {
		if span.ParentSpanID != "" {
			continue
		}
		start, err := strconv.ParseInt(span.StartTimeUnixNano, 10, 64)
		if err != nil {
			if !found {
				root = span
				found = true
			}
			continue
		}
		if !found || start < earliest {
			root = span
			earliest = start
			found = true
		}
	}
	return root, found
}

func findChildSpanData(spans []tempo.SpanData, parentSpanID string) []tempo.SpanData {
	children := make([]tempo.SpanData, 0)
	for _, span := range spans {
		if span.ParentSpanID == parentSpanID {
			children = append(children, span)
		}
	}
	return children
}
