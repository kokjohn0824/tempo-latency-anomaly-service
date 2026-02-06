package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alexchang/tempo-latency-anomaly-service/internal/config"
	"github.com/alexchang/tempo-latency-anomaly-service/internal/domain"
	"github.com/alexchang/tempo-latency-anomaly-service/internal/service"
	"github.com/alexchang/tempo-latency-anomaly-service/internal/store"
	smocks "github.com/alexchang/tempo-latency-anomaly-service/internal/store/mocks"
	"github.com/alexchang/tempo-latency-anomaly-service/internal/tempo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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

	TraceLookup(client, nil).ServeHTTP(rr, req)

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

	TraceLookup(client, nil).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func TestTraceLookup_AnnotatesIsAnomaly(t *testing.T) {
	t.Parallel()

	serviceName := "api-gateway"
	endpoint := "GET /api/users"
	startNano := "1736928000000000000"

	// Tempo test server returns two traces, one anomalous, one normal.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tempo.TempoResponse{
			Traces: []tempo.TraceData{
				{
					TraceID:           "trace-anom",
					RootServiceName:   serviceName,
					RootTraceName:     endpoint,
					StartTimeUnixNano: startNano,
					DurationMs:        250,
				},
				{
					TraceID:           "trace-ok",
					RootServiceName:   serviceName,
					RootTraceName:     endpoint,
					StartTimeUnixNano: startNano,
					DurationMs:        150,
				},
			},
		})
	}))
	t.Cleanup(srv.Close)

	cfg := &config.Config{
		Timezone: "Asia/Taipei",
		Stats: config.StatsConfig{
			Factor:     1.0,
			K:          0,
			MinSamples: 1,
		},
		Fallback: config.FallbackConfig{
			NearbyHoursEnabled:      false,
			DayTypeGlobalEnabled:    false,
			FullGlobalEnabled:       false,
			NearbyHoursRange:        2,
			NearbyMinSamples:        20,
			DayTypeGlobalMinSamples: 50,
			FullGlobalMinSamples:    30,
		},
	}

	// Baseline threshold: max(p95*factor, p50 + k*MAD) = max(200, 100) = 200.
	b := &store.Baseline{P50: 100, P95: 200, MAD: 0, SampleCount: 100, UpdatedAt: time.Now().UTC()}

	// Compute the expected bucket/key for exact match.
	bucket, err := domain.ParseTimeBucket(fmt.Sprintf("%s", startNano), cfg.Timezone)
	if !assert.NoError(t, err) {
		return
	}
	key := domain.MakeBaselineKey(serviceName, endpoint, bucket)

	st := new(smocks.MockStore)
	st.On("GetBaseline", mock.Anything, key).Return(b, nil).Once()

	bl := service.NewBaselineLookup(st, cfg)
	checker := service.NewCheck(st, cfg, bl)

	client := tempo.NewClient(config.TempoConfig{URL: srv.URL})
	req := httptest.NewRequest(http.MethodGet, "/v1/traces?service=api-gateway&endpoint=GET%20%2Fapi%2Fusers&start=1736928000&end=1736931600&limit=10", nil)
	rr := httptest.NewRecorder()

	TraceLookup(client, checker).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var resp domain.TraceLookupResponse
	err = json.NewDecoder(rr.Body).Decode(&resp)
	if assert.NoError(t, err) && assert.Len(t, resp.Traces, 2) {
		assert.Equal(t, "trace-anom", resp.Traces[0].TraceID)
		assert.True(t, resp.Traces[0].IsAnomaly)

		assert.Equal(t, "trace-ok", resp.Traces[1].TraceID)
		assert.False(t, resp.Traces[1].IsAnomaly)
	}

	st.AssertExpectations(t)
}

func TestTraceLookupAnomalies_FiltersToOnlyAnomalies(t *testing.T) {
	t.Parallel()

	serviceName := "api-gateway"
	endpoint := "GET /api/users"
	startNano := "1736928000000000000"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tempo.TempoResponse{
			Traces: []tempo.TraceData{
				{
					TraceID:           "trace-anom",
					RootServiceName:   serviceName,
					RootTraceName:     endpoint,
					StartTimeUnixNano: startNano,
					DurationMs:        250,
				},
				{
					TraceID:           "trace-ok",
					RootServiceName:   serviceName,
					RootTraceName:     endpoint,
					StartTimeUnixNano: startNano,
					DurationMs:        150,
				},
			},
		})
	}))
	t.Cleanup(srv.Close)

	cfg := &config.Config{
		Timezone: "Asia/Taipei",
		Stats: config.StatsConfig{
			Factor:     1.0,
			K:          0,
			MinSamples: 1,
		},
		Fallback: config.FallbackConfig{
			NearbyHoursEnabled:   false,
			DayTypeGlobalEnabled: false,
			FullGlobalEnabled:    false,
		},
	}

	b := &store.Baseline{P50: 100, P95: 200, MAD: 0, SampleCount: 100, UpdatedAt: time.Now().UTC()}
	bucket, err := domain.ParseTimeBucket(fmt.Sprintf("%s", startNano), cfg.Timezone)
	if !assert.NoError(t, err) {
		return
	}
	key := domain.MakeBaselineKey(serviceName, endpoint, bucket)

	st := new(smocks.MockStore)
	st.On("GetBaseline", mock.Anything, key).Return(b, nil).Once()

	bl := service.NewBaselineLookup(st, cfg)
	checker := service.NewCheck(st, cfg, bl)

	client := tempo.NewClient(config.TempoConfig{URL: srv.URL})
	req := httptest.NewRequest(http.MethodGet, "/v1/traces/anomalies?service=api-gateway&endpoint=GET%20%2Fapi%2Fusers&start=1736928000&end=1736931600&limit=10", nil)
	rr := httptest.NewRecorder()

	TraceLookupAnomalies(client, checker).ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)

	var resp domain.TraceLookupResponse
	err = json.NewDecoder(rr.Body).Decode(&resp)
	if assert.NoError(t, err) {
		assert.Equal(t, 1, resp.Count)
		if assert.Len(t, resp.Traces, 1) {
			assert.Equal(t, "trace-anom", resp.Traces[0].TraceID)
			assert.True(t, resp.Traces[0].IsAnomaly)
		}
	}

	st.AssertExpectations(t)
}
