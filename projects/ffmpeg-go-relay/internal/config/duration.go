package config

import (
	"encoding/json"
	"fmt"
	"time"
)

// Duration is a custom time.Duration that marshals/unmarshals to/from JSON strings.
// Examples: "30s", "1m", "2h", "500ms"
type Duration time.Duration

// UnmarshalJSON parses duration from JSON string format (e.g., "30s", "1m")
func (d *Duration) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	parsed, err := time.ParseDuration(s)
	if err != nil {
		return fmt.Errorf("invalid duration %q: %w", s, err)
	}

	*d = Duration(parsed)
	return nil
}

// MarshalJSON encodes duration to JSON string format
func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(d).String())
}

// String returns the string representation of the duration
func (d Duration) String() string {
	return time.Duration(d).String()
}

// AsDuration returns the underlying time.Duration
func (d Duration) AsDuration() time.Duration {
	return time.Duration(d)
}
