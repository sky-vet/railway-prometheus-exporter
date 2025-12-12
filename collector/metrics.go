package collector

import "github.com/prometheus/client_golang/prometheus"

// Metric descriptors for Railway service metrics
var (
	labelNames = []string{"environment_id", "environment_name", "service_id", "service_name", "project_name"}

	// Core metrics from Railway API
	cpuUsageDesc = prometheus.NewDesc(
		"railway_service_cpu_usage",
		"Current CPU usage of the Railway service",
		labelNames,
		nil,
	)

	cpuLimitDesc = prometheus.NewDesc(
		"railway_service_cpu_limit",
		"Configured CPU limit for the Railway service",
		labelNames,
		nil,
	)

	memoryUsageDesc = prometheus.NewDesc(
		"railway_service_memory_usage_gb",
		"Current memory usage of the Railway service in gigabytes",
		labelNames,
		nil,
	)

	memoryLimitDesc = prometheus.NewDesc(
		"railway_service_memory_limit_gb",
		"Configured memory limit for the Railway service in gigabytes",
		labelNames,
		nil,
	)

	networkRxDesc = prometheus.NewDesc(
		"railway_service_network_rx_gb_total",
		"Total network received data for the Railway service in gigabytes",
		labelNames,
		nil,
	)

	networkTxDesc = prometheus.NewDesc(
		"railway_service_network_tx_gb_total",
		"Total network transmitted data for the Railway service in gigabytes",
		labelNames,
		nil,
	)

	// Derived metrics
	cpuUtilizationDesc = prometheus.NewDesc(
		"railway_service_cpu_utilization",
		"CPU utilization percentage of the Railway service (0-1)",
		labelNames,
		nil,
	)

	memoryUtilizationDesc = prometheus.NewDesc(
		"railway_service_memory_utilization",
		"Memory utilization percentage of the Railway service (0-1)",
		labelNames,
		nil,
	)

	// Exporter health metrics
	lastScrapeSuccessDesc = prometheus.NewDesc(
		"railway_exporter_last_scrape_success",
		"Unix timestamp of the last successful scrape",
		nil,
		nil,
	)

	lastScrapeDurationDesc = prometheus.NewDesc(
		"railway_exporter_last_scrape_duration_seconds",
		"Duration of the last scrape in seconds",
		nil,
		nil,
	)
)
