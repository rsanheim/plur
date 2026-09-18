package framework

import (
	"fmt"
)

// Job represents a command to run with optional environment variables
// Used by both parallel execution (plur spec) and watch mode (plur watch)
type Job struct {
	Name            string    `toml:"-" json:"name"`
	Cmd             []string  `toml:"cmd" json:"cmd"`
	Env             []string  `toml:"env,omitempty" json:"env,omitempty"`
	FrameworkName   string    `toml:"framework,omitempty" json:"framework,omitempty"`
	Framework       Framework `toml:"-" json:"-"`
	TargetPattern   string    `toml:"target_pattern,omitempty" json:"target_pattern,omitempty"`     // Glob pattern for file discovery (e.g., "spec/**/*_spec.rb")
	ExcludePatterns []string  `toml:"exclude_patterns,omitempty" json:"exclude_patterns,omitempty"` // Glob patterns to exclude during file discovery
}

func (j Job) TargetPatterns() ([]string, error) {
	if j.TargetPattern != "" {
		return []string{j.TargetPattern}, nil
	}
	fw := j.Framework
	if fw.Name == "" {
		return nil, fmt.Errorf("job %q has no resolved framework", j.Name)
	}
	if len(fw.DetectPatterns) == 0 {
		return nil, fmt.Errorf("job %q has no target_pattern and framework %q has no detect patterns", j.Name, fw.Name)
	}
	return fw.DetectPatterns, nil
}
