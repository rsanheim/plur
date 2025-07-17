package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"github.com/rsanheim/plur/types"
)

// RuntimeTracker accumulates runtime data for spec files
type RuntimeTracker struct {
	mu       sync.Mutex
	runtimes map[string]float64 // map[filepath]total_runtime_seconds
}

// NewRuntimeTracker creates a new runtime tracker
func NewRuntimeTracker() *RuntimeTracker {
	return &RuntimeTracker{
		runtimes: make(map[string]float64),
	}
}

// AddRuntime adds runtime for a spec file
func (rt *RuntimeTracker) AddRuntime(filePath string, runtime float64) {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	// Accumulate runtimes for the same file
	rt.runtimes[filePath] += runtime
}

// AddTestNotification adds runtime from a test notification
func (rt *RuntimeTracker) AddTestNotification(notification types.TestCaseNotification) {
	if notification.FilePath != "" && notification.Duration > 0 {
		rt.AddRuntime(notification.FilePath, notification.Duration.Seconds())
	}
}

// SaveToFile writes the runtime data to a project-specific JSON file
func (rt *RuntimeTracker) SaveToFile() error {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	// Get the project-specific runtime file path
	runtimeFile, err := getRuntimeFilePath()
	if err != nil {
		return err
	}

	// Create the file
	file, err := os.Create(runtimeFile)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(rt.runtimes)
}

// GetRuntimeFilePath returns the runtime file path (exported for messages)
func GetRuntimeFilePath() (string, error) {
	return getRuntimeFilePath()
}

// GetRuntimes returns a copy of the runtime data
func (rt *RuntimeTracker) GetRuntimes() map[string]float64 {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	// Return a copy to avoid concurrent access issues
	result := make(map[string]float64)
	for k, v := range rt.runtimes {
		result[k] = v
	}
	return result
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

// getRuntimeFilePath returns the project-specific runtime file path
func getRuntimeFilePath() (string, error) {
	configPaths := InitConfigPaths()
	runtimesDir := configPaths.RuntimeDir

	// Get project hash for filename
	projectHash, err := getProjectHash()
	if err != nil {
		return "", err
	}

	return filepath.Join(runtimesDir, projectHash+".json"), nil
}

// LoadRuntimeData loads runtime data from the cache directory
func LoadRuntimeData() (map[string]float64, error) {
	runtimeFile, err := getRuntimeFilePath()
	if err != nil {
		return nil, err
	}

	// Check if file exists
	if _, err := os.Stat(runtimeFile); os.IsNotExist(err) {
		// No runtime data yet, return empty map
		return make(map[string]float64), nil
	}

	file, err := os.Open(runtimeFile)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var runtimes map[string]float64
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&runtimes); err != nil {
		return nil, err
	}

	return runtimes, nil
}
