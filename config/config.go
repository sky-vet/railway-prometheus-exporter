package config

import (
	"github.com/caarlos0/env/v11"
)

type Config struct {
	RailwayApiKey string   `env:"RAILWAY_API_KEY,required"`
	EnvironmentId string   `env:"ENVIRONMENT_ID,required"`
	ProjectIds    []string `env:"PROJECT_IDS,required" envSeparator:","`
}

func GetConfig() (*Config, error) {
	config := Config{}

	err := env.Parse(&config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}
