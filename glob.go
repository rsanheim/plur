package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/rsanheim/plur/framework"
	"github.com/rsanheim/plur/job"
)

// FindFilesFromJob discovers all files based on the job's target pattern,
// then removes any files matching the exclude patterns.
// Returns kept files, excluded files, and any error.
func FindFilesFromJob(j job.Job, excludePatterns []string) ([]string, []string, error) {
	patterns, err := framework.TargetPatternsForJob(j)
	if err != nil {
		return nil, nil, err
	}
	files, err := expandGlobPatterns(patterns)
	if err != nil {
		return nil, nil, err
	}
	return filterExcludedFiles(files, excludePatterns)
}

// ExpandPatternsFromJob takes a list of file paths/patterns and expands any glob patterns.
// Directories expand using the job's target pattern or framework detect patterns.
// Any files matching excludePatterns are removed from the result.
// Returns kept files, excluded files, and any error.
func ExpandPatternsFromJob(patternsInput []string, j job.Job, excludePatterns []string) ([]string, []string, error) {
	spec, err := framework.Get(j.Framework)
	if err != nil {
		return nil, nil, err
	}

	var allFiles []string

	for _, pattern := range patternsInput {
		var matches []string
		var err error

		// Check if it's a plain path (no glob characters)
		if !strings.ContainsAny(pattern, "*?[{") && !strings.Contains(pattern, "**") {
			fileInfo, statErr := os.Stat(pattern)
			if statErr != nil {
				return nil, nil, fmt.Errorf("file not found: %s", pattern)
			}

			if fileInfo.IsDir() {
				targetPatterns, err := framework.TargetPatternsForJobWithSpec(j, spec)
				if err != nil {
					return nil, nil, err
				}
				for _, targetPattern := range targetPatterns {
					_, tail := doublestar.SplitPattern(targetPattern)
					dirPattern := filepath.Join(pattern, filepath.FromSlash(tail))
					dirMatches, globErr := doublestar.FilepathGlob(dirPattern)
					if globErr != nil {
						return nil, nil, fmt.Errorf("error expanding pattern %q: %v", dirPattern, globErr)
					}
					matches = append(matches, dirMatches...)
				}
			} else {
				matches = []string{pattern}
			}
		} else {
			matches, err = doublestar.FilepathGlob(pattern)
		}

		if err != nil {
			return nil, nil, fmt.Errorf("error expanding pattern %q: %v", pattern, err)
		}

		allFiles = append(allFiles, matches...)
	}

	return filterExcludedFiles(allFiles, excludePatterns)
}

func filterExcludedFiles(files []string, excludePatterns []string) (kept []string, excluded []string, err error) {
	sorted := dedupeAndSort(files)
	if len(excludePatterns) == 0 {
		return sorted, nil, nil
	}

	// Normalize patterns once upfront
	normalized := make([]string, len(excludePatterns))
	for i, p := range excludePatterns {
		normalized[i] = filepath.ToSlash(p)
	}

	for _, file := range sorted {
		filePath := filepath.ToSlash(file)
		matched := false
		for _, pattern := range normalized {
			m, matchErr := doublestar.PathMatch(pattern, filePath)
			if matchErr != nil {
				return nil, nil, fmt.Errorf("invalid exclude pattern %q: %w", pattern, matchErr)
			}
			if m {
				matched = true
				break
			}
		}
		if matched {
			excluded = append(excluded, file)
		} else {
			kept = append(kept, file)
		}
	}

	return kept, excluded, nil
}

func dedupeAndSort(files []string) []string {
	seen := make(map[string]struct{}, len(files))
	result := make([]string, 0, len(files))
	for _, f := range files {
		if _, ok := seen[f]; !ok {
			seen[f] = struct{}{}
			result = append(result, f)
		}
	}
	sort.Strings(result)
	return result
}

func expandGlobPatterns(patterns []string) ([]string, error) {
	var allFiles []string
	for _, pattern := range patterns {
		matches, err := doublestar.FilepathGlob(pattern)
		if err != nil {
			return nil, fmt.Errorf("error finding files with pattern %q: %w", pattern, err)
		}
		allFiles = append(allFiles, matches...)
	}
	return allFiles, nil
}
