package tempo

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alexchang/tempo-latency-anomaly-service/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestClient_SearchTraces(t *testing.T) {
	t.Parallel()

	service := "api-gateway"
	endpoint := "GET /api/users?foo=bar"
	start := int64(1736928000)
	end := int64(1736931600)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/search", r.URL.Path)

		q := r.URL.Query()
		assert.Equal(t, "1736928000", q.Get("start"))
		assert.Equal(t, "1736931600", q.Get("end"))
		assert.Equal(t, "200", q.Get("limit"))
		assert.Equal(t, "service.name=api-gateway name=\"GET /api/users?foo=bar\"", q.Get("tags"))

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(TempoResponse{
			Traces: []TraceData{
				{
					TraceID:           "trace-1",
					RootServiceName:   service,
					RootTraceName:     endpoint,
					StartTimeUnixNano: "1736928000000000000",
					DurationMs:        321,
				},
			},
		})
	}))
	t.Cleanup(srv.Close)

	client := NewClient(config.TempoConfig{URL: srv.URL})
	traces, err := client.SearchTraces(context.Background(), service, endpoint, start, end, 200)
	if assert.NoError(t, err) {
		if assert.Len(t, traces, 1) {
			assert.Equal(t, "trace-1", traces[0].TraceID)
			assert.Equal(t, service, traces[0].RootServiceName)
			assert.Equal(t, endpoint, traces[0].RootTraceName)
			assert.Equal(t, "1736928000000000000", traces[0].StartTimeUnixNano)
			assert.Equal(t, int64(321), traces[0].DurationMs)
		}
	}
}
