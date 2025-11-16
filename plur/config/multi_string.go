package config

import (
	"encoding/json"
	"fmt"
)

// MultiString allows single string or array in TOML configuration
// This enables both: jobs = "rspec" and jobs = ["rspec", "lint"]
type MultiString []string

// UnmarshalTOML implements custom TOML unmarshaling for MultiString
// Used when go-toml directly unmarshals (e.g., loading embedded defaults.toml)
func (ms *MultiString) UnmarshalTOML(value any) error {
	switch v := value.(type) {
	case string:
		*ms = []string{v}
		return nil
	case []any:
		strs := make([]string, len(v))
		for i, item := range v {
			str, ok := item.(string)
			if !ok {
				return fmt.Errorf("expected string in array, got %T", item)
			}
			strs[i] = str
		}
		*ms = strs
		return nil
	default:
		return fmt.Errorf("expected string or array of strings, got %T", v)
	}
}

// UnmarshalJSON implements json.Unmarshaler for Kong's type conversion
// Delegates to UnmarshalTOML to keep conversion logic in one place
func (ms *MultiString) UnmarshalJSON(data []byte) error {
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	return ms.UnmarshalTOML(value)
}

// Slice returns a copy of the underlying string slice
func (ms MultiString) Slice() []string {
	return append([]string(nil), ms...)
}
