package config

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/invopop/jsonschema"
)

// Duration is a custom duration type that can be unmarshaled from strings
// like "5s", "1h", "30m", etc.
type Duration struct {
	time.Duration
}

// UnmarshalYAML implements yaml.Unmarshaler interface.
// This method works with any YAML parser that supports the standard interface.
func (d *Duration) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return fmt.Errorf("duration must be a string: %w", err)
	}

	dur, err := time.ParseDuration(s)
	if err != nil {
		return fmt.Errorf("invalid duration format: %w", err)
	}

	d.Duration = dur
	return nil
}

// MarshalYAML implements yaml.Marshaler interface.
func (d Duration) MarshalYAML() (interface{}, error) {
	return d.String(), nil
}

// UnmarshalJSON implements json.Unmarshaler interface.
func (d *Duration) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	dur, err := time.ParseDuration(s)
	if err != nil {
		return fmt.Errorf("invalid duration format: %w", err)
	}

	d.Duration = dur
	return nil
}

// MarshalJSON implements json.Marshaler interface.
func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

// JSONSchema returns the JSON schema for Duration type.
func (Duration) JSONSchema() *jsonschema.Schema {
	return &jsonschema.Schema{
		Title: "Human readable duration",
		Type:  "string",
		// Keep "duration" for tools; pattern clarifies allowed units.
		Format: "duration",
		Description: "Duration string: a positive sequence of <number><unit> tokens. " +
			"Units:\n" +
			"* `s` — seconds\n" +
			"* `m` — minutes (`60` s)\n" +
			"* `h` — hours (`60` m)\n" +
			"* `d` — days (`24` h)\n" +
			"* `w` — weeks (`7` d)\n",
		Pattern:  `^(?:\d+(?:\.\d+)?(?:s|m|h|d|w))+$`,
		Examples: []any{"2h45m", "1.5d", "2w", "90m", "18h"},
	}
}

// Std returns the standard time.Duration value.
func (d Duration) Std() time.Duration {
	return d.Duration
}
