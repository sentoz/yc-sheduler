package web

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
)

// Server represents an HTTP server for metrics and health endpoints.
type Server struct {
	srv    *http.Server
	ln     net.Listener
	ctx    context.Context
	cancel context.CancelFunc
}

// NewServer creates a new Server instance.
func NewServer(ctx context.Context, addr string, metricsEnabled bool) (*Server, error) {
	mux := http.NewServeMux()

	// Register metrics endpoint if enabled (must be before /)
	if metricsEnabled {
		mux.Handle("/metrics", promhttp.Handler())
	}

	// Register health endpoints
	mux.HandleFunc("/health", HealthHandler)
	mux.HandleFunc("/health/live", HealthHandler)
	mux.HandleFunc("/health/ready", HealthHandler)

	// Register build info endpoint (must be last as it matches all paths)
	mux.HandleFunc("/", BuildInfoHandler)

	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	serverCtx, cancel := context.WithCancel(ctx)

	server := &Server{
		srv:    srv,
		ln:     ln,
		ctx:    serverCtx,
		cancel: cancel,
	}

	log.Info().
		Str("addr", addr).
		Bool("metrics_enabled", metricsEnabled).
		Msg("Starting metrics and health HTTP server")

	return server, nil
}

// Start starts the HTTP server in a separate goroutine.
func (s *Server) Start() {
	go func() {
		if err := s.srv.Serve(s.ln); err != nil && err != http.ErrServerClosed {
			log.Warn().Err(err).Msg("Metrics/health HTTP server stopped with error")
		}
	}()

	go func() {
		<-s.ctx.Done()
		if err := s.Shutdown(context.Background()); err != nil {
			log.Warn().Err(err).Msg("Failed to shutdown metrics/health HTTP server")
		}
	}()
}

// Shutdown gracefully shuts down the HTTP server.
func (s *Server) Shutdown(ctx context.Context) error {
	s.cancel()
	if s.srv == nil {
		return nil
	}
	return s.srv.Shutdown(ctx)
}
