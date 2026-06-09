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

type MatchedWatch struct {
	Watch   WatchMapping
	Targets []string
}

// NewEventProcessor creates a new EventProcessor with the given jobs and watch mappings
func NewEventProcessor(jobs map[string]framework.Job, watches []WatchMapping) *EventProcessor {
	return &EventProcessor{
		jobs:    jobs,
		watches: watches,
	}
}

// ProcessPath maps a file path to target files per job
// Returns a map of jobName -> []targetFiles
// If a watch mapping has no targets configured, the source file itself is used.
// If no_targets is true, the job is still returned with an empty target list.
func (processor *EventProcessor) ProcessPath(path string) (map[string][]string, error) {
	results := make(map[string][]string)

	matches, err := processor.MatchPath(path)
	if err != nil {
		return nil, err
	}

	for _, match := range matches {
		// Add targets to each job specified in this watch
		for _, jobName := range match.Watch.Jobs {
			// Validate job exists
			if _, exists := processor.jobs[jobName]; !exists {
				return nil, fmt.Errorf("watch %q references undefined job %q", match.Watch.Name, jobName)
			}

			results[jobName] = append(results[jobName], match.Targets...)
		}
	}

	// Deduplicate targets per job
	for jobName := range results {
		results[jobName] = deduplicate(results[jobName])
	}

	return results, nil
}

func (processor *EventProcessor) MatchPath(path string) ([]MatchedWatch, error) {
	matches := make([]MatchedWatch, 0)
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

		// Determine target files
		targets, err := processor.renderTargets(watch, normalizedPath)
		logger.Logger.Debug("renderTargets result", "normalizedPath", normalizedPath, "watch", watch.Source, "targets", targets)
		if err != nil {
			return nil, fmt.Errorf("error rendering targets for watch %q: %w", watch.Name, err)
		}

		matches = append(matches, MatchedWatch{Watch: watch, Targets: targets})
	}

	return matches, nil
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
