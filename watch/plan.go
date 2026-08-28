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
	Rule     WatchMapping
	Existing TargetSet
	Missing  TargetSet
}

// JobRun is one job a plan executes. An empty target set runs the job without
// target arguments.
type JobRun struct {
	Job     framework.Job
	Targets TargetSet
}

// Plan is the complete answer to "what would watch do for these paths?"
// plur watch executes Runs; plur watch find prints them.
type Plan struct {
	Matches []Match
	Runs    []JobRun
}

// Admit normalizes a changed path to be relative to CWD and applies the
// global ignore patterns. Paths outside CWD are rejected: live watch roots
// are filtered to the project, so a plan for an outside file could never
// happen in a real watch session. The live event loop and watch find both
// route paths through here so they agree on what gets processed. The
// returned path is valid for display even when ok is false.
func (p Planner) Admit(path string) (string, bool) {
	if filepath.IsAbs(path) {
		// CWD is symlink-resolved; resolve the input the same way so
		// logical paths (macOS /tmp, symlinked checkouts) relativize
		// correctly. Fall back to the raw path when it does not exist.
		if resolved, err := filepath.EvalSymlinks(path); err == nil {
			path = resolved
		}
		rel, err := filepath.Rel(p.CWD, path)
		if err != nil {
			logger.Logger.Debug("Rejecting unrelativizable path", "path", path, "cwd", p.CWD, "error", err)
			return path, false
		}
		path = rel
	} else {
		path = filepath.Clean(path)
		if filepath.IsLocal(path) && p.CWD != "" {
			absPath := filepath.Join(p.CWD, path)
			if resolved, err := filepath.EvalSymlinks(absPath); err == nil {
				rel, err := filepath.Rel(p.CWD, resolved)
				if err != nil || !filepath.IsLocal(rel) {
					logger.Logger.Warn("Skipping path outside project", "path", path, "resolved", resolved, "cwd", p.CWD)
					return path, false
				}
			}
		}
	}
	if !filepath.IsLocal(path) {
		logger.Logger.Warn("Skipping path outside project", "path", path, "cwd", p.CWD)
		return path, false
	}
	if matchesAny(filepath.ToSlash(path), p.IgnorePatterns) {
		return path, false
	}
	return path, true
}

// Plan decides which jobs run, with which targets, for a batch of changed
// paths. Paths must already be CWD-relative (see Admit).
func (p Planner) Plan(paths TargetSet) Plan {
	plan := Plan{}
	for path := range paths.All() {
		matches := p.matchPath(path)
		if !anyRunnable(matches) {
			logger.Logger.Debug("No existing targets for file", "path", path)
		}
		plan.Matches = append(plan.Matches, matches...)
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

		m := Match{Rule: rule}
		if len(rule.Jobs) > 0 {
			var existing, missing []string
			for target := range renderRuleTargets(rule, normalized).All() {
				if targetExists(target, p.CWD) {
					existing = append(existing, target)
				} else {
					logger.Logger.Info("Skipping non-existent target", "target", target, "rule", rule.Name, "source", rule.Source)
					missing = append(missing, target)
				}
			}
			m.Existing = NewTargetSet(existing...)
			m.Missing = NewTargetSet(missing...)
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

// anyRunnable reports whether any match can produce a job run: it has
// existing targets or is a no_targets rule that runs its jobs bare.
func anyRunnable(matches []Match) bool {
	for _, m := range matches {
		if m.Existing.Len() > 0 {
			return true
		}
		if m.Rule.NoTargets && len(m.Rule.Jobs) > 0 {
			return true
		}
	}
	return false
}

// collectJobTargets gathers deduplicated existing targets for a job across
// all matches. runnable is true when targets exist or a no_targets rule
// matched for the job.
func collectJobTargets(matches []Match, jobName string) (targets TargetSet, runnable bool) {
	var collected []string
	for _, m := range matches {
		if !slices.Contains(m.Rule.Jobs, jobName) {
			continue
		}
		if m.Rule.NoTargets {
			runnable = true
		}
		for target := range m.Existing.All() {
			collected = append(collected, target)
		}
	}
	targets = NewTargetSet(collected...)
	if targets.Len() > 0 {
		runnable = true
	}
	return targets, runnable
}

// renderRuleTargets renders a rule's target templates for a matched path.
// no_targets rules render nothing; rules without targets use the source
// file itself.
func renderRuleTargets(rule WatchMapping, normalizedPath string) TargetSet {
	if rule.NoTargets {
		return TargetSet{}
	}
	if len(rule.Targets) == 0 {
		return NewTargetSet(filepath.FromSlash(normalizedPath))
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
	return NewTargetSet(targets...)
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
