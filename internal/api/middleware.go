package api

import (
    "context"
    "log"
    "math/rand"
    "net/http"
    "time"
)

type ctxKey string

const requestIDKey ctxKey = "reqID"

func loggingMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        next.ServeHTTP(w, r)
        dur := time.Since(start)
        rid := RequestIDFromContext(r.Context())
        log.Printf("%s %s %dms rid=%s", r.Method, r.URL.Path, dur.Milliseconds(), rid)
    })
}

func requestIDMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        rid := r.Header.Get("X-Request-ID")
        if rid == "" {
            rid = randomID()
        }
        ctx := context.WithValue(r.Context(), requestIDKey, rid)
        w.Header().Set("X-Request-ID", rid)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}

func recoverMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        defer func() {
            if rec := recover(); rec != nil {
                log.Printf("panic: %v", rec)
                http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
            }
        }()
        next.ServeHTTP(w, r)
    })
}

func RequestIDFromContext(ctx context.Context) string {
    if v, ok := ctx.Value(requestIDKey).(string); ok {
        return v
    }
    return ""
}

func randomID() string {
    const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
    b := make([]byte, 16)
    for i := range b {
        b[i] = letters[rand.Intn(len(letters))]
    }
    return string(b)
}

