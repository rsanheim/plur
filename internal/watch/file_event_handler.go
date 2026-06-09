package watch

import (
	"github.com/rsanheim/plur/internal/framework"
	"github.com/rsanheim/plur/logger"
)

// JobExecutor is a function that executes a job with target files
type JobExecutor func(j framework.Job, targets []string, cwd string) error

// FileEventHandler processes file change events and executes jobs
type FileEventHandler struct {
	Jobs    map[string]framework.Job
	Watches []WatchMapping
	CWD     string

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

// PathPlanError records a planning error for one changed path.
type PathPlanError struct {
	Path string
	Err  error
}

// BatchPlan is the side-effect-free execution plan for a batch of file changes.
type BatchPlan struct {
	Jobs            []JobPlan
	MatchedRules    []WatchMapping
	MissingTargets  map[string][]string
	NoRunnablePaths []string
	ShouldReload    bool
	Errors          []PathPlanError
}

// PlanBatch aggregates file changes into explicit runnable jobs.
func PlanBatch(paths []string, jobs map[string]framework.Job, watches []WatchMapping, cwd string) BatchPlan {
	plan := BatchPlan{
		Jobs:            make([]JobPlan, 0),
		MatchedRules:    make([]WatchMapping, 0),
		MissingTargets:  make(map[string][]string),
		NoRunnablePaths: make([]string, 0),
		Errors:          make([]PathPlanError, 0),
	}

	if len(watches) == 0 {
		return plan
	}

	allTargets := make(map[string][]string)
	runnableJobs := make(map[string]framework.Job)
	allMatchedRules := []WatchMapping{}

	for _, path := range paths {
		result, err := FindTargetsForFile(path, jobs, watches, cwd)
		if err != nil {
			plan.Errors = append(plan.Errors, PathPlanError{Path: path, Err: err})
			continue
		}

		allMatchedRules = append(allMatchedRules, result.MatchedRules...)

		if len(result.RunnableJobs) == 0 {
			plan.NoRunnablePaths = append(plan.NoRunnablePaths, path)
		}

		for jobName, targets := range result.MissingTargets {
			plan.MissingTargets[jobName] = append(plan.MissingTargets[jobName], targets...)
		}

		for _, jobPlan := range result.RunnableJobs {
			runnableJobs[jobPlan.Name] = jobPlan.Job
			allTargets[jobPlan.Name] = append(allTargets[jobPlan.Name], jobPlan.Targets...)
		}
	}

	for _, rule := range allMatchedRules {
		if rule.Reload {
			plan.ShouldReload = true
			break
		}
	}

	for jobName := range allTargets {
		allTargets[jobName] = deduplicate(allTargets[jobName])
	}
	for jobName := range plan.MissingTargets {
		plan.MissingTargets[jobName] = deduplicate(plan.MissingTargets[jobName])
	}

	seenJobs := make(map[string]bool)
	for _, rule := range allMatchedRules {
		for _, jobName := range rule.Jobs {
			if seenJobs[jobName] {
				continue
			}

			job, runnable := runnableJobs[jobName]
			if !runnable {
				continue
			}

			plan.Jobs = append(plan.Jobs, JobPlan{
				Name:    jobName,
				Job:     job,
				Targets: allTargets[jobName],
			})
			seenJobs[jobName] = true
		}
	}

	plan.MatchedRules = allMatchedRules

	return plan
}

// HandleBatch processes multiple file paths, aggregates targets, and executes jobs
func (h *FileEventHandler) HandleBatch(paths []string) HandleResult {
	if len(h.Watches) == 0 {
		return HandleResult{}
	}

	plan := PlanBatch(paths, h.Jobs, h.Watches, h.CWD)

	for _, err := range plan.Errors {
		logger.Logger.Warn("Error processing file change", "path", err.Path, "error", err.Err)
	}
	for _, path := range plan.NoRunnablePaths {
		logger.Logger.Debug("No existing targets for file", "path", path)
	}
	for jobName, targets := range plan.MissingTargets {
		for _, target := range targets {
			logger.Logger.Info("Skipping non-existent target", "target", target, "job", jobName)
		}
	}
	if plan.ShouldReload {
		for _, rule := range plan.MatchedRules {
			if rule.Reload {
				logger.Logger.Info("Watch rule triggered reload", "source", rule.Source)
				break
			}
		}
	}

	var executedJobs []string
	for _, jobPlan := range plan.Jobs {
		if err := h.executor()(jobPlan.Job, jobPlan.Targets, h.CWD); err != nil {
			logger.Logger.Warn("Job execution error", "job", jobPlan.Name, "error", err)
		}
		executedJobs = append(executedJobs, jobPlan.Name)
	}

	return HandleResult{
		ExecutedJobs: executedJobs,
		ShouldReload: plan.ShouldReload,
	}
}
