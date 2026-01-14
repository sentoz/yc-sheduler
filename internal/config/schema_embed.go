package config

import (
	_ "embed"
	"os"
	"path/filepath"
)

// schemaData contains the JSON schema for configuration.
// It is loaded at runtime from the static/schemas/config.json file
// relative to the module root.
var schemaData []byte

func init() {
	// Try to load schema from static/schemas/config.json relative to module root
	// This works when running from the project root
	if data, err := os.ReadFile("static/schemas/config.json"); err == nil {
		schemaData = data
		return
	}
	// Fallback: try relative to current working directory
	if data, err := os.ReadFile(filepath.Join("..", "..", "static", "schemas", "config.json")); err == nil {
		schemaData = data
		return
	}
	// If both fail, schemaData will be empty and validation will fail with ErrSchemaLoad
}
