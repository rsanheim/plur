package main

import (
	"os"
	"sort"
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


// GroupSpecFilesByRuntime distributes spec files based on their historical runtime
func GroupSpecFilesByRuntime(specFiles []string, numWorkers int, runtimeData map[string]float64) []FileGroup {
	// Create a struct to hold file and its runtime
	type fileWithRuntime struct {
		path    string
		runtime float64
	}

	// Get runtime for each file
	filesWithRuntimes := make([]fileWithRuntime, 0, len(specFiles))
	totalRuntime := 0.0

	for _, file := range specFiles {
		runtime, ok := runtimeData[file]
		if !ok {
			// Default runtime for files without history (1 second)
			runtime = 1.0
		}
		filesWithRuntimes = append(filesWithRuntimes, fileWithRuntime{file, runtime})
		totalRuntime += runtime
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
