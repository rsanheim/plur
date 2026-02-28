package watch

import (
	"github.com/rsanheim/plur/job"
	"github.com/rsanheim/plur/logger"
)

// JobExecutor is a function that executes a job with target files
type JobExecutor func(j job.Job, targets []string, cwd string) error

// FileEventHandler processes file change events and executes jobs
type FileEventHandler struct {
	Jobs    map[string]job.Job
	Watches []WatchMapping
	CWD     string

	// Executor runs jobs. Defaults to ExecuteJob if nil.
	Executor JobExecutor
}

func (h *FileEventHandler) executor() JobExecutor {
	if h.Executor != nil {
		return h.Executor
	}
	return ExecuteJob
}

// HandleResult contains the outcomes of processing file events
type HandleResult struct {
	ExecutedJobs []string // job names that were run
	ShouldReload bool     // true if any matched rule has Reload: true
}

// HandleBatch processes multiple file paths, aggregates targets, and executes jobs
func (h *FileEventHandler) HandleBatch(paths []string) HandleResult {
	if len(h.Watches) == 0 {
		return HandleResult{}
	}

	// 1. Aggregate results from all source files
	allExistingTargets := make(map[string][]string)
	allMatchedRules := []WatchMapping{}

	for _, path := range paths {
		result, err := FindTargetsForFile(path, h.Jobs, h.Watches, h.CWD)
		if err != nil {
			logger.Logger.Warn("Error processing file change", "path", path, "error", err)
			continue
		}

		// Always collect matched rules (needed for reload detection)
		allMatchedRules = append(allMatchedRules, result.MatchedRules...)

		if !result.HasExistingTargets() {
			logger.Logger.Debug("No existing targets for file", "path", path)
			continue
		}

		// Log missing targets
		for jobName, targets := range result.MissingTargets {
			for _, target := range targets {
				logger.Logger.Info("Skipping non-existent target", "target", target, "job", jobName)
			}
		}

		// Merge targets per job
		for jobName, targets := range result.ExistingTargets {
			allExistingTargets[jobName] = append(allExistingTargets[jobName], targets...)
		}
	}

	// Check for reload first (can happen even with no targets)
	shouldReload := false
	for _, rule := range allMatchedRules {
		if rule.Reload {
			logger.Logger.Info("Watch rule triggered reload", "source", rule.Source)
			shouldReload = true
			break
		}
	}

	if len(allExistingTargets) == 0 {
		return HandleResult{ShouldReload: shouldReload}
	}

	// 2. Dedupe targets per job
	for jobName := range allExistingTargets {
		allExistingTargets[jobName] = Deduplicate(allExistingTargets[jobName])
	}

	// 3. Execute jobs in matched rule order
	var executedJobs []string
	seenJobs := make(map[string]bool)
	for _, rule := range allMatchedRules {
		for _, jobName := range rule.Jobs {
			if seenJobs[jobName] {
				continue
			}
			seenJobs[jobName] = true

			j, exists := h.Jobs[jobName]
			if !exists {
				logger.Logger.Warn("Job not found", "job", jobName)
				continue
			}

			targets := allExistingTargets[jobName]
			if err := h.executor()(j, targets, h.CWD); err != nil {
				logger.Logger.Warn("Job execution error", "job", jobName, "error", err)
			}
			executedJobs = append(executedJobs, jobName)
		}
	}

	return HandleResult{
		ExecutedJobs: executedJobs,
		ShouldReload: shouldReload,
	}
}
