package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rsanheim/plur/types"
)

const RuntimeSchemaVersion = 2

// RuntimeData is the top-level schema for the runtime JSON file.
type RuntimeData struct {
	SchemaVersion int                    `json:"schema_version"`
	ProjectRoot   string                 `json:"project_root"`
	ProjectHash   string                 `json:"project_hash"`
	GeneratedAt   string                 `json:"generated_at"`
	PlurVersion   string                 `json:"plur_version"`
	Framework     string                 `json:"framework"`
	Files         map[string]FileRuntime `json:"files"`
}

// FileRuntime stores per-file runtime metadata.
type FileRuntime struct {
	TotalSeconds float64 `json:"total_seconds"`
	ExampleCount int     `json:"example_count"`
	LastSeen     string  `json:"last_seen"`
}

// RuntimeTracker accumulates runtime data for spec files and manages persistence.
// It loads existing data at construction and merges with new data on save.
// Note: This is used single-threaded after all workers complete, so no mutex needed.
type RuntimeTracker struct {
	runtimes    map[string]FileRuntime // collected this run
	loadedData  map[string]FileRuntime // loaded from file at construction
	runtimeFile string                 // computed once at creation
	projectRoot string                 // absolute path of project
	projectHash string                 // 8-char SHA-256 hash of projectRoot
	framework   string                 // set before save (e.g. "rspec", "minitest")
}

// NewRuntimeTracker creates a new runtime tracker with the given runtime directory.
// It computes the project-specific file path and loads any existing runtime data.
func NewRuntimeTracker(runtimeDir string) (*RuntimeTracker, error) {
	runtimeFile, projectRoot, projectHash, err := computeRuntimeFilePath(runtimeDir)
	if err != nil {
		return nil, err
	}

	loadedData := loadExistingData(runtimeFile)

	return &RuntimeTracker{
		runtimes:    make(map[string]FileRuntime),
		loadedData:  loadedData,
		runtimeFile: runtimeFile,
		projectRoot: projectRoot,
		projectHash: projectHash,
	}, nil
}

// RuntimeFilePath returns the path where runtime data is stored
func (rt *RuntimeTracker) RuntimeFilePath() string {
	return rt.runtimeFile
}

// SetFramework sets the framework name (e.g. "rspec", "minitest") for metadata.
func (rt *RuntimeTracker) SetFramework(fw string) {
	rt.framework = fw
}

// LoadedData returns the runtime data that was loaded from file at construction.
// This is used for grouping files by runtime before tests run.
// Returns map[string]float64 to preserve the grouper's interface.
func (rt *RuntimeTracker) LoadedData() map[string]float64 {
	result := make(map[string]float64, len(rt.loadedData))
	for k, v := range rt.loadedData {
		result[k] = v.TotalSeconds
	}
	return result
}

// AddRuntime adds runtime for a spec file with its example count.
func (rt *RuntimeTracker) AddRuntime(filePath string, seconds float64, exampleCount int) {
	existing := rt.runtimes[filePath]
	existing.TotalSeconds += seconds
	existing.ExampleCount += exampleCount
	existing.LastSeen = time.Now().UTC().Format(time.RFC3339)
	rt.runtimes[filePath] = existing
}

// AddTestNotification adds runtime from a test notification
func (rt *RuntimeTracker) AddTestNotification(notification types.TestCaseNotification) {
	if notification.FilePath != "" && notification.Duration > 0 {
		rt.AddRuntime(notification.FilePath, notification.Duration.Seconds(), 1)
	}
}

// SaveToFile writes the runtime data to the project-specific JSON file.
// It merges existing data with new measurements (new data takes precedence).
func (rt *RuntimeTracker) SaveToFile() error {
	merged := make(map[string]FileRuntime)
	for k, v := range rt.loadedData {
		merged[k] = v
	}
	for k, v := range rt.runtimes {
		merged[k] = v
	}

	data := RuntimeData{
		SchemaVersion: RuntimeSchemaVersion,
		ProjectRoot:   rt.projectRoot,
		ProjectHash:   rt.projectHash,
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
		PlurVersion:   GetVersionInfo(),
		Framework:     rt.framework,
		Files:         merged,
	}

	file, err := os.Create(rt.runtimeFile)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

// computeRuntimeFilePath computes the project-specific runtime file path
// using a human-readable sanitized version of the absolute project path.
func computeRuntimeFilePath(runtimeDir string) (filePath, projectRoot, projectHash string, err error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", "", "", err
	}

	absPath, err := filepath.Abs(cwd)
	if err != nil {
		return "", "", "", err
	}

	hash := sha256.Sum256([]byte(absPath))
	projectHash = hex.EncodeToString(hash[:])[:8]
	sanitized := sanitizeProjectPath(absPath)
	filePath = filepath.Join(runtimeDir, sanitized+".json")
	return filePath, absPath, projectHash, nil
}

// sanitizeProjectPath converts an absolute path to a human-readable filename component.
// "/Users/rob/src/myapp" → "Users_rob_src_myapp"
func sanitizeProjectPath(absPath string) string {
	sanitized := strings.ReplaceAll(absPath, string(filepath.Separator), "_")
	sanitized = strings.TrimLeft(sanitized, "_")
	return sanitized
}

// loadExistingData loads runtime data from file, returning empty map if not found
// or if the file uses an unsupported schema version (hard cutover, no legacy loading).
func loadExistingData(runtimeFile string) map[string]FileRuntime {
	rawBytes, err := os.ReadFile(runtimeFile)
	if err != nil {
		return make(map[string]FileRuntime)
	}

	var data RuntimeData
	if err := json.Unmarshal(rawBytes, &data); err != nil {
		return make(map[string]FileRuntime)
	}

	if data.SchemaVersion != RuntimeSchemaVersion {
		return make(map[string]FileRuntime)
	}

	if data.Files == nil {
		return make(map[string]FileRuntime)
	}
	return data.Files
}
