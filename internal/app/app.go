package app

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/sentoz/yc-sheduler/internal/config"
	"github.com/sentoz/yc-sheduler/internal/metrics"
	"github.com/sentoz/yc-sheduler/internal/reloader"
	"github.com/sentoz/yc-sheduler/internal/resource"
	"github.com/sentoz/yc-sheduler/internal/scheduler"
	"github.com/sentoz/yc-sheduler/internal/validator"
	"github.com/sentoz/yc-sheduler/internal/web"
	"github.com/sentoz/yc-sheduler/internal/yc"
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
	reloader     *reloader.Reloader
	dryRun       bool
}

const schedulesReloadInterval = 10 * time.Second

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

	schedulesReloader, err := reloader.New(cfg.SchedulesDir, schedulesReloadInterval, func(ctx context.Context) error {
		return reloadSchedules(ctx, cfg.SchedulesDir, sched, stateChecker, operator, val, dryRun, m, cfg)
	})
	if err != nil {
		return nil, fmt.Errorf("create schedules reloader: %w", err)
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
		reloader:     schedulesReloader,
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
	go a.reloader.Start(ctx)

	log.Info().Msg("yc-scheduler started")

	// Start scheduler (blocks until context is canceled)
	if err := a.scheduler.Start(ctx); err != nil {
		return fmt.Errorf("scheduler stopped with error: %w", err)
	}

	log.Info().Msg("yc-scheduler stopped")
	return nil
}

func reloadSchedules(
	ctx context.Context,
	schedulesDir string,
	sched *scheduler.Scheduler,
	stateChecker resource.StateChecker,
	operator resource.Operator,
	val *validator.Validator,
	dryRun bool,
	m *metrics.Metrics,
	cfg *config.Config,
) error {
	schedules, err := config.LoadSchedules(ctx, schedulesDir)
	if err != nil {
		return fmt.Errorf("load schedules: %w", err)
	}

	if err := sched.ReplaceSchedules(stateChecker, operator, schedules, dryRun, m); err != nil {
		return fmt.Errorf("replace schedules: %w", err)
	}

	cfg.Schedules = append([]config.Schedule(nil), schedules...)
	val.UpdateSchedules(schedules)

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
