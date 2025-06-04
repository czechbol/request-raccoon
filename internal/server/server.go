package server

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/czechbol/request-raccoon/internal/config"
	"github.com/czechbol/request-raccoon/internal/handler"
	"github.com/czechbol/request-raccoon/internal/middleware"
)

// Server holds the HTTP server and its dependencies
type Server struct {
	config     config.Config
	middleware *middleware.Manager
	handler    *handler.Handler
	server     *http.Server
}

// New creates a new server instance with configuration
func New(cfg config.Config) *Server {
	// Create middleware manager
	middlewareManager := middleware.NewManager(cfg)

	// Create handlers
	h := handler.New()

	s := &Server{
		config:     cfg,
		middleware: middlewareManager,
		handler:    h,
	}

	s.setupRoutes()
	return s
}

// setupRoutes configures all HTTP routes and middleware
func (s *Server) setupRoutes() {
	mux := http.NewServeMux()

	// Health check endpoints
	mux.HandleFunc("/health", s.handler.Health)

	// Catch-all handler for logging all other requests
	mux.HandleFunc("/", s.handler.Universal)

	// Apply middleware in the correct order
	finalHandler := s.middleware.Logging(mux)

	s.server = &http.Server{
		Addr:              s.config.Host + ":" + s.config.Port,
		Handler:           finalHandler,
		ReadHeaderTimeout: 10 * time.Second, // Set a read header timeout
	}
}

// Start starts the HTTP server
func (s *Server) Start() error {
	slog.Info("Starting HTTP logger server",
		"address", s.server.Addr,
		"config", s.config)

	return s.server.ListenAndServe()
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	slog.Info("Shutting down HTTP logger server")
	return s.server.Shutdown(ctx)
}
