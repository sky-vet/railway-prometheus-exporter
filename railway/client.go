package railway

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const (
	railwayAPIURL = "https://backboard.railway.com/graphql/v2"
)

// Client is a Railway GraphQL API client
type Client struct {
	apiKey     string
	httpClient *http.Client
}

// NewClient creates a new Railway API client
func NewClient(apiKey string) *Client {
	return &Client{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// executeQuery executes a GraphQL query and unmarshals the response
func (c *Client) executeQuery(query string, variables map[string]interface{}, response interface{}) error {
	reqBody := GraphQLRequest{
		Query:     query,
		Variables: variables,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", railwayAPIURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(response); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	return nil
}

// GetProjectServices retrieves all services for a given project
func (c *Client) GetProjectServices(projectID string) ([]ServiceNode, error) {
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

	return response.Data.Project.Services.Edges, nil
}

// GetServiceInfo retrieves metadata about a specific service
func (c *Client) GetServiceInfo(serviceID string) (*ServiceInfo, error) {
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

	return &response.Data.Service, nil
}

// GetEnvironmentInfo retrieves metadata about a specific environment
func (c *Client) GetEnvironmentInfo(environmentID string) (*EnvironmentInfo, error) {
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

	return &response.Data.Environment, nil
}

// GetMetrics retrieves current metrics for a specific service
func (c *Client) GetMetrics(environmentID, serviceID string) (map[string]float64, error) {
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

	return metrics, nil
}
