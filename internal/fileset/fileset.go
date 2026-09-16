package fileset

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/rsanheim/plur/internal/framework"
	"github.com/rsanheim/plur/internal/framework/rspec"
	"github.com/rsanheim/plur/internal/fsutil"
)

// Discover expands files, directories, globs, and RSpec selectors, then returns
// sorted, deduplicated files with excludes applied. Empty inputs use job defaults.
func Discover(j framework.Job, inputs, excludes []string) ([]string, error) {
	for _, ex := range excludes {
		if !doublestar.ValidatePathPattern(ex) {
			return nil, fmt.Errorf("invalid exclude pattern %q: %w", ex, doublestar.ErrBadPattern)
		}
	}
	targetPath := func(target string) string {
		if j.Framework.Name == "rspec" {
			return rspec.TargetPath(target)
		}
		return target
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
		if targetPath(input) == input && strings.ContainsAny(input, "*?[{") {
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
		path := filepath.ToSlash(targetPath(file))
		return slices.ContainsFunc(excludes, func(ex string) bool {
			return doublestar.PathMatchUnvalidated(ex, path)
		})
	})
	slices.Sort(files)
	return slices.Compact(files), nil
}
