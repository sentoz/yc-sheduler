package web

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
)

// StartServer starts an HTTP server that exposes Prometheus metrics (if enabled)
// and health-check endpoints. It runs the server in a separate goroutine
// and handles graceful shutdown via context.
// If metricsEnabled is false, only health endpoints are available.
func StartServer(ctx context.Context, addr string, metricsEnabled bool) {
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
		log.Warn().
			Str("addr", addr).
			Err(err).
			Msg("Failed to bind metrics/health server listener, metrics will be disabled")
		return
	}

	log.Info().
		Str("addr", addr).
		Bool("metrics_enabled", metricsEnabled).
		Msg("Starting metrics and health HTTP server")

	go func() {
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			log.Warn().Err(err).Msg("Metrics/health HTTP server stopped with error")
		}
	}()

	go func() {
		<-ctx.Done()
		if err := srv.Shutdown(context.Background()); err != nil {
			log.Warn().Err(err).Msg("Failed to shutdown metrics/health HTTP server")
		}
	}()
}
