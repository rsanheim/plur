package main

import (
	"crypto/sha256"
	"encoding/hex"
	"maps"
	"os"
	"path/filepath"

	"github.com/rsanheim/plur/internal/buildinfo"
	"github.com/rsanheim/plur/types"
)

// RuntimeTracker accumulates runtime data and persists it to the v2 runtime
// cache. It is used single-threaded after all workers complete, so no mutex
// is required.
//
// Today's responsibilities cover file-level aggregates. Per-example metadata
// passes through in Task 3+: examples observed during a run are buffered in
// pendingExamples until SaveToFile decides (per RunKind) whether to merge them
// as an aggregate-eligible full run or a partial observation.
type RuntimeTracker struct {
	cache           *RuntimeCache
	fileRuntimes    map[string]float64                  // collected this run, by project-relative file path
	pendingExamples map[string]map[string]*ExampleEntry // collected this run, file -> example.id -> entry
	runtimeFile     string
}

// NewRuntimeTracker creates a tracker, computing the project-specific cache
// file path and loading any existing v2 data. Missing, v1, or corrupt cache
// files are silently replaced by an empty cache.
func NewRuntimeTracker(runtimeDir string) (*RuntimeTracker, error) {
	runtimeFile, err := computeRuntimeFilePath(runtimeDir)
	if err != nil {
		return nil, err
	}

	cache := LoadRuntimeCache(runtimeFile)

	return &RuntimeTracker{
		cache:           cache,
		fileRuntimes:    make(map[string]float64),
		pendingExamples: make(map[string]map[string]*ExampleEntry),
		runtimeFile:     runtimeFile,
	}, nil
}

// RuntimeFilePath returns the path where runtime data is stored.
func (rt *RuntimeTracker) RuntimeFilePath() string {
	return rt.runtimeFile
}

// LoadedData returns file-level runtime data loaded from the cache. Used by
// the grouper to balance workers before tests run.
func (rt *RuntimeTracker) LoadedData() map[string]float64 {
	return rt.cache.FileRuntimes()
}

// Cache returns the underlying v2 cache. Read-only for callers that need
// per-example data (the splitter).
func (rt *RuntimeTracker) Cache() *RuntimeCache {
	return rt.cache
}

// AddRuntime accumulates per-file runtime collected during the current run.
func (rt *RuntimeTracker) AddRuntime(filePath string, runtime float64) {
	rt.fileRuntimes[filePath] += runtime
}

// AddTestNotification accumulates runtime from a test notification. Examples
// with file path and example identity are also buffered for example-level
// merging at save time.
func (rt *RuntimeTracker) AddTestNotification(notification types.TestCaseNotification) {
	if notification.FilePath == "" {
		return
	}
	if notification.Duration > 0 {
		rt.AddRuntime(notification.FilePath, notification.Duration.Seconds())
	}
	if notification.TestID != "" && notification.LineNumber > 0 {
		if rt.pendingExamples[notification.FilePath] == nil {
			rt.pendingExamples[notification.FilePath] = make(map[string]*ExampleEntry)
		}
		rt.pendingExamples[notification.FilePath][notification.TestID] = &ExampleEntry{
			LineNumber:            notification.LineNumber,
			LocationRerunArgument: notification.LocationRerunArgument,
			ScopedID:              notification.ScopedID,
			RuntimeSeconds:        notification.Duration.Seconds(),
			Status:                notification.Status,
		}
	}
}

// SaveToFile persists the runtime data to the v2 cache file. runKind dictates
// whether the file-level aggregates are updated (RunKindAggregate) or
// preserved (RunKindPartial). See runtime_cache.go for the full lifecycle.
func (rt *RuntimeTracker) SaveToFile(runKind RunKind) error {
	for filePath, runtime := range rt.fileRuntimes {
		mtime, size, ok := SourceFreshness(filePath)
		if !ok {
			continue
		}
		examples := rt.pendingExamples[filePath]
		if runKind.IsAggregateEligible() {
			rt.cache.MergeAggregateRun(filePath, mtime, size, runtime, examples)
		} else {
			rt.cache.MergeObservations(filePath, examples)
		}
	}

	// Files observed only via partial example data (no aggregated runtime)
	// still merge their example observations.
	for filePath, examples := range rt.pendingExamples {
		if _, alreadyHandled := rt.fileRuntimes[filePath]; alreadyHandled {
			continue
		}
		rt.cache.MergeObservations(filePath, examples)
	}

	return SaveRuntimeCache(rt.cache, rt.runtimeFile, buildinfo.GetVersionInfo())
}

// PendingFileRuntimes returns a copy of the file-runtime observations
// accumulated this run. Useful for tests and diagnostics.
func (rt *RuntimeTracker) PendingFileRuntimes() map[string]float64 {
	return maps.Clone(rt.fileRuntimes)
}

func computeRuntimeFilePath(runtimeDir string) (string, error) {
	projectHash, err := getProjectHash()
	if err != nil {
		return "", err
	}
	return filepath.Join(runtimeDir, projectHash+".json"), nil
}

func getProjectHash() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	absPath, err := filepath.Abs(cwd)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256([]byte(absPath))
	return hex.EncodeToString(hash[:])[:8], nil
}
