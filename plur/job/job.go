package job

import "strings"

// Job represents a command to run with optional environment variables
// Used by both parallel execution (plur spec) and watch mode (plur watch)
type Job struct {
	Name          string   `toml:"-" json:"name"`
	Cmd           []string `toml:"cmd" json:"cmd"`
	Env           []string `toml:"env,omitempty" json:"env,omitempty"`
	Framework     string   `toml:"framework,omitempty" json:"framework,omitempty"`
	TargetPattern string   `toml:"target_pattern,omitempty" json:"target_pattern,omitempty"` // Glob pattern for file discovery (e.g., "spec/**/*_spec.rb")
}

// BuildJobCmd builds the command array for a job with specific targets
// If the job.Cmd contains a {{target}} token, it replaces it with all targets
// Otherwise, it appends the targets as the last arguments
//
// Examples:
//
//	BuildJobCmd(job, []string{"spec/foo.rb", "spec/bar.rb"})
//	  with Cmd = ["bundle", "exec", "rspec", "{{target}}"]
//	  → ["bundle", "exec", "rspec", "spec/foo.rb", "spec/bar.rb"]
//
//	BuildJobCmd(job, []string{"test1.rb", "test2.rb"})
//	  with Cmd = ["my-runner", "--file={{target}}"]
//	  → ["my-runner", "--file=test1.rb", "--file=test2.rb"]
func BuildJobCmd(job Job, targets []string) []string {
	result := []string{}
	foundToken := false

	for _, part := range job.Cmd {
		if part == "{{target}}" {
			// Replace entire {{target}} element with all targets
			result = append(result, targets...)
			foundToken = true
		} else if strings.Contains(part, "{{target}}") {
			// Token is part of a string (e.g., "--file={{target}}")
			// Expand to multiple args: ["--file=spec/foo.rb", "--file=spec/bar.rb"]
			for _, target := range targets {
				result = append(result, strings.ReplaceAll(part, "{{target}}", target))
			}
			foundToken = true
		} else {
			result = append(result, part)
		}
	}

	// If no {{target}} token found, append all targets at end
	if !foundToken {
		result = append(result, targets...)
	}

	return result
}

// BuildJobAllCmd builds the command for running a job without a specific target
// This removes any {{target}} tokens from the command
func BuildJobAllCmd(job Job) []string {
	result := []string{}

	for _, part := range job.Cmd {
		if !strings.Contains(part, "{{target}}") {
			result = append(result, part)
		}
	}

	return result
}

// GetTargetPattern returns the glob pattern for file discovery
func (j Job) GetTargetPattern() string {
	return j.TargetPattern
}

// UsesTargets returns true if the job command expects target files
// (i.e., contains {{target}} placeholder)
func (j Job) UsesTargets() bool {
	for _, part := range j.Cmd {
		if strings.Contains(part, "{{target}}") {
			return true
		}
	}
	return false
}
