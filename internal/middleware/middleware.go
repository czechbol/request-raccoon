package middleware

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"unicode/utf8"

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
		// Prepare log entry
		logFields := []any{
			"method", r.Method,
			"path", r.URL.Path,
			"query", r.URL.RawQuery,
			"remote_addr", r.RemoteAddr,
		}

		// Add request body if enabled
		if m.config.EnableRequestBody && r.Body != nil {
			bodyFields, err := m.peekBody(r)
			if err != nil {
				slog.Error("Failed to read request body",
					"error", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			logFields = append(logFields, bodyFields...)
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

// peekBody reads just enough of the request body to log it, restores the body
// for downstream handlers and returns the log fields describing it.
func (m *Manager) peekBody(r *http.Request) ([]any, error) {
	limit := m.config.MaxLoggedBody

	// The one byte past the limit tells whether the body has to be truncated.
	prefix, err := io.ReadAll(io.LimitReader(r.Body, int64(limit)+1))
	if err != nil {
		return nil, err
	}

	r.Body = struct {
		io.Reader
		io.Closer
	}{io.MultiReader(bytes.NewReader(prefix), r.Body), r.Body}

	switch {
	case len(prefix) == 0:
		return nil, nil
	case len(prefix) <= limit:
		return []any{"request_body", string(prefix)}, nil
	}

	fields := []any{
		"request_body", string(truncateUTF8(prefix, limit)),
		"request_body_truncated", true,
	}
	// ContentLength is -1 when unknown, for example with chunked encoding.
	if r.ContentLength > 0 {
		fields = append(fields, "request_body_size", r.ContentLength)
	}
	return fields, nil
}

// Utility functions

// truncateUTF8 returns at most limit bytes of b without splitting a UTF-8
// character. b must be longer than limit.
func truncateUTF8(b []byte, limit int) []byte {
	// Only a character starting in the last utf8.UTFMax-1 bytes can cross the limit.
	for i := limit - 1; i >= 0 && i > limit-utf8.UTFMax; i-- {
		if utf8.RuneStart(b[i]) {
			if !utf8.FullRune(b[i:limit]) {
				return b[:i]
			}
			break
		}
	}
	return b[:limit]
}

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
