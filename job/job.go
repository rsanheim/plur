package job

// Job represents a command to run with optional environment variables
// Used by both parallel execution (plur spec) and watch mode (plur watch)
type Job struct {
	Name            string   `toml:"-" json:"name"`
	Cmd             []string `toml:"cmd" json:"cmd"`
	Env             []string `toml:"env,omitempty" json:"env,omitempty"`
	FrameworkName   string   `toml:"framework,omitempty" json:"framework,omitempty"`
	TargetPattern   string   `toml:"target_pattern,omitempty" json:"target_pattern,omitempty"`     // Glob pattern for file discovery (e.g., "spec/**/*_spec.rb")
	ExcludePatterns []string `toml:"exclude_patterns,omitempty" json:"exclude_patterns,omitempty"` // Glob patterns to exclude during file discovery
}

// BuildJobCmd builds the command array for a job with target arguments appended.
//
// Examples:
//
//	BuildJobCmd(job, []string{"spec/foo.rb", "spec/bar.rb"})
//	  with Cmd = ["bundle", "exec", "rspec"]
//	  → ["bundle", "exec", "rspec", "spec/foo.rb", "spec/bar.rb"]
func BuildJobCmd(job Job, targets []string) []string {
	result := append([]string{}, job.Cmd...)
	result = append(result, targets...)
	return result
}
