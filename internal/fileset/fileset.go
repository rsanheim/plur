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

// Discover returns sorted, deduped, exclude-filtered files for a job.
// When inputs is empty, framework target patterns drive discovery; otherwise
// each input is classified as a glob, an existing file (passthrough), or a
// directory (joined with framework target tails). Exclude patterns are applied
// after expansion using doublestar semantics.
func Discover(j job.Job, inputs, excludes []string) ([]string, error) {
	patterns, err := classifyInputs(j, inputs)
	if err != nil {
		return nil, err
	}

	var files []string
	for _, p := range patterns {
		if !hasGlobMeta(p) {
			files = append(files, p)
			continue
		}
		matches, err := doublestar.FilepathGlob(p)
		if err != nil {
			return nil, fmt.Errorf("error finding files with pattern %q: %w", p, err)
		}
		files = append(files, matches...)
	}

	for _, ex := range excludes {
		if _, err := doublestar.PathMatch(ex, ""); err != nil {
			return nil, fmt.Errorf("invalid exclude pattern %q: %w", ex, err)
		}
	}
	files = slices.DeleteFunc(files, func(f string) bool {
		s := filepath.ToSlash(f)
		for _, ex := range excludes {
			if ok, _ := doublestar.PathMatch(ex, s); ok {
				return true
			}
		}
		return false
	})

	slices.Sort(files)
	return slices.Compact(files), nil
}

// hasGlobMeta reports whether s contains any doublestar metacharacters.
func hasGlobMeta(s string) bool { return strings.ContainsAny(s, "*?[{") }

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
