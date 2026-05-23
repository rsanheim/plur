package fileset

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/rsanheim/plur/framework"
	"github.com/rsanheim/plur/job"
)

type DiscoverResult struct {
	Files          []string
	ExcludeMatches map[string]int
}

type TargetMismatch struct {
	Target string
	Path   string
}

// Discover returns sorted, deduped, exclude-filtered files for a job.
// When inputs is empty, framework target patterns drive discovery; otherwise
// each input is classified as a glob, an existing file (passthrough), or a
// directory (joined with framework target tails). Exclude patterns are applied
// after expansion using doublestar semantics.
func Discover(j job.Job, inputs, excludes []string) ([]string, error) {
	result, err := DiscoverWithDetails(j, inputs, excludes)
	if err != nil {
		return nil, err
	}
	return result.Files, nil
}

func DiscoverWithDetails(j job.Job, inputs, excludes []string) (DiscoverResult, error) {
	patterns, err := classifyInputs(j, inputs)
	if err != nil {
		return DiscoverResult{}, err
	}

	var files []string
	for _, p := range patterns {
		if !hasGlobMeta(p) {
			files = append(files, p)
			continue
		}
		matches, err := doublestar.FilepathGlob(p)
		if err != nil {
			return DiscoverResult{}, fmt.Errorf("error finding files with pattern %q: %w", p, err)
		}
		files = append(files, matches...)
	}

	for _, ex := range excludes {
		if _, err := doublestar.PathMatch(ex, ""); err != nil {
			return DiscoverResult{}, fmt.Errorf("invalid exclude pattern %q: %w", ex, err)
		}
	}

	slices.Sort(files)
	files = slices.Compact(files)

	excludeMatches := make(map[string]int, len(excludes))
	for _, ex := range excludes {
		excludeMatches[ex] = 0
	}

	files = slices.DeleteFunc(files, func(f string) bool {
		s := filepath.ToSlash(filePathForExcludeMatch(f))
		excluded := false
		for _, ex := range excludes {
			if ok, _ := doublestar.PathMatch(ex, s); ok {
				excludeMatches[ex]++
				excluded = true
			}
		}
		return excluded
	})

	return DiscoverResult{Files: files, ExcludeMatches: excludeMatches}, nil
}

// hasGlobMeta reports whether s contains any doublestar metacharacters.
func hasGlobMeta(s string) bool { return strings.ContainsAny(s, "*?[{") }

func filePathForExcludeMatch(s string) string {
	if isFileLineTarget(s) {
		return s[:strings.IndexByte(s, ':')]
	}
	return s
}

func ExplicitTargetMismatches(inputs, targetPatterns []string) ([]TargetMismatch, error) {
	if len(inputs) == 0 || len(targetPatterns) == 0 {
		return nil, nil
	}

	var mismatches []TargetMismatch
	for _, in := range inputs {
		targetPath, ok := explicitFileTargetPath(in)
		if !ok {
			continue
		}
		matched, err := matchesAnyTargetPattern(targetPath, targetPatterns)
		if err != nil {
			return nil, err
		}
		if !matched {
			mismatches = append(mismatches, TargetMismatch{Target: in, Path: targetPath})
		}
	}
	return mismatches, nil
}

func explicitFileTargetPath(input string) (string, bool) {
	if hasGlobMeta(input) {
		return "", false
	}
	if isFileLineTarget(input) {
		return filePathForExcludeMatch(input), true
	}
	info, err := os.Stat(input)
	if err != nil || info.IsDir() {
		return "", false
	}
	return input, true
}

func matchesAnyTargetPattern(path string, targetPatterns []string) (bool, error) {
	normalized := filepath.ToSlash(path)
	for _, pattern := range targetPatterns {
		matched, err := doublestar.Match(filepath.ToSlash(pattern), normalized)
		if err != nil {
			return false, fmt.Errorf("invalid target pattern %q: %w", pattern, err)
		}
		if matched {
			return true, nil
		}
	}
	return false, nil
}

func classifyInputs(j job.Job, inputs []string) ([]string, error) {
	if len(inputs) == 0 {
		return framework.TargetPatternsForJob(j)
	}
	spec, err := framework.Get(j.Framework)
	if err != nil {
		return nil, err
	}
	var targets []string
	var out []string
	for _, in := range inputs {
		if hasGlobMeta(in) {
			out = append(out, in)
			continue
		}
		if isFileLineTarget(in) {
			out = append(out, in)
			continue
		}
		info, err := os.Stat(in)
		if err != nil {
			return nil, fmt.Errorf("file not found: %s", in)
		}
		if !info.IsDir() {
			out = append(out, in)
			continue
		}
		if targets == nil {
			targets, err = framework.TargetPatternsForJobWithSpec(j, spec)
			if err != nil {
				return nil, err
			}
		}
		for _, t := range targets {
			_, tail := doublestar.SplitPattern(t)
			out = append(out, filepath.Join(in, filepath.FromSlash(tail)))
		}
	}
	return out, nil
}

// isFileLineTarget reports whether s looks like an RSpec focused target:
// the substring before the first ':' is an existing regular file. We pass the
// full string through to the framework and let RSpec interpret the suffix
// (line numbers, scoped IDs, etc.) — plur does not parse what comes after the
// colon.
func isFileLineTarget(s string) bool {
	idx := strings.IndexByte(s, ':')
	if idx <= 0 {
		return false
	}
	info, err := os.Stat(s[:idx])
	return err == nil && !info.IsDir()
}
