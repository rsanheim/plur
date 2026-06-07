package runtime

import (
	"fmt"
	"maps"
	"os"
	"slices"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/rsanheim/plur/internal/framework"
	"github.com/rsanheim/plur/logger"
	"github.com/rsanheim/plur/watch"
)

type CLIInput struct {
	Use           string
	Jobs          map[string]framework.Job
	WatchMappings []watch.WatchMapping
	ConfigFiles   []string
}

type RuntimeConfig struct {
	Use       string
	Jobs      map[string]framework.Job
	Watches   []watch.WatchMapping
	Inherited map[string]InheritedFields
	Sources   []string
}

func BuildRuntimeConfig(cli *CLIInput) (*RuntimeConfig, error) {
	jobs, inherited, err := buildResolvedJobs(cli.Jobs)
	if err != nil {
		return nil, err
	}

	var sources []string
	for _, configFile := range cli.ConfigFiles {
		expanded := kong.ExpandPath(configFile)
		if _, err := os.Stat(expanded); err == nil {
			sources = append(sources, expanded)
		}
	}

	rc := &RuntimeConfig{
		Use:       cli.Use,
		Jobs:      jobs,
		Inherited: inherited,
		Sources:   sources,
	}

	jobName := rc.Use
	if jobName == "" {
		jobName, _ = autodetectJobName(rc.Jobs)
	}
	var builtins []watch.WatchMapping
	for _, w := range builtinDefaults.Defaults.Watches {
		if jobName != "" && slices.Contains(w.Jobs, jobName) {
			builtins = append(builtins, w)
		}
	}
	rc.Watches = watch.MergeWatches(builtins, cli.WatchMappings)

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
		for _, arg := range j.Cmd {
			if strings.Contains(arg, "{{") || strings.Contains(arg, "}}") {
				return fmt.Errorf("configuration error in %v: job %q command must not contain template tokens", rc.Sources, name)
			}
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
	Job       framework.Job
	Reason    ResolveReason
	Inherited InheritedFields
}

func SelectJobFromRuntimeConfig(rc *RuntimeConfig, patterns []string) (*SelectedJob, error) {
	if rc.Use != "" {
		return buildSelectedJob(rc, rc.Use, ResolveReasonExplicitName)
	}

	if len(patterns) > 0 {
		if frameworkName, err := inferFrameworkFromPatterns(patterns); err != nil {
			return nil, err
		} else if frameworkName != "" {
			return buildSelectedJob(rc, frameworkName, ResolveReasonExplicitPatterns)
		}
	}

	name, err := autodetectJobName(rc.Jobs)
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
		available := slices.Sorted(maps.Keys(rc.Jobs))
		return nil, fmt.Errorf("job '%s' not found. Available jobs: %s", name, strings.Join(available, ", "))
	}
	resolved, err := j.ResolveFramework()
	if err != nil {
		return nil, err
	}
	return &SelectedJob{
		Name:      name,
		Job:       resolved,
		Reason:    reason,
		Inherited: rc.Inherited[name],
	}, nil
}

func LogInheritedFields(jobName string, inherited InheritedFields) {
	if inherited == (InheritedFields{}) {
		return
	}
	logger.Logger.Info("job inherited defaults",
		"job", jobName,
		"cmd", inherited.Cmd,
		"env", inherited.Env,
		"framework", inherited.Framework,
		"target_pattern", inherited.TargetPattern,
		"exclude_patterns", inherited.ExcludePatterns)
}
