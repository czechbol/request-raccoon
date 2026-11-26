package config

import (
	"errors"
	"os"
	"strconv"
)

const (
	defaultMaxLoggedBody = 64 * 1024
	maxLoggedBodyCap     = 10 * 1024 * 1024
)

// Config holds all configuration for the HTTP logger
type Config struct {
	Port              string `json:"port"`
	Host              string `json:"host"`
	LogLevel          string `json:"log_level"`
	EnableRequestBody bool   `json:"enable_request_body"`
	// MaxLoggedBody is the number of request body bytes that are logged at most.
	MaxLoggedBody int `json:"max_logged_body"`
}

// Load returns a configuration with values from environment variables or defaults
func Load() Config {
	return Config{
		Port:              getEnv("PORT", "8080"),
		Host:              getEnv("HOST", "0.0.0.0"),
		LogLevel:          getEnv("LOG_LEVEL", "info"),
		EnableRequestBody: getBoolEnv("ENABLE_REQUEST_BODY", true),
		MaxLoggedBody:     getSizeEnv("MAX_LOGGED_BODY", defaultMaxLoggedBody, maxLoggedBodyCap),
	}
}

// getSizeEnv reads a non-negative byte count, clamping values above limit.
// Negative and non-numeric values yield defaultValue.
func getSizeEnv(key string, defaultValue, limit int) int {
	n, err := strconv.Atoi(os.Getenv(key))
	// For out-of-range numbers Atoi returns the nearest int, which still has the right sign.
	if (err != nil && !errors.Is(err, strconv.ErrRange)) || n < 0 {
		return defaultValue
	}
	return min(n, limit)
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
