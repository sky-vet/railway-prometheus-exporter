package config

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/caarlos0/env/v11"
)

type EnvironmentTarget struct {
	ProjectID     string
	EnvironmentID string
}

type Config struct {
	RailwayApiKey string `env:"RAILWAY_API_KEY,required"`
	TargetsRaw    string `env:"ENVIRONMENT_TARGETS,required"`
	Port          string `env:"PORT" envDefault:"9090"`

	// Cache configuration
	MetricsCacheTTL   int  `env:"METRICS_CACHE_TTL" envDefault:"30"`
	DiscoveryCacheTTL int  `env:"DISCOVERY_CACHE_TTL" envDefault:"300"`
	CacheMaxSize      int  `env:"CACHE_MAX_SIZE" envDefault:"10000"`
	EnableCache       bool `env:"ENABLE_CACHE" envDefault:"true"`

	// Retry configuration
	MaxRetries     int `env:"MAX_RETRIES" envDefault:"3"`
	InitialBackoff int `env:"INITIAL_BACKOFF_MS" envDefault:"100"`

	// Parsed targets (not from env, populated by GetConfig())
	Targets []EnvironmentTarget
}

func GetConfig() (*Config, error) {
	config := Config{}

	err := env.Parse(&config)
	if err != nil {
		return nil, err
	}

	// Parse targets
	targets, err := parseTargets(config.TargetsRaw)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ENVIRONMENT_TARGETS: %w", err)
	}
	config.Targets = targets

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

// Validate checks that all configuration values are valid
func (c *Config) Validate() error {
	// Validate port
	port, err := strconv.Atoi(c.Port)
	if err != nil {
		return fmt.Errorf("invalid PORT value '%s': must be a number", c.Port)
	}
	if port < 1 || port > 65535 {
		return fmt.Errorf("invalid PORT value %d: must be between 1 and 65535", port)
	}

	// Validate cache TTLs
	if c.MetricsCacheTTL <= 0 {
		return fmt.Errorf("invalid METRICS_CACHE_TTL value %d: must be greater than 0", c.MetricsCacheTTL)
	}
	if c.MetricsCacheTTL > 3600 {
		return fmt.Errorf("invalid METRICS_CACHE_TTL value %d: must be less than 3600 seconds (1 hour)", c.MetricsCacheTTL)
	}

	if c.DiscoveryCacheTTL <= 0 {
		return fmt.Errorf("invalid DISCOVERY_CACHE_TTL value %d: must be greater than 0", c.DiscoveryCacheTTL)
	}
	if c.DiscoveryCacheTTL > 3600 {
		return fmt.Errorf("invalid DISCOVERY_CACHE_TTL value %d: must be less than 3600 seconds (1 hour)", c.DiscoveryCacheTTL)
	}

	// Validate cache max size
	if c.CacheMaxSize <= 0 {
		return fmt.Errorf("invalid CACHE_MAX_SIZE value %d: must be greater than 0", c.CacheMaxSize)
	}
	if c.CacheMaxSize > 1000000 {
		return fmt.Errorf("invalid CACHE_MAX_SIZE value %d: must be less than 1,000,000", c.CacheMaxSize)
	}

	// Validate retry configuration
	if c.MaxRetries < 0 {
		return fmt.Errorf("invalid MAX_RETRIES value %d: must be 0 or greater", c.MaxRetries)
	}
	if c.MaxRetries > 10 {
		return fmt.Errorf("invalid MAX_RETRIES value %d: must be 10 or less", c.MaxRetries)
	}

	if c.InitialBackoff <= 0 {
		return fmt.Errorf("invalid INITIAL_BACKOFF_MS value %d: must be greater than 0", c.InitialBackoff)
	}
	if c.InitialBackoff > 10000 {
		return fmt.Errorf("invalid INITIAL_BACKOFF_MS value %d: must be less than 10,000ms (10 seconds)", c.InitialBackoff)
	}

	return nil
}

func parseTargets(raw string) ([]EnvironmentTarget, error) {
	pairs := strings.Split(raw, ",")
	targets := make([]EnvironmentTarget, 0, len(pairs))

	for _, pair := range pairs {
		parts := strings.Split(strings.TrimSpace(pair), ":")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid target format: %s (expected projectId:environmentId)", pair)
		}

		projectID := strings.TrimSpace(parts[0])
		environmentID := strings.TrimSpace(parts[1])

		if projectID == "" || environmentID == "" {
			return nil, fmt.Errorf("invalid target: both projectId and environmentId must be non-empty in %s", pair)
		}

		targets = append(targets, EnvironmentTarget{
			ProjectID:     projectID,
			EnvironmentID: environmentID,
		})
	}

	if len(targets) == 0 {
		return nil, fmt.Errorf("at least one target must be specified")
	}

	return targets, nil
}
