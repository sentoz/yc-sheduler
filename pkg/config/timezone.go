package config

import (
	"encoding/json"

	"github.com/invopop/jsonschema"
)

// Timezone represents an IANA timezone name.
type Timezone string

// UnmarshalYAML implements yaml.Unmarshaler interface.
func (t *Timezone) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}
	*t = Timezone(s)
	return nil
}

// MarshalYAML implements yaml.Marshaler interface.
func (t Timezone) MarshalYAML() (interface{}, error) {
	return string(t), nil
}

// UnmarshalJSON implements json.Unmarshaler interface.
func (t *Timezone) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	*t = Timezone(s)
	return nil
}

// MarshalJSON implements json.Marshaler interface.
func (t Timezone) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(t))
}

// String returns the string representation of the timezone.
func (t Timezone) String() string {
	return string(t)
}

// JSONSchema returns the JSON schema for Timezone type.
func (Timezone) JSONSchema() *jsonschema.Schema {
	return &jsonschema.Schema{
		Type:        "string",
		Description: "IANA timezone name (e.g., Europe/Moscow, America/New_York, UTC)",
		Examples:    []any{"UTC", "Europe/Moscow", "America/New_York", "Asia/Tokyo"},
	}
}
