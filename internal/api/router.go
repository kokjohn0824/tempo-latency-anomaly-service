package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/alexchang/tempo-latency-anomaly-service/internal/api/handlers"
	"github.com/alexchang/tempo-latency-anomaly-service/internal/service"
	"github.com/alexchang/tempo-latency-anomaly-service/internal/store"
	"github.com/alexchang/tempo-latency-anomaly-service/internal/tempo"

	_ "github.com/alexchang/tempo-latency-anomaly-service/docs"
	httpSwagger "github.com/swaggo/http-swagger"
)

// NewRouter builds an http.Handler with routes and middleware wired.
func NewRouter(checkSvc *service.Check, listSvc *service.ListAvailable, st store.Store, tempoClient *tempo.Client) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/healthz", handlers.Healthz)

	mux.HandleFunc("/v1/anomaly/check", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		handlers.Check(checkSvc).ServeHTTP(w, r)
	})

	mux.HandleFunc("/v1/baseline", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		handlers.Baseline(st).ServeHTTP(w, r)
	})

	mux.HandleFunc("/v1/available", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		handlers.ListAvailable(listSvc).ServeHTTP(w, r)
	})

	mux.HandleFunc("/v1/traces", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		handlers.TraceLookup(tempoClient).ServeHTTP(w, r)
	})

	mux.HandleFunc("/v1/traces/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		
		// Check if it's a child spans request (POST /v1/traces/child-spans)
		if path == "/v1/traces/child-spans" {
			if r.Method != http.MethodPost {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			handlers.TraceChildSpans(tempoClient).ServeHTTP(w, r)
			return
		}
		
		// All other /v1/traces/* endpoints are GET only
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		
		// Check if it's a longest span request
		if strings.HasSuffix(path, "/longest-span") {
			handlers.TraceLongestSpan(tempoClient).ServeHTTP(w, r)
			return
		}
		
		// If no match, return 404
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "endpoint not found"})
	})

	// Swagger UI endpoint
	mux.HandleFunc("/swagger/", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))

	h := recoverMiddleware(requestIDMiddleware(loggingMiddleware(mux)))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		h.ServeHTTP(w, r)
	})
}

// writeJSON is a small helper to encode JSON with a timeout to avoid stuck writers.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(true)
	_ = enc.Encode(v)
}
