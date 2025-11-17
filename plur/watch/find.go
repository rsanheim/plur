package watch

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/rsanheim/plur/job"
	"github.com/rsanheim/plur/logger"
)

// FindResult contains the results of finding targets for a file change
type FindResult struct {
	FilePath        string
	MatchedRules    []*WatchMapping     // Watch rules that matched the file
	ExistingTargets map[string][]string // jobName -> target files that exist
	MissingTargets  map[string][]string // jobName -> target files that don't exist
	Jobs            map[string]*job.Job // All jobs referenced
}

// HasExistingTargets returns true if any targets exist
func (r *FindResult) HasExistingTargets() bool {
	for _, targets := range r.ExistingTargets {
		if len(targets) > 0 {
			return true
		}
	}
	return false
}

// HasMissingTargets returns true if any targets are missing
func (r *FindResult) HasMissingTargets() bool {
	for _, targets := range r.MissingTargets {
		if len(targets) > 0 {
			return true
		}
	}
	return false
}

// FindTargetsForFile determines what would be executed for a given file change
// It returns all matched rules and separates existing vs missing target files
func FindTargetsForFile(filePath string, jobs map[string]*job.Job, watches []*WatchMapping) (*FindResult, error) {
	processor := NewEventProcessor(jobs, watches)

	// Get candidate targets from event processor
	candidateTargets, err := processor.ProcessPath(filePath)
	if err != nil {
		return nil, err
	}

	result := &FindResult{
		FilePath:        filePath,
		MatchedRules:    make([]*WatchMapping, 0),
		ExistingTargets: make(map[string][]string),
		MissingTargets:  make(map[string][]string),
		Jobs:            jobs,
	}

	// Find which rules actually matched
	for _, w := range watches {
		if matchesWatch(filePath, w) {
			result.MatchedRules = append(result.MatchedRules, w)
		}
	}

	// Filter targets by existence
	for jobName, targets := range candidateTargets {
		for _, target := range targets {
			fmt.Println("target", target)
			if _, err := os.Stat(target); err == nil {
				result.ExistingTargets[jobName] = append(result.ExistingTargets[jobName], target)
			} else {
				result.MissingTargets[jobName] = append(result.MissingTargets[jobName], target)
				logger.LogVerbose("Skipping non-existent target", "target", target, "job", jobName)
			}
		}
	}

	return result, nil
}

// matchesWatch checks if a file path matches a watch mapping
// This duplicates some logic from EventProcessor.ProcessPath but is needed
// to identify which rules matched without re-implementing the whole thing
func matchesWatch(filePath string, w *WatchMapping) bool {
	// Check exclude patterns first
	for _, exclude := range w.Exclude {
		if matches(filePath, exclude) {
			return false
		}
	}

	// Check if matches source pattern
	return matches(filePath, w.Source)
}

// matches checks if a path matches a glob pattern
func matches(path, pattern string) bool {
	normalizedPath := filepath.ToSlash(path)
	matched, err := doublestar.Match(filepath.ToSlash(pattern), normalizedPath)
	if err != nil {
		return false
	}
	return matched
}
