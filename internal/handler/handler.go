package handler

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"time"
)

// Handler contains the HTTP handlers.
type Handler struct{}

// New creates a new handler instance.
func New() *Handler {
	return &Handler{}
}

// Universal responds with 200 OK to all requests.
func (h *Handler) Universal(w http.ResponseWriter, r *http.Request) {
	drainBody(r)

	response := map[string]interface{}{
		"status":    "success",
		"message":   "Request logged successfully",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"method":    r.Method,
		"path":      r.URL.Path,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(w).Encode(response)
	if err != nil {
		slog.Error("Failed to encode response", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// Health provides health check endpoint.
func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	drainBody(r)

	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	err := json.NewEncoder(w).Encode(health)
	if err != nil {
		slog.Error("Failed to encode response", "error", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// drainBody receives the whole request body. It has to happen before responding,
// because net/http cuts off large unread uploads once the response is sent.
func drainBody(r *http.Request) {
	if r.Body != nil {
		// Read failures do not change the response.
		_, _ = io.Copy(io.Discard, r.Body)
	}
}
