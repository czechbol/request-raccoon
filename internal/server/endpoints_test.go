package server

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/czechbol/request-raccoon/internal/config"
	"github.com/czechbol/request-raccoon/internal/endpoint"
)

func TestServer_CustomEndpoints(t *testing.T) {
	logs := &syncBuffer{}
	captureLogs(t, logs)

	endpoints, err := endpoint.Parse(`[
		{"pattern":"POST /hook","status":202,"headers":{"Content-Type":"text/plain","X-Custom":"yes"},"body":"accepted"},
		{"pattern":"/users/{id}","status":404}
	]`)
	if err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{EnableRequestBody: true, MaxLoggedBody: 4, MaxRequestBody: 8 << 20}
	ts := startServer(t, cfg, endpoints...)

	tests := []struct {
		name, method, path string
		size               int
		status             int
		body               string // substring; empty means an empty response body
		custom             string
	}{
		{"match", "POST", "/hook", 3, http.StatusAccepted, "accepted", "yes"},
		{"large body", "POST", "/hook", 4 << 20, http.StatusAccepted, "accepted", "yes"},
		{"too large", "POST", "/hook", 8<<20 + 1, http.StatusRequestEntityTooLarge, "Too Large", ""},
		{"other method", "GET", "/hook", 0, http.StatusOK, "Request logged successfully", ""},
		{"wildcard, any method", "DELETE", "/users/7", 0, http.StatusNotFound, "", ""},
		{"unmatched path", "POST", "/other", 3, http.StatusOK, "Request logged successfully", ""},
		{"health", "GET", "/health", 0, http.StatusOK, "healthy", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, err := http.NewRequestWithContext(
				t.Context(), tt.method, ts.URL+tt.path, strings.NewReader(strings.Repeat("a", tt.size)))
			if err != nil {
				t.Fatal(err)
			}
			resp, err := ts.Client().Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				t.Fatal(err)
			}
			if resp.StatusCode != tt.status {
				t.Errorf("status = %d, want %d", resp.StatusCode, tt.status)
			}
			if (tt.body == "" && len(body) != 0) || !strings.Contains(string(body), tt.body) {
				t.Errorf("body = %q, want %q", body, tt.body)
			}
			if got := resp.Header.Get("X-Custom"); got != tt.custom {
				t.Errorf("X-Custom = %q, want %q", got, tt.custom)
			}
		})
	}

	if got := string(logs.Bytes()); !strings.Contains(got, "/users/7") {
		t.Errorf("custom endpoint request was not logged: %s", got)
	}
}

func TestEndpointSummary(t *testing.T) {
	got := endpointSummary([]endpoint.Endpoint{
		{Pattern: "POST /hook", Status: 202, Headers: map[string]string{"X-Token": "secret"}, Body: "secret"},
	})
	if len(got) != 1 || got[0] != "POST /hook -> 202" {
		t.Errorf("endpointSummary = %q", got)
	}
}
