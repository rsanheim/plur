package watch

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/rsanheim/plur/internal/framework"
	"github.com/rsanheim/plur/logger"
)

type JobRun struct {
	JobName   string
	Job       framework.Job
	Targets   []string
	NoTargets bool
}

type MissingTarget struct {
	JobName string
	Target  string
}

type WatchPlan struct {
	FilePath       string
	MatchedRules   []WatchMapping
	JobRuns        []JobRun
	MissingTargets []MissingTarget
}

func (p *WatchPlan) ShouldReload() bool {
	for _, rule := range p.MatchedRules {
		if rule.Reload {
			return true
		}
	}
	return false
}

// FindResult contains the results of finding targets for a file change
type FindResult struct {
	FilePath        string
	MatchedRules    []WatchMapping           // Watch rules that matched the file
	ExistingTargets map[string][]string      // jobName -> target files that exist
	MissingTargets  map[string][]string      // jobName -> target files that don't exist
	Jobs            map[string]framework.Job // All jobs referenced
	JobRuns         []JobRun                 // Explicit executable job plan
}

// HasExistingTargets returns true if any job has executable targets, including
// a matched no-target job represented by a present key with an empty slice.
func (r *FindResult) HasExistingTargets() bool {
	return len(r.ExistingTargets) > 0
}

// HasMissingTargets returns true if any targets are missing
func (r *FindResult) HasMissingTargets() bool {
	for _, targets := range r.MissingTargets {
		if len(targets) > 0 {
			return true
		}
	}
	return false
}

// FindTargetsForFile determines what would be executed for a given file change.
// The cwd parameter is used to resolve relative target paths for existence checks.
// It returns all matched rules and separates existing vs missing target files.
func FindTargetsForFile(filePath string, jobs map[string]framework.Job, watches []WatchMapping, cwd string) (*FindResult, error) {
	plan, err := PlanWatchForFile(filePath, jobs, watches, cwd)
	if err != nil {
		return nil, err
	}

	result := &FindResult{
		FilePath:        filePath,
		MatchedRules:    plan.MatchedRules,
		ExistingTargets: make(map[string][]string),
		MissingTargets:  make(map[string][]string),
		Jobs:            jobs,
		JobRuns:         plan.JobRuns,
	}

	for _, run := range plan.JobRuns {
		if run.NoTargets {
			if _, exists := result.ExistingTargets[run.JobName]; !exists {
				result.ExistingTargets[run.JobName] = nil
			}
			continue
		}
		result.ExistingTargets[run.JobName] = deduplicate(append(result.ExistingTargets[run.JobName], run.Targets...))
	}

	for _, missing := range plan.MissingTargets {
		result.MissingTargets[missing.JobName] = append(result.MissingTargets[missing.JobName], missing.Target)
	}

	return result, nil
}

func PlanWatchForFile(filePath string, jobs map[string]framework.Job, watches []WatchMapping, cwd string) (*WatchPlan, error) {
	processor := NewEventProcessor(jobs, watches)
	matches, err := processor.MatchPath(filePath)
	if err != nil {
		return nil, err
	}

	plan := &WatchPlan{
		FilePath:       filePath,
		MatchedRules:   make([]WatchMapping, 0, len(matches)),
		JobRuns:        make([]JobRun, 0),
		MissingTargets: make([]MissingTarget, 0),
	}

	for _, match := range matches {
		plan.MatchedRules = append(plan.MatchedRules, match.Watch)

		for _, jobName := range match.Watch.Jobs {
			job, exists := jobs[jobName]
			if !exists {
				return nil, fmt.Errorf("watch %q references undefined job %q", match.Watch.Name, jobName)
			}

			if match.Watch.NoTargets {
				plan.JobRuns = append(plan.JobRuns, JobRun{
					JobName:   jobName,
					Job:       job,
					NoTargets: true,
				})
				continue
			}

			existingTargets := make([]string, 0, len(match.Targets))
			for _, target := range match.Targets {
				targetPath := target
				if cwd != "" && !filepath.IsAbs(target) {
					targetPath = filepath.Join(cwd, target)
				}

				if _, err := os.Stat(targetPath); err == nil {
					existingTargets = append(existingTargets, target)
				} else {
					plan.MissingTargets = append(plan.MissingTargets, MissingTarget{JobName: jobName, Target: target})
					logger.Logger.Info("Skipping non-existent target", "target", target, "job", jobName)
				}
			}

			if len(existingTargets) > 0 {
				plan.JobRuns = append(plan.JobRuns, JobRun{
					JobName: jobName,
					Job:     job,
					Targets: existingTargets,
				})
			}
		}
	}

	return plan, nil
}
