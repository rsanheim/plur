package watch

import (
	_ "embed"
	"fmt"
	"os"

	"github.com/pelletier/go-toml"
	"github.com/rsanheim/plur/job"
)

//go:embed defaults.toml
var defaultsFile []byte

// DefaultsConfig holds all default profiles
type DefaultsConfig struct {
	Defaults map[string]DefaultProfile `toml:"defaults"`
}

// DefaultProfile represents a complete configuration profile for a project type
type DefaultProfile struct {
	Jobs    map[string]job.Job `toml:"job"`
	Watches []WatchMapping     `toml:"watch"`
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

	// Check for Ruby project with RSpec
	if fileExists("Gemfile") && dirExists("spec") {
		return "ruby"
	}

	// Check for Ruby project with Minitest
	if fileExists("Gemfile") && dirExists("test") {
		return "ruby"
	}

	// Check for Ruby project with just lib directory
	if dirExists("lib") && (dirExists("spec") || dirExists("test")) {
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

		watchesCopy := make([]WatchMapping, len(profile.Watches))
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
func GetAutodetectedDefaults() (map[string]*job.Job, []*WatchMapping) {
	profileName := AutodetectProfile()
	if profileName == "" {
		return make(map[string]*job.Job), []*WatchMapping{}
	}

	profile := GetDefaultProfile(profileName)
	if profile == nil {
		return make(map[string]*job.Job), []*WatchMapping{}
	}

	// Convert to pointer maps for consistency with rest of codebase
	jobs := make(map[string]*job.Job, len(profile.Jobs))
	for name, j := range profile.Jobs {
		jobCopy := j
		jobCopy.Name = name
		jobs[name] = &jobCopy
	}

	// Convert to pointer slice
	watches := make([]*WatchMapping, len(profile.Watches))
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
