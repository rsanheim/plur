package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/rsanheim/plur/framework"
	"github.com/rsanheim/plur/job"
)

// FindFilesFromJob discovers all files based on the job's target pattern
func FindFilesFromJob(j job.Job) ([]string, error) {
	patterns, err := targetPatternsForJob(j)
	if err != nil {
		return nil, err
	}
	return expandGlobPatterns(patterns)
}

// ExpandPatternsFromJob takes a list of file paths/patterns and expands any glob patterns.
// Directories expand using the job's target pattern or framework detect patterns.
func ExpandPatternsFromJob(patternsInput []string, j job.Job) ([]string, error) {
	spec, err := framework.Get(j.Framework)
	if err != nil {
		return nil, err
	}

	seenFiles := make(map[string]struct{})

	for _, pattern := range patternsInput {
		var matches []string
		var err error

		// Check if it's a plain path (no glob characters)
		if !strings.ContainsAny(pattern, "*?[{") && !strings.Contains(pattern, "**") {
			fileInfo, statErr := os.Stat(pattern)
			if statErr != nil {
				return nil, fmt.Errorf("file not found: %s", pattern)
			}

			if fileInfo.IsDir() {
				targetPatterns, err := targetPatternsForJobWithSpec(j, spec)
				if err != nil {
					return nil, err
				}
				for _, targetPattern := range targetPatterns {
					_, tail := doublestar.SplitPattern(targetPattern)
					dirPattern := filepath.Join(pattern, filepath.FromSlash(tail))
					dirMatches, globErr := doublestar.FilepathGlob(dirPattern)
					if globErr != nil {
						return nil, fmt.Errorf("error expanding pattern %q: %v", dirPattern, globErr)
					}
					matches = append(matches, dirMatches...)
				}
			} else {
				// Single file: pass it through
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

func targetPatternsForJob(j job.Job) ([]string, error) {
	if j.TargetPattern != "" {
		return []string{j.TargetPattern}, nil
	}
	spec, err := framework.Get(j.Framework)
	if err != nil {
		return nil, err
	}
	return targetPatternsForJobWithSpec(j, spec)
}

func targetPatternsForJobWithSpec(j job.Job, spec framework.Spec) ([]string, error) {
	if len(spec.DetectPatterns) == 0 {
		return nil, fmt.Errorf("job %q has no target_pattern and framework %q has no detect patterns", j.Name, spec.Name)
	}
	return spec.DetectPatterns, nil
}

func expandGlobPatterns(patterns []string) ([]string, error) {
	seenFiles := make(map[string]struct{})
	for _, pattern := range patterns {
		matches, err := doublestar.FilepathGlob(pattern)
		if err != nil {
			return nil, fmt.Errorf("error finding files with pattern %q: %w", pattern, err)
		}
		for _, match := range matches {
			seenFiles[match] = struct{}{}
		}
	}
	allFiles := make([]string, 0, len(seenFiles))
	for file := range seenFiles {
		allFiles = append(allFiles, file)
	}
	return allFiles, nil
}
