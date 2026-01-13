package config

import (
	"encoding/json"
	"time"

	"github.com/invopop/jsonschema"
)

// RFC3339Time represents a time in RFC3339 format.
type RFC3339Time string

// UnmarshalYAML implements yaml.Unmarshaler interface.
func (t *RFC3339Time) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}
	if _, err := time.Parse(time.RFC3339, s); err != nil {
		return err
	}
	*t = RFC3339Time(s)
	return nil
}

// MarshalYAML implements yaml.Marshaler interface.
func (t RFC3339Time) MarshalYAML() (interface{}, error) {
	return string(t), nil
}

// UnmarshalJSON implements json.Unmarshaler interface.
func (t *RFC3339Time) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	if _, err := time.Parse(time.RFC3339, s); err != nil {
		return err
	}
	*t = RFC3339Time(s)
	return nil
}

// MarshalJSON implements json.Marshaler interface.
func (t RFC3339Time) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(t))
}

// String returns the string representation of the time.
func (t RFC3339Time) String() string {
	return string(t)
}

// Time returns the parsed time.Time value.
func (t RFC3339Time) Time() (time.Time, error) {
	return time.Parse(time.RFC3339, string(t))
}

// JSONSchema returns the JSON schema for RFC3339Time type.
func (RFC3339Time) JSONSchema() *jsonschema.Schema {
	minLen := uint64(20)
	return &jsonschema.Schema{
		Type:        "string",
		Format:      "date-time",
		Description: "Time in RFC3339 format (e.g., 2024-01-01T09:00:00Z)",
		Pattern:     `^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(Z|[+-]\d{2}:\d{2})$`,
		Examples:    []any{"2024-01-01T09:00:00Z", "2024-12-31T23:59:59+03:00"},
		MinLength:   &minLen,
	}
}
