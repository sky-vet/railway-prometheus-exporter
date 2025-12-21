package collector

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mykalmachon/railway-prometheus-exporter/config"
	"github.com/mykalmachon/railway-prometheus-exporter/logger"
	"github.com/mykalmachon/railway-prometheus-exporter/railway"
	"github.com/prometheus/client_golang/prometheus"
)

// RailwayCollector implements the prometheus.Collector interface
type RailwayCollector struct {
	client  *railway.Client
	targets []config.EnvironmentTarget
}

// NewRailwayCollector creates a new Railway metrics collector
func NewRailwayCollector(client *railway.Client, targets []config.EnvironmentTarget) *RailwayCollector {
	return &RailwayCollector{
		client:  client,
		targets: targets,
	}
}

// Describe implements prometheus.Collector
func (c *RailwayCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- cpuUsageDesc
	ch <- cpuLimitDesc
	ch <- memoryUsageDesc
	ch <- memoryLimitDesc
	ch <- networkRxDesc
	ch <- networkTxDesc
	ch <- cpuUtilizationDesc
	ch <- memoryUtilizationDesc
	ch <- lastScrapeSuccessDesc
	ch <- lastScrapeDurationDesc
}

// Collect implements prometheus.Collector
func (c *RailwayCollector) Collect(ch chan<- prometheus.Metric) {
	startTime := time.Now()
	var successCount atomic.Int32 // Track number of successful target collections

	defer func() {
		scrapeDuration := time.Since(startTime).Seconds()
		ch <- prometheus.MustNewConstMetric(
			lastScrapeDurationDesc,
			prometheus.GaugeValue,
			scrapeDuration,
		)

		// Only report success if at least one target succeeded
		if successCount.Load() > 0 {
			ch <- prometheus.MustNewConstMetric(
				lastScrapeSuccessDesc,
				prometheus.GaugeValue,
				float64(time.Now().Unix()),
			)
		}
	}()

	var wg sync.WaitGroup
	metricsChan := make(chan prometheus.Metric, 100)

	// Collect from all targets in parallel
	for _, target := range c.targets {
		wg.Add(1)
		go func(t config.EnvironmentTarget) {
			// Panic recovery MUST be first to ensure wg.Done() always executes
			defer func() {
				if r := recover(); r != nil {
					logger.Stderr.Error("Panic recovered in collector goroutine",
						"project_id", t.ProjectID,
						"environment_id", t.EnvironmentID,
						"panic", r)
				}
			}()
			defer wg.Done()

			if err := c.collectTargetMetrics(metricsChan, t); err != nil {
				logger.Stderr.Error("Failed to collect target metrics",
					"project_id", t.ProjectID,
					"environment_id", t.EnvironmentID,
					"error", err)
			} else {
				// Target succeeded, increment success counter
				successCount.Add(1)
			}
		}(target)
	}

	// Close metrics channel when all targets are done
	go func() {
		wg.Wait()
		close(metricsChan)
	}()

	// Forward all metrics to Prometheus
	for metric := range metricsChan {
		ch <- metric
	}
}

// collectTargetMetrics collects metrics for all services in a project-environment pair
func (c *RailwayCollector) collectTargetMetrics(ch chan<- prometheus.Metric, target config.EnvironmentTarget) error {
	// Get environment info for this target
	envInfo, envCacheHit, err := c.client.GetEnvironmentInfo(target.EnvironmentID)
	if err != nil {
		return fmt.Errorf("failed to get environment info: %w", err)
	}

	// Validate that environment belongs to the specified project
	if envInfo.ProjectID != target.ProjectID {
		return fmt.Errorf("environment %s does not belong to project %s (belongs to %s)",
			target.EnvironmentID, target.ProjectID, envInfo.ProjectID)
	}

	// Get all services for this project
	services, servicesCacheHit, err := c.client.GetProjectServices(target.ProjectID)
	if err != nil {
		return fmt.Errorf("failed to get services: %w", err)
	}

	logger.Stdout.Info("Collecting metrics for target",
		"project_id", target.ProjectID,
		"environment_id", target.EnvironmentID,
		"environment_name", envInfo.Name,
		"service_count", len(services),
		"env_cache_hit", envCacheHit,
		"services_cache_hit", servicesCacheHit,
	)

	// Collect metrics for each service
	for _, serviceNode := range services {
		serviceID := serviceNode.Node.ID
		serviceName := serviceNode.Node.Name

		if err := c.collectServiceMetrics(ch, serviceID, serviceName, target.EnvironmentID, envInfo); err != nil {
			logger.Stderr.Error("Failed to collect metrics for service",
				"service_id", serviceID,
				"service_name", serviceName,
				"project_id", target.ProjectID,
				"environment_id", target.EnvironmentID,
				"error", err,
			)
			continue
		}
	}

	return nil
}

// collectServiceMetrics collects metrics for a single service
func (c *RailwayCollector) collectServiceMetrics(ch chan<- prometheus.Metric, serviceID, serviceName, environmentID string, envInfo *railway.EnvironmentInfo) error {
	// Get service info to get project name
	serviceInfo, serviceInfoCacheHit, err := c.client.GetServiceInfo(serviceID)
	if err != nil {
		return fmt.Errorf("failed to get service info: %w", err)
	}

	// Get current metrics
	metrics, metricsCacheHit, err := c.client.GetMetrics(environmentID, serviceID)
	if err != nil {
		return fmt.Errorf("failed to get metrics: %w", err)
	}

	// Prepare labels
	labels := []string{
		envInfo.ID,
		envInfo.Name,
		serviceID,
		serviceName,
		serviceInfo.Project.Name,
	}

	// Emit core metrics
	if cpuUsage, ok := metrics[railway.MetricCPUUsage]; ok {
		ch <- prometheus.MustNewConstMetric(cpuUsageDesc, prometheus.GaugeValue, cpuUsage, labels...)
	}

	if cpuLimit, ok := metrics[railway.MetricCPULimit]; ok {
		ch <- prometheus.MustNewConstMetric(cpuLimitDesc, prometheus.GaugeValue, cpuLimit, labels...)
	}

	if memUsage, ok := metrics[railway.MetricMemoryUsageGB]; ok {
		ch <- prometheus.MustNewConstMetric(memoryUsageDesc, prometheus.GaugeValue, memUsage, labels...)
	}

	if memLimit, ok := metrics[railway.MetricMemoryLimitGB]; ok {
		ch <- prometheus.MustNewConstMetric(memoryLimitDesc, prometheus.GaugeValue, memLimit, labels...)
	}

	if networkRx, ok := metrics[railway.MetricNetworkRxGB]; ok {
		ch <- prometheus.MustNewConstMetric(networkRxDesc, prometheus.CounterValue, networkRx, labels...)
	}

	if networkTx, ok := metrics[railway.MetricNetworkTxGB]; ok {
		ch <- prometheus.MustNewConstMetric(networkTxDesc, prometheus.CounterValue, networkTx, labels...)
	}

	// Calculate and emit derived metrics
	cpuUsage, hasCPUUsage := metrics[railway.MetricCPUUsage]
	cpuLimit, hasCPULimit := metrics[railway.MetricCPULimit]
	if hasCPUUsage && hasCPULimit && cpuLimit > 0 {
		cpuUtilization := cpuUsage / cpuLimit
		ch <- prometheus.MustNewConstMetric(cpuUtilizationDesc, prometheus.GaugeValue, cpuUtilization, labels...)
	}

	memUsage, hasMemUsage := metrics[railway.MetricMemoryUsageGB]
	memLimit, hasMemLimit := metrics[railway.MetricMemoryLimitGB]
	if hasMemUsage && hasMemLimit && memLimit > 0 {
		memUtilization := memUsage / memLimit
		ch <- prometheus.MustNewConstMetric(memoryUtilizationDesc, prometheus.GaugeValue, memUtilization, labels...)
	}

	logger.Stdout.Info("Collected metrics for service",
		"service_id", serviceID,
		"service_name", serviceName,
		"project_name", serviceInfo.Project.Name,
		"environment_id", environmentID,
		"metric_count", len(metrics),
		"service_info_cache_hit", serviceInfoCacheHit,
		"metrics_cache_hit", metricsCacheHit,
	)

	return nil
}
