package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"testing/iotest"

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
			manager := NewManager(&cfg)

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

			req := httptest.NewRequestWithContext(t.Context(), tt.method, url, body)

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
	manager := NewManager(&cfg)

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
	req := httptest.NewRequestWithContext(t.Context(), "POST", "/test", strings.NewReader(largeBody))
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
	manager := NewManager(&cfg)

	var handlerCalled bool
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	handler := manager.Logging(nextHandler)
	req := httptest.NewRequestWithContext(t.Context(), "GET", "/test", nil)
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
	tests := []struct {
		name     string
		received string
		// wantBody is nil when no request_body field is expected.
		wantBody      any
		wantTruncated any
	}{
		{name: "nothing received"},
		{name: "partial body", received: "partial", wantBody: "partial"},
		{name: "partial body at limit", received: "12345678", wantBody: "12345678"},
		{
			name: "partial body over limit", received: "123456789",
			wantBody: "12345678", wantTruncated: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logs := captureLogs(t)
			manager := NewManager(&config.Config{EnableRequestBody: true, MaxLoggedBody: 8})

			handler := manager.Logging(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
				t.Error("Next handler should not have been called due to body read error")
			}))

			// DataErrReader reports the error together with the last bytes of data.
			source := iotest.DataErrReader(io.MultiReader(strings.NewReader(tt.received), &errorReader{}))
			req := httptest.NewRequestWithContext(t.Context(), "POST", "/test", source)
			req.Header.Set("User-Agent", "test-agent")
			req.Header.Set("Authorization", "Bearer secret")
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			if rr.Code != http.StatusBadRequest {
				t.Errorf("Expected status 400, got %d", rr.Code)
			}

			entry := requestLogEntry(t, logs)
			want := map[string]any{
				"method":                 "POST",
				"path":                   "/test",
				"request_body":           tt.wantBody,
				"request_body_truncated": tt.wantTruncated,
				"request_body_error":     io.ErrUnexpectedEOF.Error(),
			}
			for field, value := range want {
				if got := entry[field]; got != value {
					t.Errorf("%s: expected %#v, got %#v", field, value, got)
				}
			}
			wantHeaders := map[string]any{"User-Agent": "test-agent", "Authorization": "[REDACTED]"}
			if got := entry["headers"]; !reflect.DeepEqual(got, wantHeaders) {
				t.Errorf("headers: expected %#v, got %#v", wantHeaders, got)
			}
		})
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
	manager := NewManager(&cfg)

	var handlerCalled bool
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	handler := manager.Logging(nextHandler)
	req := httptest.NewRequestWithContext(t.Context(), "GET", "/test", nil)

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
	manager := NewManager(&cfg)

	var handlerCalled bool
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	handler := manager.Logging(nextHandler)
	req := httptest.NewRequestWithContext(t.Context(), "GET", "/test", nil)
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

func TestManager_Logging_BodyLimit(t *testing.T) {
	tests := []struct {
		name    string
		limit   int
		body    string
		chunked bool
		// wantBody is nil when no request_body field is expected.
		wantBody      any
		wantTruncated bool
		// wantSize is nil when no request_body_size field is expected.
		wantSize any
	}{
		{name: "under limit", limit: 8, body: "abc", wantBody: "abc"},
		{name: "exactly limit", limit: 8, body: "abcdefgh", wantBody: "abcdefgh"},
		{
			name: "limit plus one", limit: 8, body: "abcdefghi",
			wantBody: "abcdefgh", wantTruncated: true, wantSize: float64(9),
		},
		{
			name: "over 1024 bytes within limit", limit: 65536, body: strings.Repeat("a", 1500),
			wantBody: strings.Repeat("a", 1500),
		},
		{name: "zero limit", limit: 0, body: "abc", wantBody: "", wantTruncated: true, wantSize: float64(3)},
		{name: "zero limit without body", limit: 0, body: ""},
		{name: "empty body", limit: 8, body: ""},
		{
			name: "two byte character split", limit: 4, body: "abcédef",
			wantBody: "abc", wantTruncated: true, wantSize: float64(8),
		},
		{
			name: "three byte character split after two bytes", limit: 5, body: "abc€def",
			wantBody: "abc", wantTruncated: true, wantSize: float64(9),
		},
		{
			name: "four byte character split after three bytes", limit: 6, body: "abc🦝def",
			wantBody: "abc", wantTruncated: true, wantSize: float64(10),
		},
		{
			name: "character ending at limit is kept", limit: 6, body: "abc€def",
			wantBody: "abc€", wantTruncated: true, wantSize: float64(9),
		},
		{
			name: "first character longer than limit", limit: 2, body: "🦝",
			wantBody: "", wantTruncated: true, wantSize: float64(4),
		},
		{name: "multibyte body exactly limit", limit: 6, body: "abc€", wantBody: "abc€"},
		{
			// The JSON log handler renders the invalid byte as U+FFFD.
			name: "invalid byte is not treated as a split character", limit: 3, body: "ab\xffcd",
			wantBody: "ab�", wantTruncated: true, wantSize: float64(5),
		},
		{name: "chunked within limit", limit: 8, body: "abcdefgh", chunked: true, wantBody: "abcdefgh"},
		{
			name: "chunked over limit", limit: 8, body: "abcdefghijkl", chunked: true,
			wantBody: "abcdefgh", wantTruncated: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logs := captureLogs(t)
			manager := NewManager(&config.Config{EnableRequestBody: true, MaxLoggedBody: tt.limit})

			var received []byte
			handler := manager.Logging(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
				var err error
				if received, err = io.ReadAll(r.Body); err != nil {
					t.Errorf("Failed to read body in next handler: %v", err)
				}
			}))

			req := httptest.NewRequestWithContext(t.Context(), "POST", "/test", strings.NewReader(tt.body))
			if tt.chunked {
				req.ContentLength = -1
			}
			handler.ServeHTTP(httptest.NewRecorder(), req)

			if string(received) != tt.body {
				t.Errorf("Next handler: expected body %q, got %q", tt.body, received)
			}

			entry := requestLogEntry(t, logs)
			if got := entry["request_body"]; got != tt.wantBody {
				t.Errorf("request_body: expected %#v, got %#v", tt.wantBody, got)
			}
			var wantTruncated any
			if tt.wantTruncated {
				wantTruncated = true
			}
			if got := entry["request_body_truncated"]; got != wantTruncated {
				t.Errorf("request_body_truncated: expected %#v, got %#v", wantTruncated, got)
			}
			if got := entry["request_body_size"]; got != tt.wantSize {
				t.Errorf("request_body_size: expected %#v, got %#v", tt.wantSize, got)
			}
		})
	}
}

func TestManager_Logging_BodyDisabled(t *testing.T) {
	logs := captureLogs(t)
	manager := NewManager(&config.Config{EnableRequestBody: false, MaxLoggedBody: 4})

	source := &countingReader{reader: strings.NewReader("abcdefgh")}
	handler := manager.Logging(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		if source.count != 0 {
			t.Errorf("Expected body to be left unread, %d bytes were consumed", source.count)
		}
	}))
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequestWithContext(t.Context(), "POST", "/test", source))

	entry := requestLogEntry(t, logs)
	for _, field := range []string{"request_body", "request_body_truncated", "request_body_size"} {
		if value, ok := entry[field]; ok {
			t.Errorf("Expected no %s field, got %#v", field, value)
		}
	}
}

func TestManager_Logging_BoundedBuffering(t *testing.T) {
	const limit = 1000

	logs := captureLogs(t)
	manager := NewManager(&config.Config{EnableRequestBody: true, MaxLoggedBody: limit})
	body := strings.Repeat("abcdefgh", 1<<17) // 1 MiB
	source := &countingReader{reader: strings.NewReader(body)}

	handler := manager.Logging(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		if source.count > limit+1 {
			t.Errorf("Expected at most %d bytes buffered, got %d", limit+1, source.count)
		}
		if logs.Len() == 0 {
			t.Error("Expected the request to be logged before the next handler runs")
		}

		received, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("Failed to read body in next handler: %v", err)
		}
		if string(received) != body {
			t.Errorf("Next handler: expected %d bytes of body, got %d", len(body), len(received))
		}
	}))
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequestWithContext(t.Context(), "POST", "/test", source))
}

func TestManager_Logging_OmitsEmptyHeaders(t *testing.T) {
	logs := captureLogs(t)
	handler := NewManager(&config.Config{}).Logging(http.NotFoundHandler())

	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequestWithContext(t.Context(), "GET", "/test", nil))

	if headers, ok := requestLogEntry(t, logs)["headers"]; ok {
		t.Errorf("Expected no headers field, got %v", headers)
	}
}

// Helper types and functions for testing

// captureLogs redirects the default logger to a buffer for the duration of the test.
func captureLogs(t *testing.T) *bytes.Buffer {
	t.Helper()

	previous, output, flags := slog.Default(), log.Writer(), log.Flags()
	t.Cleanup(func() {
		slog.SetDefault(previous)
		// slog.SetDefault also redirects the log package, which restoring the default does not undo.
		log.SetOutput(output)
		log.SetFlags(flags)
	})

	buf := &bytes.Buffer{}
	slog.SetDefault(slog.New(slog.NewJSONHandler(buf, nil)))
	return buf
}

// requestLogEntry decodes the single log line written for a request.
func requestLogEntry(t *testing.T, logs *bytes.Buffer) map[string]any {
	t.Helper()

	var entry map[string]any
	if err := json.Unmarshal(logs.Bytes(), &entry); err != nil {
		t.Fatalf("Expected exactly one JSON log line, got %q: %v", logs.String(), err)
	}
	if entry["msg"] != "HTTP request received" {
		t.Fatalf("Unexpected log message: %v", entry["msg"])
	}
	return entry
}

type countingReader struct {
	reader io.Reader
	count  int
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.reader.Read(p)
	c.count += n
	return n, err
}

type errorReader struct{}

func (e *errorReader) Read(p []byte) (n int, err error) {
	return 0, io.ErrUnexpectedEOF
}

func TestManager_Logging_MultipleHeaderValues(t *testing.T) {
	cfg := config.Config{
		EnableRequestBody: false,
	}
	manager := NewManager(&cfg)

	var handlerCalled bool
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	})

	handler := manager.Logging(nextHandler)
	req := httptest.NewRequestWithContext(t.Context(), "GET", "/test", nil)

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
