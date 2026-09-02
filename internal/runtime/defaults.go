package runtime

import (
	_ "embed"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/bmatcuk/doublestar/v4"
	"github.com/rsanheim/plur/internal/framework"
	"github.com/rsanheim/plur/internal/fsutil"
	"github.com/rsanheim/plur/watch"
)

//go:embed defaults.toml
var defaultsFile []byte

type defaultsConfig struct {
	Defaults struct {
		Jobs    map[string]framework.Job `toml:"job"`
		Watches []watch.WatchMapping     `toml:"watch"`
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
	Cmd             bool
	Env             bool
	Framework       bool
	TargetPattern   bool
	ExcludePatterns bool
}

// autodetectJobName runs autodetection against the given resolved jobs and returns the
// name of the best-matching job based on file system presence.
func autodetectJobName(resolvedJobs map[string]framework.Job) (string, error) {
	priority := []string{"rspec", "minitest", "go-test"}
	for _, name := range priority {
		j, exists := resolvedJobs[name]
		if !exists {
			continue
		}
		patterns := []string{j.TargetPattern}
		if j.TargetPattern == "" {
			patterns = framework.DetectPatterns(j.FrameworkName)
		}
		if len(patterns) == 0 {
			continue
		}
		for _, pattern := range patterns {
			found, err := anyFileMatches(pattern)
			if err != nil {
				return "", fmt.Errorf("error finding files with pattern %q: %w", pattern, err)
			}
			if found {
				return name, nil
			}
		}
	}
	return "", errors.New("no default spec/test files found using default patterns")
}

// detectIgnoredDirs are never descended into when checking for test file
// presence. They hold third-party or generated code whose test files must
// not drive detection, and they dominate walk time when present.
var detectIgnoredDirs = map[string]bool{
	"node_modules": true,
	"vendor":       true,
	"tmp":          true,
}

// detectFS hides detectIgnoredDirs from doublestar, whose ** expansion has
// no other hook for pruning a subtree. It also memoizes the last ReadDir,
// because that expansion reads every directory twice in a row: once to match
// files in it and once to enumerate its subdirectories.
type detectFS struct {
	fs.FS
	lastName string
	last     []fs.DirEntry
}

func (d *detectFS) ReadDir(name string) ([]fs.DirEntry, error) {
	if name == d.lastName {
		return d.last, nil
	}
	entries, err := fs.ReadDir(d.FS, name)
	if err != nil {
		return nil, err
	}
	d.lastName = name
	d.last = slices.DeleteFunc(entries, func(e fs.DirEntry) bool { return detectIgnoredDirs[e.Name()] })
	return d.last, nil
}

// anyFileMatches reports whether at least one file matches the doublestar
// pattern. Detection only needs existence, so the walk stops at the first
// hit and never descends into hidden or ignored directories.
func anyFileMatches(pattern string) (bool, error) {
	base, rest := doublestar.SplitPattern(filepath.ToSlash(pattern))
	found := false
	err := doublestar.GlobWalk(&detectFS{FS: os.DirFS(filepath.FromSlash(base))}, rest, func(string, fs.DirEntry) error {
		found = true
		return fs.SkipAll
	}, doublestar.WithFilesOnly(), doublestar.WithNoHidden())
	if errors.Is(err, fs.SkipAll) {
		err = nil
	}
	return found, err
}

// buildResolvedJobs merges built-in defaults and user jobs into a resolved jobs map.
// It applies framework and target pattern defaulting and normalizes frameworks.
func buildResolvedJobs(userJobs map[string]framework.Job) (map[string]framework.Job, map[string]InheritedFields, error) {
	resolved := make(map[string]framework.Job)
	inherited := make(map[string]InheritedFields)

	jobNames := make(map[string]struct{})
	for name := range builtinDefaults.Defaults.Jobs {
		jobNames[name] = struct{}{}
	}
	for name := range userJobs {
		jobNames[name] = struct{}{}
	}

	for jobName := range jobNames {
		builtin, hasBuiltin := builtinDefaults.Defaults.Jobs[jobName]
		user, hasUser := framework.Job{}, false
		if userJobs != nil {
			user, hasUser = userJobs[jobName]
		}

		inherit := InheritedFields{}
		resolvedJob := framework.Job{}

		// Start with builtin if present
		if hasBuiltin {
			resolvedJob = builtin
			if !hasUser {
				inherit.Cmd = len(builtin.Cmd) > 0
				inherit.Env = len(builtin.Env) > 0
				inherit.Framework = builtin.FrameworkName != ""
				inherit.TargetPattern = builtin.TargetPattern != ""
				inherit.ExcludePatterns = len(builtin.ExcludePatterns) > 0
			}
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

			if user.FrameworkName != "" {
				resolvedJob.FrameworkName = user.FrameworkName
			} else if resolvedJob.FrameworkName != "" {
				inherit.Framework = true
			}

			if user.TargetPattern != "" {
				resolvedJob.TargetPattern = user.TargetPattern
			} else if resolvedJob.TargetPattern != "" {
				inherit.TargetPattern = true
			}

			if len(user.ExcludePatterns) > 0 {
				resolvedJob.ExcludePatterns = user.ExcludePatterns
			} else if len(resolvedJob.ExcludePatterns) > 0 {
				inherit.ExcludePatterns = true
			}
		}

		resolvedJob.Name = jobName

		// Framework defaulting (only affects pure user jobs without builtin)
		if resolvedJob.FrameworkName == "" {
			resolvedJob.FrameworkName = "passthrough"
		}

		// Validate framework
		normalizedFramework := framework.Normalize(resolvedJob.FrameworkName)
		if !framework.IsKnown(normalizedFramework) {
			return nil, nil, fmt.Errorf("job %q has unknown framework %q", jobName, resolvedJob.FrameworkName)
		}
		resolvedJob.FrameworkName = normalizedFramework

		resolved[jobName] = resolvedJob
		inherited[jobName] = inherit
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
		frameworks := slices.Sorted(maps.Keys(union))
		return "", fmt.Errorf("explicit patterns match multiple frameworks (%s). Split the command or pass --use to select one", strings.Join(frameworks, ", "))
	}

	return "", nil
}

func frameworksMatchingPattern(pattern string, candidates []string) (map[string]struct{}, error) {
	matched := make(map[string]struct{})
	for _, name := range candidates {
		detectPatterns := framework.DetectPatterns(name)
		if len(detectPatterns) == 0 {
			continue
		}
		ok, err := patternMatchesFramework(pattern, detectPatterns)
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
		found, err := anyFileMatches(dirPattern)
		if err != nil {
			return false, err
		}
		if found {
			return true, nil
		}
	}
	return false, nil
}
