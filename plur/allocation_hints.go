package main

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// AllocationHints provides size hints for pre-allocation based on historical data
type AllocationHints struct {
	EstimatedTests    int
	EstimatedFailures int
	EstimatedOutput   int
}

// ProjectStats stores historical statistics for a project
type ProjectStats struct {
	AverageTests        float64 `json:"avg_tests"`
	AverageFailures     float64 `json:"avg_failures"`
	AverageOutputSize   float64 `json:"avg_output_size"`
	AverageTestsPerFile float64 `json:"avg_tests_per_file"`
	RunCount            int     `json:"run_count"`
}

// GetAllocationHints returns allocation hints based on test suite characteristics and history
func GetAllocationHints(projectPath string, numFiles int, runtimeDir string) AllocationHints {
	// Start with defaults
	hints := AllocationHints{
		EstimatedTests:    numFiles * 10, // Default: 10 tests per file
		EstimatedFailures: numFiles,      // Default: 1 failure per file max
		EstimatedOutput:   4096,          // Default: 4KB
	}

	// Try to load historical stats
	stats := loadProjectStats(projectPath, runtimeDir)
	if stats != nil && stats.RunCount > 0 {
		// Use historical data for better estimates
		hints.EstimatedTests = int(stats.AverageTests * 1.2)       // Add 20% buffer
		hints.EstimatedFailures = int(stats.AverageFailures * 2)   // Double for safety
		hints.EstimatedOutput = int(stats.AverageOutputSize * 1.5) // 50% buffer

		// If we have per-file stats, use that
		if stats.AverageTestsPerFile > 0 {
			estimatedFromFiles := int(float64(numFiles) * stats.AverageTestsPerFile * 1.2)
			if estimatedFromFiles > hints.EstimatedTests {
				hints.EstimatedTests = estimatedFromFiles
			}
		}
	}

	// Apply reasonable bounds
	hints = applyBounds(hints)

	return hints
}

// loadProjectStats loads historical statistics for a project
func loadProjectStats(projectPath, runtimeDir string) *ProjectStats {
	// Generate a project identifier (could be hash of path or project name)
	projectID := filepath.Base(projectPath)
	statsFile := filepath.Join(runtimeDir, projectID+".stats.json")

	data, err := os.ReadFile(statsFile)
	if err != nil {
		return nil // No history available
	}

	var stats ProjectStats
	if err := json.Unmarshal(data, &stats); err != nil {
		return nil
	}

	return &stats
}

// UpdateProjectStats updates the historical statistics after a test run
func UpdateProjectStats(projectPath, runtimeDir string, result WorkerResult) {
	projectID := filepath.Base(projectPath)
	statsFile := filepath.Join(runtimeDir, projectID+".stats.json")

	// Load existing stats or create new
	stats := loadProjectStats(projectPath, runtimeDir)
	if stats == nil {
		stats = &ProjectStats{}
	}

	// Update with exponential moving average (more weight to recent runs)
	alpha := 0.3 // Weight for new data
	if stats.RunCount == 0 {
		// First run, use all new data
		stats.AverageTests = float64(result.ExampleCount)
		stats.AverageFailures = float64(result.FailureCount)
		stats.AverageOutputSize = float64(len(result.Output))
	} else {
		// Blend with historical data
		stats.AverageTests = alpha*float64(result.ExampleCount) + (1-alpha)*stats.AverageTests
		stats.AverageFailures = alpha*float64(result.FailureCount) + (1-alpha)*stats.AverageFailures
		stats.AverageOutputSize = alpha*float64(len(result.Output)) + (1-alpha)*stats.AverageOutputSize
	}

	stats.RunCount++

	// Save updated stats
	data, _ := json.MarshalIndent(stats, "", "  ")
	os.WriteFile(statsFile, data, 0644)
}

// applyBounds ensures allocation hints are within reasonable limits
func applyBounds(hints AllocationHints) AllocationHints {
	// Minimum values
	if hints.EstimatedTests < 10 {
		hints.EstimatedTests = 10
	}
	if hints.EstimatedFailures < 5 {
		hints.EstimatedFailures = 5
	}
	if hints.EstimatedOutput < 1024 {
		hints.EstimatedOutput = 1024
	}

	// Maximum values (to prevent over-allocation)
	if hints.EstimatedTests > 10000 {
		hints.EstimatedTests = 10000
	}
	if hints.EstimatedFailures > 1000 {
		hints.EstimatedFailures = 1000
	}
	if hints.EstimatedOutput > 10*1024*1024 { // 10MB max
		hints.EstimatedOutput = 10 * 1024 * 1024
	}

	return hints
}
