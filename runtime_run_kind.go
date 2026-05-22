package main

import (
	"strings"

	"github.com/rsanheim/plur/internal/testruntime"
)

// classifyRunKind determines whether the current invocation should be treated
// as aggregate-eligible (full default run) or partial. Partial classification
// is intentionally inclusive: any signal that the run is selective, filtered,
// or aborted demotes it so that a non-default run cannot overwrite the
// full-file aggregates produced by a true default run.
//
// Inputs:
//   - patterns:        positional Patterns from the CLI
//   - tags:            --tag values
//   - passthroughArgs: anything after `--`
//   - aborted:         true if the run did not complete naturally (fail-fast,
//     ctrl-c, worker error)
func classifyRunKind(patterns, tags, passthroughArgs []string, aborted bool) testruntime.RunKind {
	if aborted {
		return testruntime.RunKindPartial
	}
	if len(tags) > 0 {
		return testruntime.RunKindPartial
	}
	if hasFileLinePattern(patterns) {
		return testruntime.RunKindPartial
	}
	if hasAggregateBreakingArg(passthroughArgs) {
		return testruntime.RunKindPartial
	}
	return testruntime.RunKindAggregate
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

// hasAggregateBreakingArg reports whether passthrough args contain RSpec
// options that demote the run to partial. We intentionally err on the side of
// "partial" rather than enumerate every safe flag.
func hasAggregateBreakingArg(args []string) bool {
	for _, a := range args {
		if a == "--fail-fast" || strings.HasPrefix(a, "--fail-fast=") {
			return true
		}
		if a == "-e" || a == "--example" || strings.HasPrefix(a, "--example=") {
			return true
		}
		if a == "-t" || a == "--tag" || strings.HasPrefix(a, "--tag=") {
			return true
		}
		if strings.HasPrefix(a, "--only-failures") || strings.HasPrefix(a, "--next-failure") {
			return true
		}
	}
	return len(args) > 0
}
