package watch

import (
	"fmt"
	"strings"
)

// Job represents a command to run with optional environment variables
type Job struct {
	Name string   `toml:"-" json:"name"`
	Cmd  []string `toml:"cmd" json:"cmd"`
	Env  []string `toml:"env,omitempty" json:"env,omitempty"`
}

// WatchMapping represents a source->target mapping rule for watch mode
type WatchMapping struct {
	Name    string       `toml:"name,omitempty" json:"name,omitempty"`
	Source  string       `toml:"source" json:"source"`
	Targets *MultiString `toml:"targets,omitempty" json:"targets,omitempty"`
	Jobs    MultiString  `toml:"jobs" json:"jobs"`
	Exclude []string     `toml:"exclude,omitempty" json:"exclude,omitempty"`
}

// MultiString allows single string or array in TOML configuration
// This enables both: jobs = "rspec" and jobs = ["rspec", "lint"]
type MultiString []string

// UnmarshalTOML implements custom TOML unmarshaling for MultiString
// It accepts both a single string and an array of strings
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

// Slice returns a copy of the underlying string slice
func (ms MultiString) Slice() []string {
	return append([]string(nil), ms...)
}

// BuildJobCmd builds the command array for a job with a specific target
// If the job.Cmd contains a {{target}} token, it replaces it
// Otherwise, it appends the target as the last argument
func BuildJobCmd(job *Job, target string) []string {
	hasToken := false
	result := make([]string, len(job.Cmd))

	for i, part := range job.Cmd {
		if strings.Contains(part, "{{target}}") {
			result[i] = strings.ReplaceAll(part, "{{target}}", target)
			hasToken = true
		} else {
			result[i] = part
		}
	}

	// If no {{target}} token found, append target as last arg
	if !hasToken {
		result = append(result, target)
	}

	return result
}

// BuildJobAllCmd builds the command for running a job without a specific target
// This removes any {{target}} tokens from the command
func BuildJobAllCmd(job *Job) []string {
	result := []string{}

	for _, part := range job.Cmd {
		if !strings.Contains(part, "{{target}}") {
			result = append(result, part)
		}
	}

	return result
}
