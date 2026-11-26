package server

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/czechbol/request-raccoon/internal/config"
	"github.com/czechbol/request-raccoon/internal/endpoint"
)

// declaredLength is the Content-Length announced by startUpload.
const declaredLength = 100

func TestServer_MaxRequestBody(t *testing.T) {
	tests := []struct {
		name       string
		maxBody    int64
		maxLogged  int
		logBody    bool
		size       int
		wantStatus int
		// wantInEntry means the failure is reported in the request line instead of a warning.
		wantInEntry bool
	}{
		{name: "exact limit within prefix", maxBody: 16, maxLogged: 64, logBody: true, size: 16, wantStatus: 200},
		{
			name: "limit plus one within prefix", maxBody: 16, maxLogged: 64, logBody: true, size: 17,
			wantStatus: 413, wantInEntry: true,
		},
		{
			name: "exact limit equal to logged limit", maxBody: 16, maxLogged: 16, logBody: true, size: 16,
			wantStatus: 200,
		},
		{
			name: "limit plus one equal to logged limit", maxBody: 16, maxLogged: 16, logBody: true, size: 17,
			wantStatus: 413, wantInEntry: true,
		},
		{name: "exact limit after prefix", maxBody: 16, maxLogged: 4, logBody: true, size: 16, wantStatus: 200},
		{name: "limit plus one after prefix", maxBody: 16, maxLogged: 4, logBody: true, size: 17, wantStatus: 413},
		{name: "exact limit with body logging disabled", maxBody: 16, size: 16, wantStatus: 200},
		{name: "limit plus one with body logging disabled", maxBody: 16, size: 17, wantStatus: 413},
		{
			name: "large upload over limit", maxBody: 1 << 20, maxLogged: 4, logBody: true, size: 8 << 20,
			wantStatus: 413,
		},
		{name: "large upload with body logging disabled", maxBody: 1 << 20, size: 8 << 20, wantStatus: 413},
		{name: "disabled limit", maxLogged: 4, logBody: true, size: 8 << 20, wantStatus: 200},
		{name: "disabled limit with body logging disabled", size: 8 << 20, wantStatus: 200},
	}

	for _, tt := range tests {
		for _, path := range []string{"/upload", "/health"} {
			for _, chunked := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s %s chunked=%t", tt.name, path, chunked), func(t *testing.T) {
					logs := &syncBuffer{}
					captureLogs(t, logs)
					ts := startServer(t, &config.Config{
						EnableRequestBody: tt.logBody,
						MaxLoggedBody:     tt.maxLogged,
						MaxRequestBody:    tt.maxBody,
					})

					var upload io.Reader = strings.NewReader(strings.Repeat("a", tt.size))
					if chunked {
						// Hiding the concrete type leaves the length unknown to the client.
						upload = io.MultiReader(upload)
					}
					req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, ts.URL+path, upload)
					if err != nil {
						t.Fatalf("Failed to create request: %v", err)
					}
					resp, err := ts.Client().Do(req)
					if err != nil {
						t.Fatalf("Upload failed: %v", err)
					}
					defer resp.Body.Close()

					if resp.StatusCode != tt.wantStatus {
						t.Errorf("Expected status %d, got %d", tt.wantStatus, resp.StatusCode)
					}
					assertLogs(t, logs, path, tt.wantStatus != http.StatusOK, tt.wantInEntry)
				})
			}
		}
	}
}

func TestServer_ReadTimeout(t *testing.T) {
	tests := []struct {
		name        string
		logBody     bool
		sent        string
		wantInEntry bool
	}{
		{name: "nothing sent", logBody: true, wantInEntry: true},
		{name: "stalled before the logged prefix", logBody: true, sent: "ab", wantInEntry: true},
		{name: "stalled after the logged prefix", logBody: true, sent: "abcdefgh"},
		{name: "stalled with body logging disabled", sent: "ab"},
	}

	for _, tt := range tests {
		for _, path := range []string{"/upload", "/health"} {
			for _, chunked := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s %s chunked=%t", tt.name, path, chunked), func(t *testing.T) {
					logs := &syncBuffer{}
					captureLogs(t, logs)
					ts := startServer(t, &config.Config{
						EnableRequestBody: tt.logBody,
						MaxLoggedBody:     4,
						ReadTimeout:       200 * time.Millisecond,
					})

					conn := startUpload(t, ts, path, tt.sent, chunked)
					resp := readResponse(t, conn)
					defer resp.Body.Close()

					if resp.StatusCode != http.StatusRequestTimeout {
						t.Errorf("Expected status %d, got %d", http.StatusRequestTimeout, resp.StatusCode)
					}
					assertLogs(t, logs, path, true, tt.wantInEntry)
				})
			}
		}
	}
}

func TestServer_DisabledReadTimeout(t *testing.T) {
	logs := &syncBuffer{}
	captureLogs(t, logs)
	ts := startServer(t, &config.Config{EnableRequestBody: true, MaxLoggedBody: 4})

	if got := ts.Config.ReadTimeout; got != 0 {
		t.Errorf("Expected no read timeout, got %v", got)
	}

	conn := startUpload(t, ts, "/upload", "ab", false)
	time.Sleep(300 * time.Millisecond)
	if _, err := io.WriteString(conn, strings.Repeat("c", declaredLength-2)); err != nil {
		t.Fatalf("Failed to finish the upload: %v", err)
	}
	resp := readResponse(t, conn)
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}
}

// startServer serves the application, including its server settings, on a loopback address.
func startServer(t *testing.T, cfg *config.Config, endpoints ...endpoint.Endpoint) *httptest.Server {
	t.Helper()

	ts := httptest.NewUnstartedServer(nil)
	ts.Config = New(cfg, endpoints...).server
	ts.Start()
	t.Cleanup(ts.Close)
	return ts
}

// startUpload sends the headers and the beginning of a body, leaving the upload unfinished.
func startUpload(t *testing.T, ts *httptest.Server, path, sent string, chunked bool) net.Conn {
	t.Helper()

	conn, err := (&net.Dialer{}).DialContext(t.Context(), "tcp", ts.Listener.Addr().String())
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	t.Cleanup(func() { conn.Close() })

	length := fmt.Sprintf("Content-Length: %d", declaredLength)
	if chunked {
		length = "Transfer-Encoding: chunked"
		if sent != "" {
			sent = fmt.Sprintf("%x\r\n%s\r\n", len(sent), sent)
		}
	}
	if _, err := fmt.Fprintf(conn, "POST %s HTTP/1.1\r\nHost: test\r\n%s\r\n\r\n%s", path, length, sent); err != nil {
		t.Fatalf("Failed to send the request: %v", err)
	}
	return conn
}

func readResponse(t *testing.T, conn net.Conn) *http.Response {
	t.Helper()

	if err := conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatalf("Failed to set a deadline: %v", err)
	}
	resp, err := http.ReadResponse(bufio.NewReader(conn), nil)
	if err != nil {
		t.Fatalf("Failed to read the response: %v", err)
	}
	return resp
}

// assertLogs checks that the request line was logged and, for a rejected
// request, that the failure was reported in it or in a following warning.
func assertLogs(t *testing.T, logs *syncBuffer, path string, rejected, inEntry bool) {
	t.Helper()

	var entries []map[string]any
	for decoder := json.NewDecoder(bytes.NewReader(logs.Bytes())); decoder.More(); {
		var entry map[string]any
		if err := decoder.Decode(&entry); err != nil {
			t.Fatalf("Failed to decode log line: %v", err)
		}
		entries = append(entries, entry)
	}

	wantEntries := 1
	if rejected && !inEntry {
		wantEntries = 2
	}
	if len(entries) != wantEntries {
		t.Fatalf("Expected %d log lines, got %d: %v", wantEntries, len(entries), entries)
	}
	if entries[0]["msg"] != "HTTP request received" || entries[0]["path"] != path {
		t.Errorf("Expected the request line for %s first, got %v", path, entries[0])
	}
	if _, got := entries[0]["request_body_error"]; got != inEntry {
		t.Errorf("request_body_error present: expected %t, got %t", inEntry, got)
	}
	if wantEntries == 2 && entries[1]["msg"] != "Failed to read request body" {
		t.Errorf("Expected a read warning, got %v", entries[1])
	}
}

// syncBuffer is a log destination shared between the server and the test.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

// Bytes returns a copy of everything written so far.
func (b *syncBuffer) Bytes() []byte {
	b.mu.Lock()
	defer b.mu.Unlock()
	return bytes.Clone(b.buf.Bytes())
}
