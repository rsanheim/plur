package fileset

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/rsanheim/plur/internal/framework"
)

type DiscoverResult struct {
	Files []string
}

// Discover returns sorted, deduped, exclude-filtered files for a job.
// When inputs is empty, framework target patterns drive discovery; otherwise
// each input is classified as a glob, an existing file (passthrough), or a
// directory (joined with framework target tails). Exclude patterns are applied
// after expansion using doublestar semantics.
func Discover(j framework.Job, inputs, excludes []string) (DiscoverResult, error) {
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

	files = slices.DeleteFunc(files, func(f string) bool {
		s := filepath.ToSlash(filePathForExcludeMatch(f))
		for _, ex := range excludes {
			if ok, _ := doublestar.PathMatch(ex, s); ok {
				return true
			}
		}
		return false
	})

	return DiscoverResult{Files: files}, nil
}

// hasGlobMeta reports whether s contains any doublestar metacharacters.
func hasGlobMeta(s string) bool { return strings.ContainsAny(s, "*?[{") }

func filePathForExcludeMatch(s string) string {
	if isFileLineTarget(s) {
		return s[:strings.IndexByte(s, ':')]
	}
	return s
}

func classifyInputs(j framework.Job, inputs []string) ([]string, error) {
	if len(inputs) == 0 {
		return j.TargetPatterns()
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
			var err error
			targets, err = j.TargetPatterns()
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
