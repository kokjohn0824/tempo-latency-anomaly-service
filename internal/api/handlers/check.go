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
// @Description Fallback: Automatically applies a 5-level strategy to select a baseline
// @Description (exact → nearby → daytype → global → unavailable) when the exact bucket has insufficient data.
// @Description Response fields:
// @Description - baselineSource: which source was used (exact/nearby/daytype/global/unavailable)
// @Description - fallbackLevel: numeric level 1-5 mapping to the source (1=exact, 2=nearby, 3=daytype, 4=global, 5=unavailable)
// @Description - sourceDetails: human-readable details about the selected source (e.g. "exact match: 17|weekday")
// @Description - cannotDetermine: true if no sufficient baseline exists to decide
// @Tags Anomaly Detection
// @Accept json
// @Produce json
// @Param request body domain.AnomalyCheckRequest true "Anomaly check request"
// @Success 200 {object} domain.AnomalyCheckResponse "Includes baselineSource, fallbackLevel, sourceDetails, cannotDetermine"
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
