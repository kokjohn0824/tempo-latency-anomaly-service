package handlers

import (
    "encoding/json"
    "net/http"

    "github.com/alexchang/tempo-latency-anomaly-service/internal/domain"
    "github.com/alexchang/tempo-latency-anomaly-service/internal/service"
)

// Check godoc
// @Summary Check for latency anomaly
// @Description Evaluate if a given request latency is anomalous based on historical baselines
// @Description The service uses time-bucketed baselines (per hour and day type) and statistical methods (P50, P95, MAD) to detect anomalies
// @Tags Anomaly Detection
// @Accept json
// @Produce json
// @Param request body domain.AnomalyCheckRequest true "Anomaly check request"
// @Success 200 {object} domain.AnomalyCheckResponse
// @Failure 400 {object} map[string]string "Invalid JSON"
// @Failure 500 {object} map[string]string "Internal server error"
// @Failure 503 {object} map[string]string "Service not available"
// @Router /v1/anomaly/check [post]
func Check(svc *service.Check) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if svc == nil {
            http.Error(w, "service not available", http.StatusServiceUnavailable)
            return
        }
        var req domain.AnomalyCheckRequest
        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
            http.Error(w, "invalid json", http.StatusBadRequest)
            return
        }
        resp, err := svc.Evaluate(r.Context(), req)
        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(resp)
    })
}
