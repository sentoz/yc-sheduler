package static

import _ "embed"

// ConfigSchema contains the JSON schema for configuration validation.
// It is embedded at build time from schemas/config.json.
//
//go:embed schemas/config.json
var ConfigSchema []byte
