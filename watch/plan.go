package watch

import (
	"os"
	"path/filepath"
	"slices"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/rsanheim/plur/internal/framework"
	"github.com/rsanheim/plur/logger"
)

// Planner holds everything needed to decide what a file change does.
// Both plur watch and plur watch find build it the same way from
// validated runtime config. Patterns and target templates are validated
// at config load, so planning cannot fail.
type Planner struct {
	Jobs           map[string]framework.Job
	Watches        []WatchMapping
	IgnorePatterns []string
	CWD            string
}

// Match is one watch rule that matched one changed file, with rendered
// targets split by whether they exist on disk. Rules without jobs match
// for reload and reporting purposes but never render targets.
type Match struct {
	Path     string
	Rule     WatchMapping
	Existing []string
	Missing  []string
}

// JobRun is one job a plan executes, with merged, deduplicated existing
// targets. Empty Targets means the job runs with no target arguments.
type JobRun struct {
	Job     framework.Job
	Targets []string
}

// Plan is the complete answer to "what would watch do for these paths?"
// plur watch executes Runs; plur watch find prints them.
type Plan struct {
	Matches []Match
	Runs    []JobRun
	Reload  bool
}

// Plan decides which jobs run, with which targets, for a batch of changed
// paths. Paths must already be CWD-relative (see Admit).
func (p Planner) Plan(paths []string) Plan {
	plan := Plan{}
	for _, path := range paths {
		plan.Matches = append(plan.Matches, p.matchPath(path)...)
	}

	for _, m := range plan.Matches {
		if m.Rule.Reload {
			logger.Logger.Info("Watch rule triggered reload", "source", m.Rule.Source)
			plan.Reload = true
			break
		}
	}

	plan.Runs = p.buildRuns(plan.Matches)
	return plan
}

// matchPath finds every rule matching one path and renders its targets.
func (p Planner) matchPath(path string) []Match {
	var matches []Match
	normalized := filepath.ToSlash(path)

	for _, rule := range p.Watches {
		if matchesAny(normalized, rule.Ignore) || !matchesPattern(normalized, rule.Source) {
			continue
		}

		m := Match{Path: path, Rule: rule}
		if len(rule.Jobs) > 0 {
			for _, target := range renderRuleTargets(rule, normalized) {
				if targetExists(target, p.CWD) {
					m.Existing = append(m.Existing, target)
				} else {
					logger.Logger.Info("Skipping non-existent target", "target", target, "rule", rule.Name, "source", rule.Source)
					m.Missing = append(m.Missing, target)
				}
			}
		}
		matches = append(matches, m)
	}
	return matches
}

// buildRuns merges match targets per job, preserving first-match job order.
// A job runs when it has existing targets or when a matching rule is
// no_targets (which runs the job bare).
func (p Planner) buildRuns(matches []Match) []JobRun {
	seen := make(map[string]bool)
	var runs []JobRun

	for _, m := range matches {
		for _, jobName := range m.Rule.Jobs {
			if seen[jobName] {
				continue
			}
			seen[jobName] = true

			targets, runnable := collectJobTargets(matches, jobName)
			if !runnable {
				continue
			}
			job, ok := p.Jobs[jobName]
			if !ok {
				// Watch job references are validated at config load; log rather
				// than executing a zero-value job if that invariant is ever broken.
				logger.Logger.Error("watch rule references unknown job", "job", jobName)
				continue
			}
			runs = append(runs, JobRun{Job: job, Targets: targets})
		}
	}
	return runs
}

// collectJobTargets gathers deduplicated existing targets for a job across
// all matches. runnable is true when targets exist or a no_targets rule
// matched for the job.
func collectJobTargets(matches []Match, jobName string) (targets []string, runnable bool) {
	for _, m := range matches {
		if !slices.Contains(m.Rule.Jobs, jobName) {
			continue
		}
		if m.Rule.NoTargets {
			runnable = true
		}
		targets = append(targets, m.Existing...)
	}
	targets = deduplicate(targets)
	if len(targets) > 0 {
		runnable = true
	}
	return targets, runnable
}

// renderRuleTargets renders a rule's target templates for a matched path.
// no_targets rules render nothing; rules without targets use the source
// file itself.
func renderRuleTargets(rule WatchMapping, normalizedPath string) []string {
	if rule.NoTargets {
		return nil
	}
	if len(rule.Targets) == 0 {
		return []string{filepath.FromSlash(normalizedPath)}
	}

	tokens := BuildTokens(normalizedPath, rule.Source)
	targets := make([]string, 0, len(rule.Targets))
	for _, tmpl := range rule.Targets {
		rendered, err := RenderTemplate(tmpl, tokens)
		if err != nil {
			// Templates are validated at config load; log rather than
			// silently dropping if that invariant is ever broken.
			logger.Logger.Error("failed to render target template", "template", tmpl, "error", err)
			continue
		}
		targets = append(targets, rendered)
	}
	return deduplicate(targets)
}

func targetExists(target, cwd string) bool {
	path := target
	if cwd != "" && !filepath.IsAbs(target) {
		path = filepath.Join(cwd, target)
	}
	_, err := os.Stat(path)
	return err == nil
}

func matchesPattern(normalizedPath, pattern string) bool {
	matched, err := doublestar.Match(filepath.ToSlash(pattern), normalizedPath)
	return err == nil && matched
}

func matchesAny(normalizedPath string, patterns []string) bool {
	for _, pattern := range patterns {
		if matchesPattern(normalizedPath, pattern) {
			return true
		}
	}
	return false
}
