package watch

import (
	"github.com/rsanheim/plur/job"
	"github.com/rsanheim/plur/logger"
)

// JobExecutor is a function that executes a planned watch job.
type JobExecutor func(plan ExecutionPlan) error

type NoRunnableReason string

const (
	NoRunnableNoRule         NoRunnableReason = "no_matching_rule"
	NoRunnableMissingTargets NoRunnableReason = "missing_targets"
)

type NoRunnableChange struct {
	Path           string
	Reason         NoRunnableReason
	MissingTargets []string
}

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

func (h *FileEventHandler) planner() Planner {
	return Planner{
		Jobs:    h.Jobs,
		Watches: h.Watches,
		CWD:     h.CWD,
	}
}

// HandleResult contains the outcomes of processing file events
type HandleResult struct {
	ExecutedJobs      []string // job names that were run
	ExecutedPlans     []ExecutionPlan
	ShouldReload      bool // true if any matched rule has Reload: true
	PlanningErrors    []PlanError
	NoRunnableChanges []NoRunnableChange
}

// HandleBatch processes multiple file paths, aggregates targets, and executes jobs
func (h *FileEventHandler) HandleBatch(paths []string) HandleResult {
	plan := h.planner().PlanBatch(paths)
	var executedJobs []string

	if plan.ShouldReload {
		for _, rule := range plan.MatchedRules {
			if rule.Reload {
				logger.Logger.Info("Watch rule triggered reload", "source", rule.Source)
				break
			}
		}
	}

	for jobName, targets := range plan.MissingTargets {
		for _, target := range targets {
			logger.Logger.Info("Skipping non-existent target", "target", target, "job", jobName)
		}
	}
	for _, err := range plan.Errors {
		if err.Err != nil {
			logger.Logger.Warn("Watch planning error", "path", err.Path, "error", err.Err)
		}
	}

	executionPlans := BuildExecutionPlans(plan.JobPlans, h.CWD)
	for _, executionPlan := range executionPlans {
		if err := h.executor()(executionPlan); err != nil {
			logger.Logger.Warn("Job execution error", "job", executionPlan.JobName, "error", err)
		}
		executedJobs = append(executedJobs, executionPlan.JobName)
	}

	return HandleResult{
		ExecutedJobs:      executedJobs,
		ExecutedPlans:     executionPlans,
		ShouldReload:      plan.ShouldReload,
		PlanningErrors:    plan.Errors,
		NoRunnableChanges: plan.NoRunnableChanges,
	}
}

func hasReloadRule(rules []WatchMapping) bool {
	for _, rule := range rules {
		if rule.Reload {
			return true
		}
	}
	return false
}

func missingTargetList(targetsByJob map[string][]string) []string {
	var targets []string
	for _, jobTargets := range targetsByJob {
		targets = append(targets, jobTargets...)
	}
	return Deduplicate(targets)
}
