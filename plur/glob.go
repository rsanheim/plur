package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/rsanheim/plur/config"
)

// FindTestFiles discovers all test files based on the framework
func FindTestFiles(framework config.TestFramework) ([]string, error) {
	pattern := getDefaultPattern(framework)
	matches, err := doublestar.FilepathGlob(pattern)
	if err != nil {
		return nil, fmt.Errorf("error finding test files: %v", err)
	}
	return matches, nil
}

// FindSpecFiles discovers all spec files in the spec directory
func FindSpecFiles() ([]string, error) {
	// Check if spec directory exists
	if _, err := os.Stat("spec"); os.IsNotExist(err) {
		return []string{}, nil // Return empty list if no spec directory
	}

	return doublestar.FilepathGlob("spec/**/*_spec.rb")
}

// FindMinitestFiles discovers all test files in the test directory
func FindMinitestFiles() ([]string, error) {
	// Check if test directory exists
	if _, err := os.Stat("test"); os.IsNotExist(err) {
		return []string{}, nil // Return empty list if no test directory
	}

	return doublestar.FilepathGlob("test/**/*_test.rb")
}

// ExpandGlobPatterns takes a list of file paths/patterns and expands any glob patterns
// Supports ** for recursive directory matching, brace expansion, and more
// Like RSpec, when given patterns or directories, filters to only test files
func ExpandGlobPatterns(patterns []string, framework config.TestFramework) ([]string, error) {
	seenFiles := make(map[string]struct{})
	suffix := getTestFileSuffix(framework)

	for _, pattern := range patterns {
		var matches []string
		var err error

		// Check if it's a plain path (no glob characters)
		if !strings.ContainsAny(pattern, "*?[{") && !strings.Contains(pattern, "**") {
			fileInfo, statErr := os.Stat(pattern)
			if statErr != nil {
				return nil, fmt.Errorf("file not found: %s", pattern)
			}

			if fileInfo.IsDir() {
				// Directory: append test pattern (like RSpec's gather_directories)
				pattern = filepath.Join(pattern, "**", "*"+suffix)
				matches, err = doublestar.FilepathGlob(pattern)
			} else {
				// Single file: pass it through (like RSpec's extract_location)
				// RSpec will handle it - finds 0 examples for non-test files
				if !strings.HasSuffix(pattern, suffix) {
					fmt.Fprintf(os.Stderr, "Warning: %s does not end with %s\n", pattern, suffix)
				}
				matches = []string{pattern}
			}
		} else {
			// It's a glob pattern - use GlobWalk to expand and filter in one pass
			// This matches RSpec's behavior of applying its pattern filter
			matches = []string{}
			err = doublestar.GlobWalk(os.DirFS("."), pattern, func(path string, d fs.DirEntry) error {
				// Only add test files (like RSpec does)
				if !d.IsDir() && strings.HasSuffix(path, suffix) {
					matches = append(matches, path)
				}
				return nil
			})
		}

		if err != nil {
			return nil, fmt.Errorf("error expanding pattern %q: %v", pattern, err)
		}

		// Add matches to set
		for _, match := range matches {
			seenFiles[match] = struct{}{}
		}
	}

	// Convert map keys to slice
	allFiles := make([]string, 0, len(seenFiles))
	for file := range seenFiles {
		allFiles = append(allFiles, file)
	}

	return allFiles, nil
}

// getTestFileSuffix returns the test file suffix for the framework
func getTestFileSuffix(framework config.TestFramework) string {
	switch framework {
	case config.FrameworkRSpec:
		return "_spec.rb"
	case config.FrameworkMinitest:
		return "_test.rb"
	default:
		return "_spec.rb"
	}
}

// getDefaultPattern returns the default glob pattern for the framework
func getDefaultPattern(framework config.TestFramework) string {
	switch framework {
	case config.FrameworkRSpec:
		return "spec/**/*_spec.rb"
	case config.FrameworkMinitest:
		return "test/**/*_test.rb"
	default:
		return "spec/**/*_spec.rb"
	}
}
