package config

import (
	"os"
	"testing"
	"time"
)

func TestLoad(t *testing.T) {
	// Save original environment
	originalPort := os.Getenv("PORT")
	originalHost := os.Getenv("HOST")
	originalLogLevel := os.Getenv("LOG_LEVEL")
	originalEnableRequestBody := os.Getenv("ENABLE_REQUEST_BODY")

	// Clean up after test
	defer func() {
		os.Setenv("PORT", originalPort)
		os.Setenv("HOST", originalHost)
		os.Setenv("LOG_LEVEL", originalLogLevel)
		os.Setenv("ENABLE_REQUEST_BODY", originalEnableRequestBody)
	}()

	t.Run("default values", func(t *testing.T) {
		// Clear environment variables
		os.Unsetenv("PORT")
		os.Unsetenv("HOST")
		os.Unsetenv("LOG_LEVEL")
		os.Unsetenv("ENABLE_REQUEST_BODY")

		cfg := Load()

		if cfg.Port != "8080" {
			t.Errorf("Expected default port 8080, got %s", cfg.Port)
		}
		if cfg.Host != "0.0.0.0" {
			t.Errorf("Expected default host 0.0.0.0, got %s", cfg.Host)
		}
		if cfg.LogLevel != "info" {
			t.Errorf("Expected default log level info, got %s", cfg.LogLevel)
		}
		if !cfg.EnableRequestBody {
			t.Errorf("Expected default EnableRequestBody true, got %v", cfg.EnableRequestBody)
		}
	})

	t.Run("environment variables override", func(t *testing.T) {
		os.Setenv("PORT", "3000")
		os.Setenv("HOST", "127.0.0.1")
		os.Setenv("LOG_LEVEL", "debug")
		os.Setenv("ENABLE_REQUEST_BODY", "false")

		cfg := Load()

		if cfg.Port != "3000" {
			t.Errorf("Expected port 3000, got %s", cfg.Port)
		}
		if cfg.Host != "127.0.0.1" {
			t.Errorf("Expected host 127.0.0.1, got %s", cfg.Host)
		}
		if cfg.LogLevel != "debug" {
			t.Errorf("Expected log level debug, got %s", cfg.LogLevel)
		}
		if cfg.EnableRequestBody {
			t.Errorf("Expected EnableRequestBody false, got %v", cfg.EnableRequestBody)
		}
	})
}

func TestLoad_MaxLoggedBody(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		expected int
	}{
		{name: "unset or empty uses default", envValue: "", expected: 65536},
		{name: "custom value", envValue: "2048", expected: 2048},
		{name: "zero", envValue: "0", expected: 0},
		{name: "exactly the cap", envValue: "10485760", expected: 10485760},
		{name: "above the cap is clamped", envValue: "10485761", expected: 10485760},
		{name: "beyond integer range is clamped", envValue: "99999999999999999999", expected: 10485760},
		{name: "negative uses default", envValue: "-1", expected: 65536},
		{name: "negative beyond integer range uses default", envValue: "-99999999999999999999", expected: 65536},
		{name: "non-numeric uses default", envValue: "lots", expected: 65536},
		{name: "unit suffix uses default", envValue: "64k", expected: 65536},
		{name: "fraction uses default", envValue: "1.5", expected: 65536},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("MAX_LOGGED_BODY", tt.envValue)

			if got := Load().MaxLoggedBody; got != tt.expected {
				t.Errorf("MAX_LOGGED_BODY=%q: expected %d, got %d", tt.envValue, tt.expected, got)
			}
		})
	}
}

func TestLoad_MaxRequestBody(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		expected int64
	}{
		{name: "unset or empty uses default", envValue: "", expected: 33554432},
		{name: "custom value", envValue: "2048", expected: 2048},
		{name: "zero disables the limit", envValue: "0", expected: 0},
		{name: "above the default is kept", envValue: "1073741824", expected: 1073741824},
		{name: "negative uses default", envValue: "-1", expected: 33554432},
		{name: "beyond integer range uses default", envValue: "99999999999999999999", expected: 33554432},
		{name: "non-numeric uses default", envValue: "lots", expected: 33554432},
		{name: "unit suffix uses default", envValue: "32M", expected: 33554432},
		{name: "fraction uses default", envValue: "1.5", expected: 33554432},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("MAX_REQUEST_BODY", tt.envValue)

			if got := Load().MaxRequestBody; got != tt.expected {
				t.Errorf("MAX_REQUEST_BODY=%q: expected %d, got %d", tt.envValue, tt.expected, got)
			}
		})
	}
}

func TestLoad_ReadTimeout(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		expected time.Duration
	}{
		{name: "unset or empty uses default", envValue: "", expected: time.Minute},
		{name: "custom value", envValue: "1m30s", expected: 90 * time.Second},
		{name: "sub-second value", envValue: "250ms", expected: 250 * time.Millisecond},
		{name: "zero disables the timeout", envValue: "0", expected: 0},
		{name: "zero with unit disables the timeout", envValue: "0s", expected: 0},
		{name: "negative uses default", envValue: "-1s", expected: time.Minute},
		{name: "missing unit uses default", envValue: "30", expected: time.Minute},
		{name: "non-duration uses default", envValue: "soon", expected: time.Minute},
		{name: "beyond duration range uses default", envValue: "9999999999h", expected: time.Minute},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("READ_TIMEOUT", tt.envValue)

			if got := Load().ReadTimeout; got != tt.expected {
				t.Errorf("READ_TIMEOUT=%q: expected %v, got %v", tt.envValue, tt.expected, got)
			}
		})
	}
}

func TestGetEnv(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue string
		envValue     string
		expected     string
	}{
		{
			name:         "environment variable set",
			key:          "TEST_KEY",
			defaultValue: "default",
			envValue:     "custom",
			expected:     "custom",
		},
		{
			name:         "environment variable not set",
			key:          "NONEXISTENT_KEY",
			defaultValue: "default",
			envValue:     "",
			expected:     "default",
		},
		{
			name:         "empty environment variable",
			key:          "EMPTY_KEY",
			defaultValue: "default",
			envValue:     "",
			expected:     "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original value
			original := os.Getenv(tt.key)
			defer os.Setenv(tt.key, original)

			if tt.envValue != "" {
				os.Setenv(tt.key, tt.envValue)
			} else {
				os.Unsetenv(tt.key)
			}

			result := getEnv(tt.key, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestGetBoolEnv(t *testing.T) {
	tests := []struct {
		name         string
		key          string
		defaultValue bool
		envValue     string
		expected     bool
	}{
		{
			name:         "true value",
			key:          "TEST_BOOL",
			defaultValue: false,
			envValue:     "true",
			expected:     true,
		},
		{
			name:         "false value",
			key:          "TEST_BOOL",
			defaultValue: true,
			envValue:     "false",
			expected:     false,
		},
		{
			name:         "1 value",
			key:          "TEST_BOOL",
			defaultValue: false,
			envValue:     "1",
			expected:     true,
		},
		{
			name:         "0 value",
			key:          "TEST_BOOL",
			defaultValue: true,
			envValue:     "0",
			expected:     false,
		},
		{
			name:         "invalid value uses default",
			key:          "TEST_BOOL",
			defaultValue: true,
			envValue:     "invalid",
			expected:     true,
		},
		{
			name:         "empty value uses default",
			key:          "TEST_BOOL",
			defaultValue: false,
			envValue:     "",
			expected:     false,
		},
		{
			name:         "unset value uses default",
			key:          "NONEXISTENT_BOOL",
			defaultValue: true,
			envValue:     "",
			expected:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original value
			original := os.Getenv(tt.key)
			defer os.Setenv(tt.key, original)

			if tt.envValue != "" {
				os.Setenv(tt.key, tt.envValue)
			} else {
				os.Unsetenv(tt.key)
			}

			result := getBoolEnv(tt.key, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}
