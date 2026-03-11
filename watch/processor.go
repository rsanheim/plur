package watch

import (
	"fmt"
	"path/filepath"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/rsanheim/plur/job"
	"github.com/rsanheim/plur/logger"
)

// EventProcessor maps file change events to jobs with target files
// It does NOT watch files (that's WatcherManager's job)
// It only determines: "given a file changed, what jobs should run and with what targets?"
type EventProcessor struct {
	jobs    map[string]job.Job
	watches []WatchMapping
}

// NewEventProcessor creates a new EventProcessor with the given jobs and watch mappings
func NewEventProcessor(jobs map[string]job.Job, watches []WatchMapping) *EventProcessor {
	return &EventProcessor{
		jobs:    jobs,
		watches: watches,
	}
}

// ProcessPath maps a file path to target files per job
// Returns a map of jobName -> []targetFiles
// If a watch mapping has no targets configured, the source file itself is used
func (ep *EventProcessor) ProcessPath(path string) (map[string][]string, error) {
	results := make(map[string][]string)
	normalizedPath := filepath.ToSlash(path)

	for _, watch := range ep.watches {
		if ep.isIgnored(normalizedPath, watch.Ignore) {
			continue
		}

		// Check if path matches the source pattern
		matched, err := doublestar.Match(filepath.ToSlash(watch.Source), normalizedPath)
		if err != nil {
			return nil, fmt.Errorf("error matching pattern %q: %w", watch.Source, err)
		}

		if !matched {
			// trace: logger.Logger.Debug("path does not match", "watch", watch.Source, "normalizedPath", normalizedPath)
			continue
		}

		// Determine target files
		targets, err := ep.renderTargets(watch, normalizedPath)
		logger.Logger.Debug("renderTargets result", "normalizedPath", normalizedPath, "watch", watch.Source, "targets", targets)
		if err != nil {
			return nil, fmt.Errorf("error rendering targets for watch %q: %w", watch.Name, err)
		}

		// Add targets to each job specified in this watch
		for _, jobName := range watch.Jobs {
			if _, exists := ep.jobs[jobName]; !exists {
				return nil, fmt.Errorf("watch %q references undefined job %q", watch.Name, jobName)
			}
			results[jobName] = append(results[jobName], targets...)
		}
	}

	for jobName := range results {
		results[jobName] = Deduplicate(results[jobName])
	}

	return results, nil
}

// renderTargets renders the target templates for a watch mapping
// If no targets are specified (empty slice), returns the source path
func (ep *EventProcessor) renderTargets(watch WatchMapping, path string) ([]string, error) {
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
func (ep *EventProcessor) isIgnored(path string, ignorePatterns []string) bool {
	for _, pattern := range ignorePatterns {
		matched, err := doublestar.Match(filepath.ToSlash(pattern), path)
		if err == nil && matched {
			return true
		}
	}
	return false
}

// Deduplicate removes duplicate strings from a slice while preserving order
func Deduplicate(items []string) []string {
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

// ValidateConfig validates the configuration before creating the processor
// It checks that all jobs referenced in watches exist
func ValidateConfig(jobs map[string]job.Job, watches []WatchMapping) error {
	for _, watch := range watches {
		for _, jobName := range watch.Jobs {
			if _, exists := jobs[jobName]; !exists {
				name := watch.Name
				if name == "" {
					name = fmt.Sprintf("watch with source %q", watch.Source)
				}
				return fmt.Errorf("%s references undefined job %q", name, jobName)
			}
		}

		// Validate target templates
		for _, target := range watch.Targets {
			if err := ValidateTemplate(target); err != nil {
				name := watch.Name
				if name == "" {
					name = fmt.Sprintf("watch with source %q", watch.Source)
				}
				return fmt.Errorf("%s has invalid target template %q: %w", name, target, err)
			}
		}
	}

	return nil
}
