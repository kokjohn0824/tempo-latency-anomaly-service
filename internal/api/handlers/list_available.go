package handlers

import (
    "encoding/json"
    "net/http"

    "github.com/alexchang/tempo-latency-anomaly-service/internal/service"
)

// ListAvailable godoc
// @Summary List available services and endpoints
// @Description Retrieve all services and endpoints that have sufficient baseline samples for anomaly detection
// @Description Returns a list of service/endpoint pairs grouped by service, along with their available time buckets
// @Description Only includes baselines with sufficient samples (configured via min_samples)
// @Tags Available Services
// @Accept json
// @Produce json
// @Success 200 {object} domain.AvailableServicesResponse
// @Failure 500 {object} map[string]string "Internal server error"
// @Failure 503 {object} map[string]string "Service not available"
// @Router /v1/available [get]
func ListAvailable(svc *service.ListAvailable) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if svc == nil {
            http.Error(w, `{"error":"service not available"}`, http.StatusServiceUnavailable)
            return
        }

        ctx := r.Context()
        resp, err := svc.GetAvailableServices(ctx)
        if err != nil {
            http.Error(w, `{"error":"failed to retrieve available services"}`, http.StatusInternalServerError)
            return
        }

        w.Header().Set("Content-Type", "application/json")
        w.WriteHeader(http.StatusOK)
        _ = json.NewEncoder(w).Encode(resp)
    })
}
