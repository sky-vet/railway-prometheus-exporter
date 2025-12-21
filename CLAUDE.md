# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Railway Prometheus Exporter is a Go-based Prometheus exporter that monitors Railway services. It uses a pull-based model where Prometheus periodically scrapes the `/metrics` endpoint, and the exporter fetches the latest metrics from Railway's GraphQL API.

The exporter collects CPU usage, memory usage, and network traffic metrics, then enriches them with labels (service name, environment, project) for filtering and aggregation.

## Development Commands

### Build
```bash
go build -o railway-prometheus-exporter
```

### Run
```bash
# Requires .env file with RAILWAY_API_KEY, ENVIRONMENT_TARGETS, and PORT
go run main.go
```

### Dependencies
```bash
# Install/update dependencies
go mod tidy

# Download dependencies
go mod download
```

## Architecture

### Core Components

**main.go** - Application entry point
- Loads configuration from environment variables
- Initializes cache and Railway API client
- Registers Prometheus collector
- Starts HTTP server with `/metrics`, `/health`, and `/` endpoints
- Implements cache warmup on startup (parallel pre-fetching of environment, service, and metrics data)

**config/config.go** - Configuration management
- Parses environment variables using `caarlos0/env`
- Validates and parses `ENVIRONMENT_TARGETS` format: `projectId:environmentId,projectId:environmentId`
- Provides cache TTL configuration (metrics: 30s default, discovery: 300s default)

**railway/client.go** - Railway GraphQL API client
- Executes GraphQL queries against Railway's API (`https://backboard.railway.com/graphql/v2`)
- Implements caching with separate TTLs for metrics (short-lived) and discovery data (long-lived)
- Handles rate limiting with header checking and logging
- Four main methods:
  - `GetProjectServices()` - Lists all services in a project (cached with discovery TTL)
  - `GetServiceInfo()` - Gets service metadata including project name (cached with discovery TTL)
  - `GetEnvironmentInfo()` - Gets environment metadata (cached with discovery TTL)
  - `GetMetrics()` - Fetches current metrics for a service (cached with metrics TTL)

**railway/types.go** - GraphQL response types
- Defines structs for unmarshaling GraphQL responses
- Metric types: `CPU_USAGE`, `CPU_LIMIT`, `MEMORY_USAGE_GB`, `MEMORY_LIMIT_GB`, `NETWORK_RX_GB`, `NETWORK_TX_GB`

**collector/collector.go** - Prometheus collector implementation
- Implements `prometheus.Collector` interface with `Describe()` and `Collect()` methods
- Collects metrics from all configured project-environment targets in parallel using goroutines
- Validates that environments belong to their specified projects
- Emits both raw metrics (CPU, memory, network) and derived metrics (CPU/memory utilization percentages)
- Tracks scrape success timestamp and duration as exporter health metrics

**collector/metrics.go** - Prometheus metric descriptors
- Defines `prometheus.Desc` for all exported metrics with consistent labels: `environment_id`, `environment_name`, `service_id`, `service_name`, `project_name`

**cache/cache.go** - Thread-safe TTL cache
- Built on `sync.Map` for concurrent access
- Stores cache entries with expiration timestamps
- Auto-expires entries on `Get()` if TTL has elapsed
- Used for both metrics data (short TTL) and discovery data (long TTL) to reduce Railway API calls

**logger/logger.go** - Structured logging
- Provides `Stdout` and `Stderr` loggers with structured key-value logging

### Data Flow

1. **Startup**: Application warms up cache by pre-fetching all environment info, services, and initial metrics in parallel
2. **Prometheus Scrape**: When `/metrics` is called:
   - Collector spawns goroutines for each project-environment target
   - Each goroutine fetches environment info (cached), services list (cached), then metrics for each service (cached)
   - Metrics are validated, labeled, and emitted to Prometheus
   - Scrape duration and success timestamp are recorded
3. **Caching Strategy**:
   - Discovery data (environments, services, project info) cached for 5 minutes by default
   - Metrics data cached for 30 seconds by default
   - Cache can be disabled via `ENABLE_CACHE=false`

### Concurrency Patterns

- **Cache warmup**: Uses `sync.WaitGroup` to parallelize fetching across all targets
- **Metric collection**: Each target is processed in parallel with goroutines, writing to a buffered channel
- **Rate limiting**: Uses `sync.Mutex` to safely check/log rate limit headers from Railway API

### Configuration

Environment variables:
- `RAILWAY_API_KEY` (required) - Railway API token
- `ENVIRONMENT_TARGETS` (required) - Comma-separated `projectId:environmentId` pairs
- `PORT` (default: `9090`) - HTTP server port
- `ENABLE_CACHE` (default: `true`) - Enable/disable caching
- `METRICS_CACHE_TTL` (default: `30`) - Seconds to cache metrics
- `DISCOVERY_CACHE_TTL` (default: `300`) - Seconds to cache service/environment data

## Key Implementation Details

- The exporter validates that each environment belongs to its specified project during collection (collector/collector.go:104-107)
- Railway API queries fetch metrics from the last 5 minutes with 60-second sampling/averaging windows, then extract the most recent value (railway/client.go:286-330)
- All Railway API methods check for rate limiting headers and log warnings when approaching limits (railway/client.go:62-89)
- Cache keys are namespaced by type (e.g., `metrics:envId:svcId`, `project_services:projId`, `env_info:envId`)
- The collector continues processing remaining services even if one service fails, logging errors without failing the entire scrape (collector/collector.go:127-136)
