package job

import (
	"strings"

	"github.com/rsanheim/plur/minitest"
	"github.com/rsanheim/plur/passthrough"
	"github.com/rsanheim/plur/rspec"
	"github.com/rsanheim/plur/types"
)

// Job represents a command to run with optional environment variables
// Used by both parallel execution (plur spec) and watch mode (plur watch)
type Job struct {
	Name          string   `toml:"-" json:"name"`
	Cmd           []string `toml:"cmd" json:"cmd"`
	Env           []string `toml:"env,omitempty" json:"env,omitempty"`
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

// GetConventionBasedTargetPattern returns a target pattern based on job name conventions
// Jobs containing "rspec" get "spec/**/*_spec.rb", jobs containing "minitest" get "test/**/*_test.rb"
// Returns empty string if no convention matches
func (j Job) GetConventionBasedTargetPattern() string {
	// Apply conventions based on job name (case-insensitive)
	nameLower := strings.ToLower(j.Name)
	if strings.Contains(nameLower, "rspec") {
		return "spec/**/*_spec.rb"
	}
	if strings.Contains(nameLower, "minitest") {
		return "test/**/*_test.rb"
	}

	return ""
}

// GetTargetPattern returns the glob pattern for file discovery
// Falls back to convention-based pattern if not explicitly set
func (j Job) GetTargetPattern() string {
	if j.TargetPattern != "" {
		return j.TargetPattern
	}
	return j.GetConventionBasedTargetPattern()
}

// GetTargetSuffix extracts the file suffix from the target pattern
// Used by ExpandGlobPatterns when user passes a directory: "spec/models" → "spec/models/**/*_spec.rb"
//
// Examples:
//
//	"spec/**/*_spec.rb" → "_spec.rb"
//	"test/**/*_test.rb" → "_test.rb"
func (j Job) GetTargetSuffix() string {
	pattern := j.GetTargetPattern()
	if pattern == "" {
		return ""
	}

	// Find the last * in the pattern
	lastStar := strings.LastIndex(pattern, "*")
	if lastStar == -1 {
		return ""
	}

	// Get everything after the last *
	suffix := pattern[lastStar+1:]

	// Validate it looks like a test suffix (contains _ and .)
	if strings.Contains(suffix, "_") && strings.Contains(suffix, ".") {
		return suffix
	}

	return ""
}

// CreateParser creates the appropriate test output parser for this job
// Returns passthrough parser for custom jobs (non-rspec/minitest)
func (j Job) CreateParser() (types.TestOutputParser, error) {
	switch j.Name {
	case "rspec":
		return rspec.NewOutputParser(), nil
	case "minitest":
		return minitest.NewOutputParser(), nil
	default:
		return passthrough.NewOutputParser(), nil
	}
}

// IsMinitestStyle returns true if this job is minitest-style (for formatting decisions)
func (j Job) IsMinitestStyle() bool {
	return j.Name == "minitest"
}
