package collector

import (
	"fmt"
	"time"

	"github.com/mykalmachon/railway-prometheus-exporter/logger"
	"github.com/mykalmachon/railway-prometheus-exporter/railway"
	"github.com/prometheus/client_golang/prometheus"
)

// RailwayCollector implements the prometheus.Collector interface
type RailwayCollector struct {
	client        *railway.Client
	environmentID string
	projectIDs    []string
}

// NewRailwayCollector creates a new Railway metrics collector
func NewRailwayCollector(client *railway.Client, environmentID string, projectIDs []string) *RailwayCollector {
	return &RailwayCollector{
		client:        client,
		environmentID: environmentID,
		projectIDs:    projectIDs,
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
	scrapeSuccess := false

	defer func() {
		scrapeDuration := time.Since(startTime).Seconds()
		ch <- prometheus.MustNewConstMetric(
			lastScrapeDurationDesc,
			prometheus.GaugeValue,
			scrapeDuration,
		)

		if scrapeSuccess {
			ch <- prometheus.MustNewConstMetric(
				lastScrapeSuccessDesc,
				prometheus.GaugeValue,
				float64(time.Now().Unix()),
			)
		}
	}()

	// Get environment info once (used for all services)
	envInfo, err := c.client.GetEnvironmentInfo(c.environmentID)
	if err != nil {
		logger.Stderr.Error("Failed to get environment info",
			"environment_id", c.environmentID,
			"error", err,
		)
		return
	}

	// Iterate through all configured projects
	for _, projectID := range c.projectIDs {
		if err := c.collectProjectMetrics(ch, projectID, envInfo); err != nil {
			logger.Stderr.Error("Failed to collect metrics for project",
				"project_id", projectID,
				"error", err,
			)
			continue
		}
	}

	scrapeSuccess = true
}

// collectProjectMetrics collects metrics for all services in a project
func (c *RailwayCollector) collectProjectMetrics(ch chan<- prometheus.Metric, projectID string, envInfo *railway.EnvironmentInfo) error {
	// Get all services for this project
	services, err := c.client.GetProjectServices(projectID)
	if err != nil {
		return fmt.Errorf("failed to get services: %w", err)
	}

	logger.Stdout.Info("Collecting metrics for project",
		"project_id", projectID,
		"service_count", len(services),
	)

	// Collect metrics for each service
	for _, serviceNode := range services {
		serviceID := serviceNode.Node.ID
		serviceName := serviceNode.Node.Name

		if err := c.collectServiceMetrics(ch, serviceID, serviceName, projectID, envInfo); err != nil {
			logger.Stderr.Error("Failed to collect metrics for service",
				"service_id", serviceID,
				"service_name", serviceName,
				"error", err,
			)
			continue
		}
	}

	return nil
}

// collectServiceMetrics collects metrics for a single service
func (c *RailwayCollector) collectServiceMetrics(ch chan<- prometheus.Metric, serviceID, serviceName, projectID string, envInfo *railway.EnvironmentInfo) error {
	// Get service info to get project name
	serviceInfo, err := c.client.GetServiceInfo(serviceID)
	if err != nil {
		return fmt.Errorf("failed to get service info: %w", err)
	}

	// Get current metrics
	metrics, err := c.client.GetMetrics(c.environmentID, serviceID)
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
		"metric_count", len(metrics),
	)

	return nil
}
