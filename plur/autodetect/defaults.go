package autodetect

import (
	_ "embed"
	"fmt"
	"os"
	"strings"

	"github.com/pelletier/go-toml"
	"github.com/rsanheim/plur/job"
	"github.com/rsanheim/plur/watch"
)

//go:embed defaults.toml
var defaultsFile []byte

// DefaultsConfig holds all default profiles
type DefaultsConfig struct {
	Defaults map[string]DefaultProfile `toml:"defaults"`
}

// DefaultProfile represents a complete configuration profile for a project type
type DefaultProfile struct {
	Jobs    map[string]job.Job   `toml:"job"`
	Watches []watch.WatchMapping `toml:"watch"`
}

var builtinDefaults DefaultsConfig

func init() {
	if err := toml.Unmarshal(defaultsFile, &builtinDefaults); err != nil {
		panic(fmt.Errorf("failed to load embedded defaults: %w", err))
	}
}

// AutodetectProfile determines the best default profile based on project structure
// Returns the profile name (e.g., "ruby", "go") or empty string if no match
func AutodetectProfile() string {
	// Check for Go project
	if fileExists("go.mod") {
		return "go"
	}

	// Check for Ruby project - be permissive for backward compatibility
	// Accept any of: Gemfile, spec/, test/, or lib/ directory
	if fileExists("Gemfile") || dirExists("spec") || dirExists("test") || dirExists("lib") {
		return "ruby"
	}

	return ""
}

// GetDefaultProfile returns the default profile for a given name
// Returns nil if the profile doesn't exist
func GetDefaultProfile(name string) *DefaultProfile {
	if profile, exists := builtinDefaults.Defaults[name]; exists {
		// Deep copy to avoid modifications to builtin defaults
		jobsCopy := make(map[string]job.Job, len(profile.Jobs))
		for k, v := range profile.Jobs {
			jobsCopy[k] = v
		}

		watchesCopy := make([]watch.WatchMapping, len(profile.Watches))
		copy(watchesCopy, profile.Watches)

		return &DefaultProfile{
			Jobs:    jobsCopy,
			Watches: watchesCopy,
		}
	}
	return nil
}

// GetAutodetectedDefaults returns jobs and watches for the autodetected project type
// Returns empty maps/slices if no profile is detected
func GetAutodetectedDefaults() (map[string]*job.Job, []*watch.WatchMapping) {
	profileName := AutodetectProfile()
	if profileName == "" {
		return make(map[string]*job.Job), []*watch.WatchMapping{}
	}

	profile := GetDefaultProfile(profileName)
	if profile == nil {
		return make(map[string]*job.Job), []*watch.WatchMapping{}
	}

	// Convert to pointer maps for consistency with rest of codebase
	jobs := make(map[string]*job.Job, len(profile.Jobs))
	for name, j := range profile.Jobs {
		jobCopy := j
		jobCopy.Name = name
		jobs[name] = &jobCopy
	}

	// Convert to pointer slice
	watches := make([]*watch.WatchMapping, len(profile.Watches))
	for i := range profile.Watches {
		watches[i] = &profile.Watches[i]
	}

	return jobs, watches
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// dirExists checks if a directory exists
func dirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// DetectFramework intelligently detects the test framework based on:
// 1. File patterns (if provided) - infers from suffixes like *_spec.rb or *_test.rb
// 2. Directory structure (spec/ vs test/)
// 3. Profile autodetection (Gemfile, go.mod, etc.)
//
// Returns: jobName, *job.Job, wasInferredFromFiles, error
func DetectFramework(patterns []string) (string, *job.Job, bool, error) {
	// Step 1: Try to infer from file patterns if provided
	if len(patterns) > 0 {
		frameworkFromFiles := inferFrameworkFromPatterns(patterns)
		if frameworkFromFiles != "" {
			// Get the default job for this framework from the "ruby" profile
			// This works even without Gemfile/spec/test directories
			rubyProfile := GetDefaultProfile("ruby")
			if rubyProfile != nil {
				if j, exists := rubyProfile.Jobs[frameworkFromFiles]; exists {
					jobCopy := j
					jobCopy.Name = frameworkFromFiles
					return frameworkFromFiles, &jobCopy, true, nil
				}
			}
		}
	}

	// Step 2: Try autodetection based on current directory
	autodetectedJobs, _ := GetAutodetectedDefaults()

	// Smart framework selection based on directory structure
	// If only spec/ exists, use RSpec
	// If only test/ exists, use Minitest
	// If both exist, prefer RSpec (more common in modern Ruby projects)
	hasSpecDir := dirExists("spec")
	hasTestDir := dirExists("test")

	var jobName string
	var currentJob *job.Job

	if hasSpecDir && !hasTestDir {
		// Only spec/ directory - use RSpec
		if j, exists := autodetectedJobs["rspec"]; exists {
			jobName = "rspec"
			currentJob = j
		}
	} else if hasTestDir && !hasSpecDir {
		// Only test/ directory - use Minitest
		if j, exists := autodetectedJobs["minitest"]; exists {
			jobName = "minitest"
			currentJob = j
		}
	} else {
		// Both exist or neither exist - use priority order: rspec > minitest > other
		if j, exists := autodetectedJobs["rspec"]; exists {
			jobName = "rspec"
			currentJob = j
		} else if j, exists := autodetectedJobs["minitest"]; exists {
			jobName = "minitest"
			currentJob = j
		} else {
			// Fall back to any job with a target_pattern
			for name, j := range autodetectedJobs {
				if j.GetTargetPattern() != "" {
					jobName = name
					currentJob = j
					break
				}
			}
		}
	}

	if currentJob == nil {
		return "", nil, false, fmt.Errorf("no test framework detected. Please create a .plur.toml with a job configuration")
	}

	return jobName, currentJob, false, nil
}

// inferFrameworkFromPatterns examines file patterns to infer the framework
// Returns "rspec", "minitest", or "" if unable to determine
func inferFrameworkFromPatterns(patterns []string) string {
	if len(patterns) == 0 {
		return ""
	}

	hasRSpecFiles := false
	hasMinitestFiles := false

	for _, pattern := range patterns {
		// Skip glob patterns and directories - only check actual files
		if containsGlobChars(pattern) || dirExists(pattern) {
			continue
		}

		// Check if it's a file that exists
		if !fileExists(pattern) {
			continue
		}

		// Check suffix
		if len(pattern) >= 8 && pattern[len(pattern)-8:] == "_spec.rb" {
			hasRSpecFiles = true
		} else if len(pattern) >= 8 && pattern[len(pattern)-8:] == "_test.rb" {
			hasMinitestFiles = true
		}
	}

	// Only infer if all files are of one type
	if hasRSpecFiles && !hasMinitestFiles {
		return "rspec"
	}
	if hasMinitestFiles && !hasRSpecFiles {
		return "minitest"
	}

	// Mixed or unclear - don't infer
	return ""
}

// containsGlobChars checks if a string contains glob characters
func containsGlobChars(s string) bool {
	return strings.Contains(s, "*") || strings.Contains(s, "?") || strings.Contains(s, "[")
}
