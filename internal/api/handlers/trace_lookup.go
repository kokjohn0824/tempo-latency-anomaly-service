package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/alexchang/tempo-latency-anomaly-service/internal/domain"
	"github.com/alexchang/tempo-latency-anomaly-service/internal/tempo"
)

// TraceLookup godoc
// @Summary Look up traces
// @Description Retrieve trace IDs and metadata for a service/endpoint within a time range
// @Tags Traces
// @Accept json
// @Produce json
// @Param service query string true "Service name" example("api-gateway")
// @Param endpoint query string true "Endpoint name" example("GET /api/users")
// @Param start query int true "Start time (unix seconds)" example(1736928000)
// @Param end query int true "End time (unix seconds)" example(1736931600)
// @Param limit query int false "Max traces to return" example(200)
// @Success 200 {object} domain.TraceLookupResponse
// @Failure 400 {object} map[string]string "Invalid parameters"
// @Failure 500 {object} map[string]string "Tempo error"
// @Failure 503 {object} map[string]string "Tempo not available"
// @Router /v1/traces [get]
func TraceLookup(client *tempo.Client) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if client == nil {
			http.Error(w, "tempo client not available", http.StatusServiceUnavailable)
			return
		}

		q := r.URL.Query()
		service := q.Get("service")
		endpoint := q.Get("endpoint")
		startStr := q.Get("start")
		endStr := q.Get("end")
		limitStr := q.Get("limit")
		if service == "" || endpoint == "" || startStr == "" || endStr == "" {
			http.Error(w, "missing query params: service, endpoint, start, end", http.StatusBadRequest)
			return
		}

		start, err := strconv.ParseInt(startStr, 10, 64)
		if err != nil || start <= 0 {
			http.Error(w, "invalid start", http.StatusBadRequest)
			return
		}
		end, err := strconv.ParseInt(endStr, 10, 64)
		if err != nil || end <= 0 {
			http.Error(w, "invalid end", http.StatusBadRequest)
			return
		}
		if end < start {
			http.Error(w, "end must be >= start", http.StatusBadRequest)
			return
		}

		limit := 500
		if limitStr != "" {
			parsed, err := strconv.Atoi(limitStr)
			if err != nil || parsed <= 0 {
				http.Error(w, "invalid limit", http.StatusBadRequest)
				return
			}
			limit = parsed
		}

		traces, err := client.SearchTraces(r.Context(), service, endpoint, start, end, limit)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		resp := domain.TraceLookupResponse{
			Service:  service,
			Endpoint: endpoint,
			Start:    start,
			End:      end,
			Count:    len(traces),
			Traces:   traces,
		}
		json.NewEncoder(w).Encode(resp)
	})
}
