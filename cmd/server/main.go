package main

import (
    "context"
    "flag"
    "log"
    "os"
    "os/signal"
    "syscall"

    "github.com/alexchang/tempo-latency-anomaly-service/internal/app"
    "github.com/alexchang/tempo-latency-anomaly-service/internal/config"
)

// @title Tempo Latency Anomaly Detection Service API
// @version 1.0
// @description Time-Aware API Latency Anomaly Detection Service based on Grafana Tempo traces
// @description
// @description This service provides real-time latency anomaly detection using statistical methods (P50, P95, MAD).
// @description It automatically ingests traces from Tempo and maintains time-bucketed baselines per (service, endpoint, hour, dayType).

// @contact.name API Support
// @contact.email support@example.com

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8080
// @BasePath /

// @schemes http https

// @tag.name Health
// @tag.description Health check endpoints

// @tag.name Anomaly Detection
// @tag.description Anomaly detection and evaluation endpoints

// @tag.name Baseline
// @tag.description Baseline statistics query endpoints

func main() {
    // Config path via flag or env CONFIG_FILE
    var cfgPath string
    flag.StringVar(&cfgPath, "config", os.Getenv("CONFIG_FILE"), "path to config yaml")
    flag.Parse()

    cfg, err := config.Load(cfgPath)
    if err != nil {
        log.Fatalf("load config: %v", err)
    }

    a, err := app.New(cfg)
    if err != nil {
        log.Fatalf("init app: %v", err)
    }

    ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
    defer stop()

    if err := a.Run(ctx); err != nil {
        log.Fatalf("run: %v", err)
    }
}

