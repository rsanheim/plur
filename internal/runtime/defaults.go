package runtime

import (
	_ "embed"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/bmatcuk/doublestar/v4"
	"github.com/rsanheim/plur/framework"
	"github.com/rsanheim/plur/internal/fsutil"
	"github.com/rsanheim/plur/job"
	"github.com/rsanheim/plur/watch"
)

//go:embed defaults.toml
var defaultsFile []byte

type defaultsConfig struct {
	Defaults struct {
		Jobs    map[string]job.Job   `toml:"job"`
		Watches []watch.WatchMapping `toml:"watch"`
	} `toml:"defaults"`
}

var builtinDefaults defaultsConfig

func init() {
	if _, err := toml.Decode(string(defaultsFile), &builtinDefaults); err != nil {
		panic(fmt.Errorf("failed to load embedded defaults: %w", err))
	}
}

// InheritedFields indicates which fields were inherited from a built-in default.
type InheritedFields struct {
	Cmd           bool
	Env           bool
	Framework     bool
	TargetPattern bool
}

// autodetectJobName runs autodetection against the given resolved jobs and returns the
// name of the best-matching job based on file system presence.
func autodetectJobName(resolvedJobs map[string]job.Job) (string, error) {
	priority := []string{"rspec", "minitest", "go-test"}
	for _, name := range priority {
		j, exists := resolvedJobs[name]
		if !exists {
			continue
		}
		patterns, err := framework.TargetPatternsForJob(j)
		if err != nil || len(patterns) == 0 {
			continue
		}
		for _, pattern := range patterns {
			matches, err := doublestar.FilepathGlob(pattern)
			if err != nil {
				return "", fmt.Errorf("error finding files with pattern %q: %w", pattern, err)
			}
			if len(matches) > 0 {
				return name, nil
			}
		}
	}
	return "", fmt.Errorf("no default spec/test files found using default patterns")
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
	if strings.ContainsAny(pattern, "*?[") {
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

