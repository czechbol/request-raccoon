package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/czechbol/request-raccoon/internal/config"
)

func TestManager_Logging(t *testing.T) {
	tests := []struct {
		name              string
		enableRequestBody bool
		method            string
		path              string
		query             string
		body              string
		headers           map[string]string
		expectNext        bool
	}{
		{
			name:              "GET request without body",
			enableRequestBody: false,
			method:            "GET",
			path:              "/test",
			query:             "param=value",
			body:              "",
			headers:           map[string]string{"User-Agent": "test-agent"},
			expectNext:        true,
		},
		{
			name:              "POST request with body enabled",
			enableRequestBody: true,
			method:            "POST",
			path:              "/webhook",
			query:             "",
			body:              `{"message": "test"}`,
			headers:           map[string]string{"Content-Type": "application/json"},
			expectNext:        true,
		},
		{
			name:              "POST request with body disabled",
			enableRequestBody: false,
			method:            "POST",
			path:              "/webhook",
			query:             "",
			body:              `{"message": "test"}`,
			headers:           map[string]string{"Content-Type": "application/json"},
			expectNext:        true,
		},
		{
			name:              "request with sensitive headers",
			enableRequestBody: false,
			method:            "GET",
			path:              "/secure",
			query:             "",
			body:              "",
			headers: map[string]string{
				"Authorization": "Bearer token123",
				"Cookie":        "session=abc123",
				"User-Agent":    "test-agent",
			},
			expectNext: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.Config{
				EnableRequestBody: tt.enableRequestBody,
			}
			manager := NewManager(cfg)

			// Create a test handler that records if it was called
			var handlerCalled bool
			var receivedBody []byte
			nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				handlerCalled = true
				if r.Body != nil {
					body, _ := io.ReadAll(r.Body)
					receivedBody = body
				}
				w.WriteHeader(http.StatusOK)
			})

			// Wrap with logging middleware
			handler := manager.Logging(nextHandler)

			// Create request
			var body io.Reader
			if tt.body != "" {
				body = strings.NewReader(tt.body)
			}

			url := tt.path
			if tt.query != "" {
				url += "?" + tt.query
			}

			req := httptest.NewRequest(tt.method, url, body)

			// Add headers
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, req)

			// Check if next handler was called
			if handlerCalled != tt.expectNext {
				t.Errorf("Expected next handler called: %v, got: %v", tt.expectNext, handlerCalled)
			}

			// If body was provided and body reading is enabled, verify body was preserved
			if tt.body != "" && tt.enableRequestBody && handlerCalled {
				if string(receivedBody) != tt.body {
					t.Errorf("Expected body %s, got %s", tt.body, string(receivedBody))
				}
			}
		})
	}
}

func TestManager_Logging_LargeBody(t *testing.T) {
	cfg := config.Config{
		EnableRequestBody: true,
	}
	manager := NewManager(cfg)

	// Create a large body (over 1024 bytes)
	largeBody := strings.Repeat("a", 1500)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Read the body to verify it's still available
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("Failed to read body in next handler: %v", err)
		}
		if string(body) != largeBody {
			t.Error("Body was not properly preserved for large content")
		}
		w.WriteHeader(http.StatusOK)
	})

	handler := manager.Logging(nextHandler)
	req := httptest.NewRequest("POST", "/test", strings.NewReader(largeBody))
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}
}

func TestManager_Logging_NilBody(t *testing.T) {
	cfg := config.Config{
		EnableRequestBody: true,
	}
	manager := NewManager(cfg)

	var handlerCalled bool
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	handler := manager.Logging(nextHandler)
	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if !handlerCalled {
		t.Error("Next handler should have been called")
	}

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}
}

func TestManager_Logging_BodyReadError(t *testing.T) {
	cfg := config.Config{
		EnableRequestBody: true,
	}
	manager := NewManager(cfg)

	var handlerCalled bool
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	handler := manager.Logging(nextHandler)

	// Create a reader that always returns an error
	errorReader := &errorReader{}
	req := httptest.NewRequest("POST", "/test", errorReader)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	// Handler should not be called due to error
	if handlerCalled {
		t.Error("Next handler should not have been called due to body read error")
	}

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", rr.Code)
	}
}

func TestIsSensitiveHeader(t *testing.T) {
	tests := []struct {
		header   string
		expected bool
	}{
		// Sensitive headers
		{"Authorization", true},
		{"authorization", true},
		{"AUTHORIZATION", true},
		{"Cookie", true},
		{"cookie", true},
		{"Set-Cookie", true},
		{"set-cookie", true},
		{"X-API-Key", true},
		{"x-api-key", true},
		{"X-Auth-Token", true},
		{"x-auth-token", true},
		{"Proxy-Authorization", true},
		{"proxy-authorization", true},
		{"WWW-Authenticate", true},
		{"www-authenticate", true},
		{"Proxy-Authenticate", true},
		{"proxy-authenticate", true},

		// Non-sensitive headers
		{"Content-Type", false},
		{"content-type", false},
		{"User-Agent", false},
		{"user-agent", false},
		{"Accept", false},
		{"accept", false},
		{"Host", false},
		{"host", false},
		{"X-Forwarded-For", false},
		{"x-forwarded-for", false},
		{"Content-Length", false},
		{"content-length", false},
		{"", false},
		{"Random-Header", false},
		{"random-header", false},
	}

	for _, tt := range tests {
		t.Run(tt.header, func(t *testing.T) {
			result := isSensitiveHeader(tt.header)
			if result != tt.expected {
				t.Errorf("isSensitiveHeader(%q) = %v, expected %v", tt.header, result, tt.expected)
			}
		})
	}
}

func TestManager_Logging_HeaderFiltering(t *testing.T) {
	cfg := config.Config{
		EnableRequestBody: false,
	}
	manager := NewManager(cfg)

	var handlerCalled bool
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	handler := manager.Logging(nextHandler)
	req := httptest.NewRequest("GET", "/test", nil)

	// Add both sensitive and non-sensitive headers
	req.Header.Set("Authorization", "Bearer secret")
	req.Header.Set("User-Agent", "test-agent")
	req.Header.Set("Cookie", "session=abc123")
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if !handlerCalled {
		t.Error("Next handler should have been called")
	}

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}
}

func TestManager_Logging_EmptyHeaders(t *testing.T) {
	cfg := config.Config{
		EnableRequestBody: false,
	}
	manager := NewManager(cfg)

	var handlerCalled bool
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	handler := manager.Logging(nextHandler)
	req := httptest.NewRequest("GET", "/test", nil)
	// Don't add any headers

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if !handlerCalled {
		t.Error("Next handler should have been called")
	}

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}
}

// Helper types and functions for testing

type errorReader struct{}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, io.ErrUnexpectedEOF
}

func TestManager_Logging_MultipleHeaderValues(t *testing.T) {
	cfg := config.Config{
		EnableRequestBody: false,
	}
	manager := NewManager(cfg)

	var handlerCalled bool
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	handler := manager.Logging(nextHandler)
	req := httptest.NewRequest("GET", "/test", nil)

	// Add header with multiple values
	req.Header.Add("Accept", "application/json")
	req.Header.Add("Accept", "text/html")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if !handlerCalled {
		t.Error("Next handler should have been called")
	}

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", rr.Code)
	}
}
