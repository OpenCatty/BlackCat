package config

import (
	"fmt"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

// Duration wraps time.Duration and supports YAML unmarshalling from both
// string format ("30m", "300s") and integer format (raw seconds).
type Duration struct {
	time.Duration
}

// UnmarshalYAML implements yaml.Unmarshaler for gopkg.in/yaml.v3.
// Accepts both string values (e.g., "30m", "300s") via time.ParseDuration
// and integer values (raw seconds).
func (d *Duration) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.ScalarNode {
		// Try to parse as string first (handles "30m", "300s", etc.)
		if value.Tag == "!!str" || value.Tag == "" {
			// Could be a string duration or a number
			parsed, err := time.ParseDuration(value.Value)
			if err == nil {
				d.Duration = parsed
				return nil
			}
		}

		// Try to parse as integer (raw seconds)
		if value.Tag == "!!int" || value.Tag == "" {
			// Attempt to convert to int64 for seconds
			seconds, err := strconv.ParseInt(value.Value, 10, 64)
			if err == nil {
				d.Duration = time.Duration(seconds) * time.Second
				return nil
			}
		}

		return fmt.Errorf("cannot unmarshal %q into Duration", value.Value)
	}

	return fmt.Errorf("cannot unmarshal non-scalar node into Duration")
}

// MarshalYAML implements yaml.Marshaler for gopkg.in/yaml.v3.
// Returns the duration as a string representation.
func (d Duration) MarshalYAML() (interface{}, error) {
	return d.String(), nil
}

// String returns the string representation of the duration.
func (d Duration) String() string {
	return d.Duration.String()
}

// IsZero reports whether d is zero.
func (d Duration) IsZero() bool {
	return d.Duration == 0
}
