package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jessevdk/go-flags"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	iconfig "github.com/woozymasta/yc-scheduler/internal/config"
	"github.com/woozymasta/yc-scheduler/internal/metrics"
	"github.com/woozymasta/yc-scheduler/internal/scheduler"
	"github.com/woozymasta/yc-scheduler/internal/validator"
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
		Token  string `short:"t" long:"token" description:"Yandex Cloud OAuth/IAM token (discouraged; prefer --sa-key)"`
		SaKey  string `long:"sa-key" description:"Path to Yandex Cloud service account key JSON file (preferred)"`
		DryRun bool   `short:"n" long:"dry-run" description:"Dry run mode: log planned actions without calling YC APIs"`
	}

	if _, err := flags.Parse(&opts); err != nil {
		// go-flags returns an error even for --help; in that case do not treat
		// it as a failure exit code.
		if ferr, ok := err.(*flags.Error); ok && ferr.Type == flags.ErrHelp {
			return nil
		}
		return err
	}

	setupLogger()

	cfg, err := iconfig.Load(context.Background(), opts.Config)
	if err != nil {
		return fmt.Errorf("yc-scheduler: load config: %w", err)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	saKey := opts.SaKey
	if saKey == "" {
		// Try common environment variables for service account key file.
		saKey = os.Getenv("YC_SERVICE_ACCOUNT_KEY_FILE")
		if saKey == "" {
			saKey = os.Getenv("YC_SA_KEY_FILE")
		}
	}

	token := opts.Token
	if token == "" {
		token = os.Getenv("YC_TOKEN")
	}

	auth := yc.AuthConfig{
		ServiceAccountKeyFile: saKey,
		Token:                 token,
	}

	client, err := yc.NewClient(ctx, auth)
	if err != nil {
		return fmt.Errorf("yc-scheduler: create YC client: %w", err)
	}

	shutdownTimeout := 5 * time.Minute
	if cfg.ShutdownTimeout.Duration > 0 {
		shutdownTimeout = cfg.ShutdownTimeout.Std()
	}

	defer func() {
		shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancelShutdown()
		if err := client.Shutdown(shutdownCtx); err != nil {
			log.Warn().Err(err).Msg("Failed to shutdown YC client")
		}
	}()

	timezone := ""
	if cfg.Timezone != "" {
		timezone = cfg.Timezone.String()
	}

	maxConcurrentJobs := cfg.MaxConcurrentJobs
	if maxConcurrentJobs <= 0 {
		maxConcurrentJobs = 5
	}

	sched, err := scheduler.New(timezone, maxConcurrentJobs)
	if err != nil {
		return fmt.Errorf("yc-scheduler: init scheduler: %w", err)
	}

	if cfg.MetricsEnabled {
		metrics.Init()
		port := cfg.MetricsPort
		if port == 0 {
			port = 9090
		}
		addr := fmt.Sprintf(":%d", port)
		metrics.StartServer(ctx, addr)
	}

	if err := registerSchedules(sched, client, cfg, opts.DryRun); err != nil {
		return err
	}

	// Start state validator with interval from config (default 10 minutes).
	validationInterval := 10 * time.Minute
	if cfg.ValidationInterval.Duration > 0 {
		validationInterval = cfg.ValidationInterval.Std()
	}
	v := validator.New(client, cfg)
	v.Start(ctx, validationInterval)

	log.Info().Msg("yc-scheduler started")
	if err := sched.Start(ctx); err != nil {
		return fmt.Errorf("yc-scheduler: scheduler stopped with error: %w", err)
	}
	log.Info().Msg("yc-scheduler stopped")

	return nil
}

func setupLogger() {
	output := zerolog.ConsoleWriter{
		Out:        os.Stderr,
		TimeFormat: time.RFC3339,
	}
	log.Logger = zerolog.New(output).With().Timestamp().Logger()
}

func registerSchedules(s *scheduler.Scheduler, client *yc.Client, cfg *pkgconfig.Config, dryRun bool) error {
	for _, sch := range cfg.Schedules {
		def, err := scheduler.ScheduleToJobDefinition(sch)
		if err != nil {
			return fmt.Errorf("register schedule %q: %w", sch.Name, err)
		}

		if sch.Actions.Start != nil && sch.Actions.Start.Enabled {
			name := sch.Name + ":start"
			if err := s.AddJob(def, name, makeExecutor(client, sch, "start", dryRun)); err != nil {
				return err
			}
		}
		if sch.Actions.Stop != nil && sch.Actions.Stop.Enabled {
			name := sch.Name + ":stop"
			if err := s.AddJob(def, name, makeExecutor(client, sch, "stop", dryRun)); err != nil {
				return err
			}
		}
		if sch.Actions.Restart != nil && sch.Actions.Restart.Enabled {
			name := sch.Name + ":restart"
			if err := s.AddJob(def, name, makeExecutor(client, sch, "restart", dryRun)); err != nil {
				return err
			}
		}
	}
	return nil
}

func makeExecutor(client *yc.Client, sch pkgconfig.Schedule, action string, dryRun bool) func(context.Context) {
	resource := sch.Resource

	return func(ctx context.Context) {
		if dryRun {
			log.Info().
				Str("schedule", sch.Name).
				Str("resource_type", resource.Type).
				Str("resource_id", resource.ID).
				Str("action", action).
				Msg("Dry-run: planned operation")
			metrics.IncOperation(resource.Type, action, "dry_run")
			return
		}

		switch resource.Type {
		case "vm":
			switch action {
			case "start":
				if err := client.StartInstance(ctx, resource.FolderID, resource.ID); err != nil {
					log.Error().Err(err).
						Str("resource_type", resource.Type).
						Str("resource_id", resource.ID).
						Str("action", action).
						Msg("VM operation failed")
					metrics.IncOperation(resource.Type, action, "error")
					return
				}
			case "stop":
				if err := client.StopInstance(ctx, resource.FolderID, resource.ID); err != nil {
					log.Error().Err(err).
						Str("resource_type", resource.Type).
						Str("resource_id", resource.ID).
						Str("action", action).
						Msg("VM operation failed")
					metrics.IncOperation(resource.Type, action, "error")
					return
				}
			case "restart":
				if err := client.RestartInstance(ctx, resource.FolderID, resource.ID); err != nil {
					log.Error().Err(err).
						Str("resource_type", resource.Type).
						Str("resource_id", resource.ID).
						Str("action", action).
						Msg("VM operation failed")
					metrics.IncOperation(resource.Type, action, "error")
					return
				}
			}
			metrics.IncOperation(resource.Type, action, "success")
		case "k8s_node_group":
			const desiredSize = int64(1)
			switch action {
			case "start":
				if err := client.StartNodeGroup(ctx, resource.FolderID, resource.ID, desiredSize); err != nil {
					log.Error().Err(err).
						Str("resource_type", resource.Type).
						Str("resource_id", resource.ID).
						Str("action", action).
						Msg("Node group operation failed")
					metrics.IncOperation(resource.Type, action, "error")
					return
				}
			case "stop":
				if err := client.StopNodeGroup(ctx, resource.FolderID, resource.ID); err != nil {
					log.Error().Err(err).
						Str("resource_type", resource.Type).
						Str("resource_id", resource.ID).
						Str("action", action).
						Msg("Node group operation failed")
					metrics.IncOperation(resource.Type, action, "error")
					return
				}
			case "restart":
				if err := client.RestartNodeGroup(ctx, resource.FolderID, resource.ID, desiredSize); err != nil {
					log.Error().Err(err).
						Str("resource_type", resource.Type).
						Str("resource_id", resource.ID).
						Str("action", action).
						Msg("Node group operation failed")
					metrics.IncOperation(resource.Type, action, "error")
					return
				}
			}
			metrics.IncOperation(resource.Type, action, "success")
		case "k8s_cluster":
			switch action {
			case "start":
				if err := client.StartCluster(ctx, resource.FolderID, resource.ID); err != nil {
					log.Error().Err(err).
						Str("resource_type", resource.Type).
						Str("resource_id", resource.ID).
						Str("action", action).
						Msg("Cluster operation failed")
					metrics.IncOperation(resource.Type, action, "error")
					return
				}
			case "stop":
				if err := client.StopCluster(ctx, resource.FolderID, resource.ID); err != nil {
					log.Error().Err(err).
						Str("resource_type", resource.Type).
						Str("resource_id", resource.ID).
						Str("action", action).
						Msg("Cluster operation failed")
					metrics.IncOperation(resource.Type, action, "error")
					return
				}
			case "restart":
				if err := client.RestartCluster(ctx, resource.FolderID, resource.ID); err != nil {
					log.Error().Err(err).
						Str("resource_type", resource.Type).
						Str("resource_id", resource.ID).
						Str("action", action).
						Msg("Cluster operation failed")
					metrics.IncOperation(resource.Type, action, "error")
					return
				}
			}
			metrics.IncOperation(resource.Type, action, "success")
		default:
			log.Error().
				Str("resource_type", resource.Type).
				Str("schedule", sch.Name).
				Msg("Unsupported resource type")
			metrics.IncOperation(resource.Type, action, "error")
		}
	}
}
