package config

import "errors"

// Named errors used by the configuration loader.
var (
	// ErrConfigNotFound is returned when the configuration file does not exist.
	ErrConfigNotFound = errors.New("config file not found")

	// ErrInvalidConfig is returned when the configuration cannot be parsed or validated.
	ErrInvalidConfig = errors.New("invalid configuration")

	// ErrSchemaLoad is returned when the embedded JSON schema cannot be loaded or compiled.
	ErrSchemaLoad = errors.New("failed to load configuration schema")

	// ErrSchemaValidation is returned when configuration does not match the JSON schema.
	ErrSchemaValidation = errors.New("configuration schema validation failed")
)
