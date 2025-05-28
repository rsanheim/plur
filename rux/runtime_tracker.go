package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
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

// AddExample adds runtime from an RSpec example
func (rt *RuntimeTracker) AddExample(example RSpecExample) {
	if example.FilePath != "" && example.RunTime > 0 {
		rt.AddRuntime(example.FilePath, example.RunTime)
	}
}

// SaveToFile writes the runtime data to a JSON file
func (rt *RuntimeTracker) SaveToFile(dir string) error {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	// Ensure directory exists
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Write runtime data to runtime.json
	runtimeFile := filepath.Join(dir, "runtime.json")
	file, err := os.Create(runtimeFile)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(rt.runtimes)
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

// LoadRuntimeData loads runtime data from the cache directory
func LoadRuntimeData() (map[string]float64, error) {
	cacheDir, err := getRuxCacheDir()
	if err != nil {
		return nil, err
	}

	runtimeFile := filepath.Join(cacheDir, "runtime.json")

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
