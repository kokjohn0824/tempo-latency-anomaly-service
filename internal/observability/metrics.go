package observability

import (
    "fmt"
    "net/http"
    "time"
)

// MetricsHandler exposes a minimal Prometheus-compatible metrics endpoint.
// This is a lightweight placeholder without external deps.
func MetricsHandler(w http.ResponseWriter, r *http.Request) {
    w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
    now := time.Now().Unix()
    // Simple up metric and current unix time gauge
    _, _ = fmt.Fprintf(w, "# HELP app_up Application up status\n")
    _, _ = fmt.Fprintf(w, "# TYPE app_up gauge\n")
    _, _ = fmt.Fprintf(w, "app_up 1\n")
    _, _ = fmt.Fprintf(w, "# HELP app_now_unixtime Current unix time\n")
    _, _ = fmt.Fprintf(w, "# TYPE app_now_unixtime gauge\n")
    _, _ = fmt.Fprintf(w, "app_now_unixtime %d\n", now)
}

