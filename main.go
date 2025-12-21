package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/joho/godotenv"
	"github.com/mykalmachon/railway-prometheus-exporter/cache"
	"github.com/mykalmachon/railway-prometheus-exporter/collector"
	"github.com/mykalmachon/railway-prometheus-exporter/config"
	"github.com/mykalmachon/railway-prometheus-exporter/logger"
	"github.com/mykalmachon/railway-prometheus-exporter/railway"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// IndexResponse is the JSON response for the home page
type IndexResponse struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Version     string            `json:"version"`
	Endpoints   map[string]string `json:"endpoints"`
	Config      IndexConfig       `json:"config"`
}

// IndexConfig contains configuration information for the home page
type IndexConfig struct {
	TargetCount       int  `json:"target_count"`
	CacheEnabled      bool `json:"cache_enabled"`
	MetricsCacheTTL   int  `json:"metrics_cache_ttl_seconds"`
	DiscoveryCacheTTL int  `json:"discovery_cache_ttl_seconds"`
}

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
		"target_count", len(cfg.Targets),
	)

	// Log configured targets
	for i, target := range cfg.Targets {
		logger.Stdout.Info("Configured target",
			"index", i+1,
			"project_id", target.ProjectID,
			"environment_id", target.EnvironmentID,
		)
	}

	// Create cache
	metricsCache := cache.NewCache()

	// Create Railway API client with cache
	railwayClient := railway.NewClientWithCache(
		cfg.RailwayApiKey,
		metricsCache,
		railway.CacheTTLConfig{
			MetricsTTL:   time.Duration(cfg.MetricsCacheTTL) * time.Second,
			DiscoveryTTL: time.Duration(cfg.DiscoveryCacheTTL) * time.Second,
			EnableCache:  cfg.EnableCache,
		},
	)

	logger.Stdout.Info("Cache configuration",
		"enabled", cfg.EnableCache,
		"metrics_ttl_seconds", cfg.MetricsCacheTTL,
		"discovery_ttl_seconds", cfg.DiscoveryCacheTTL,
	)

	// Warm up cache if enabled
	if cfg.EnableCache {
		warmupCache(railwayClient, cfg.Targets)
	}

	// Create and register Prometheus collector
	railwayCollector := collector.NewRailwayCollector(
		railwayClient,
		cfg.Targets,
	)
	prometheus.MustRegister(railwayCollector)

	// Setup HTTP routes
	http.Handle("/metrics", promhttp.Handler())
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/", indexHandler(cfg))

	// Start HTTP server
	addr := ":" + cfg.Port
	logger.Stdout.Info("HTTP server listening", "address", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		logger.Stderr.Error("HTTP server failed", "error", err)
		os.Exit(1)
	}
}

// warmupCache pre-populates the cache with initial data
func warmupCache(client *railway.Client, targets []config.EnvironmentTarget) {
	logger.Stdout.Info("Warming up cache...", "target_count", len(targets))

	var wg sync.WaitGroup
	for _, target := range targets {
		wg.Add(1)
		go func(t config.EnvironmentTarget) {
			defer wg.Done()

			// Pre-fetch environment info and services
			envInfo, err := client.GetEnvironmentInfo(t.EnvironmentID)
			if err != nil {
				logger.Stderr.Error("Failed to warm up environment info",
					"environment_id", t.EnvironmentID,
					"error", err)
				return
			}

			services, err := client.GetProjectServices(t.ProjectID)
			if err != nil {
				logger.Stderr.Error("Failed to warm up project services",
					"project_id", t.ProjectID,
					"error", err)
				return
			}

			// Pre-fetch service info and metrics for each service
			for _, svc := range services {
				client.GetServiceInfo(svc.Node.ID)
				client.GetMetrics(envInfo.ID, svc.Node.ID)
			}
		}(target)
	}

	wg.Wait()
	logger.Stdout.Info("Cache warmup complete")
}

// healthHandler provides a simple health check endpoint
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "OK")
}

// indexHandler provides basic information about the exporter in JSON format
func indexHandler(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		response := IndexResponse{
			Name:        "Railway Prometheus Exporter",
			Description: "A Prometheus exporter for Railway service metrics",
			Version:     "1.0.0",
			Endpoints: map[string]string{
				"metrics": "/metrics",
				"health":  "/health",
			},
			Config: IndexConfig{
				TargetCount:       len(cfg.Targets),
				CacheEnabled:      cfg.EnableCache,
				MetricsCacheTTL:   cfg.MetricsCacheTTL,
				DiscoveryCacheTTL: cfg.DiscoveryCacheTTL,
			},
		}

		json.NewEncoder(w).Encode(response)
	}
}
