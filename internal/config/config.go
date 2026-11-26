package config

import (
	"errors"
	"os"
	"strconv"
	"time"
)

const (
	defaultMaxLoggedBody = 64 * 1024
	maxLoggedBodyCap     = 10 * 1024 * 1024

	defaultMaxRequestBody = 32 * 1024 * 1024
	defaultReadTimeout    = 60 * time.Second

	int64Bits = 64
)

// Config holds all configuration for the HTTP logger
type Config struct {
	Port              string `json:"port"`
	Host              string `json:"host"`
	LogLevel          string `json:"log_level"`
	EnableRequestBody bool   `json:"enable_request_body"`
	// MaxLoggedBody is the number of request body bytes that are logged at most.
	MaxLoggedBody int `json:"max_logged_body"`
	// MaxRequestBody is the largest accepted request body in bytes, 0 means no limit.
	MaxRequestBody int64 `json:"max_request_body"`
	// ReadTimeout is the time allowed for reading a whole request, 0 means no timeout.
	ReadTimeout time.Duration `json:"read_timeout"`
}

// Load returns a configuration with values from environment variables or defaults
func Load() Config {
	return Config{
		Port:              getEnv("PORT", "8080"),
		Host:              getEnv("HOST", "0.0.0.0"),
		LogLevel:          getEnv("LOG_LEVEL", "info"),
		EnableRequestBody: getBoolEnv("ENABLE_REQUEST_BODY", true),
		MaxLoggedBody:     getSizeEnv("MAX_LOGGED_BODY", defaultMaxLoggedBody, maxLoggedBodyCap),
		MaxRequestBody:    getInt64Env("MAX_REQUEST_BODY", defaultMaxRequestBody),
		ReadTimeout:       getDurationEnv("READ_TIMEOUT", defaultReadTimeout),
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

// getInt64Env reads a non-negative integer.
// Negative, non-numeric and out-of-range values yield defaultValue.
func getInt64Env(key string, defaultValue int64) int64 {
	n, err := strconv.ParseInt(os.Getenv(key), 10, int64Bits)
	if err != nil || n < 0 {
		return defaultValue
	}
	return n
}

// getDurationEnv reads a non-negative Go duration such as "1m30s".
// Negative and unparsable values yield defaultValue.
func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	d, err := time.ParseDuration(os.Getenv(key))
	if err != nil || d < 0 {
		return defaultValue
	}
	return d
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
