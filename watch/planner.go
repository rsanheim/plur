package watch

import "github.com/rsanheim/plur/job"

type JobPlan struct {
	JobName string
	Job     job.Job
	Targets []string
}

type PlanError struct {
	Path string
	Err  error
}

type Plan struct {
	Paths             []string
	MatchedRules      []WatchMapping
	ExistingTargets   map[string][]string
	MissingTargets    map[string][]string
	JobPlans          []JobPlan
	Errors            []PlanError
	ShouldReload      bool
	NoRunnableChanges []NoRunnableChange
}

type Planner struct {
	Jobs    map[string]job.Job
	Watches []WatchMapping
	CWD     string
}

func (p Planner) PlanPath(path string) Plan {
	return p.PlanBatch([]string{path})
}

func (p Planner) PlanBatch(paths []string) Plan {
	plan := Plan{
		Paths:           append([]string{}, paths...),
		ExistingTargets: make(map[string][]string),
		MissingTargets:  make(map[string][]string),
	}
	if len(p.Watches) == 0 {
		return plan
	}

	allExistingTargets := make(map[string][]string)
	allMatchedRules := []WatchMapping{}
	noRunnableChanges := []NoRunnableChange{}

	for _, path := range paths {
		result, err := FindTargetsForFile(path, p.Jobs, p.Watches, p.CWD)
		if err != nil {
			plan.Errors = append(plan.Errors, PlanError{Path: path, Err: err})
			continue
		}

		allMatchedRules = append(allMatchedRules, result.MatchedRules...)
		mergeTargetMap(plan.MissingTargets, result.MissingTargets)

		if len(result.MatchedRules) == 0 {
			noRunnableChanges = append(noRunnableChanges, NoRunnableChange{
				Path:   path,
				Reason: NoRunnableNoRule,
			})
			continue
		}

		if !result.HasExistingTargets() {
			if !hasReloadRule(result.MatchedRules) {
				noRunnableChanges = append(noRunnableChanges, NoRunnableChange{
					Path:           path,
					Reason:         NoRunnableMissingTargets,
					MissingTargets: missingTargetList(result.MissingTargets),
				})
			}
			continue
		}

		mergeTargetMap(allExistingTargets, result.ExistingTargets)
	}

	plan.MatchedRules = allMatchedRules
	plan.NoRunnableChanges = noRunnableChanges
	plan.ShouldReload = hasReloadRule(allMatchedRules)

	if len(allExistingTargets) == 0 {
		return plan
	}

	for jobName := range allExistingTargets {
		allExistingTargets[jobName] = Deduplicate(allExistingTargets[jobName])
	}
	plan.ExistingTargets = allExistingTargets

	seenJobs := make(map[string]bool)
	for _, rule := range allMatchedRules {
		for _, jobName := range rule.Jobs {
			if seenJobs[jobName] {
				continue
			}
			seenJobs[jobName] = true

			j, exists := p.Jobs[jobName]
			if !exists {
				continue
			}

			plan.JobPlans = append(plan.JobPlans, JobPlan{
				JobName: jobName,
				Job:     j,
				Targets: allExistingTargets[jobName],
			})
		}
	}

	return plan
}

func mergeTargetMap(dest, source map[string][]string) {
	for jobName, targets := range source {
		dest[jobName] = append(dest[jobName], targets...)
	}
}
