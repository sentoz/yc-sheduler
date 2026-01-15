package config

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/creasty/defaults"
	"github.com/rs/zerolog/log"
	jschema "github.com/santhosh-tekuri/jsonschema/v6"
	jamle "github.com/woozymasta/jamle"

	pkgconfig "github.com/woozymasta/yc-scheduler/pkg/config"
	"github.com/woozymasta/yc-scheduler/static"
)

var (
	schemaOnce sync.Once
	schema     *jschema.Schema
	schemaErr  error
)

// getSchema lazily compiles the embedded JSON schema and returns it.
func getSchema() (*jschema.Schema, error) {
	schemaOnce.Do(func() {
		if len(static.ConfigSchema) == 0 {
			schemaErr = ErrSchemaLoad
			return
		}
		compiler := jschema.NewCompiler()

		const schemaURL = "embedded://config-schema"

		// static.ConfigSchema contains raw JSON bytes. AddResource expects a valid
		// JSON value (map, slice, etc.), not raw []byte. We need to unmarshal it first.
		var schemaDoc interface{}
		if err := json.Unmarshal(static.ConfigSchema, &schemaDoc); err != nil {
			schemaErr = fmt.Errorf("%w: unmarshal schema: %v", ErrSchemaLoad, err)
			return
		}

		if err := compiler.AddResource(schemaURL, schemaDoc); err != nil {
			schemaErr = fmt.Errorf("%w: add resource: %v", ErrSchemaLoad, err)
			return
		}

		schema, schemaErr = compiler.Compile(schemaURL)
		if schemaErr != nil {
			schemaErr = fmt.Errorf("%w: compile: %v", ErrSchemaLoad, schemaErr)
		}
	})

	return schema, schemaErr
}

// Load reads, parses and validates configuration from the given path.
// The path must point to a YAML or JSON file. Environment variables inside
// the configuration are expanded by jamle.
func Load(_ context.Context, path string) (*pkgconfig.Config, error) {
	if path == "" {
		return nil, fmt.Errorf("%w: empty path", ErrConfigNotFound)
	}

	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("%w: %s", ErrConfigNotFound, path)
		}
		return nil, fmt.Errorf("stat config %q: %w", path, err)
	}

	if info.IsDir() {
		return nil, fmt.Errorf("%w: %s is a directory, expected file", ErrInvalidConfig, path)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %q: %w", path, err)
	}

	var cfg pkgconfig.Config
	if err := jamle.Unmarshal(raw, &cfg); err != nil {
		return nil, fmt.Errorf("%w: decode: %v", ErrInvalidConfig, err)
	}

	// Apply default values for fields that weren't set in the config.
	if err := defaults.Set(&cfg); err != nil {
		return nil, fmt.Errorf("%w: apply defaults: %v", ErrInvalidConfig, err)
	}

	if err := validate(&cfg); err != nil {
		return nil, err
	}

	log.Info().
		Str("config_path", path).
		Int("schedules", len(cfg.Schedules)).
		Msg("Configuration loaded and validated")

	return &cfg, nil
}

// validate checks configuration against the embedded JSON schema and
// returns a wrapped ErrSchemaValidation on failure.
func validate(cfg *pkgconfig.Config) error {
	schema, err := getSchema()
	if err != nil {
		return err
	}

	// Marshal config to JSON, then unmarshal to interface{} so Validate receives
	// a valid JSON value (map/slice), not *bytes.Reader.
	data, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("%w: marshal for validation: %v", ErrInvalidConfig, err)
	}

	var cfgDoc interface{}
	if err := json.Unmarshal(data, &cfgDoc); err != nil {
		return fmt.Errorf("%w: unmarshal for validation: %v", ErrInvalidConfig, err)
	}

	if err := schema.Validate(cfgDoc); err != nil {
		return fmt.Errorf("%w: %v", ErrSchemaValidation, err)
	}

	return nil
}
