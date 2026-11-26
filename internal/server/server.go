package server

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/czechbol/request-raccoon/internal/config"
	"github.com/czechbol/request-raccoon/internal/endpoint"
	"github.com/czechbol/request-raccoon/internal/handler"
	"github.com/czechbol/request-raccoon/internal/middleware"
)

// Server holds the HTTP server and its dependencies
type Server struct {
	config     config.Config
	middleware *middleware.Manager
	handler    *handler.Handler
	endpoints  []endpoint.Endpoint
	server     *http.Server
}

// New creates a new server instance with configuration.
// The custom endpoints have to come from endpoint.Parse, which validates them.
func New(cfg *config.Config, endpoints ...endpoint.Endpoint) *Server {
	// Create middleware manager
	middlewareManager := middleware.NewManager(cfg)

	// Create handlers
	h := handler.New()

	s := &Server{
		config:     *cfg,
		middleware: middlewareManager,
		handler:    h,
		endpoints:  endpoints,
	}

	s.setupRoutes()
	return s
}

// setupRoutes configures all HTTP routes and middleware
func (s *Server) setupRoutes() {
	mux := http.NewServeMux()

	// Health check endpoints
	mux.HandleFunc("/health", s.handler.Health)

	// Custom endpoints take precedence over the catch-all handler
	for _, e := range s.endpoints {
		mux.Handle(e.Pattern, s.handler.Custom(e))
	}

	// Catch-all handler for logging all other requests
	mux.HandleFunc("/", s.handler.Universal)

	// Apply middleware in the correct order
	finalHandler := s.middleware.Logging(mux)
	// MaxBytesHandler treats 0 as an empty body limit, not as no limit.
	if s.config.MaxRequestBody > 0 {
		finalHandler = http.MaxBytesHandler(finalHandler, s.config.MaxRequestBody)
	}

	s.server = &http.Server{
		Addr:              s.config.Host + ":" + s.config.Port,
		Handler:           finalHandler,
		ReadHeaderTimeout: 10 * time.Second, // Set a read header timeout
		ReadTimeout:       s.config.ReadTimeout,
	}
}

// Start starts the HTTP server
func (s *Server) Start() error {
	slog.Info("Starting HTTP logger server",
		"address", s.server.Addr,
		"config", s.config,
		"endpoints", endpointSummary(s.endpoints))

	return s.server.ListenAndServe()
}

// endpointSummary describes the custom endpoints without headers or bodies, which may hold secrets.
func endpointSummary(endpoints []endpoint.Endpoint) []string {
	summary := make([]string, 0, len(endpoints))
	for _, e := range endpoints {
		summary = append(summary, fmt.Sprintf("%s -> %d", e.Pattern, e.Status))
	}
	return summary
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	slog.Info("Shutting down HTTP logger server")
	return s.server.Shutdown(ctx)
}
