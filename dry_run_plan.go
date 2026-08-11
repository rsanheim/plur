package main

import (
	"os"
	"os/exec"
	"strings"
)

// planEnv returns the env entries a worker command adds beyond the inherited
// environment — the configured job env plus plur-managed vars — deduplicated
// with the final value winning, so a JSON plan's argv+env reproduces the run.
func planEnv(cmd *exec.Cmd) []string {
	if cmd.Env == nil {
		return nil
	}

	envs := cmd.Env
	if inherited := os.Environ(); hasInheritedEnvPrefix(envs, inherited) {
		return withInheritedManagedEnv(dedupeEnvByKey(validEnvEntries(envs[len(inherited):])), envs)
	}

	var extras []string
	for _, env := range cmd.Environ() {
		if isManagedPlanEnvEntry(env) {
			extras = append(extras, env)
		}
	}
	return dedupeEnvByKey(extras)
}

func hasInheritedEnvPrefix(envs, inherited []string) bool {
	if len(envs) < len(inherited) {
		return false
	}
	for i, env := range inherited {
		if envs[i] != env {
			return false
		}
	}
	return true
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

func withInheritedManagedEnv(extras, envs []string) []string {
	seen := make(map[string]struct{}, len(extras))
	for _, env := range extras {
		if key, _, ok := strings.Cut(env, "="); ok {
			seen[key] = struct{}{}
		}
	}
	for _, env := range envs {
		key, _, ok := strings.Cut(env, "=")
		if !ok || key != "RAILS_ENV" {
			continue
		}
		if _, exists := seen[key]; !exists {
			return append([]string{env}, extras...)
		}
	}
	return extras
}

func isManagedPlanEnvEntry(env string) bool {
	key, _, ok := strings.Cut(env, "=")
	return ok && (key == EnvTestEnvNumber || key == EnvParallelTestGroups || key == "RAILS_ENV")
}

type DryRunPlan struct {
	Version  int                `json:"version"`
	Mode     string             `json:"mode"`
	Job      DryRunPlanJob      `json:"job"`
	Targets  []string           `json:"targets"`
	Warnings []string           `json:"warnings"`
	Workers  []DryRunPlanWorker `json:"workers"`
}

type DryRunPlanJob struct {
	Name      string `json:"name"`
	Framework string `json:"framework"`
	Reason    string `json:"reason"`
}

type DryRunPlanWorker struct {
	Index   int      `json:"index"`
	Targets []string `json:"targets"`
	Argv    []string `json:"argv"`
	Env     []string `json:"env"`
	Shell   string   `json:"shell"`
}

type RunnerDryRunPlan struct {
	Targets []string
	Workers []DryRunPlanWorker
}
