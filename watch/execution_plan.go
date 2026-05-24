package watch

import (
	"slices"
	"strings"

	"github.com/rsanheim/plur/job"
)

type ExecutionPlan struct {
	JobName string
	Job     job.Job
	Targets []string
	Argv    []string
	Env     []string
	CWD     string
}

func BuildExecutionPlans(jobPlans []JobPlan, cwd string) []ExecutionPlan {
	plans := make([]ExecutionPlan, 0, len(jobPlans))
	for _, plan := range jobPlans {
		plans = append(plans, BuildExecutionPlan(plan.Job, plan.Targets, cwd))
	}
	return plans
}

func BuildExecutionPlan(j job.Job, targets []string, cwd string) ExecutionPlan {
	targets = slices.Clone(targets)
	return ExecutionPlan{
		JobName: j.Name,
		Job:     j,
		Targets: targets,
		Argv:    job.BuildJobCmd(j, targets),
		Env:     dedupeEnvByKey(validEnvEntries(j.Env)),
		CWD:     cwd,
	}
}

func validEnvEntries(envs []string) []string {
	entries := make([]string, 0, len(envs))
	for _, env := range envs {
		if strings.Contains(env, "=") {
			entries = append(entries, env)
		}
	}
	return entries
}

func dedupeEnvByKey(envs []string) []string {
	lastIndex := make(map[string]int, len(envs))
	for i, env := range envs {
		key, _, ok := strings.Cut(env, "=")
		if ok {
			lastIndex[key] = i
		}
	}

	entries := make([]string, 0, len(lastIndex))
	for i, env := range envs {
		key, _, ok := strings.Cut(env, "=")
		if ok && lastIndex[key] == i {
			entries = append(entries, env)
		}
	}

	return entries
}
