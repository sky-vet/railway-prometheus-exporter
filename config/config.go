package config

import (
	"fmt"
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
	EnableCache       bool `env:"ENABLE_CACHE" envDefault:"true"`

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

	return &config, nil
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
