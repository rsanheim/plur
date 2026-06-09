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

type BatchWatchPlan struct {
	MatchedRules []WatchMapping
	JobRuns      []JobRun
	ShouldReload bool
}

// HandleBatch processes multiple file paths, aggregates targets, and executes jobs
func (h *FileEventHandler) HandleBatch(paths []string) HandleResult {
	if len(h.Watches) == 0 {
		return HandleResult{}
	}

	plan := h.PlanBatch(paths)
	if len(plan.JobRuns) == 0 {
		return HandleResult{ShouldReload: plan.ShouldReload}
	}

	var executedJobs []string
	for _, run := range plan.JobRuns {
		if err := h.executor()(run.Job, run.Targets, h.CWD); err != nil {
			logger.Logger.Warn("Job execution error", "job", run.JobName, "error", err)
		}
		executedJobs = append(executedJobs, run.JobName)
	}

	return HandleResult{
		ExecutedJobs: executedJobs,
		ShouldReload: plan.ShouldReload,
	}
}

func (h *FileEventHandler) PlanBatch(paths []string) BatchWatchPlan {
	plan := BatchWatchPlan{
		MatchedRules: make([]WatchMapping, 0),
		JobRuns:      make([]JobRun, 0),
	}

	runsByJob := make(map[string]int)

	for _, path := range paths {
		filePlan, err := PlanWatchForFile(path, h.Jobs, h.Watches, h.CWD)
		if err != nil {
			logger.Logger.Warn("Error processing file change", "path", path, "error", err)
			continue
		}

		plan.MatchedRules = append(plan.MatchedRules, filePlan.MatchedRules...)

		if filePlan.ShouldReload() {
			plan.ShouldReload = true
		}

		if len(filePlan.JobRuns) == 0 {
			logger.Logger.Debug("No existing targets for file", "path", path)
			continue
		}

		for _, run := range filePlan.JobRuns {
			index, exists := runsByJob[run.JobName]
			if !exists {
				if !run.NoTargets {
					run.Targets = deduplicate(run.Targets)
				}
				runsByJob[run.JobName] = len(plan.JobRuns)
				plan.JobRuns = append(plan.JobRuns, run)
				continue
			}

			existingRun := &plan.JobRuns[index]
			if existingRun.NoTargets {
				if !run.NoTargets {
					existingRun.Targets = deduplicate(run.Targets)
					existingRun.NoTargets = false
				}
				continue
			}
			if run.NoTargets {
				continue
			}

			existingRun.Targets = deduplicate(append(existingRun.Targets, run.Targets...))
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

	return plan
}
