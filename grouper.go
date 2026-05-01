package main

import (
	"os"
	"sort"

	"github.com/rsanheim/plur/logger"
)

// FileGroup represents a group of spec files that will run in one process
type FileGroup struct {
	Files     []string
	TotalSize int64
	WorkerID  int
}

// GroupSpecFilesBySize distributes spec files into groups to minimize total processes
// while balancing the workload across workers
func GroupSpecFilesBySize(specFiles []string, numWorkers int) []FileGroup {
	// Get file sizes
	type fileWithSize struct {
		path string
		size int64
	}

	filesWithSizes := make([]fileWithSize, 0, len(specFiles))
	for _, file := range specFiles {
		info, err := os.Stat(file)
		if err != nil {
			// Default size for missing files
			filesWithSizes = append(filesWithSizes, fileWithSize{file, 1000})
			continue
		}
		filesWithSizes = append(filesWithSizes, fileWithSize{file, info.Size()})
	}

	// Sort by size descending (largest first)
	sort.Slice(filesWithSizes, func(i, j int) bool {
		return filesWithSizes[i].size > filesWithSizes[j].size
	})

	// Initialize groups
	groups := make([]FileGroup, numWorkers)
	for i := range groups {
		groups[i] = FileGroup{
			Files:     make([]string, 0),
			TotalSize: 0,
			WorkerID:  i,
		}
	}

	// Distribute files using "smallest group first" algorithm
	// This ensures balanced distribution
	for _, file := range filesWithSizes {
		// Find the group with smallest total size
		minIdx := 0
		minSize := groups[0].TotalSize
		for i := 1; i < len(groups); i++ {
			if groups[i].TotalSize < minSize {
				minIdx = i
				minSize = groups[i].TotalSize
			}
		}

		// Add file to smallest group
		groups[minIdx].Files = append(groups[minIdx].Files, file.path)
		groups[minIdx].TotalSize += file.size
	}

	// Remove empty groups (when we have more workers than files)
	nonEmptyGroups := make([]FileGroup, 0)
	for _, group := range groups {
		if len(group.Files) > 0 {
			nonEmptyGroups = append(nonEmptyGroups, group)
		}
	}

	return nonEmptyGroups
}

// GroupOpts controls optional behavior for the runtime grouper.
type GroupOpts struct {
	// RspecCmd, when non-empty, is the base command (e.g. ["bundle", "exec",
	// "rspec"]) used for an optional `--dry-run --format json` pass over
	// long-pole files. The exact line numbers it returns are used in place of
	// the regex-based extractor, eliminating fuzzy-match over-counting.
	RspecCmd []string
}

// GroupSpecFilesByRuntime distributes spec files based on their historical runtime.
func GroupSpecFilesByRuntime(specFiles []string, numWorkers int, runtimeData map[string]float64) []FileGroup {
	return GroupSpecFilesByRuntimeWithOpts(specFiles, numWorkers, runtimeData, GroupOpts{})
}

// GroupSpecFilesByRuntimeWithOpts is the variant that accepts opts. Callers
// that have an rspec command available should use this entry point so the
// dry-run-based exact line lookup engages.
func GroupSpecFilesByRuntimeWithOpts(specFiles []string, numWorkers int, runtimeData map[string]float64, opts GroupOpts) []FileGroup {
	// Create a struct to hold file and its runtime
	type fileWithRuntime struct {
		path    string
		runtime float64
	}

	// Get runtime for each file, tracking hits/misses
	filesWithRuntimes := make([]fileWithRuntime, 0, len(specFiles))
	hits, misses := 0, 0

	for _, file := range specFiles {
		runtime, ok := runtimeData[file]
		if !ok {
			// Default runtime for files without history (1 second)
			runtime = 1.0
			misses++
		} else {
			hits++
		}
		filesWithRuntimes = append(filesWithRuntimes, fileWithRuntime{file, runtime})
	}

	logger.Logger.Debug("runtime data lookup",
		"hits", hits,
		"misses", misses,
		"hit_rate", float64(hits)/float64(len(specFiles))*100)

	// Long-pole splitting: any file whose runtime exceeds 1.5x the per-worker
	// average is the wall-time bottleneck even with optimal LPT bin packing. Split
	// such files into rspec file:line invocations distributed round-robin so
	// multiple workers can run them concurrently.
	totalRuntime := 0.0
	for _, f := range filesWithRuntimes {
		totalRuntime += f.runtime
	}
	if numWorkers > 1 && totalRuntime > 0 {
		perWorkerTarget := totalRuntime / float64(numWorkers)
		splitThreshold := perWorkerTarget * 1.5

		// First pass: collect long-pole file paths so we can do a single
		// batched dry-run for all of them (one gem-load amortized).
		var longPoleFiles []string
		for _, f := range filesWithRuntimes {
			if f.runtime > splitThreshold {
				longPoleFiles = append(longPoleFiles, f.path)
			}
		}

		// resolveExampleLines is a no-op when RspecCmd is empty (tests).
		var exactLines map[string][]int
		if len(longPoleFiles) > 0 && len(opts.RspecCmd) > 0 {
			exactLines = resolveExampleLines(opts.RspecCmd, longPoleFiles)
		}

		var expanded []fileWithRuntime
		splits := 0
		for _, f := range filesWithRuntimes {
			if f.runtime > splitThreshold {
				numChunks := int(f.runtime/perWorkerTarget + 0.5)
				if numChunks < 2 {
					numChunks = 2
				}
				if numChunks > numWorkers {
					numChunks = numWorkers
				}
				chunks, ok := splitFileByExamples(f.path, numChunks, exactLines[f.path])
				if ok {
					chunkRuntime := f.runtime / float64(len(chunks))
					for _, c := range chunks {
						expanded = append(expanded, fileWithRuntime{c, chunkRuntime})
					}
					splits++
					continue
				}
			}
			expanded = append(expanded, f)
		}
		if splits > 0 {
			logger.Logger.Debug("long-pole splitting", "files_split", splits, "per_worker_target_s", perWorkerTarget, "exact_lines_used", len(exactLines) > 0)
			filesWithRuntimes = expanded
		}
	}

	// Sort by runtime descending (slowest first)
	sort.Slice(filesWithRuntimes, func(i, j int) bool {
		return filesWithRuntimes[i].runtime > filesWithRuntimes[j].runtime
	})

	// Initialize groups
	groups := make([]FileGroup, numWorkers)
	groupRuntimes := make([]float64, numWorkers)
	for i := range groups {
		groups[i] = FileGroup{
			Files:     make([]string, 0),
			TotalSize: 0,
			WorkerID:  i,
		}
	}

	// Distribute files using "smallest runtime first" algorithm
	// This ensures balanced distribution based on actual test runtime
	for _, file := range filesWithRuntimes {
		// Find the group with smallest total runtime
		minIdx := 0
		minRuntime := groupRuntimes[0]
		for i := 1; i < len(groups); i++ {
			if groupRuntimes[i] < minRuntime {
				minIdx = i
				minRuntime = groupRuntimes[i]
			}
		}

		// Add file to group with smallest runtime
		groups[minIdx].Files = append(groups[minIdx].Files, file.path)
		groupRuntimes[minIdx] += file.runtime
		// Store runtime as int64 (milliseconds) for compatibility
		groups[minIdx].TotalSize += int64(file.runtime * 1000)
	}

	// Remove empty groups (when we have more workers than files)
	nonEmptyGroups := make([]FileGroup, 0)
	for _, group := range groups {
		if len(group.Files) > 0 {
			nonEmptyGroups = append(nonEmptyGroups, group)
		}
	}

	return nonEmptyGroups
}
