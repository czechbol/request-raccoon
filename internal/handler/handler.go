package handler

import (
	"encoding/json"
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
func (h *Handler) Health(w http.ResponseWriter, _ *http.Request) {
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
