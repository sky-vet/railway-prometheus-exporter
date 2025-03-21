package main

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/mykalmachon/railway-prometheus-exporter/config"
	"github.com/mykalmachon/railway-prometheus-exporter/logger"
)

func main() {
	fmt.Println("Hello Railway Prometheus Exporter")

	if err := godotenv.Load(); err != nil {
		logger.Stderr.Error("Error loading .env file into environment", "error", err.Error())
		os.Exit(1)
	}

	config, configErr := config.GetConfig()
	if configErr != nil {
		logger.Stderr.Error("Error parsing config", "error", configErr.Error())
		os.Exit(1)
	}

	fmt.Println("Config is as follows")
	fmt.Printf("%v", config)
}
