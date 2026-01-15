package handlers

import (
    "encoding/json"
    "net/http"
    "strconv"

    "github.com/alexchang/tempo-latency-anomaly-service/internal/domain"
    "github.com/alexchang/tempo-latency-anomaly-service/internal/store"
)

// Baseline godoc
// @Summary Get baseline statistics
// @Description Retrieve baseline statistics (P50, P95, MAD) for a specific service, endpoint, hour, and day type
// @Description Used for debugging and monitoring baseline data
// @Tags Baseline
// @Accept json
// @Produce json
// @Param service query string true "Service name" example("twdiw-customer-service-prod")
// @Param endpoint query string true "Endpoint name" example("GET /actuator/health")
// @Param hour query int true "Hour of day (0-23)" example(16)
// @Param dayType query string true "Day type (weekday or weekend)" example("weekday")
// @Success 200 {object} store.Baseline
// @Failure 400 {object} map[string]string "Invalid parameters"
// @Failure 404 {object} map[string]string "Baseline not found"
// @Failure 500 {object} map[string]string "Internal server error"
// @Failure 503 {object} map[string]string "Store not available"
// @Router /v1/baseline [get]
func Baseline(st store.Store) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if st == nil {
            http.Error(w, "store not available", http.StatusServiceUnavailable)
            return
        }
        q := r.URL.Query()
        service := q.Get("service")
        endpoint := q.Get("endpoint")
        hourStr := q.Get("hour")
        dayType := q.Get("dayType")
        if service == "" || endpoint == "" || hourStr == "" || dayType == "" {
            http.Error(w, "missing query params: service, endpoint, hour, dayType", http.StatusBadRequest)
            return
        }
        hour, err := strconv.Atoi(hourStr)
        if err != nil || hour < 0 || hour > 23 {
            http.Error(w, "invalid hour", http.StatusBadRequest)
            return
        }
        key := domain.MakeBaselineKey(service, endpoint, domain.TimeBucket{Hour: hour, DayType: dayType})
        b, err := st.GetBaseline(r.Context(), key)
        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }
        if b == nil {
            w.WriteHeader(http.StatusNotFound)
            json.NewEncoder(w).Encode(map[string]string{"error": "not found"})
            return
        }
        json.NewEncoder(w).Encode(b)
    })
}
