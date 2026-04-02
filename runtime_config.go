package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/rsanheim/plur/autodetect"
	"github.com/rsanheim/plur/job"
	"github.com/rsanheim/plur/logger"
	"github.com/rsanheim/plur/watch"
)

type RuntimeConfig struct {
	Use       string
	Jobs      map[string]job.Job
	Watches   []watch.WatchMapping
	Inherited map[string]autodetect.InheritedFields
	Sources   []string
}

func buildRuntimeConfig(cli *PlurCLI) (*RuntimeConfig, error) {
	jobs, inherited, err := autodetect.BuildResolvedJobs(cli.Job)
	if err != nil {
		return nil, err
	}

	rc := &RuntimeConfig{
		Use:       cli.Use,
		Jobs:      jobs,
		Inherited: inherited,
		Sources:   runtimeConfigSources(cli.configFiles),
	}

	if len(cli.WatchMappings) > 0 {
		rc.Watches = cli.WatchMappings
	} else if selected, err := selectJobFromRuntimeConfig(rc, nil); err == nil {
		rc.Watches = autodetect.BuiltinWatchesForJob(selected.Name)
	}

	if err := validateRuntimeConfig(rc); err != nil {
		return nil, err
	}
	return rc, nil
}

func validateRuntimeConfig(rc *RuntimeConfig) error {
	for name, j := range rc.Jobs {
		if len(j.Cmd) == 0 {
			return fmt.Errorf("configuration error in %v: job %q must define a command", rc.Sources, name)
		}
	}

	for _, w := range rc.Watches {
		for _, jobName := range w.Jobs {
			if _, ok := rc.Jobs[jobName]; !ok {
				return fmt.Errorf("configuration error in %v: watch %q references undefined job %q", rc.Sources, w.Name, jobName)
			}
		}
		for _, target := range w.Targets {
			if err := watch.ValidateTemplate(target); err != nil {
				return fmt.Errorf("configuration error in %v: watch %q has invalid target template %q: %w", rc.Sources, w.Name, target, err)
			}
		}
	}

	return nil
}

func runtimeConfigSources(configFiles []string) []string {
	var out []string
	for _, configFile := range configFiles {
		expanded := kong.ExpandPath(configFile)
		if _, err := os.Stat(expanded); err == nil {
			out = append(out, expanded)
		}
	}
	return out
}

// Job selection from RuntimeConfig

type ResolveReason string

const (
	ResolveReasonExplicitName            ResolveReason = "explicit_name"
	ResolveReasonExplicitPatterns        ResolveReason = "explicit_patterns"
	ResolveReasonAutodetect              ResolveReason = "autodetect"
	ResolveReasonAutodetectAfterPatterns ResolveReason = "autodetect_after_patterns"
)

type SelectedJob struct {
	Name      string
	Job       job.Job
	Reason    ResolveReason
	Inherited autodetect.InheritedFields
}

func selectJobFromRuntimeConfig(rc *RuntimeConfig, patterns []string) (*SelectedJob, error) {
	if rc.Use != "" {
		return buildSelectedJob(rc, rc.Use, ResolveReasonExplicitName)
	}

	if len(patterns) > 0 {
		if frameworkName, err := autodetect.InferFrameworkFromPatterns(patterns); err != nil {
			return nil, err
		} else if frameworkName != "" {
			return buildSelectedJob(rc, frameworkName, ResolveReasonExplicitPatterns)
		}
	}

	name, err := autodetect.AutodetectJobName(rc.Jobs)
	if err != nil {
		return nil, err
	}

	reason := ResolveReasonAutodetect
	if len(patterns) > 0 {
		reason = ResolveReasonAutodetectAfterPatterns
	}
	return buildSelectedJob(rc, name, reason)
}

func buildSelectedJob(rc *RuntimeConfig, name string, reason ResolveReason) (*SelectedJob, error) {
	j, ok := rc.Jobs[name]
	if !ok {
		return nil, buildJobNotFoundError(name, rc.Jobs)
	}
	return &SelectedJob{
		Name:      name,
		Job:       j,
		Reason:    reason,
		Inherited: rc.Inherited[name],
	}, nil
}

func buildJobNotFoundError(name string, jobs map[string]job.Job) error {
	available := make([]string, 0, len(jobs))
	for jobName := range jobs {
		available = append(available, jobName)
	}
	sort.Strings(available)
	return fmt.Errorf("job '%s' not found. Available jobs: %s", name, strings.Join(available, ", "))
}

func logInheritedFields(jobName string, inherited autodetect.InheritedFields) {
	if !inherited.Cmd && !inherited.Env && !inherited.Framework && !inherited.TargetPattern {
		return
	}
	logger.Logger.Info("job inherited defaults",
		"job", jobName,
		"cmd", inherited.Cmd,
		"env", inherited.Env,
		"framework", inherited.Framework,
		"target_pattern", inherited.TargetPattern)
}
