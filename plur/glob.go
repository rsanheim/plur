package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/rsanheim/plur/job"
)

// FindFilesFromJob discovers all files based on the job's target pattern
func FindFilesFromJob(j *job.Job) ([]string, error) {
	pattern := j.GetTargetPattern()
	if pattern == "" {
		return nil, fmt.Errorf("job %q has no target_pattern configured and job name does not match any conventions (rspec/minitest)", j.Name)
	}

	matches, err := doublestar.FilepathGlob(pattern)
	if err != nil {
		return nil, fmt.Errorf("error finding files with pattern %q: %w", pattern, err)
	}
	return matches, nil
}

// ExpandPatternsFromJob takes a list of file paths/patterns and expands any glob patterns
// Uses the job's target suffix for directory expansion
func ExpandPatternsFromJob(patterns []string, j *job.Job) ([]string, error) {
	seenFiles := make(map[string]struct{})
	suffix := j.GetTargetSuffix()

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
				// Directory: use job's target suffix within this directory
				dirPattern := filepath.Join(pattern, "**", "*"+suffix)
				matches, err = doublestar.FilepathGlob(dirPattern)
			} else {
				// Single file: pass it through but warn if it doesn't match expected pattern
				if suffix != "" && !strings.HasSuffix(pattern, suffix) {
					fmt.Fprintf(os.Stderr, "Warning: %s does not end with %s\n", pattern, suffix)
				}
				matches = []string{pattern}
			}
		} else {
			// It's already a glob pattern - expand it directly
			matches, err = doublestar.FilepathGlob(pattern)
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
