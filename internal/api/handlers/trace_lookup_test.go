package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alexchang/tempo-latency-anomaly-service/internal/config"
	"github.com/alexchang/tempo-latency-anomaly-service/internal/domain"
	"github.com/alexchang/tempo-latency-anomaly-service/internal/tempo"
	"github.com/stretchr/testify/assert"
)

func TestTraceLookup_OK(t *testing.T) {
	t.Parallel()

	service := "api-gateway"
	endpoint := "GET /api/users"
	start := int64(1736928000)
	end := int64(1736931600)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tempo.TempoResponse{
			Traces: []tempo.TraceData{
				{
					TraceID:           "trace-1",
					RootServiceName:   service,
					RootTraceName:     endpoint,
					StartTimeUnixNano: "1736928000000000000",
					DurationMs:        250,
				},
			},
		})
	}))
	t.Cleanup(srv.Close)

	client := tempo.NewClient(config.TempoConfig{URL: srv.URL})
	req := httptest.NewRequest(http.MethodGet, "/v1/traces?service=api-gateway&endpoint=GET%20%2Fapi%2Fusers&start=1736928000&end=1736931600&limit=10", nil)
	rr := httptest.NewRecorder()

	TraceLookup(client).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var resp domain.TraceLookupResponse
	err := json.NewDecoder(rr.Body).Decode(&resp)
	if assert.NoError(t, err) {
		assert.Equal(t, service, resp.Service)
		assert.Equal(t, endpoint, resp.Endpoint)
		assert.Equal(t, start, resp.Start)
		assert.Equal(t, end, resp.End)
		assert.Equal(t, 1, resp.Count)
		if assert.Len(t, resp.Traces, 1) {
			assert.Equal(t, "trace-1", resp.Traces[0].TraceID)
			assert.Equal(t, endpoint, resp.Traces[0].RootTraceName)
		}
	}
}

func TestTraceLookup_MissingParams(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tempo.TempoResponse{Traces: []tempo.TraceData{}})
	}))
	t.Cleanup(srv.Close)

	client := tempo.NewClient(config.TempoConfig{URL: srv.URL})
	req := httptest.NewRequest(http.MethodGet, "/v1/traces?service=api-gateway", nil)
	rr := httptest.NewRecorder()

	TraceLookup(client).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}
