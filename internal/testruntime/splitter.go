package testruntime

import (
	"fmt"
	"slices"
	"strings"
)

// SplitDecision describes how a long-running RSpec file was (or was not)
// expanded into focused file:line targets.
//
// When Chunks == 1 the file was passed through unchanged.
// When Chunks  > 1, Targets holds Chunks file:line:line... strings, and
// ChunkRuntimeSeconds is the estimated per-chunk runtime to feed back into
// worker grouping.
type SplitDecision struct {
	Targets             []string
	Chunks              int
	ChunkRuntimeSeconds float64
}

// SplitFile decides whether to split a single RSpec file into focused
// file:line targets. The threshold is intentionally KISS for the experimental
// rollout: split when the file's historical runtime exceeds the per-worker
// budget, the worker count is greater than 1, and we have at least two known
// example lines. Chunk count is bounded by worker count and by the number of
// known example lines. Chunks are distributed round-robin so each chunk gets
// a similar mix of early and late examples, which tends to balance under the
// assumption that examples within a file have comparable runtimes.
//
// Inputs:
//   - filePath:               project-relative spec file path
//   - runtimeSeconds:         historical runtime_seconds for the file
//   - exampleLines:           known example line numbers (e.g. from the v2 cache)
//   - workerCount:            total workers
//   - targetPerWorkerRuntime: per-worker runtime budget the splitter aims for
//
// Returns a SplitDecision. Repeated calls with the same inputs produce the
// same targets in the same order.
func SplitFile(filePath string, runtimeSeconds float64, exampleLines []int, workerCount int, targetPerWorkerRuntime float64) SplitDecision {
	noSplit := SplitDecision{
		Targets:             []string{filePath},
		Chunks:              1,
		ChunkRuntimeSeconds: runtimeSeconds,
	}

	if workerCount <= 1 || targetPerWorkerRuntime <= 0 {
		return noSplit
	}
	if runtimeSeconds <= targetPerWorkerRuntime {
		return noSplit
	}
	if len(exampleLines) < 2 {
		return noSplit
	}

	chunks := min(workerCount, len(exampleLines))
	if chunks < 2 {
		return noSplit
	}

	// Use a defensive sorted copy: callers should pass sorted lines, but we do
	// not want to mutate their slice and we want deterministic output even if
	// they don't.
	lines := slices.Clone(exampleLines)
	slices.Sort(lines)

	buckets := make([][]int, chunks)
	for i, line := range lines {
		bucket := i % chunks
		buckets[bucket] = append(buckets[bucket], line)
	}

	targets := make([]string, 0, chunks)
	for _, bucket := range buckets {
		if len(bucket) == 0 {
			continue
		}
		targets = append(targets, formatTarget(filePath, bucket))
	}

	return SplitDecision{
		Targets:             targets,
		Chunks:              len(targets),
		ChunkRuntimeSeconds: runtimeSeconds / float64(len(targets)),
	}
}

// formatTarget produces an RSpec file:line:line... target like
// "spec/slow_spec.rb:12:38:91".
func formatTarget(filePath string, lines []int) string {
	parts := make([]string, 0, len(lines)+1)
	parts = append(parts, filePath)
	for _, line := range lines {
		parts = append(parts, fmt.Sprintf("%d", line))
	}
	return strings.Join(parts, ":")
}
