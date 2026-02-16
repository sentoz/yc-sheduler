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

	// ErrScheduleSchemaLoad is returned when the embedded schedule schema cannot be loaded or compiled.
	ErrScheduleSchemaLoad = errors.New("failed to load schedule schema")

	// ErrScheduleSchemaValidation is returned when a schedule document does not match the JSON schema.
	ErrScheduleSchemaValidation = errors.New("schedule schema validation failed")
)
