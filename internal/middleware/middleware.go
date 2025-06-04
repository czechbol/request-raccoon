package middleware

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/czechbol/request-raccoon/internal/config"
)

// Manager handles all middleware functionality
type Manager struct {
	config config.Config
}

// NewManager creates a new middleware manager
func NewManager(cfg config.Config) *Manager {
	return &Manager{
		config: cfg,
	}
}

// Logging logs all HTTP requests with comprehensive details
func (m *Manager) Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Read and restore request body if enabled
		var bodyContent string
		if m.config.EnableRequestBody && r.Body != nil {
			bodyBytes, err := io.ReadAll(r.Body)
			if err != nil {
				slog.Error("Failed to read request body",
					"error", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			bodyContent = string(bodyBytes)

			// Restore the request body for downstream handlers
			r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		}

		// Prepare log entry
		logFields := []any{
			"method", r.Method,
			"path", r.URL.Path,
			"query", r.URL.RawQuery,
			"remote_addr", r.RemoteAddr,
		}

		// Add request body if enabled and not too large
		if m.config.EnableRequestBody && bodyContent != "" && len(bodyContent) <= 1024 {
			logFields = append(logFields, "request_body", bodyContent)
		}

		// Add all headers (except sensitive ones)
		headers := make(map[string]string)
		for k, v := range r.Header {
			if !isSensitiveHeader(k) && len(v) > 0 {
				headers[k] = v[0]
			} else {
				headers[k] = "[REDACTED]"
			}
		}
		if len(headers) > 0 {
			logFields = append(logFields, "headers", headers)
		}

		// Log with appropriate level based on status code
		slog.Info("HTTP request received", logFields...)

		// Call the next handler
		next.ServeHTTP(w, r)
	})
}

// Utility functions

func isSensitiveHeader(key string) bool {
	sensitive := []string{
		"authorization", "cookie", "set-cookie", "x-api-key", "x-auth-token",
		"proxy-authorization", "www-authenticate", "proxy-authenticate",
	}
	lowerKey := strings.ToLower(key)
	for _, s := range sensitive {
		if lowerKey == s {
			return true
		}
	}
	return false
}
