package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/mykalmachon/railway-prometheus-exporter/collector"
	"github.com/mykalmachon/railway-prometheus-exporter/config"
	"github.com/mykalmachon/railway-prometheus-exporter/logger"
	"github.com/mykalmachon/railway-prometheus-exporter/railway"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	// Load .env file (ignore error in production where env vars are set directly)
	_ = godotenv.Load()

	// Parse configuration
	cfg, err := config.GetConfig()
	if err != nil {
		logger.Stderr.Error("Failed to parse configuration", "error", err)
		os.Exit(1)
	}

	logger.Stdout.Info("Starting Railway Prometheus Exporter",
		"port", cfg.Port,
		"environment_id", cfg.EnvironmentId,
		"project_count", len(cfg.ProjectIds),
	)

	// Create Railway API client
	railwayClient := railway.NewClient(cfg.RailwayApiKey)

	// Create and register Prometheus collector
	railwayCollector := collector.NewRailwayCollector(
		railwayClient,
		cfg.EnvironmentId,
		cfg.ProjectIds,
	)
	prometheus.MustRegister(railwayCollector)

	// Setup HTTP routes
	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/", indexHandler)

	// Start HTTP server
	addr := ":" + cfg.Port
	logger.Stdout.Info("HTTP server listening", "address", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		logger.Stderr.Error("HTTP server failed", "error", err)
		os.Exit(1)
	}
}

// healthHandler provides a simple health check endpoint
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "OK")
}

// indexHandler provides basic information about the exporter
func indexHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, `<!DOCTYPE html>
<html>
<head>
	<title>Railway Prometheus Exporter</title>
</head>
<body>
	<h1>Railway Prometheus Exporter</h1>
	<p>A Prometheus exporter for Railway service metrics.</p>
	<ul>
		<li><a href="/metrics">Metrics</a></li>
		<li><a href="/health">Health</a></li>
	</ul>
</body>
</html>`)
}
