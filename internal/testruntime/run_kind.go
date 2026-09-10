package testruntime

import "strings"

// RunKind describes the selection a Plur invocation made. The cache uses this
// to decide whether to update file-level aggregates.
type RunKind int

const (
	// RunKindAggregate is a default/full-file run. It may rewrite file-level
	// aggregates.
	RunKindAggregate RunKind = iota
	// RunKindPartial is any non-aggregate run: focused (file:line), tag
	// filtered, fail-fast, or custom-arg. It may merge per-example
	// observations but must not touch file-level aggregates.
	RunKindPartial
)

// IsAggregateEligible reports whether a run of the given kind may update
// file-level aggregates.
func (k RunKind) IsAggregateEligible() bool {
	return k == RunKindAggregate
}

// ClassifyRunKind determines whether the current invocation should be treated
// as aggregate-eligible (full default run) or partial. Partial classification
// is intentionally inclusive: tags, focused targets, and any passthrough args
// prevent a non-default run from overwriting full-file aggregates.
//
// Inputs:
//   - patterns:        positional Patterns from the CLI
//   - tags:            --tag values
//   - passthroughArgs: anything after `--`
func ClassifyRunKind(patterns, tags, passthroughArgs []string) RunKind {
	if len(tags) > 0 {
		return RunKindPartial
	}
	if hasFileLinePattern(patterns) {
		return RunKindPartial
	}
	if len(passthroughArgs) > 0 {
		return RunKindPartial
	}
	return RunKindAggregate
}

// hasFileLinePattern reports whether any positional pattern looks like a
// focused target (file:line, file[1:2], file[1:2,1:3]).
func hasFileLinePattern(patterns []string) bool {
	for _, p := range patterns {
		if strings.Contains(p, ":") || strings.Contains(p, "[") {
			return true
		}
	}
	return false
}
