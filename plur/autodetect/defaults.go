package autodetect

import (
	_ "embed"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/pelletier/go-toml"
	"github.com/rsanheim/plur/framework"
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
	Job          job.Job
	Name         string
	Reason       ResolveReason        // reason for job selection
	Watches      []watch.WatchMapping // watches that reference this job
	ResolvedJobs map[string]job.Job
	Inherited    InheritedFields
}

// ResolveReason indicates how a job was selected.
type ResolveReason string

const (
	// ResolveReasonExplicitName is selected when --use (or config use) names a job directly.
	// Example: "plur --use rspec"
	ResolveReasonExplicitName ResolveReason = "explicit_name"
	// ResolveReasonExplicitPatterns is selected when CLI patterns infer a single framework.
	// Examples: "plur spec/models", "plur test/**/user*_test.rb"
	ResolveReasonExplicitPatterns ResolveReason = "explicit_patterns"
	// ResolveReasonAutodetect is selected when no explicit name/patterns are provided.
	// Example: "plur"
	ResolveReasonAutodetect ResolveReason = "autodetect"
	// ResolveReasonAutodetectAfterPatterns is selected when patterns are provided but don't
	// match any framework, so selection falls back to autodetect.
	// Example: "plur README.md"
	ResolveReasonAutodetectAfterPatterns ResolveReason = "autodetect_after_patterns"
)

// InheritedFields indicates which fields were inherited from a built-in default.
type InheritedFields struct {
	Cmd           bool
	Env           bool
	Framework     bool
	TargetPattern bool
}

// ResolveJob determines which job to use based on explicit selection or autodetection.
//
// Resolution order:
//  1. If explicitName provided → look up in userJobs, then built-in defaults
//  2. If patterns provided → infer framework using detect patterns (dir/file/glob rules)
//  3. Autodetect → check which jobs have matching files (priority: rspec > minitest > go-test)
func ResolveJob(explicitName string, userJobs map[string]job.Job, patterns []string) (*ResolveJobResult, error) {
	resolvedJobs, inherited, err := buildResolvedJobs(userJobs)
	if err != nil {
		return nil, err
	}

	// 1. If explicit name provided, look it up (user config first, then defaults)
	if explicitName != "" {
		return resolveExplicitJob(explicitName, resolvedJobs, inherited)
	}

	// 2. If file patterns provided, infer from suffixes
	if len(patterns) > 0 {
		result, err := resolveFromPatterns(patterns, resolvedJobs, inherited)
		if err != nil {
			return nil, err
		}
		if result != nil {
			return result, nil
		}
	}

	// 3. Autodetect from file system (with priority order)
	result, err := autodetectJob(resolvedJobs, inherited)
	if err != nil || result == nil {
		return result, err
	}
	if len(patterns) > 0 {
		result.Reason = ResolveReasonAutodetectAfterPatterns
	}
	return result, nil
}

func resolveExplicitJob(name string, resolvedJobs map[string]job.Job, inherited map[string]InheritedFields) (*ResolveJobResult, error) {
	if j, exists := resolvedJobs[name]; exists {
		return &ResolveJobResult{
			Job:          j,
			Name:         name,
			Reason:       ResolveReasonExplicitName,
			Watches:      getWatchesForJob(name),
			ResolvedJobs: resolvedJobs,
			Inherited:    inherited[name],
		}, nil
	}
	return nil, buildJobNotFoundError(name, resolvedJobs)
}

func resolveFromPatterns(patterns []string, resolvedJobs map[string]job.Job, inherited map[string]InheritedFields) (*ResolveJobResult, error) {
	frameworkName, err := inferFrameworkFromPatterns(patterns)
	if err != nil || frameworkName == "" {
		return nil, err
	}
	if j, exists := resolvedJobs[frameworkName]; exists {
		return &ResolveJobResult{
			Job:          j,
			Name:         frameworkName,
			Reason:       ResolveReasonExplicitPatterns,
			Watches:      getWatchesForJob(frameworkName),
			ResolvedJobs: resolvedJobs,
			Inherited:    inherited[frameworkName],
		}, nil
	}
	return nil, nil
}

func autodetectJob(resolvedJobs map[string]job.Job, inherited map[string]InheritedFields) (*ResolveJobResult, error) {
	// Explicit priority order: rspec > minitest > go-test
	priority := []string{"rspec", "minitest", "go-test"}

	// First pass: check for actual test files
	for _, name := range priority {
		j, exists := resolvedJobs[name]
		if !exists {
			continue
		}
		patterns, err := discoveryPatternsForJob(j)
		if err != nil || len(patterns) == 0 {
			continue
		}
		matches, err := anyPatternMatches(patterns)
		if err != nil {
			return nil, err
		}
		if matches {
			return &ResolveJobResult{
				Job:          j,
				Name:         name,
				Reason:       ResolveReasonAutodetect,
				Watches:      getWatchesForJob(name),
				ResolvedJobs: resolvedJobs,
				Inherited:    inherited[name],
			}, nil
		}
	}

	return nil, fmt.Errorf("no default spec/test files found using default patterns")
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

func discoveryPatternsForJob(j job.Job) ([]string, error) {
	if j.TargetPattern != "" {
		return []string{j.TargetPattern}, nil
	}
	spec, err := framework.Get(j.Framework)
	if err != nil {
		return nil, err
	}
	return spec.DetectPatterns, nil
}

func anyPatternMatches(patterns []string) (bool, error) {
	for _, pattern := range patterns {
		matches, err := doublestar.FilepathGlob(pattern)
		if err != nil {
			return false, fmt.Errorf("error finding files with pattern %q: %w", pattern, err)
		}
		if len(matches) > 0 {
			return true, nil
		}
	}
	return false, nil
}

// buildResolvedJobs merges built-in defaults and user jobs into a resolved jobs map.
// It applies framework and target pattern defaulting and normalizes frameworks.
func buildResolvedJobs(userJobs map[string]job.Job) (map[string]job.Job, map[string]InheritedFields, error) {
	resolved := make(map[string]job.Job)
	inherited := make(map[string]InheritedFields)

	names := make(map[string]struct{})
	for name := range builtinDefaults.Defaults.Jobs {
		names[name] = struct{}{}
	}
	for name := range userJobs {
		names[name] = struct{}{}
	}

	for name := range names {
		builtin, hasBuiltin := builtinDefaults.Defaults.Jobs[name]
		user, hasUser := job.Job{}, false
		if userJobs != nil {
			user, hasUser = userJobs[name]
		}

		inherit := InheritedFields{}
		resolvedJob := job.Job{}

		// Start with builtin if present
		if hasBuiltin {
			resolvedJob = builtin
		}

		// Overlay user fields, tracking inheritance at decision point
		if hasUser {
			if len(user.Cmd) > 0 {
				resolvedJob.Cmd = user.Cmd
			} else if len(resolvedJob.Cmd) > 0 {
				inherit.Cmd = true
			}

			if len(user.Env) > 0 {
				resolvedJob.Env = user.Env
			} else if len(resolvedJob.Env) > 0 {
				inherit.Env = true
			}

			if user.Framework != "" {
				resolvedJob.Framework = user.Framework
			} else if resolvedJob.Framework != "" {
				inherit.Framework = true
			}

			if user.TargetPattern != "" {
				resolvedJob.TargetPattern = user.TargetPattern
			} else if resolvedJob.TargetPattern != "" {
				inherit.TargetPattern = true
			}
		}

		resolvedJob.Name = name

		// Framework defaulting (only affects pure user jobs without builtin)
		if resolvedJob.Framework == "" {
			resolvedJob.Framework = "passthrough"
		}

		// Validate framework
		normalizedFramework := framework.Normalize(resolvedJob.Framework)
		if !framework.IsKnown(normalizedFramework) {
			return nil, nil, fmt.Errorf("job %q has unknown framework %q", name, resolvedJob.Framework)
		}
		resolvedJob.Framework = normalizedFramework

		resolved[name] = resolvedJob
		inherited[name] = inherit
	}

	return resolved, inherited, nil
}

// ValidateConfig validates jobs and watch mappings at config load time.
func ValidateConfig(userJobs map[string]job.Job, userWatches []watch.WatchMapping) error {
	resolvedJobs, _, err := buildResolvedJobs(userJobs)
	if err != nil {
		return err
	}

	if err := validateResolvedJobs(resolvedJobs); err != nil {
		return err
	}

	watches := userWatches
	if len(watches) == 0 {
		watches = builtinDefaults.Defaults.Watches
	}
	return watch.ValidateConfig(resolvedJobs, watches)
}

func validateResolvedJobs(resolvedJobs map[string]job.Job) error {
	for name, j := range resolvedJobs {
		if len(j.Cmd) == 0 {
			return fmt.Errorf("job %q must define a command", name)
		}
	}
	return nil
}

func buildJobNotFoundError(name string, resolvedJobs map[string]job.Job) error {
	availableJobs := make([]string, 0, len(resolvedJobs))
	for jobName := range resolvedJobs {
		availableJobs = append(availableJobs, jobName)
	}
	sort.Strings(availableJobs)
	return fmt.Errorf("job '%s' not found. Available jobs: %s", name, strings.Join(availableJobs, ", "))
}

// inferFrameworkFromPatterns examines file patterns to infer the framework
// Returns "rspec", "minitest", or "" if unable to determine
func inferFrameworkFromPatterns(patterns []string) (string, error) {
	if len(patterns) == 0 {
		return "", nil
	}

	candidates := []string{"rspec", "minitest", "go-test"}
	counts := make(map[string]int)
	union := make(map[string]struct{})

	for _, pattern := range patterns {
		matched, err := frameworksMatchingPattern(pattern, candidates)
		if err != nil {
			return "", err
		}
		if len(matched) == 0 {
			return "", nil
		}

		for name := range matched {
			counts[name]++
			union[name] = struct{}{}
		}
	}

	intersection := make([]string, 0, len(counts))
	for name, count := range counts {
		if count == len(patterns) {
			intersection = append(intersection, name)
		}
	}

	if len(intersection) == 1 {
		return intersection[0], nil
	}

	if len(union) > 1 {
		frameworks := make([]string, 0, len(union))
		for name := range union {
			frameworks = append(frameworks, name)
		}
		sort.Strings(frameworks)
		return "", fmt.Errorf("explicit patterns match multiple frameworks (%s). Split the command or pass --use to select one", strings.Join(frameworks, ", "))
	}

	return "", nil
}

func frameworksMatchingPattern(pattern string, candidates []string) (map[string]struct{}, error) {
	matched := make(map[string]struct{})
	for _, name := range candidates {
		spec, err := framework.Get(name)
		if err != nil {
			return nil, err
		}
		if len(spec.DetectPatterns) == 0 {
			continue
		}
		ok, err := patternMatchesFramework(pattern, spec.DetectPatterns)
		if err != nil {
			return nil, err
		}
		if ok {
			matched[name] = struct{}{}
		}
	}
	return matched, nil
}

func patternMatchesFramework(pattern string, detectPatterns []string) (bool, error) {
	if containsGlobChars(pattern) {
		return globMatchesFramework(pattern, detectPatterns)
	}
	if fsutil.DirExists(pattern) {
		return dirMatchesFramework(pattern, detectPatterns)
	}
	if fsutil.FileExists(pattern) {
		return fileMatchesFramework(pattern, detectPatterns)
	}
	return false, nil
}

func fileMatchesFramework(path string, detectPatterns []string) (bool, error) {
	normalized := filepath.ToSlash(path)
	for _, pattern := range detectPatterns {
		matched, err := doublestar.Match(pattern, normalized)
		if err != nil {
			return false, err
		}
		if matched {
			return true, nil
		}
	}
	return false, nil
}

func globMatchesFramework(pattern string, detectPatterns []string) (bool, error) {
	matches, err := doublestar.FilepathGlob(pattern)
	if err != nil {
		return false, err
	}
	for _, match := range matches {
		ok, err := fileMatchesFramework(match, detectPatterns)
		if err != nil {
			return false, err
		}
		if ok {
			return true, nil
		}
	}
	return false, nil
}

func dirMatchesFramework(dir string, detectPatterns []string) (bool, error) {
	for _, pattern := range detectPatterns {
		_, tail := doublestar.SplitPattern(pattern)
		dirPattern := filepath.Join(dir, filepath.FromSlash(tail))
		matches, err := doublestar.FilepathGlob(dirPattern)
		if err != nil {
			return false, err
		}
		if len(matches) > 0 {
			return true, nil
		}
	}
	return false, nil
}

// containsGlobChars checks if a string contains glob characters
func containsGlobChars(s string) bool {
	return strings.Contains(s, "*") || strings.Contains(s, "?") || strings.Contains(s, "[")
}
