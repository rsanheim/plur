package autodetect

import (
	_ "embed"
	"fmt"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/pelletier/go-toml"
	"github.com/rsanheim/plur/internal/fsutil"
	"github.com/rsanheim/plur/job"
	"github.com/rsanheim/plur/watch"
)

//go:embed defaults.toml
var defaultsFile []byte

// DefaultsConfig holds embedded default jobs and watches (flat structure)
type DefaultsConfig struct {
	Defaults struct {
		Jobs    map[string]job.Job   `toml:"job"`
		Watches []watch.WatchMapping `toml:"watch"`
	} `toml:"defaults"`
}

var builtinDefaults DefaultsConfig

func init() {
	if err := toml.Unmarshal(defaultsFile, &builtinDefaults); err != nil {
		panic(fmt.Errorf("failed to load embedded defaults: %w", err))
	}
}

// ResolveJobResult contains the resolved job and metadata
type ResolveJobResult struct {
	Job         job.Job
	Name        string
	WasInferred bool                 // true if inferred from file patterns
	Watches     []watch.WatchMapping // watches that reference this job
}

// ResolveJob determines which job to use based on explicit selection or autodetection.
//
// Resolution order:
//  1. If explicitName provided → look up in userJobs, then built-in defaults
//  2. If patterns provided → infer from file suffixes (_spec.rb, _test.rb)
//  3. Autodetect → check which jobs have matching files (priority: rspec > minitest > go-test)
func ResolveJob(explicitName string, userJobs map[string]job.Job, patterns []string) (*ResolveJobResult, error) {
	// 1. If explicit name provided, look it up (user config first, then defaults)
	if explicitName != "" {
		return resolveExplicitJob(explicitName, userJobs)
	}

	// 2. If file patterns provided, infer from suffixes
	if len(patterns) > 0 {
		if result := resolveFromPatterns(patterns); result != nil {
			return result, nil
		}
	}

	// 3. Autodetect from file system (with priority order)
	return autodetectJob()
}

func resolveExplicitJob(name string, userJobs map[string]job.Job) (*ResolveJobResult, error) {
	// Check user config first
	if j, exists := userJobs[name]; exists {
		j.Name = name
		return &ResolveJobResult{Job: j, Name: name, Watches: getWatchesForJob(name)}, nil
	}
	// Fall back to built-in defaults
	if j, exists := builtinDefaults.Defaults.Jobs[name]; exists {
		j.Name = name
		return &ResolveJobResult{Job: j, Name: name, Watches: getWatchesForJob(name)}, nil
	}
	return nil, buildJobNotFoundError(name, userJobs)
}

func resolveFromPatterns(patterns []string) *ResolveJobResult {
	framework := inferFrameworkFromPatterns(patterns)
	if framework == "" {
		return nil
	}
	if j, exists := builtinDefaults.Defaults.Jobs[framework]; exists {
		j.Name = framework
		return &ResolveJobResult{Job: j, Name: framework, WasInferred: true, Watches: getWatchesForJob(framework)}
	}
	return nil
}

func autodetectJob() (*ResolveJobResult, error) {
	// Explicit priority order: rspec > minitest > go-test
	priority := []string{"rspec", "minitest", "go-test"}

	// First pass: check for actual test files
	for _, name := range priority {
		j, exists := builtinDefaults.Defaults.Jobs[name]
		if !exists || j.TargetPattern == "" {
			continue
		}
		matches, _ := doublestar.FilepathGlob(j.TargetPattern)
		if len(matches) > 0 {
			j.Name = name
			return &ResolveJobResult{Job: j, Name: name, Watches: getWatchesForJob(name)}, nil
		}
	}

	// Second pass: check for directories (for watch mode setup before tests exist)
	if fsutil.DirExists("spec") {
		if j, exists := builtinDefaults.Defaults.Jobs["rspec"]; exists {
			j.Name = "rspec"
			return &ResolveJobResult{Job: j, Name: "rspec", Watches: getWatchesForJob("rspec")}, nil
		}
	}
	if fsutil.DirExists("test") {
		if j, exists := builtinDefaults.Defaults.Jobs["minitest"]; exists {
			j.Name = "minitest"
			return &ResolveJobResult{Job: j, Name: "minitest", Watches: getWatchesForJob("minitest")}, nil
		}
	}
	if fsutil.FileExists("go.mod") {
		if j, exists := builtinDefaults.Defaults.Jobs["go-test"]; exists {
			j.Name = "go-test"
			return &ResolveJobResult{Job: j, Name: "go-test", Watches: getWatchesForJob("go-test")}, nil
		}
	}

	return nil, fmt.Errorf("No default spec/test files found using default patterns")
}

func getWatchesForJob(jobName string) []watch.WatchMapping {
	var result []watch.WatchMapping
	for _, w := range builtinDefaults.Defaults.Watches {
		for _, j := range w.Jobs {
			if j == jobName {
				result = append(result, w)
				break
			}
		}
	}
	return result
}

func buildJobNotFoundError(name string, userJobs map[string]job.Job) error {
	availableJobs := make([]string, 0, len(userJobs)+len(builtinDefaults.Defaults.Jobs))
	for jobName := range userJobs {
		availableJobs = append(availableJobs, jobName)
	}
	for jobName := range builtinDefaults.Defaults.Jobs {
		// Avoid duplicates
		found := false
		for _, existing := range availableJobs {
			if existing == jobName {
				found = true
				break
			}
		}
		if !found {
			availableJobs = append(availableJobs, jobName)
		}
	}
	sort.Strings(availableJobs)
	return fmt.Errorf("job '%s' not found. Available jobs: %s", name, strings.Join(availableJobs, ", "))
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
		if containsGlobChars(pattern) || fsutil.DirExists(pattern) {
			continue
		}

		// Check if it's a file that exists
		if !fsutil.FileExists(pattern) {
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
