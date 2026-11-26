package handler

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/czechbol/request-raccoon/internal/bodyerror"
)

// Handler contains the HTTP handlers.
type Handler struct{}

// New creates a new handler instance.
func New() *Handler {
	return &Handler{}
}

// Universal responds with 200 OK to all requests whose body is not rejected.
func (h *Handler) Universal(w http.ResponseWriter, r *http.Request) {
	if !drainBody(w, r) {
		return
	}

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
	if !drainBody(w, r) {
		return
	}

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
// It reports false after rejecting a body that is too large or too slow.
func drainBody(w http.ResponseWriter, r *http.Request) bool {
	if r.Body == nil {
		return true
	}
	_, err := io.Copy(io.Discard, r.Body)
	if err == nil {
		return true
	}
	slog.Warn("Failed to read request body", "path", r.URL.Path, "error", err)

	// Other read failures do not change the response.
	status := bodyerror.Status(err)
	if status == http.StatusBadRequest {
		return true
	}
	http.Error(w, http.StatusText(status), status)
	return false
}
