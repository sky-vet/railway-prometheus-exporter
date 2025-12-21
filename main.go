package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
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

	// Create cache with configured max size
	metricsCache := cache.NewCacheWithMaxSize(cfg.CacheMaxSize)
	defer metricsCache.Close() // Ensure cleanup on shutdown

	// Create Railway API client with cache
	railwayClient := railway.NewClientWithCache(
		cfg.RailwayApiKey,
		metricsCache,
		railway.CacheTTLConfig{
			MetricsTTL:   time.Duration(cfg.MetricsCacheTTL) * time.Second,
			DiscoveryTTL: time.Duration(cfg.DiscoveryCacheTTL) * time.Second,
			EnableCache:  cfg.EnableCache,
		},
		railway.RetryConfig{
			MaxRetries:     cfg.MaxRetries,
			InitialBackoff: time.Duration(cfg.InitialBackoff) * time.Millisecond,
		},
	)

	logger.Stdout.Info("Cache configuration",
		"enabled", cfg.EnableCache,
		"metrics_ttl_seconds", cfg.MetricsCacheTTL,
		"discovery_ttl_seconds", cfg.DiscoveryCacheTTL,
		"max_size", cfg.CacheMaxSize,
	)

	logger.Stdout.Info("Retry configuration",
		"max_retries", cfg.MaxRetries,
		"initial_backoff_ms", cfg.InitialBackoff,
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

	// Create HTTP server with timeouts
	addr := ":" + cfg.Port
	server := &http.Server{
		Addr:         addr,
		Handler:      nil, // Use default ServeMux
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Channel to listen for OS signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	// Start server in background goroutine
	go func() {
		logger.Stdout.Info("HTTP server listening", "address", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Stderr.Error("HTTP server failed", "error", err)
			os.Exit(1)
		}
	}()

	// Wait for shutdown signal
	sig := <-sigChan
	logger.Stdout.Info("Received shutdown signal, gracefully shutting down...", "signal", sig)

	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Attempt graceful shutdown
	if err := server.Shutdown(ctx); err != nil {
		logger.Stderr.Error("Server shutdown failed", "error", err)
		os.Exit(1)
	}

	logger.Stdout.Info("Server stopped gracefully")
}

// warmupCache pre-populates the cache with initial data
func warmupCache(client *railway.Client, targets []config.EnvironmentTarget) {
	logger.Stdout.Info("Warming up cache...", "target_count", len(targets))

	var wg sync.WaitGroup
	for _, target := range targets {
		wg.Add(1)
		go func(t config.EnvironmentTarget) {
			// Panic recovery MUST be first to ensure wg.Done() always executes
			defer func() {
				if r := recover(); r != nil {
					logger.Stderr.Error("Panic recovered in cache warmup goroutine",
						"project_id", t.ProjectID,
						"environment_id", t.EnvironmentID,
						"panic", r)
				}
			}()
			defer wg.Done()

			// Pre-fetch environment info and services
			envInfo, _, err := client.GetEnvironmentInfo(t.EnvironmentID)
			if err != nil {
				logger.Stderr.Error("Failed to warm up environment info",
					"environment_id", t.EnvironmentID,
					"error", err)
				return
			}

			services, _, err := client.GetProjectServices(t.ProjectID)
			if err != nil {
				logger.Stderr.Error("Failed to warm up project services",
					"project_id", t.ProjectID,
					"error", err)
				return
			}

			// Pre-fetch service info and metrics for each service
			for _, svc := range services {
				if _, _, err := client.GetServiceInfo(svc.Node.ID); err != nil {
					logger.Stderr.Error("Failed to warm up service info",
						"project_id", t.ProjectID,
						"environment_id", envInfo.ID,
						"service_id", svc.Node.ID,
						"error", err)
				}

				if _, _, err := client.GetMetrics(envInfo.ID, svc.Node.ID); err != nil {
					logger.Stderr.Error("Failed to warm up service metrics",
						"project_id", t.ProjectID,
						"environment_id", envInfo.ID,
						"service_id", svc.Node.ID,
						"error", err)
				}
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

		if err := json.NewEncoder(w).Encode(response); err != nil {
			logger.Stderr.Error("Failed to encode index response", "error", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
	}
}
