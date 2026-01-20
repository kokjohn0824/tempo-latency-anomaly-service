package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alexchang/tempo-latency-anomaly-service/internal/config"
	"github.com/alexchang/tempo-latency-anomaly-service/internal/domain"
	"github.com/alexchang/tempo-latency-anomaly-service/internal/tempo"
	"github.com/stretchr/testify/assert"
)

func TestTraceLongestSpan_OK(t *testing.T) {
	t.Parallel()

	traceID := "abc123"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/traces/"+traceID, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tempo.TraceByIDResponse{
			ResourceSpans: []tempo.ResourceSpan{
				{
					Resource: tempo.Resource{
						Attributes: []tempo.KeyValue{
							{Key: "service.name", Value: tempo.AttributeValue{StringValue: "orders"}},
						},
					},
					ScopeSpans: []tempo.ScopeSpan{
						{
							Spans: []tempo.Span{
								{
									TraceID:           traceID,
									SpanID:            "span-1",
									Name:              "root",
									StartTimeUnixNano: "1000000000",
									EndTimeUnixNano:   "1500000000",
								},
								{
									TraceID:           traceID,
									SpanID:            "span-2",
									ParentSpanID:      "span-1",
									Name:              "db.query",
									StartTimeUnixNano: "1000000000",
									EndTimeUnixNano:   "2500000000",
								},
							},
						},
					},
				},
			},
		})
	}))
	t.Cleanup(srv.Close)

	client := tempo.NewClient(config.TempoConfig{URL: srv.URL})
	req := httptest.NewRequest(http.MethodGet, "/v1/traces/"+traceID+"/longest-span", nil)
	rr := httptest.NewRecorder()

	TraceLongestSpan(client).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var resp domain.LongestSpanResponse
	err := json.NewDecoder(rr.Body).Decode(&resp)
	if assert.NoError(t, err) {
		assert.Equal(t, traceID, resp.TraceID)
		assert.Equal(t, "span-2", resp.LongestSpan.SpanID)
		assert.Equal(t, "db.query", resp.LongestSpan.Name)
		assert.Equal(t, "orders", resp.LongestSpan.Service)
		assert.Equal(t, int64(1500), resp.LongestSpan.DurationMs)
		assert.Equal(t, time.Unix(0, 1000000000).UTC(), resp.LongestSpan.StartTime)
		assert.Equal(t, time.Unix(0, 2500000000).UTC(), resp.LongestSpan.EndTime)
	}
}

func TestTraceLongestSpan_EmptyTrace(t *testing.T) {
	t.Parallel()

	traceID := "empty-trace"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tempo.TraceByIDResponse{})
	}))
	t.Cleanup(srv.Close)

	client := tempo.NewClient(config.TempoConfig{URL: srv.URL})
	req := httptest.NewRequest(http.MethodGet, "/v1/traces/"+traceID+"/longest-span", nil)
	rr := httptest.NewRecorder()

	TraceLongestSpan(client).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnprocessableEntity, rr.Code)
}
