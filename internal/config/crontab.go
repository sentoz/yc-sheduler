package config

import (
	"encoding/json"

	"github.com/invopop/jsonschema"
)

// Crontab represents a cron expression.
type Crontab string

// UnmarshalYAML implements yaml.Unmarshaler interface.
func (c *Crontab) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}
	*c = Crontab(s)
	return nil
}

// MarshalYAML implements yaml.Marshaler interface.
func (c Crontab) MarshalYAML() (interface{}, error) {
	return string(c), nil
}

// UnmarshalJSON implements json.Unmarshaler interface.
func (c *Crontab) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	*c = Crontab(s)
	return nil
}

// MarshalJSON implements json.Marshaler interface.
func (c Crontab) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(c))
}

// String returns the string representation of the crontab.
func (c Crontab) String() string {
	return string(c)
}

// JSONSchema returns the JSON schema for Crontab type.
func (Crontab) JSONSchema() *jsonschema.Schema {
	minLen := uint64(1)
	return &jsonschema.Schema{
		Type:        "string",
		Description: "Cron expression (5 or 6 fields: minute hour day month weekday [second])",
		Pattern:     `^(\S+\s+){4,5}\S+$`,
		Examples:    []any{"0 9 * * *", "0 0 * * 0", "*/5 * * * *"},
		MinLength:   &minLen,
	}
}
