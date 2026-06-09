package watch

import (
	"os"
	"path/filepath"

	"github.com/rsanheim/plur/internal/framework"
)

// JobPlan is an explicit executable watch job with resolved targets.
type JobPlan struct {
	Name    string
	Job     framework.Job
	Targets []string
}

// FindResult contains the results of finding targets for a file change
type FindResult struct {
	FilePath       string
	MatchedRules   []WatchMapping      // Watch rules that matched the file
	MissingTargets map[string][]string // jobName -> target files that don't exist
	RunnableJobs   []JobPlan           // Jobs that should execute for this file change
}

// HasExistingTargets returns true if any job would execute, including explicit
// no-target jobs.
func (r *FindResult) HasExistingTargets() bool {
	return len(r.RunnableJobs) > 0
}

func (r *FindResult) ExistingTargetFiles() []string {
	files := make([]string, 0)
	for _, jobPlan := range r.RunnableJobs {
		files = append(files, jobPlan.Targets...)
	}
	return deduplicate(files)
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
	processor := NewEventProcessor(jobs, watches)

	processResult, err := processor.ProcessPath(filePath)
	if err != nil {
		return nil, err
	}

	result := &FindResult{
		FilePath:       filePath,
		MatchedRules:   processResult.MatchedRules,
		MissingTargets: make(map[string][]string),
		RunnableJobs:   make([]JobPlan, 0),
	}

	existingTargets := make(map[string][]string)

	// Filter targets by existence (resolve relative paths against cwd)
	for jobName, targets := range processResult.CandidateTargets {
		for _, target := range targets {
			targetPath := target
			if cwd != "" && !filepath.IsAbs(target) {
				targetPath = filepath.Join(cwd, target)
			}
			if _, err := os.Stat(targetPath); err == nil {
				existingTargets[jobName] = append(existingTargets[jobName], target)
			} else {
				result.MissingTargets[jobName] = append(result.MissingTargets[jobName], target)
			}
		}
	}

	seen := make(map[string]bool)
	for _, rule := range result.MatchedRules {
		for _, jobName := range rule.Jobs {
			if seen[jobName] {
				continue
			}

			job, exists := jobs[jobName]
			if !exists {
				continue
			}

			if processResult.NoTargetJobs[jobName] {
				result.RunnableJobs = append(result.RunnableJobs, JobPlan{
					Name: jobName,
					Job:  job,
				})
				seen[jobName] = true
				continue
			}

			targets := existingTargets[jobName]
			if len(targets) == 0 {
				continue
			}

			result.RunnableJobs = append(result.RunnableJobs, JobPlan{
				Name:    jobName,
				Job:     job,
				Targets: targets,
			})
			seen[jobName] = true
		}
	}

	return result, nil
}
