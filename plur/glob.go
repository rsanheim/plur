package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FindTestFiles discovers all test files based on the framework
func FindTestFiles(framework TestFramework) ([]string, error) {
	switch framework {
	case FrameworkRSpec:
		return FindSpecFiles()
	case FrameworkMinitest:
		return FindMinitestFiles()
	default:
		return FindSpecFiles() // Default to RSpec
	}
}

// FindSpecFiles discovers all spec files in the spec directory
func FindSpecFiles() ([]string, error) {
	var specFiles []string

	// Check if spec directory exists
	if _, err := os.Stat("spec"); os.IsNotExist(err) {
		return specFiles, nil // Return empty list if no spec directory
	}

	// Walk the spec directory recursively
	err := filepath.WalkDir("spec", func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if d.IsDir() {
			return nil
		}

		// Check if file ends with _spec.rb
		if strings.HasSuffix(path, "_spec.rb") {
			specFiles = append(specFiles, path)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error walking spec directory: %v", err)
	}

	return specFiles, nil
}

// FindMinitestFiles discovers all test files in the test directory
func FindMinitestFiles() ([]string, error) {
	var testFiles []string

	// Check if test directory exists
	if _, err := os.Stat("test"); os.IsNotExist(err) {
		return testFiles, nil // Return empty list if no test directory
	}

	// Walk the test directory recursively
	err := filepath.WalkDir("test", func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if d.IsDir() {
			return nil
		}

		// Check if file ends with _test.rb
		if strings.HasSuffix(path, "_test.rb") {
			testFiles = append(testFiles, path)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error walking test directory: %v", err)
	}

	return testFiles, nil
}

// ExpandGlobPatterns takes a list of file paths/patterns and expands any glob patterns
// Supports ** for recursive directory matching like Ruby's Dir.glob
func ExpandGlobPatterns(patterns []string, framework TestFramework) ([]string, error) {
	var allFiles []string
	seenFiles := make(map[string]bool)

	for _, pattern := range patterns {
		// Check if pattern contains glob characters
		if strings.ContainsAny(pattern, "*?[") {
			// Handle ** for recursive matching
			if strings.Contains(pattern, "**") {
				matches, err := expandDoubleStarGlob(pattern, framework)
				if err != nil {
					return nil, fmt.Errorf("error expanding glob pattern %q: %v", pattern, err)
				}

				for _, match := range matches {
					if !seenFiles[match] {
						allFiles = append(allFiles, match)
						seenFiles[match] = true
					}
				}
			} else {
				// Use standard glob for patterns without **
				matches, err := filepath.Glob(pattern)
				if err != nil {
					return nil, fmt.Errorf("error expanding glob pattern %q: %v", pattern, err)
				}

				// Filter to only include test files based on framework
				for _, match := range matches {
					if isTestFile(match, framework) && !seenFiles[match] {
						allFiles = append(allFiles, match)
						seenFiles[match] = true
					}
				}
			}
		} else {
			// Not a glob pattern, check if it's a valid file or directory
			fileInfo, err := os.Stat(pattern)
			if err != nil {
				return nil, fmt.Errorf("file not found: %s", pattern)
			}

			if fileInfo.IsDir() {
				// If it's a directory, expand to find all test files within it
				suffix := getTestFileSuffix(framework)
				dirPattern := filepath.Join(pattern, "**", "*"+suffix)
				matches, err := expandDoubleStarGlob(dirPattern, framework)
				if err != nil {
					return nil, fmt.Errorf("error expanding directory %q: %v", pattern, err)
				}

				for _, match := range matches {
					if !seenFiles[match] {
						allFiles = append(allFiles, match)
						seenFiles[match] = true
					}
				}
			} else if isTestFile(pattern, framework) && !seenFiles[pattern] {
				allFiles = append(allFiles, pattern)
				seenFiles[pattern] = true
			} else {
				// Warn about non-test files
				suffix := getTestFileSuffix(framework)
				fmt.Fprintf(os.Stderr, "Warning: %s does not end with %s\n", pattern, suffix)
			}
		}
	}

	return allFiles, nil
}

// expandDoubleStarGlob handles ** glob patterns for recursive directory matching
func expandDoubleStarGlob(pattern string, framework TestFramework) ([]string, error) {
	// Split pattern into parts
	parts := strings.Split(pattern, "**")
	if len(parts) != 2 {
		// Multiple ** not supported
		return nil, fmt.Errorf("multiple ** in pattern not supported: %s", pattern)
	}

	prefix := strings.TrimSuffix(parts[0], "/")
	suffix := strings.TrimPrefix(parts[1], "/")

	// If prefix is empty, start from current directory
	if prefix == "" {
		prefix = "."
	}

	var matches []string

	// Walk the directory tree starting from prefix
	err := filepath.WalkDir(prefix, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		// Skip directories unless suffix is empty
		if d.IsDir() && suffix != "" {
			return nil
		}

		// Check if the path matches the suffix pattern
		if suffix != "" {
			// Get the relative path from the prefix
			relPath, err := filepath.Rel(prefix, path)
			if err != nil {
				return nil
			}

			// Check if the relative path matches the suffix pattern
			_, err = filepath.Match(suffix, relPath)
			if err != nil {
				return nil
			}

			// Also check if any parent directory + suffix matches
			// This handles cases like spec/**/models/*_spec.rb
			pathParts := strings.Split(relPath, string(filepath.Separator))
			for i := range pathParts {
				subPath := filepath.Join(pathParts[i:]...)
				if matched, _ := filepath.Match(suffix, subPath); matched {
					if isTestFile(path, framework) {
						matches = append(matches, path)
						return nil
					}
				}
			}
		} else if isTestFile(path, framework) {
			// No suffix, just match all test files
			matches = append(matches, path)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return matches, nil
}

// isTestFile checks if a file is a test file based on the framework
func isTestFile(path string, framework TestFramework) bool {
	switch framework {
	case FrameworkRSpec:
		return strings.HasSuffix(path, "_spec.rb")
	case FrameworkMinitest:
		return strings.HasSuffix(path, "_test.rb")
	default:
		return strings.HasSuffix(path, "_spec.rb")
	}
}

// getTestFileSuffix returns the test file suffix for the framework
func getTestFileSuffix(framework TestFramework) string {
	switch framework {
	case FrameworkRSpec:
		return "_spec.rb"
	case FrameworkMinitest:
		return "_test.rb"
	default:
		return "_spec.rb"
	}
}
