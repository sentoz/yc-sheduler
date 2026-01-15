package metrics

import (
	"context"
	"net"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
)

// StartServer starts an HTTP server that exposes Prometheus metrics (if enabled)
// and basic health-check endpoints. It runs the server in a separate goroutine
// and returns the http.Server so that the caller can shut it down.
// If metricsEnabled is false, only health endpoints are available.
func StartServer(ctx context.Context, addr string, metricsEnabled bool) *http.Server {
	mux := http.NewServeMux()

	if metricsEnabled {
		mux.Handle("/metrics", promhttp.Handler())
	}
	mux.HandleFunc("/health/live", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})
	mux.HandleFunc("/health/ready", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("READY"))
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("yc-scheduler metrics endpoint"))
	})

	srv := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Warn().
			Str("addr", addr).
			Err(err).
			Msg("Failed to bind metrics/health server listener, metrics will be disabled")
		return srv
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

	return srv
}
