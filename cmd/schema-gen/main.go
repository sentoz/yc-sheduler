package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/invopop/jsonschema"
	"github.com/woozymasta/yc-scheduler/pkg/config"
)

func main() {
	var (
		outFile     string
		modulePath  string
		prettyPrint bool
	)
	flag.StringVar(&outFile, "out", "", "output file path (default: stdout)")
	flag.StringVar(&modulePath, "module", "github.com/woozymasta/yc-scheduler", "go module path (for extracting comments)")
	flag.BoolVar(&prettyPrint, "pretty", true, "pretty print JSON output")
	flag.Parse()

	// Create reflector
	r := &jsonschema.Reflector{
		AllowAdditionalProperties: false,
		DoNotReference:            false,
	}

	// Add Go comments for better documentation
	if err := r.AddGoComments(modulePath, "pkg/config", jsonschema.WithFullComment()); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to add Go comments: %v\n", err)
	}

	// Generate schema for Config type
	schema := r.Reflect(new(config.Config))

	// Set schema metadata
	schema.Version = "https://json-schema.org/draft/2020-12/schema"
	schema.Title = "YC Scheduler Configuration"
	schema.Description = "Configuration schema for YC Scheduler application"

	// Prepare output
	var output *os.File
	if outFile != "" {
		// Create output directory if needed
		if dir := filepath.Dir(outFile); dir != "" {
			if err := os.MkdirAll(dir, 0o750); err != nil {
				fmt.Fprintf(os.Stderr, "Error: failed to create output directory: %v\n", err)
				os.Exit(1)
			}
		}

		f, err := os.Create(outFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: failed to create output file: %v\n", err)
			os.Exit(1)
		}
		defer func() {
			if cerr := f.Close(); cerr != nil {
				fmt.Fprintf(os.Stderr, "Error: failed to close output file: %v\n", cerr)
			}
		}()
		output = f
	} else {
		output = os.Stdout
	}

	// Encode and write
	enc := json.NewEncoder(output)
	if prettyPrint {
		enc.SetIndent("", "  ")
	}
	if err := enc.Encode(schema); err != nil {
		fmt.Fprintf(os.Stderr, "Error: failed to encode schema: %v\n", err)
		os.Exit(1)
	}

	if outFile != "" {
		fmt.Fprintf(os.Stderr, "Schema written to: %s\n", outFile)
	}
}
