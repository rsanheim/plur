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

// ShouldUseGrouping determines if we should group files or use one-file-per-process
func ShouldUseGrouping(numFiles, numWorkers int) bool {
	// If we have more workers than files, no need to group
	if numWorkers >= numFiles {
		return false
	}

	// For small test suites, grouping helps reduce overhead
	// For large test suites, one-file-per-process might be better for granular parallelism
	// This is a tunable heuristic
	avgFilesPerWorker := float64(numFiles) / float64(numWorkers)

	// Group if we'd average more than 2 files per worker
	// This balances overhead reduction vs parallelism
	return avgFilesPerWorker > 2.0
}
