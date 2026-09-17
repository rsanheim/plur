package fileset

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/rsanheim/plur/internal/framework"
	"github.com/rsanheim/plur/internal/fsutil"
)

// RSpec line and scoped-example suffixes are passed through to the runner.
var rspecSelector = regexp.MustCompile(`(?::[0-9]+)+$|\[[0-9\s:,]+\]$`)

// Discover expands files, directories, globs, and RSpec selectors, then returns
// sorted, deduplicated files with excludes applied. Empty inputs use job defaults.
func Discover(j framework.Job, inputs, excludes []string) ([]string, error) {
	for _, ex := range excludes {
		if !doublestar.ValidatePathPattern(ex) {
			return nil, fmt.Errorf("invalid exclude pattern %q: %w", ex, doublestar.ErrBadPattern)
		}
	}
	if len(inputs) == 0 {
		var err error
		inputs, err = j.TargetPatterns()
		if err != nil {
			return nil, err
		}
	}
	var files []string
	for _, input := range inputs {
		matches := []string{input}
		selector := j.Framework.Name == "rspec" && rspecSelector.MatchString(input)
		if !selector && strings.ContainsAny(input, "*?[{") {
			var err error
			matches, err = doublestar.FilepathGlob(input)
			if err != nil {
				return nil, fmt.Errorf("error finding files with pattern %q: %w", input, err)
			}
		}
		for _, match := range matches {
			if !fsutil.DirExists(match) {
				files = append(files, match)
				continue
			}
			targets, err := j.TargetPatterns()
			if err != nil {
				return nil, err
			}
			for _, target := range targets {
				_, tail := doublestar.SplitPattern(target)
				// Keep the matched directory literal, including names such as [id].
				children, err := doublestar.Glob(os.DirFS(match), tail, doublestar.WithFilesOnly())
				if err != nil {
					return nil, fmt.Errorf("error finding files in %q with pattern %q: %w", match, tail, err)
				}
				for _, child := range children {
					files = append(files, filepath.Join(match, filepath.FromSlash(child)))
				}
			}
		}
	}
	files = slices.DeleteFunc(files, func(file string) bool {
		if j.Framework.Name == "rspec" {
			file = strings.TrimSuffix(file, rspecSelector.FindString(file))
		}
		path := filepath.ToSlash(file)
		return slices.ContainsFunc(excludes, func(ex string) bool {
			return doublestar.PathMatchUnvalidated(ex, path)
		})
	})
	slices.Sort(files)
	return slices.Compact(files), nil
}
