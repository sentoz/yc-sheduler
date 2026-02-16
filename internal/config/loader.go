package config

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/creasty/defaults"
	"github.com/rs/zerolog/log"
	jschema "github.com/santhosh-tekuri/jsonschema/v6"
	jamle "github.com/woozymasta/jamle"
	"gopkg.in/yaml.v3"

	"github.com/sentoz/yc-sheduler/static"
)

var (
	schemaOnce         sync.Once
	schema             *jschema.Schema
	schemaErr          error
	scheduleSchemaOnce sync.Once
	scheduleSchema     *jschema.Schema
	scheduleSchemaErr  error
)

// getSchema lazily compiles the embedded JSON schema and returns it.
func getSchema() (*jschema.Schema, error) {
	schemaOnce.Do(func() {
		schema, schemaErr = compileEmbeddedSchema(static.ConfigSchema, "embedded://config-schema", ErrSchemaLoad)
	})

	return schema, schemaErr
}

// getScheduleSchema lazily compiles the embedded schedule schema and returns it.
func getScheduleSchema() (*jschema.Schema, error) {
	scheduleSchemaOnce.Do(func() {
		scheduleSchema, scheduleSchemaErr = compileEmbeddedSchema(static.ScheduleSchema, "embedded://schedule-schema", ErrScheduleSchemaLoad)
	})

	return scheduleSchema, scheduleSchemaErr
}

func compileEmbeddedSchema(raw []byte, schemaURL string, loadErr error) (*jschema.Schema, error) {
	if len(raw) == 0 {
		return nil, loadErr
	}

	compiler := jschema.NewCompiler()

	// Embedded schema contains raw JSON bytes. AddResource expects a decoded JSON value.
	var schemaDoc interface{}
	if err := json.Unmarshal(raw, &schemaDoc); err != nil {
		return nil, fmt.Errorf("%w: unmarshal schema: %v", loadErr, err)
	}

	if err := compiler.AddResource(schemaURL, schemaDoc); err != nil {
		return nil, fmt.Errorf("%w: add resource: %v", loadErr, err)
	}

	schema, err := compiler.Compile(schemaURL)
	if err != nil {
		return nil, fmt.Errorf("%w: compile: %v", loadErr, err)
	}

	return schema, nil
}

// Load reads, parses and validates configuration from the given path.
// The path must point to a YAML or JSON file. Environment variables inside
// the configuration are expanded by jamle.
func Load(_ context.Context, path string) (*Config, error) {
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

	var cfg Config
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

	schedulesDir := cfg.SchedulesDir
	if !filepath.IsAbs(schedulesDir) {
		schedulesDir = filepath.Join(filepath.Dir(path), schedulesDir)
	}

	schedules, err := loadSchedules(schedulesDir)
	if err != nil {
		return nil, err
	}
	cfg.SchedulesDir = schedulesDir
	cfg.Schedules = schedules

	log.Info().
		Str("config_path", path).
		Str("schedules_dir", cfg.SchedulesDir).
		Int("schedules", len(cfg.Schedules)).
		Msg("Configuration and schedules loaded and validated")

	return &cfg, nil
}

// LoadSchedules reads and validates schedule manifests from a directory.
func LoadSchedules(_ context.Context, path string) ([]Schedule, error) {
	if path == "" {
		return nil, fmt.Errorf("%w: empty schedules directory path", ErrConfigNotFound)
	}

	return loadSchedules(path)
}

// validate checks configuration against the embedded JSON schema and
// returns a wrapped ErrSchemaValidation on failure.
func validate(cfg *Config) error {
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

func loadSchedules(path string) ([]Schedule, error) {
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("%w: schedules directory not found: %s", ErrConfigNotFound, path)
		}
		return nil, fmt.Errorf("stat schedules dir %q: %w", path, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%w: %s is not a directory", ErrInvalidConfig, path)
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("read schedules dir %q: %w", path, err)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	schedules := make([]Schedule, 0, len(entries))
	names := make(map[string]string, len(entries))
	parsedFiles := 0

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext != ".yaml" && ext != ".yml" {
			continue
		}

		filePath := filepath.Join(path, entry.Name())
		raw, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("read schedule file %q: %w", filePath, err)
		}

		fileSchedules, err := parseScheduleFile(raw, filePath)
		if err != nil {
			return nil, err
		}
		parsedFiles++

		for _, sch := range fileSchedules {
			if prev, exists := names[sch.Name]; exists {
				return nil, fmt.Errorf("%w: duplicate schedule name %q in %s and %s", ErrInvalidConfig, sch.Name, prev, filePath)
			}
			names[sch.Name] = filePath
			schedules = append(schedules, sch)
		}
	}

	if parsedFiles == 0 {
		return nil, fmt.Errorf("%w: no YAML schedule files found in %s", ErrInvalidConfig, path)
	}
	if len(schedules) == 0 {
		return nil, fmt.Errorf("%w: no schedule documents found in %s", ErrInvalidConfig, path)
	}

	return schedules, nil
}

func parseScheduleFile(raw []byte, path string) ([]Schedule, error) {
	schema, err := getScheduleSchema()
	if err != nil {
		return nil, err
	}

	decoder := yaml.NewDecoder(bytes.NewReader(raw))
	schedules := make([]Schedule, 0, 1)
	docIndex := 0

	for {
		docIndex++
		var node yaml.Node
		if err := decoder.Decode(&node); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, fmt.Errorf("%w: decode YAML document %d in %s: %v", ErrInvalidConfig, docIndex, path, err)
		}

		if node.Kind == 0 || len(node.Content) == 0 {
			continue
		}

		var doc interface{}
		if err := node.Decode(&doc); err != nil {
			return nil, fmt.Errorf("%w: decode document %d in %s: %v", ErrInvalidConfig, docIndex, path, err)
		}
		if doc == nil {
			continue
		}

		if err := schema.Validate(doc); err != nil {
			return nil, fmt.Errorf("%w: document %d in %s: %v", ErrScheduleSchemaValidation, docIndex, path, err)
		}

		docBytes, err := yaml.Marshal(doc)
		if err != nil {
			return nil, fmt.Errorf("%w: marshal document %d in %s: %v", ErrInvalidConfig, docIndex, path, err)
		}

		var manifest ScheduleManifest
		if err := jamle.Unmarshal(docBytes, &manifest); err != nil {
			return nil, fmt.Errorf("%w: unmarshal document %d in %s: %v", ErrInvalidConfig, docIndex, path, err)
		}

		schedules = append(schedules, manifest.ToSchedule())
	}

	return schedules, nil
}
