package railway

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/mykalmachon/railway-prometheus-exporter/cache"
	"github.com/mykalmachon/railway-prometheus-exporter/logger"
)

const (
	railwayAPIURL = "https://backboard.railway.com/graphql/v2"
)

// CacheTTLConfig holds cache TTL configuration
type CacheTTLConfig struct {
	MetricsTTL    time.Duration
	DiscoveryTTL  time.Duration
	EnableCache   bool
}

// RetryConfig holds retry configuration
type RetryConfig struct {
	MaxRetries     int
	InitialBackoff time.Duration
}

// Client is a Railway GraphQL API client
type Client struct {
	apiKey      string
	httpClient  *http.Client
	cache       *cache.Cache
	cacheTTLs   CacheTTLConfig
	retryConfig RetryConfig
	rateLimitMu sync.Mutex
	requestSem  chan struct{} // Semaphore to limit concurrent requests
}

// NewClient creates a new Railway API client without caching
func NewClient(apiKey string) *Client {
	return &Client{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		cache: nil,
		cacheTTLs: CacheTTLConfig{
			EnableCache: false,
		},
		retryConfig: RetryConfig{
			MaxRetries:     3,
			InitialBackoff: 100 * time.Millisecond,
		},
		requestSem: make(chan struct{}, 10), // Allow max 10 concurrent requests
	}
}

// NewClientWithCache creates a new Railway API client with cache support
func NewClientWithCache(apiKey string, cacheInstance *cache.Cache, cacheTTLs CacheTTLConfig, retryConfig RetryConfig) *Client {
	return &Client{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		cache:       cacheInstance,
		cacheTTLs:   cacheTTLs,
		retryConfig: retryConfig,
		requestSem:  make(chan struct{}, 10), // Allow max 10 concurrent requests
	}
}

// checkRateLimitHeaders checks and logs Railway API rate limit headers
func (c *Client) checkRateLimitHeaders(resp *http.Response) {
	c.rateLimitMu.Lock()
	defer c.rateLimitMu.Unlock()

	// Check for rate limit remaining headers (various formats)
	remaining := resp.Header.Get("X-RateLimit-Remaining")
	if remaining == "" {
		remaining = resp.Header.Get("RateLimit-Remaining")
	}
	if remaining == "" {
		remaining = resp.Header.Get("X-Rate-Limit-Remaining")
	}

	if remaining != "" {
		if val, err := strconv.Atoi(remaining); err == nil && val < 10 {
			logger.Stdout.Warn("Railway API rate limit approaching",
				"remaining", val)
		}
	}

	// Check for Retry-After header (indicates rate limiting)
	retryAfter := resp.Header.Get("Retry-After")
	if retryAfter != "" {
		logger.Stderr.Warn("Railway API rate limited",
			"retry_after", retryAfter)
	}
}

// isRetryableError determines if an error should be retried
func isRetryableError(statusCode int) bool {
	// Retry on rate limiting, server errors, and service unavailable
	return statusCode == http.StatusTooManyRequests ||
		statusCode == http.StatusInternalServerError ||
		statusCode == http.StatusBadGateway ||
		statusCode == http.StatusServiceUnavailable ||
		statusCode == http.StatusGatewayTimeout
}

// executeQuery executes a GraphQL query and unmarshals the response with retry logic
func (c *Client) executeQuery(query string, variables map[string]interface{}, response interface{}) (err error) {
	// Acquire semaphore to limit concurrent requests
	c.requestSem <- struct{}{}

	// CRITICAL: Release semaphore and recover from panics
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic in executeQuery: %v", r)
			logger.Stderr.Error("Panic recovered in executeQuery", "panic", r)
		}
		<-c.requestSem // Always release semaphore, even on panic
	}()

	reqBody := GraphQLRequest{
		Query:     query,
		Variables: variables,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	var lastErr error
	backoff := c.retryConfig.InitialBackoff

	// Retry loop with exponential backoff
	for attempt := 0; attempt <= c.retryConfig.MaxRetries; attempt++ {
		// Add delay before retry (but not on first attempt)
		if attempt > 0 {
			logger.Stdout.Info("Retrying Railway API request",
				"attempt", attempt,
				"max_retries", c.retryConfig.MaxRetries,
				"backoff_ms", backoff.Milliseconds())
			time.Sleep(backoff)
			backoff *= 2 // Exponential backoff
		}

		req, err := http.NewRequest("POST", railwayAPIURL, bytes.NewBuffer(jsonBody))
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+c.apiKey)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("failed to execute request: %w", err)
			// Network errors are retryable
			continue
		}
		defer resp.Body.Close()

		// Check rate limit headers
		c.checkRateLimitHeaders(resp)

		// Check for retryable status codes
		if isRetryableError(resp.StatusCode) {
			lastErr = fmt.Errorf("retryable error, status code: %d", resp.StatusCode)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}

		if err := json.NewDecoder(resp.Body).Decode(response); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}

		return nil // Success!
	}

	// All retries exhausted
	return fmt.Errorf("request failed after %d retries: %w", c.retryConfig.MaxRetries, lastErr)
}

// GetProjectServices retrieves all services for a given project
func (c *Client) GetProjectServices(projectID string) ([]ServiceNode, error) {
	// Check cache first
	if c.cacheTTLs.EnableCache && c.cache != nil {
		cacheKey := fmt.Sprintf("project_services:%s", projectID)
		if cached, ok := c.cache.Get(cacheKey); ok {
			if services, okType := cached.([]ServiceNode); okType {
				return services, nil
			}
			logger.Stderr.Warn("Invalid cached type for project services, refetching",
				"project_id", projectID)
		}
	}

	query := `
		query getProjectServices($projectId: String!) {
			project(id: $projectId) {
				services {
					edges {
						node {
							id
							name
						}
					}
				}
			}
		}
	`

	variables := map[string]interface{}{
		"projectId": projectID,
	}

	var response ProjectServicesResponse
	if err := c.executeQuery(query, variables, &response); err != nil {
		return nil, fmt.Errorf("failed to get project services: %w", err)
	}

	if len(response.Errors) > 0 {
		return nil, fmt.Errorf("graphql error: %s", response.Errors[0].Message)
	}

	result := response.Data.Project.Services.Edges

	// Cache the result
	if c.cacheTTLs.EnableCache && c.cache != nil {
		cacheKey := fmt.Sprintf("project_services:%s", projectID)
		c.cache.Set(cacheKey, result, c.cacheTTLs.DiscoveryTTL)
	}

	return result, nil
}

// GetServiceInfo retrieves metadata about a specific service
func (c *Client) GetServiceInfo(serviceID string) (*ServiceInfo, error) {
	// Check cache first
	if c.cacheTTLs.EnableCache && c.cache != nil {
		cacheKey := fmt.Sprintf("service_info:%s", serviceID)
		if cached, ok := c.cache.Get(cacheKey); ok {
			if serviceInfo, okType := cached.(*ServiceInfo); okType {
				return serviceInfo, nil
			}
			logger.Stderr.Warn("Invalid cached type for service info, refetching",
				"service_id", serviceID)
		}
	}

	query := `
		query getServiceInfo($serviceId: String!) {
			service(id: $serviceId) {
				id
				name
				project {
					id
					name
				}
			}
		}
	`

	variables := map[string]interface{}{
		"serviceId": serviceID,
	}

	var response ServiceInfoResponse
	if err := c.executeQuery(query, variables, &response); err != nil {
		return nil, fmt.Errorf("failed to get service info: %w", err)
	}

	if len(response.Errors) > 0 {
		return nil, fmt.Errorf("graphql error: %s", response.Errors[0].Message)
	}

	result := &response.Data.Service

	// Cache the result
	if c.cacheTTLs.EnableCache && c.cache != nil {
		cacheKey := fmt.Sprintf("service_info:%s", serviceID)
		c.cache.Set(cacheKey, result, c.cacheTTLs.DiscoveryTTL)
	}

	return result, nil
}

// GetEnvironmentInfo retrieves metadata about a specific environment
func (c *Client) GetEnvironmentInfo(environmentID string) (*EnvironmentInfo, error) {
	// Check cache first
	if c.cacheTTLs.EnableCache && c.cache != nil {
		cacheKey := fmt.Sprintf("env_info:%s", environmentID)
		if cached, ok := c.cache.Get(cacheKey); ok {
			if envInfo, okType := cached.(*EnvironmentInfo); okType {
				return envInfo, nil
			}
			logger.Stderr.Warn("Invalid cached type for environment info, refetching",
				"environment_id", environmentID)
		}
	}

	query := `
		query getEnvironmentInfo($environmentId: String!) {
			environment(id: $environmentId) {
				id
				name
				projectId
			}
		}
	`

	variables := map[string]interface{}{
		"environmentId": environmentID,
	}

	var response EnvironmentInfoResponse
	if err := c.executeQuery(query, variables, &response); err != nil {
		return nil, fmt.Errorf("failed to get environment info: %w", err)
	}

	if len(response.Errors) > 0 {
		return nil, fmt.Errorf("graphql error: %s", response.Errors[0].Message)
	}

	result := &response.Data.Environment

	// Cache the result
	if c.cacheTTLs.EnableCache && c.cache != nil {
		cacheKey := fmt.Sprintf("env_info:%s", environmentID)
		c.cache.Set(cacheKey, result, c.cacheTTLs.DiscoveryTTL)
	}

	return result, nil
}

// GetMetrics retrieves current metrics for a specific service
func (c *Client) GetMetrics(environmentID, serviceID string) (map[string]float64, error) {
	// Check cache first
	if c.cacheTTLs.EnableCache && c.cache != nil {
		cacheKey := fmt.Sprintf("metrics:%s:%s", environmentID, serviceID)
		if cached, ok := c.cache.Get(cacheKey); ok {
			if metrics, okType := cached.(map[string]float64); okType {
				return metrics, nil
			}
			logger.Stderr.Warn("Invalid cached type for metrics, refetching",
				"environment_id", environmentID,
				"service_id", serviceID)
		}
	}

	// Get metrics from the last 5 minutes to ensure we get recent data
	startDate := time.Now().Add(-5 * time.Minute).Format(time.RFC3339)

	query := `
		query getMetrics($environmentId: String!, $serviceId: String!, $startDate: DateTime!) {
			metrics(
				startDate: $startDate
				endDate: null
				sampleRateSeconds: 60
				averagingWindowSeconds: 60
				environmentId: $environmentId
				serviceId: $serviceId
				measurements: [CPU_USAGE, CPU_LIMIT, MEMORY_USAGE_GB, MEMORY_LIMIT_GB, NETWORK_RX_GB, NETWORK_TX_GB]
			) {
				measurement
				values {
					ts
					value
				}
			}
		}
	`

	variables := map[string]interface{}{
		"environmentId": environmentID,
		"serviceId":     serviceID,
		"startDate":     startDate,
	}

	var response MetricsResponse
	if err := c.executeQuery(query, variables, &response); err != nil {
		return nil, fmt.Errorf("failed to get metrics: %w", err)
	}

	if len(response.Errors) > 0 {
		return nil, fmt.Errorf("graphql error: %s", response.Errors[0].Message)
	}

	// Extract the latest value for each metric
	metrics := make(map[string]float64)
	for _, metric := range response.Data.Metrics {
		if len(metric.Values) > 0 {
			// Get the most recent value
			latestValue := metric.Values[len(metric.Values)-1]
			metrics[metric.Measurement] = latestValue.Value
		}
	}

	// Cache the result
	if c.cacheTTLs.EnableCache && c.cache != nil {
		cacheKey := fmt.Sprintf("metrics:%s:%s", environmentID, serviceID)
		c.cache.Set(cacheKey, metrics, c.cacheTTLs.MetricsTTL)
	}

	return metrics, nil
}
