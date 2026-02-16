package static

import _ "embed"

// ConfigSchema contains the JSON schema for configuration validation.
// It is embedded at build time from schemas/config.json.
//
//go:embed schemas/config.json
var ConfigSchema []byte

// ScheduleSchema contains the JSON schema for schedule manifest validation.
// It is embedded at build time from schemas/schedule.json.
//
//go:embed schemas/schedule.json
var ScheduleSchema []byte
