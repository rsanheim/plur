package watch

import (
	"github.com/rsanheim/plur/job"
	"github.com/rsanheim/plur/logger"
)

// JobExecutor is a function that executes a job with target files
type JobExecutor func(j job.Job, targets []string, cwd string) error

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
	ShouldReload      bool     // true if any matched rule has Reload: true
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

	for _, jobPlan := range plan.JobPlans {
		if err := h.executor()(jobPlan.Job, jobPlan.Targets, h.CWD); err != nil {
			logger.Logger.Warn("Job execution error", "job", jobPlan.JobName, "error", err)
		}
		executedJobs = append(executedJobs, jobPlan.JobName)
	}

	return HandleResult{
		ExecutedJobs:      executedJobs,
		ShouldReload:      plan.ShouldReload,
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
