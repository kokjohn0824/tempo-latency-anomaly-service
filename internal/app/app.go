package app

import (
    "fmt"
    "net/http"

    "github.com/alexchang/tempo-latency-anomaly-service/internal/api"
    "github.com/alexchang/tempo-latency-anomaly-service/internal/config"
    "github.com/alexchang/tempo-latency-anomaly-service/internal/jobs"
    "github.com/alexchang/tempo-latency-anomaly-service/internal/observability"
    "github.com/alexchang/tempo-latency-anomaly-service/internal/service"
    storepkg "github.com/alexchang/tempo-latency-anomaly-service/internal/store"
    redispkg "github.com/alexchang/tempo-latency-anomaly-service/internal/store/redis"
    "github.com/alexchang/tempo-latency-anomaly-service/internal/tempo"
)

// App wires all components together and holds process-level state.
type App struct {
    Cfg          *config.Config
    Store        storepkg.Store
    Tempo        *tempo.Client
    Ingest       *service.Ingest
    Baseline     *service.Baseline
    Check        *service.Check
    ListAvail    *service.ListAvailable
    TempoPoller  *jobs.TempoPoller
    BaselineJob  *jobs.BaselineRecompute
    HTTPServer   *http.Server
}

// New constructs the full application from configuration.
func New(cfg *config.Config) (*App, error) {
    if cfg == nil {
        return nil, fmt.Errorf("nil config")
    }

    // Logger setup (stdlog) and optional metrics
    observability.SetupLogger()

    // Storage
    st, err := redispkg.New(cfg.Redis)
    if err != nil {
        return nil, fmt.Errorf("init redis: %w", err)
    }

    // External clients
    tempoClient := tempo.NewClient(cfg.Tempo)

    // Services
    ingestSvc := service.NewIngest(st, cfg)
    baselineSvc := service.NewBaseline(st, cfg)
    baselineLookup := service.NewBaselineLookup(st, cfg)
    checkSvc := service.NewCheck(st, cfg, baselineLookup)
    listAvailSvc := service.NewListAvailable(st, cfg.Stats.MinSamples)

    // Jobs
    poller := jobs.NewTempoPoller(cfg, tempoClient, ingestSvc)
    recompute := jobs.NewBaselineRecompute(cfg, baselineSvc, st, 100)

    // HTTP router and server
    apiHandler := api.NewRouter(checkSvc, listAvailSvc, st)

    mux := http.NewServeMux()
    // Mount API under root
    mux.Handle("/", apiHandler)
    // Expose metrics endpoint
    mux.HandleFunc("/metrics", observability.MetricsHandler)

    srv := &http.Server{
        Addr:              fmt.Sprintf(":%d", cfg.HTTP.Port),
        Handler:           mux,
        ReadHeaderTimeout: cfg.HTTP.Timeout,
        ReadTimeout:       cfg.HTTP.Timeout,
        WriteTimeout:      cfg.HTTP.Timeout,
        IdleTimeout:       cfg.HTTP.Timeout,
    }

    return &App{
        Cfg:         cfg,
        Store:       st,
        Tempo:       tempoClient,
        Ingest:      ingestSvc,
        Baseline:    baselineSvc,
        Check:       checkSvc,
        ListAvail:   listAvailSvc,
        TempoPoller: poller,
        BaselineJob: recompute,
        HTTPServer:  srv,
    }, nil
}
