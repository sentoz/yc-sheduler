package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/invopop/jsonschema"
	"github.com/sentoz/yc-sheduler/internal/config"
)

func main() {
	var (
		outFile         string
		scheduleOutFile string
		modulePath      string
		prettyPrint     bool
	)
	flag.StringVar(&outFile, "out", "", "output file path (default: stdout)")
	flag.StringVar(&scheduleOutFile, "schedule-out", "", "output file path for schedule schema (default: stdout)")
	flag.StringVar(&modulePath, "module", "github.com/sentoz/yc-sheduler", "go module path (for extracting comments)")
	flag.BoolVar(&prettyPrint, "pretty", true, "pretty print JSON output")
	flag.Parse()

	// Create reflector
	r := &jsonschema.Reflector{
		AllowAdditionalProperties: false,
		DoNotReference:            false,
	}

	// Add Go comments for better documentation
	if err := r.AddGoComments(modulePath, "internal/config", jsonschema.WithFullComment()); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to add Go comments: %v\n", err)
	}

	configSchema := r.Reflect(new(config.Config))
	scheduleSchema := r.Reflect(new(config.ScheduleManifest))

	// Use draft-07 which is supported by github.com/santhosh-tekuri/jsonschema/v6.
	// Newer drafts like 2020-12 are not supported and cause metaschema validation errors.
	configSchema.Version = "http://json-schema.org/draft-07/schema#"
	configSchema.Title = "YC Scheduler Configuration"
	configSchema.Description = "Configuration schema for YC Scheduler application"

	scheduleSchema.Version = "http://json-schema.org/draft-07/schema#"
	scheduleSchema.Title = "YC Scheduler Schedule Manifest"
	scheduleSchema.Description = "Schedule manifest schema for YC Scheduler application"

	if err := writeSchema(outFile, configSchema, prettyPrint); err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to write config schema: %v\n", err)
		os.Exit(1)
	}
	if err := writeSchema(scheduleOutFile, scheduleSchema, prettyPrint); err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to write schedule schema: %v\n", err)
		os.Exit(1)
	}

	if outFile != "" {
		fmt.Fprintf(os.Stderr, "Config schema written to: %s\n", outFile)
	}
	if scheduleOutFile != "" {
		fmt.Fprintf(os.Stderr, "Schedule schema written to: %s\n", scheduleOutFile)
	}
}

func writeSchema(outFile string, schema interface{}, prettyPrint bool) error {
	output := os.Stdout
	if outFile != "" {
		if dir := filepath.Dir(outFile); dir != "" {
			if err := os.MkdirAll(dir, 0o750); err != nil {
				return fmt.Errorf("create output directory: %w", err)
			}
		}

		f, err := os.Create(outFile)
		if err != nil {
			return fmt.Errorf("create output file: %w", err)
		}
		defer func() {
			if cerr := f.Close(); cerr != nil {
				fmt.Fprintf(os.Stderr, "Error: failed to close output file %s: %v\n", outFile, cerr)
			}
		}()
		output = f
	}

	enc := json.NewEncoder(output)
	if prettyPrint {
		enc.SetIndent("", "  ")
	}
	if err := enc.Encode(schema); err != nil {
		return fmt.Errorf("encode schema: %w", err)
	}

	return nil
}
