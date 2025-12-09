package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/rsanheim/plur/types"
)

// RuntimeTracker accumulates runtime data for spec files and manages persistence.
// It loads existing data at construction and merges with new data on save.
// Note: This is used single-threaded after all workers complete, so no mutex needed.
type RuntimeTracker struct {
	runtimes    map[string]float64 // collected this run
	loadedData  map[string]float64 // loaded from file at construction
	runtimeFile string             // computed once at creation
}

// NewRuntimeTracker creates a new runtime tracker with the given runtime directory.
// It computes the project-specific file path and loads any existing runtime data.
func NewRuntimeTracker(runtimeDir string) (*RuntimeTracker, error) {
	runtimeFile, err := computeRuntimeFilePath(runtimeDir)
	if err != nil {
		return nil, err
	}

	loadedData := loadExistingData(runtimeFile)

	return &RuntimeTracker{
		runtimes:    make(map[string]float64),
		loadedData:  loadedData,
		runtimeFile: runtimeFile,
	}, nil
}

// RuntimeFilePath returns the path where runtime data is stored
func (rt *RuntimeTracker) RuntimeFilePath() string {
	return rt.runtimeFile
}

// LoadedData returns the runtime data that was loaded from file at construction.
// This is used for grouping files by runtime before tests run.
func (rt *RuntimeTracker) LoadedData() map[string]float64 {
	return rt.loadedData
}

// AddRuntime adds runtime for a spec file
func (rt *RuntimeTracker) AddRuntime(filePath string, runtime float64) {
	rt.runtimes[filePath] += runtime
}

// AddTestNotification adds runtime from a test notification
func (rt *RuntimeTracker) AddTestNotification(notification types.TestCaseNotification) {
	if notification.FilePath != "" && notification.Duration > 0 {
		rt.AddRuntime(notification.FilePath, notification.Duration.Seconds())
	}
}

// SaveToFile writes the runtime data to the project-specific JSON file.
// It merges existing data with new measurements (new data takes precedence).
func (rt *RuntimeTracker) SaveToFile() error {
	// Merge: start with loaded data, overwrite with new measurements
	merged := make(map[string]float64)
	for k, v := range rt.loadedData {
		merged[k] = v
	}
	for k, v := range rt.runtimes {
		merged[k] = v
	}

	file, err := os.Create(rt.runtimeFile)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(merged)
}

// computeRuntimeFilePath computes the project-specific runtime file path
func computeRuntimeFilePath(runtimeDir string) (string, error) {
	projectHash, err := getProjectHash()
	if err != nil {
		return "", err
	}
	return filepath.Join(runtimeDir, projectHash+".json"), nil
}

// getProjectHash generates a hash of the current working directory
func getProjectHash() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Get absolute path to ensure consistency
	absPath, err := filepath.Abs(cwd)
	if err != nil {
		return "", err
	}

	// Generate SHA-256 hash of the absolute path
	hash := sha256.Sum256([]byte(absPath))
	// Use first 8 characters of hex for readability
	return hex.EncodeToString(hash[:])[:8], nil
}

// loadExistingData loads runtime data from file, returning empty map if not found
func loadExistingData(runtimeFile string) map[string]float64 {
	if _, err := os.Stat(runtimeFile); os.IsNotExist(err) {
		return make(map[string]float64)
	}

	file, err := os.Open(runtimeFile)
	if err != nil {
		return make(map[string]float64)
	}
	defer file.Close()

	var runtimes map[string]float64
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&runtimes); err != nil {
		return make(map[string]float64)
	}

	return runtimes
}
