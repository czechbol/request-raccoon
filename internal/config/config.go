package config

import (
	"os"
	"strconv"
)

// Config holds all configuration for the HTTP logger
type Config struct {
	Port              string `json:"port"`
	Host              string `json:"host"`
	LogLevel          string `json:"log_level"`
	EnableRequestBody bool   `json:"enable_request_body"`
}

// Load returns a configuration with values from environment variables or defaults
func Load() Config {
	return Config{
		Port:              getEnv("PORT", "8080"),
		Host:              getEnv("HOST", "0.0.0.0"),
		LogLevel:          getEnv("LOG_LEVEL", "info"),
		EnableRequestBody: getBoolEnv("ENABLE_REQUEST_BODY", true),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getBoolEnv(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if b, err := strconv.ParseBool(value); err == nil {
			return b
		}
	}
	return defaultValue
}
