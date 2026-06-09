package watch

import (
	"fmt"
	"path/filepath"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/rsanheim/plur/internal/framework"
	"github.com/rsanheim/plur/logger"
)

// EventProcessor maps file change events to jobs with target files
// It does NOT watch files (that's WatcherManager's job)
// It only determines: "given a file changed, what jobs should run and with what targets?"
type EventProcessor struct {
	jobs    map[string]framework.Job
	watches []WatchMapping
}

// ProcessResult is the pure watch matching output before filesystem checks.
type ProcessResult struct {
	MatchedRules     []WatchMapping
	CandidateTargets map[string][]string
	NoTargetJobs     map[string]bool
}

// NewEventProcessor creates a new EventProcessor with the given jobs and watch mappings
func NewEventProcessor(jobs map[string]framework.Job, watches []WatchMapping) *EventProcessor {
	return &EventProcessor{
		jobs:    jobs,
		watches: watches,
	}
}

// ProcessPath maps a file path to candidate target files per job.
// If a watch mapping has no targets configured, the source file itself is used.
// If no_targets is true, the job is returned through NoTargetJobs.
func (processor *EventProcessor) ProcessPath(path string) (*ProcessResult, error) {
	result := &ProcessResult{
		MatchedRules:     make([]WatchMapping, 0),
		CandidateTargets: make(map[string][]string),
		NoTargetJobs:     make(map[string]bool),
	}

	normalizedPath := filepath.ToSlash(path)

	for _, watch := range processor.watches {
		if processor.isIgnored(normalizedPath, watch.Ignore) {
			continue
		}

		matched, err := doublestar.Match(filepath.ToSlash(watch.Source), normalizedPath)
		if err != nil {
			return nil, fmt.Errorf("error matching pattern %q: %w", watch.Source, err)
		}

		if !matched {
			continue
		}

		result.MatchedRules = append(result.MatchedRules, watch)

		// Determine target files
		targets, err := processor.renderTargets(watch, normalizedPath)
		logger.Logger.Debug("renderTargets result", "normalizedPath", normalizedPath, "watch", watch.Source, "targets", targets)
		if err != nil {
			return nil, fmt.Errorf("error rendering targets for watch %q: %w", watch.Name, err)
		}

		// Add targets to each job specified in this watch
		for _, jobName := range watch.Jobs {
			// Validate job exists
			if _, exists := processor.jobs[jobName]; !exists {
				return nil, fmt.Errorf("watch %q references undefined job %q", watch.Name, jobName)
			}

			if watch.NoTargets {
				result.NoTargetJobs[jobName] = true
			} else {
				result.CandidateTargets[jobName] = append(result.CandidateTargets[jobName], targets...)
			}
		}
	}

	// Deduplicate targets per job
	for jobName := range result.CandidateTargets {
		result.CandidateTargets[jobName] = deduplicate(result.CandidateTargets[jobName])
	}

	return result, nil
}

// renderTargets renders the target templates for a watch mapping
// If no_targets is true, returns an empty target list. If no targets are
// specified, returns the source path.
func (processor *EventProcessor) renderTargets(watch WatchMapping, path string) ([]string, error) {
	if watch.NoTargets {
		return nil, nil
	}

	// If no targets specified, use the source file itself
	if len(watch.Targets) == 0 {
		return []string{filepath.FromSlash(path)}, nil
	}

	// Build tokens from path and source pattern
	tokens := BuildTokens(path, watch.Source)
	logger.Logger.Debug("tokens", "path", path, "watch", watch.Source, "tokens", fmt.Sprintf("%+v", tokens))

	// Render each target template
	targets := make([]string, 0, len(watch.Targets))
	for _, targetTemplate := range watch.Targets {
		rendered, err := RenderTemplate(targetTemplate, tokens)
		if err != nil {
			return nil, fmt.Errorf("error rendering target template %q: %w", targetTemplate, err)
		}
		targets = append(targets, rendered)
	}

	return targets, nil
}

// isIgnored checks if a path matches any of the ignore patterns
func (processor *EventProcessor) isIgnored(path string, ignorePatterns []string) bool {
	for _, pattern := range ignorePatterns {
		matched, err := doublestar.Match(filepath.ToSlash(pattern), path)
		if err == nil && matched {
			return true
		}
	}
	return false
}

// Deduplicate removes duplicate strings from a slice while preserving order
func deduplicate(items []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(items))

	for _, item := range items {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}

	return result
}
