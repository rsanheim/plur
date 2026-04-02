package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/rsanheim/plur/autodetect"
	"github.com/rsanheim/plur/job"
)

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
