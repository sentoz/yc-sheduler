package app

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"

	"github.com/woozymasta/yc-scheduler/internal/config"
	"github.com/woozymasta/yc-scheduler/internal/metrics"
	"github.com/woozymasta/yc-scheduler/internal/resource"
	"github.com/woozymasta/yc-scheduler/internal/scheduler"
	"github.com/woozymasta/yc-scheduler/internal/validator"
	"github.com/woozymasta/yc-scheduler/internal/web"
	"github.com/woozymasta/yc-scheduler/internal/yc"
)

// App represents the main application with all its dependencies.
type App struct {
	cfg          *config.Config
	client       *yc.Client
	stateChecker resource.StateChecker
	operator     resource.Operator
	scheduler    *scheduler.Scheduler
	validator    *validator.Validator
	metrics      *metrics.Metrics
	webServer    *web.Server
	dryRun       bool
}

// New creates and initializes a new App instance.
func New(cfg *config.Config, client *yc.Client, dryRun bool) (*App, error) {
	// Initialize metrics if enabled
	var m *metrics.Metrics
	if cfg.MetricsEnabled {
		m = metrics.New()
	}

	// Create resource state checker and operator
	stateChecker := resource.NewYCStateChecker(client)
	operator := resource.NewYCOperator(client)

	// Create scheduler
	timezone := cfg.Timezone.String()
	sched, err := scheduler.New(timezone, cfg.MaxConcurrentJobs)
	if err != nil {
		return nil, fmt.Errorf("create scheduler: %w", err)
	}

	// Create validator
	val := validator.New(stateChecker, operator, cfg, sched, m, dryRun)

	// Create web server
	addr := fmt.Sprintf(":%d", cfg.MetricsPort)
	webSrv, err := web.NewServer(context.Background(), addr, cfg.MetricsEnabled)
	if err != nil {
		log.Warn().
			Str("addr", addr).
			Err(err).
			Msg("Failed to create web server, metrics/health endpoints will be unavailable")
		// Continue without web server
		webSrv = nil
	}

	return &App{
		cfg:          cfg,
		client:       client,
		stateChecker: stateChecker,
		operator:     operator,
		scheduler:    sched,
		validator:    val,
		metrics:      m,
		webServer:    webSrv,
		dryRun:       dryRun,
	}, nil
}

// Run starts the application and blocks until the context is canceled.
func (a *App) Run(ctx context.Context) error {
	// Register schedules
	if err := a.scheduler.RegisterSchedules(a.stateChecker, a.operator, a.cfg, a.dryRun, a.metrics); err != nil {
		return fmt.Errorf("register schedules: %w", err)
	}

	// Start web server if available
	if a.webServer != nil {
		a.webServer.Start()
	}

	// Start validator
	a.validator.Start(ctx, a.cfg.ValidationInterval.Std())

	log.Info().Msg("yc-scheduler started")

	// Start scheduler (blocks until context is canceled)
	if err := a.scheduler.Start(ctx); err != nil {
		return fmt.Errorf("scheduler stopped with error: %w", err)
	}

	log.Info().Msg("yc-scheduler stopped")
	return nil
}

// Shutdown gracefully shuts down the application.
func (a *App) Shutdown(ctx context.Context) error {
	var errs []error

	if a.webServer != nil {
		if err := a.webServer.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("shutdown web server: %w", err))
		}
	}

	if a.scheduler != nil {
		a.scheduler.Stop()
	}

	if a.client != nil {
		if err := a.client.Shutdown(ctx); err != nil {
			errs = append(errs, fmt.Errorf("shutdown YC client: %w", err))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("shutdown errors: %v", errs)
	}

	return nil
}
