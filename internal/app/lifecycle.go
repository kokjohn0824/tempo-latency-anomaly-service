package app

import (
    "context"
    "errors"
    "log"
    "net/http"
    "time"
)

// Run starts background jobs and the HTTP server, and blocks until the context
// is cancelled. It performs a graceful shutdown on exit.
func (a *App) Run(ctx context.Context) error {
    if a == nil || a.HTTPServer == nil {
        return errors.New("app not initialized")
    }

    ctx, cancel := context.WithCancel(ctx)
    defer cancel()
    defer a.cleanup()

    // Start background jobs
    go a.TempoPoller.Run(ctx)
    go a.BaselineJob.Run(ctx)

    // Start HTTP server
    srvErr := make(chan error, 1)
    go func() {
        log.Printf("http server listening on %s", a.HTTPServer.Addr)
        if err := a.HTTPServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            srvErr <- err
        } else {
            srvErr <- nil
        }
    }()

    // Wait for stop signal or server error
    select {
    case <-ctx.Done():
        // proceed to shutdown
    case err := <-srvErr:
        if err != nil {
            log.Printf("http server error: %v", err)
            // still attempt graceful shutdown
        }
    }

    // Graceful shutdown
    shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancelShutdown()
    if err := a.HTTPServer.Shutdown(shutdownCtx); err != nil {
        log.Printf("http server shutdown error: %v", err)
    }
    return nil
}

func (a *App) cleanup() {
    if a.Store != nil {
        if err := a.Store.Close(); err != nil {
            log.Printf("store close error: %v", err)
        }
    }
}

