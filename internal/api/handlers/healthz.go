package handlers

import (
    "encoding/json"
    "net/http"
)

// HealthResponse represents the health check response
type HealthResponse struct {
    Status string `json:"status" example:"ok"`
}

// Healthz godoc
// @Summary Health check
// @Description Check if the service is running and healthy
// @Tags Health
// @Accept json
// @Produce json
// @Success 200 {object} HealthResponse
// @Router /healthz [get]
func Healthz(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
