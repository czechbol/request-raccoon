package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHandler_Universal(t *testing.T) {
	h := New()

	tests := []struct {
		name       string
		method     string
		path       string
		requestID  string
		wantStatus int
	}{
		{
			name:       "GET request",
			method:     "GET",
			path:       "/test",
			requestID:  "test-id-1",
			wantStatus: http.StatusOK,
		},
		{
			name:       "POST request",
			method:     "POST",
			path:       "/api/webhook",
			requestID:  "test-id-2",
			wantStatus: http.StatusOK,
		},
		{
			name:       "PUT request",
			method:     "PUT",
			path:       "/update",
			requestID:  "test-id-3",
			wantStatus: http.StatusOK,
		},
		{
			name:       "DELETE request",
			method:     "DELETE",
			path:       "/delete",
			requestID:  "test-id-4",
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)

			rr := httptest.NewRecorder()
			h.Universal(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("Expected status %d, got %d", tt.wantStatus, rr.Code)
			}

			// Check content type
			contentType := rr.Header().Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("Expected Content-Type application/json, got %s", contentType)
			}

			// Parse response body
			var response map[string]interface{}
			if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
				t.Errorf("Failed to unmarshal response: %v", err)
				return
			}

			// Check required fields
			if response["status"] != "success" {
				t.Errorf("Expected status 'success', got %v", response["status"])
			}

			if response["message"] != "Request logged successfully" {
				t.Errorf(
					"Expected message 'Request logged successfully', got %v",
					response["message"],
				)
			}

			if response["method"] != tt.method {
				t.Errorf("Expected method %s, got %v", tt.method, response["method"])
			}

			if response["path"] != tt.path {
				t.Errorf("Expected path %s, got %v", tt.path, response["path"])
			}

			// Check timestamp format
			timestampStr, ok := response["timestamp"].(string)
			if !ok {
				t.Error("Timestamp should be a string")
			} else {
				if _, err := time.Parse(time.RFC3339, timestampStr); err != nil {
					t.Errorf("Invalid timestamp format: %s", timestampStr)
				}
			}
		})
	}
}

func TestHandler_Universal_WithoutRequestID(t *testing.T) {
	h := New()
	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	h.Universal(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Errorf("Failed to unmarshal response: %v", err)
		return
	}
}

func TestHandler_Health(t *testing.T) {
	h := New()

	tests := []struct {
		name       string
		method     string
		wantStatus int
	}{
		{
			name:       "GET health check",
			method:     "GET",
			wantStatus: http.StatusOK,
		},
		{
			name:       "POST health check",
			method:     "POST",
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/health", nil)
			rr := httptest.NewRecorder()

			h.Health(rr, req)

			if rr.Code != tt.wantStatus {
				t.Errorf("Expected status %d, got %d", tt.wantStatus, rr.Code)
			}

			// Check content type
			contentType := rr.Header().Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("Expected Content-Type application/json, got %s", contentType)
			}

			// Parse response body
			var response map[string]interface{}
			if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
				t.Errorf("Failed to unmarshal response: %v", err)
				return
			}

			// Check required fields
			if response["status"] != "healthy" {
				t.Errorf("Expected status 'healthy', got %v", response["status"])
			}

			// Check timestamp format
			timestampStr, ok := response["timestamp"].(string)
			if !ok {
				t.Error("Timestamp should be a string")
			} else {
				if _, err := time.Parse(time.RFC3339, timestampStr); err != nil {
					t.Errorf("Invalid timestamp format: %s", timestampStr)
				}
			}
		})
	}
}

func TestHandler_ResponseFormat(t *testing.T) {
	h := New()

	t.Run("Universal response has all required fields", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		rr := httptest.NewRecorder()

		h.Universal(rr, req)

		var response map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		requiredFields := []string{"status", "message", "timestamp", "method", "path"}
		for _, field := range requiredFields {
			if _, exists := response[field]; !exists {
				t.Errorf("Missing required field: %s", field)
			}
		}
	})

	t.Run("Health response has all required fields", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/health", nil)
		rr := httptest.NewRecorder()

		h.Health(rr, req)

		var response map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		requiredFields := []string{"status", "timestamp"}
		for _, field := range requiredFields {
			if _, exists := response[field]; !exists {
				t.Errorf("Missing required field: %s", field)
			}
		}
	})
}

func TestHandler_TimestampIsRecent(t *testing.T) {
	h := New()
	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	before := time.Now().UTC()
	h.Universal(rr, req)
	after := time.Now().UTC().Add(1 * time.Second) // Add buffer for test execution time

	var response map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	timestampStr := response["timestamp"].(string)
	timestamp, err := time.Parse(time.RFC3339, timestampStr)
	if err != nil {
		t.Fatalf("Failed to parse timestamp: %v", err)
	}

	if timestamp.Before(before.Add(-1*time.Second)) || timestamp.After(after) {
		t.Errorf("Timestamp %v is not reasonably close to request time", timestamp)
	}
}

func TestHandler_JSONEncoding(t *testing.T) {
	h := New()

	t.Run("valid JSON output", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/test", nil)
		rr := httptest.NewRecorder()

		h.Universal(rr, req)

		body := rr.Body.String()
		if !json.Valid([]byte(body)) {
			t.Errorf("Response is not valid JSON: %s", body)
		}

		// Ensure no trailing whitespace issues
		if strings.HasSuffix(body, "\n\n") {
			t.Error("Response has extra newlines")
		}
	})
}
