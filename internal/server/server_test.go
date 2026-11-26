package server

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/iotest"
	"time"

	"github.com/czechbol/request-raccoon/internal/config"
)

func TestServer_SetupRoutes(t *testing.T) {
	cfg := config.Config{
		Port:              "8080",
		Host:              "localhost",
		LogLevel:          "info",
		EnableRequestBody: true,
	}

	server := New(&cfg)

	// Test that server address is correctly set
	expectedAddr := cfg.Host + ":" + cfg.Port
	if server.server.Addr != expectedAddr {
		t.Errorf("Expected server address %s, got %s", expectedAddr, server.server.Addr)
	}

	// Test that handler is set
	if server.server.Handler == nil {
		t.Error("Expected non-nil server handler")
	}
}

func TestServer_HealthEndpoint(t *testing.T) {
	cfg := config.Config{
		Port:              "8080",
		Host:              "localhost",
		LogLevel:          "info",
		EnableRequestBody: false,
	}

	server := New(&cfg)

	// Create a test request for the health endpoint
	req := httptest.NewRequest("GET", "/health", nil)
	rr := httptest.NewRecorder()

	// Serve the request
	server.server.Handler.ServeHTTP(rr, req)

	// Check status code
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}

	// Check content type
	contentType := rr.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}

	// Check that response body is not empty
	if rr.Body.Len() == 0 {
		t.Error("Expected non-empty response body")
	}
}

func TestServer_UniversalEndpoint(t *testing.T) {
	cfg := config.Config{
		Port:              "8080",
		Host:              "localhost",
		LogLevel:          "info",
		EnableRequestBody: false,
	}

	server := New(&cfg)

	tests := []struct {
		name   string
		method string
		path   string
	}{
		{
			name:   "GET root",
			method: "GET",
			path:   "/",
		},
		{
			name:   "POST webhook",
			method: "POST",
			path:   "/webhook",
		},
		{
			name:   "PUT api",
			method: "PUT",
			path:   "/api/data",
		},
		{
			name:   "DELETE resource",
			method: "DELETE",
			path:   "/resource/123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			rr := httptest.NewRecorder()

			server.server.Handler.ServeHTTP(rr, req)

			// Universal handler should return 200 for all requests
			if rr.Code != http.StatusOK {
				t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
			}

			// Check content type
			contentType := rr.Header().Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("Expected Content-Type application/json, got %s", contentType)
			}

			// Check that response body is not empty
			if rr.Body.Len() == 0 {
				t.Error("Expected non-empty response body")
			}
		})
	}
}

func TestServer_MiddlewareIntegration(t *testing.T) {
	cfg := config.Config{
		Port:              "8080",
		Host:              "localhost",
		LogLevel:          "info",
		EnableRequestBody: true,
	}

	server := New(&cfg)

	// Test with a request that has a body
	req := httptest.NewRequest("POST", "/test", nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "test-agent")

	rr := httptest.NewRecorder()
	server.server.Handler.ServeHTTP(rr, req)

	// Should still return 200 even with middleware processing
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}
}

func TestServer_ConfigIntegration(t *testing.T) {
	tests := []struct {
		name   string
		config config.Config
	}{
		{
			name: "default config",
			config: config.Config{
				Port:              "8080",
				Host:              "0.0.0.0",
				LogLevel:          "info",
				EnableRequestBody: true,
			},
		},
		{
			name: "custom config",
			config: config.Config{
				Port:              "3000",
				Host:              "127.0.0.1",
				LogLevel:          "debug",
				EnableRequestBody: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := New(&tt.config)

			expectedAddr := tt.config.Host + ":" + tt.config.Port
			if server.server.Addr != expectedAddr {
				t.Errorf("Expected server address %s, got %s", expectedAddr, server.server.Addr)
			}

			if server.config.EnableRequestBody != tt.config.EnableRequestBody {
				t.Errorf(
					"Expected EnableRequestBody %v, got %v",
					tt.config.EnableRequestBody,
					server.config.EnableRequestBody,
				)
			}
		})
	}
}

func TestServer_Shutdown(t *testing.T) {
	cfg := config.Config{
		Port:              "0", // Use port 0 for testing to get any available port
		Host:              "localhost",
		LogLevel:          "info",
		EnableRequestBody: false,
	}

	server := New(&cfg)

	// Start server in a goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- server.Start()
	}()

	// Give the server a moment to start
	time.Sleep(10 * time.Millisecond)

	// Test shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err := server.Shutdown(ctx)
	if err != nil {
		t.Errorf("Shutdown failed: %v", err)
	}

	// Wait for start to return
	select {
	case err := <-errChan:
		// server.Start() should return "http: Server closed" error on graceful shutdown
		if err != nil && err.Error() != "http: Server closed" {
			t.Errorf("Expected 'http: Server closed' error, got: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Error("Server did not shut down within timeout")
	}
}

func TestServer_ShutdownTimeout(t *testing.T) {
	cfg := config.Config{
		Port:              "0",
		Host:              "localhost",
		LogLevel:          "info",
		EnableRequestBody: false,
	}

	server := New(&cfg)

	// Start server in a goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- server.Start()
	}()

	// Give the server a moment to start
	time.Sleep(10 * time.Millisecond)

	// Test shutdown with very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	err := server.Shutdown(ctx)
	// Should return context deadline exceeded or complete successfully
	// Both are acceptable since timing can vary in tests
	if err != nil && err != context.DeadlineExceeded {
		t.Errorf("Expected nil or context.DeadlineExceeded, got: %v", err)
	}

	// Clean up - shutdown with longer timeout
	cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cleanupCancel()
	server.Shutdown(cleanupCtx)

	// Wait for start to return
	select {
	case <-errChan:
		// Server stopped
	case <-time.After(2 * time.Second):
		// Don't fail the test if cleanup takes longer
	}
}

func TestServer_HandlerChain(t *testing.T) {
	cfg := config.Config{
		Port:              "8080",
		Host:              "localhost",
		LogLevel:          "info",
		EnableRequestBody: true,
	}

	server := New(&cfg)

	// Test that middleware is properly chained with handlers
	req := httptest.NewRequest("GET", "/health", nil)
	rr := httptest.NewRecorder()

	server.server.Handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}

	// The middleware should log the request and then pass it to the health handler
	// We can't easily test the logging output without capturing logs, but we can
	// verify that the response is from the health handler
	contentType := rr.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}
}

func TestServer_LargeUpload(t *testing.T) {
	const (
		size  = 8 << 20
		limit = 1024
	)

	tests := []struct {
		name              string
		path              string
		enableRequestBody bool
		chunked           bool
		wantBody          any
		wantTruncated     any
		wantSize          any
	}{
		{
			name:              "known length",
			path:              "/upload",
			enableRequestBody: true,
			wantBody:          strings.Repeat("a", limit),
			wantTruncated:     true,
			wantSize:          float64(size),
		},
		{
			name:              "chunked",
			path:              "/upload",
			enableRequestBody: true,
			chunked:           true,
			wantBody:          strings.Repeat("a", limit),
			wantTruncated:     true,
		},
		{name: "body logging disabled", path: "/upload"},
		{
			name:              "health check",
			path:              "/health",
			enableRequestBody: true,
			wantBody:          strings.Repeat("a", limit),
			wantTruncated:     true,
			wantSize:          float64(size),
		},
		{name: "health check with body logging disabled", path: "/health"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var logs bytes.Buffer
			captureLogs(t, &logs)

			app := New(&config.Config{EnableRequestBody: tt.enableRequestBody, MaxLoggedBody: limit}).server.Handler

			// Count what the application itself reads; net/http discards only a little after it returns.
			received := make(chan int, 1)
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body := &countingBody{ReadCloser: r.Body}
				r.Body = body
				app.ServeHTTP(w, r)
				received <- body.count
			}))
			defer ts.Close()

			var upload io.Reader = strings.NewReader(strings.Repeat("a", size))
			if tt.chunked {
				// Hiding the concrete type leaves the length unknown to the client.
				upload = io.MultiReader(upload)
			}
			req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, ts.URL+tt.path, upload)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			resp, err := ts.Client().Do(req)
			if err != nil {
				t.Fatalf("Upload failed: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
			}
			if got := <-received; got != size {
				t.Errorf("Expected the server to receive %d bytes, got %d", size, got)
			}

			var entry map[string]any
			if err := json.Unmarshal(logs.Bytes(), &entry); err != nil {
				t.Fatalf("Expected exactly one JSON log line, got %d bytes: %v", logs.Len(), err)
			}
			if got := entry["request_body"]; got != tt.wantBody {
				t.Errorf("request_body: expected %#v, got %#v", tt.wantBody, got)
			}
			if got := entry["request_body_truncated"]; got != tt.wantTruncated {
				t.Errorf("request_body_truncated: expected %#v, got %#v", tt.wantTruncated, got)
			}
			if got := entry["request_body_size"]; got != tt.wantSize {
				t.Errorf("request_body_size: expected %#v, got %#v", tt.wantSize, got)
			}
		})
	}
}

func TestServer_BodyReadErrorAfterRequestLog(t *testing.T) {
	tests := []struct {
		name              string
		path              string
		enableRequestBody bool
	}{
		{name: "universal", path: "/upload", enableRequestBody: true},
		{name: "universal with body logging disabled", path: "/upload"},
		{name: "health check", path: "/health", enableRequestBody: true},
		{name: "health check with body logging disabled", path: "/health"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var logs bytes.Buffer
			captureLogs(t, &logs)

			app := New(&config.Config{EnableRequestBody: tt.enableRequestBody, MaxLoggedBody: 4}).server.Handler

			// The failure comes after more bytes than the middleware reads for the request log.
			body := io.MultiReader(strings.NewReader("abcdefgh"), iotest.ErrReader(io.ErrUnexpectedEOF))
			rr := httptest.NewRecorder()
			app.ServeHTTP(rr, httptest.NewRequestWithContext(t.Context(), http.MethodPost, tt.path, body))

			if rr.Code != http.StatusOK {
				t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
			}

			var entries []map[string]any
			for decoder := json.NewDecoder(&logs); decoder.More(); {
				var entry map[string]any
				if err := decoder.Decode(&entry); err != nil {
					t.Fatalf("Failed to decode log line: %v", err)
				}
				entries = append(entries, entry)
			}
			if len(entries) != 2 {
				t.Fatalf("Expected a request line and a warning, got %v", entries)
			}

			if got := entries[0]["msg"]; got != "HTTP request received" {
				t.Errorf("Expected the request line first, got %v", got)
			}
			if got, ok := entries[0]["request_body_error"]; ok {
				t.Errorf("Expected no request_body_error in the request line, got %v", got)
			}

			want := map[string]any{
				"level": "WARN",
				"msg":   "Failed to read request body",
				"path":  tt.path,
				"error": io.ErrUnexpectedEOF.Error(),
			}
			for field, value := range want {
				if got := entries[1][field]; got != value {
					t.Errorf("%s: expected %#v, got %#v", field, value, got)
				}
			}
		})
	}
}

// captureLogs redirects the default logger to w for the duration of the test.
func captureLogs(t *testing.T, w io.Writer) {
	t.Helper()

	previous, output, flags := slog.Default(), log.Writer(), log.Flags()
	t.Cleanup(func() {
		slog.SetDefault(previous)
		// slog.SetDefault also redirects the log package, which restoring the default does not undo.
		log.SetOutput(output)
		log.SetFlags(flags)
	})

	slog.SetDefault(slog.New(slog.NewJSONHandler(w, nil)))
}

type countingBody struct {
	io.ReadCloser
	count int
}

func (c *countingBody) Read(p []byte) (int, error) {
	n, err := c.ReadCloser.Read(p)
	c.count += n
	return n, err
}

func TestServer_RouteSpecificity(t *testing.T) {
	cfg := config.Config{
		Port:              "8080",
		Host:              "localhost",
		LogLevel:          "info",
		EnableRequestBody: false,
	}

	server := New(&cfg)

	// Test that /health is handled by health handler, not universal
	healthReq := httptest.NewRequest("GET", "/health", nil)
	healthRr := httptest.NewRecorder()
	server.server.Handler.ServeHTTP(healthRr, healthReq)

	// Test that other paths are handled by universal handler
	universalReq := httptest.NewRequest("GET", "/anything", nil)
	universalRr := httptest.NewRecorder()
	server.server.Handler.ServeHTTP(universalRr, universalReq)

	// Both should return 200 and JSON
	if healthRr.Code != http.StatusOK {
		t.Errorf("Health endpoint: expected status %d, got %d", http.StatusOK, healthRr.Code)
	}
	if universalRr.Code != http.StatusOK {
		t.Errorf("Universal endpoint: expected status %d, got %d", http.StatusOK, universalRr.Code)
	}

	// Both should have JSON content type
	if healthRr.Header().Get("Content-Type") != "application/json" {
		t.Error("Health endpoint should return JSON")
	}
	if universalRr.Header().Get("Content-Type") != "application/json" {
		t.Error("Universal endpoint should return JSON")
	}

	// Responses should be different (health vs universal)
	if healthRr.Body.String() == universalRr.Body.String() {
		t.Error("Health and universal endpoints should return different responses")
	}
}
