package signals

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"
)

// Shutdowner is an interface for objects that can be gracefully shut down.
type Shutdowner interface {
	Shutdown(ctx context.Context) error
}

// WithSignalContext returns a context that is canceled on SIGINT or SIGTERM.
func WithSignalContext(parent context.Context) (context.Context, context.CancelFunc) {
	return signal.NotifyContext(parent, os.Interrupt, syscall.SIGTERM)
}

// GracefulShutdown shuts down the given Shutdowner with the specified timeout.
// This function is intended to be used in a defer statement.
func GracefulShutdown(shutdowner Shutdowner, timeout time.Duration) {
	if shutdowner == nil {
		return
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := shutdowner.Shutdown(shutdownCtx); err != nil {
		log.Warn().Err(err).Msg("Failed to shutdown gracefully")
	}
}
