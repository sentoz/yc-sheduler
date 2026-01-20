package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jessevdk/go-flags"
	"github.com/rs/zerolog/log"

	"github.com/sentoz/yc-sheduler/internal/app"
	"github.com/sentoz/yc-sheduler/internal/config"
	"github.com/sentoz/yc-sheduler/internal/logger"
	"github.com/sentoz/yc-sheduler/internal/signals"
	"github.com/sentoz/yc-sheduler/internal/vars"
	"github.com/sentoz/yc-sheduler/internal/yc"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	var opts struct {
		Version bool   `long:"version" description:"Print version information and exit"`
		Config  string `short:"c" long:"config" env:"YC_SHEDULER_CONFIG" description:"Path to configuration file (YAML or JSON)"`
		Token   string `short:"t" long:"token" env:"YC_TOKEN" description:"Yandex Cloud OAuth/IAM token (discouraged; prefer --sa-key)"`
		SaKey   string `long:"sa-key" env:"YC_SA_KEY_FILE" description:"Path to Yandex Cloud service account key JSON file (preferred)"`
		DryRun  bool   `short:"n" long:"dry-run" description:"Dry run mode: log planned actions without calling YC APIs"`

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

	// Handle --version flag
	if opts.Version {
		vars.Print()
		return nil
	}

	// Validate that config is provided when not using --version
	if opts.Config == "" {
		return fmt.Errorf("--config is required")
	}

	opts.Setup()

	log.Debug().
		Str("config_path", opts.Config).
		Bool("dry_run", opts.DryRun).
		Msg("CLI options parsed")

	cfg, err := config.Load(context.Background(), opts.Config)
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

	// Create and initialize application
	application, err := app.New(cfg, client, opts.DryRun)
	if err != nil {
		return fmt.Errorf("yc-scheduler: create app: %w", err)
	}

	defer func() {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout.Std())
		defer shutdownCancel()
		if err := application.Shutdown(shutdownCtx); err != nil {
			log.Warn().Err(err).Msg("Failed to shutdown application gracefully")
		}
	}()

	// Run application
	return application.Run(ctx)
}
