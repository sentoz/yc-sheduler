package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jessevdk/go-flags"
	"github.com/rs/zerolog/log"

	iconfig "github.com/woozymasta/yc-scheduler/internal/config"
	"github.com/woozymasta/yc-scheduler/internal/executor"
	"github.com/woozymasta/yc-scheduler/internal/logger"
	"github.com/woozymasta/yc-scheduler/internal/metrics"
	"github.com/woozymasta/yc-scheduler/internal/scheduler"
	"github.com/woozymasta/yc-scheduler/internal/signals"
	"github.com/woozymasta/yc-scheduler/internal/validator"
	"github.com/woozymasta/yc-scheduler/internal/web"
	"github.com/woozymasta/yc-scheduler/internal/yc"
	pkgconfig "github.com/woozymasta/yc-scheduler/pkg/config"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	var opts struct {
		Config string `short:"c" long:"config" required:"true" description:"Path to configuration file (YAML or JSON)"`
		Token  string `short:"t" long:"token" env:"YC_TOKEN" description:"Yandex Cloud OAuth/IAM token (discouraged; prefer --sa-key)"`
		SaKey  string `long:"sa-key" env:"YC_SA_KEY_FILE" description:"Path to Yandex Cloud service account key JSON file (preferred)"`
		DryRun bool   `short:"n" long:"dry-run" description:"Dry run mode: log planned actions without calling YC APIs"`

		logger.Logger `group:"Logging"`
	}

	if _, err := flags.Parse(&opts); err != nil {
		// go-flags returns an error even for --help; in that case do not treat
		// it as a failure exit code.
		if ferr, ok := err.(*flags.Error); ok && ferr.Type == flags.ErrHelp {
			return nil
		}
		return err
	}

	opts.Logger.Setup()

	log.Debug().
		Str("config_path", opts.Config).
		Bool("dry_run", opts.DryRun).
		Msg("CLI options parsed")

	cfg, err := iconfig.Load(context.Background(), opts.Config)
	if err != nil {
		return fmt.Errorf("yc-scheduler: load config: %w", err)
	}

	ctx, cancel := signals.WithSignalContext(context.Background())
	defer cancel()

	auth := yc.AuthConfig{
		ServiceAccountKeyFile: opts.SaKey,
		Token:                 opts.Token,
	}

	client, err := yc.NewClient(ctx, auth)
	if err != nil {
		return fmt.Errorf("yc-scheduler: create YC client: %w", err)
	}

	// Validate credentials before proceeding
	log.Info().Msg("Validating Yandex Cloud credentials")
	if err := client.ValidateCredentials(ctx); err != nil {
		return fmt.Errorf("yc-scheduler: credentials validation failed: %w", err)
	}
	log.Info().Msg("Credentials validated successfully")

	defer signals.GracefulShutdown(client, cfg.ShutdownTimeout.Std())

	timezone := cfg.Timezone.String()

	sched, err := scheduler.New(timezone, cfg.MaxConcurrentJobs)
	if err != nil {
		return fmt.Errorf("yc-scheduler: init scheduler: %w", err)
	}

	// Start HTTP server for metrics and health endpoints on metrics_port.
	// Health endpoints are always available, metrics only if MetricsEnabled is true.
	addr := fmt.Sprintf(":%d", cfg.MetricsPort)

	if cfg.MetricsEnabled {
		metrics.Init()
	}
	web.StartServer(ctx, addr, cfg.MetricsEnabled)

	if err := registerSchedules(sched, client, cfg, opts.DryRun); err != nil {
		return err
	}

	// Start state validator with interval from config.
	v := validator.New(client, cfg, sched, opts.DryRun)
	v.Start(ctx, cfg.ValidationInterval.Std())

	log.Info().Msg("yc-scheduler started")
	if err := sched.Start(ctx); err != nil {
		return fmt.Errorf("yc-scheduler: scheduler stopped with error: %w", err)
	}
	log.Info().Msg("yc-scheduler stopped")

	return nil
}

func registerSchedules(s *scheduler.Scheduler, client *yc.Client, cfg *pkgconfig.Config, dryRun bool) error {
	for _, sch := range cfg.Schedules {
		if sch.Actions.Start != nil && sch.Actions.Start.Enabled {
			def, err := scheduler.ScheduleToJobDefinition(sch, sch.Actions.Start)
			if err != nil {
				return fmt.Errorf("register schedule %q start action: %w", sch.Name, err)
			}
			name := sch.Name + ":start"
			if err := s.AddJob(def, name, executor.Make(client, sch, "start", dryRun), ""); err != nil {
				return err
			}
		}
		if sch.Actions.Stop != nil && sch.Actions.Stop.Enabled {
			def, err := scheduler.ScheduleToJobDefinition(sch, sch.Actions.Stop)
			if err != nil {
				return fmt.Errorf("register schedule %q stop action: %w", sch.Name, err)
			}
			name := sch.Name + ":stop"
			if err := s.AddJob(def, name, executor.Make(client, sch, "stop", dryRun), ""); err != nil {
				return err
			}
		}
	}
	return nil
}
