package config

import (
	"encoding/json"

	"github.com/invopop/jsonschema"
)

// Time represents a time of day in HH:MM or HH:MM:SS format.
type Time string

// UnmarshalYAML implements yaml.Unmarshaler interface.
func (t *Time) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}
	*t = Time(s)
	return nil
}

// MarshalYAML implements yaml.Marshaler interface.
func (t Time) MarshalYAML() (interface{}, error) {
	return string(t), nil
}

// UnmarshalJSON implements json.Unmarshaler interface.
func (t *Time) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	*t = Time(s)
	return nil
}

// MarshalJSON implements json.Marshaler interface.
func (t Time) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(t))
}

// String returns the string representation of the time.
func (t Time) String() string {
	return string(t)
}

// JSONSchema returns the JSON schema for Time type.
func (Time) JSONSchema() *jsonschema.Schema {
	minLen := uint64(5)
	return &jsonschema.Schema{
		Type:        "string",
		Description: "Time of day in HH:MM or HH:MM:SS format",
		Pattern:     `^([0-1][0-9]|2[0-3]):[0-5][0-9](:[0-5][0-9])?$`,
		Examples:    []any{"09:00", "23:59", "12:30:45"},
		MinLength:   &minLen,
	}
}
