package railway

// GraphQL request and response types for Railway API

// GraphQLRequest represents a generic GraphQL query request
type GraphQLRequest struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables,omitempty"`
}

// GraphQLResponse represents a generic GraphQL response
type GraphQLResponse struct {
	Data   interface{}            `json:"data"`
	Errors []GraphQLError         `json:"errors,omitempty"`
}

// GraphQLError represents a GraphQL error
type GraphQLError struct {
	Message string `json:"message"`
	Path    []string `json:"path,omitempty"`
}

// ServiceInfo contains metadata about a Railway service
type ServiceInfo struct {
	ID      string      `json:"id"`
	Name    string      `json:"name"`
	Project ProjectInfo `json:"project"`
}

// ProjectInfo contains metadata about a Railway project
type ProjectInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// EnvironmentInfo contains metadata about a Railway environment
type EnvironmentInfo struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	ProjectID string `json:"projectId"`
}

// ServiceNode represents a service node in the GraphQL response
type ServiceNode struct {
	Node struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"node"`
}

// ProjectServicesResponse is the response structure for project services query
type ProjectServicesResponse struct {
	Data struct {
		Project struct {
			Services struct {
				Edges []ServiceNode `json:"edges"`
			} `json:"services"`
		} `json:"project"`
	} `json:"data"`
	Errors []GraphQLError `json:"errors,omitempty"`
}

// ServiceInfoResponse is the response structure for service info query
type ServiceInfoResponse struct {
	Data struct {
		Service ServiceInfo `json:"service"`
	} `json:"data"`
	Errors []GraphQLError `json:"errors,omitempty"`
}

// EnvironmentInfoResponse is the response structure for environment info query
type EnvironmentInfoResponse struct {
	Data struct {
		Environment EnvironmentInfo `json:"environment"`
	} `json:"data"`
	Errors []GraphQLError `json:"errors,omitempty"`
}

// MetricValue represents a single metric data point
type MetricValue struct {
	Timestamp int64   `json:"ts"` // Unix timestamp in milliseconds
	Value     float64 `json:"value"`
}

// MetricMeasurement represents a specific metric type with its values
type MetricMeasurement struct {
	Measurement string        `json:"measurement"`
	Values      []MetricValue `json:"values"`
}

// MetricsResponse is the response structure for metrics query
type MetricsResponse struct {
	Data struct {
		Metrics []MetricMeasurement `json:"metrics"`
	} `json:"data"`
	Errors []GraphQLError `json:"errors,omitempty"`
}

// Metric type constants
const (
	MetricCPUUsage      = "CPU_USAGE"
	MetricCPULimit      = "CPU_LIMIT"
	MetricMemoryUsageGB = "MEMORY_USAGE_GB"
	MetricMemoryLimitGB = "MEMORY_LIMIT_GB"
	MetricNetworkRxGB   = "NETWORK_RX_GB"
	MetricNetworkTxGB   = "NETWORK_TX_GB"
)
