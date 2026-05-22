package testruntime

import (
	"fmt"
	"slices"
	"strings"
)

// SplitDecision maps focused-target spec args ("spec/slow_spec.rb:12:38") to
// their bin-packed per-target runtime in seconds. When the input file is not
// a split candidate, SplitDecision has exactly one entry: the original file
// path mapped to its recorded file-level runtime.
type SplitDecision map[string]float64

// SplitFile decides how to split a long-running RSpec file across workers by
// bin-packing its cached examples using longest-processing-time greedy. Any
// example with RuntimeSeconds <= 0 falls back to the file's mean per-example
// runtime so unmeasured examples still get a sensible weight.
//
// Returns the no-split decision (one entry mapping filePath to file runtime)
// when workerCount <= 1, the budget is non-positive, the file is unknown to
// the cache, the file's runtime is at or under budget, or fewer than two
// recorded examples have usable line numbers.
//
// Repeated calls with the same cache state and inputs produce identical
// results. Map iteration order is randomized, but consumers (the grouper)
// sort by runtime, so order does not affect downstream grouping.
func (c *Cache) SplitFile(filePath string, workerCount int, targetPerWorkerRuntime float64) SplitDecision {
	file := c.Files[filePath]
	if file == nil {
		return SplitDecision{filePath: 0}
	}
	noSplit := SplitDecision{filePath: file.RuntimeSeconds}

	if workerCount <= 1 || targetPerWorkerRuntime <= 0 {
		return noSplit
	}
	if file.RuntimeSeconds <= targetPerWorkerRuntime {
		return noSplit
	}
	if len(file.Examples) < 2 {
		return noSplit
	}

	units := buildUnits(file)
	if len(units) < 2 {
		return noSplit
	}

	chunks := min(workerCount, len(units))
	if chunks < 2 {
		return noSplit
	}

	bins := make([]splitBin, chunks)
	for _, u := range units {
		best := 0
		for i := 1; i < chunks; i++ {
			if bins[i].sum < bins[best].sum {
				best = i
			}
		}
		bins[best].lines = append(bins[best].lines, u.line)
		bins[best].sum += u.runtime
	}

	decision := make(SplitDecision, chunks)
	for _, b := range bins {
		if len(b.lines) == 0 {
			continue
		}
		slices.Sort(b.lines)
		decision[formatTarget(filePath, b.lines)] = b.sum
	}
	return decision
}

type splitUnit struct {
	line    int
	runtime float64
}

type splitBin struct {
	lines []int
	sum   float64
}

// buildUnits projects a file's recorded examples into deterministic
// (line, runtime) pairs ordered by descending runtime with ascending line as
// the deterministic tiebreak. Examples with RuntimeSeconds <= 0 are given
// the file's mean per-example runtime so they still earn a place in a bin.
func buildUnits(file *FileEntry) []splitUnit {
	mean := file.RuntimeSeconds / float64(len(file.Examples))

	ids := make([]string, 0, len(file.Examples))
	for id := range file.Examples {
		ids = append(ids, id)
	}
	slices.Sort(ids) // map iteration is randomized; sort for deterministic input.

	units := make([]splitUnit, 0, len(ids))
	for _, id := range ids {
		ex := file.Examples[id]
		if ex == nil || ex.LineNumber <= 0 {
			continue
		}
		rt := ex.RuntimeSeconds
		if rt <= 0 {
			rt = mean
		}
		units = append(units, splitUnit{line: ex.LineNumber, runtime: rt})
	}

	slices.SortFunc(units, func(a, b splitUnit) int {
		if a.runtime != b.runtime {
			if a.runtime > b.runtime {
				return -1
			}
			return 1
		}
		return a.line - b.line
	})
	return units
}

// formatTarget produces an RSpec file:line:line... target like
// "spec/slow_spec.rb:12:38:91". Lines must be passed pre-sorted ascending.
func formatTarget(filePath string, lines []int) string {
	parts := make([]string, 0, len(lines)+1)
	parts = append(parts, filePath)
	for _, line := range lines {
		parts = append(parts, fmt.Sprintf("%d", line))
	}
	return strings.Join(parts, ":")
}
