package testruntime

import (
	"strconv"
	"strings"
	"time"

	"github.com/rsanheim/plur/internal/buildinfo"
	"github.com/rsanheim/plur/types"
)

// RuntimeTracker collects timings after workers finish; no mutex is needed.
type RuntimeTracker struct {
	cache           *Cache
	fileRuntimes    map[string]float64                  // collected this run, by project-relative file path
	pendingExamples map[string]map[string]*ExampleEntry // collected this run, file -> example.id -> entry
	runtimeFile     string
	cwd             string
}

// NewRuntimeTracker loads this project's runtime cache.
func NewRuntimeTracker(runtimeDir string) (*RuntimeTracker, error) {
	runtimeFile, cwd, err := computeRuntimeFilePath(runtimeDir)
	if err != nil {
		return nil, err
	}

	cache := LoadCache(runtimeFile)

	return &RuntimeTracker{
		cache:           cache,
		fileRuntimes:    make(map[string]float64),
		pendingExamples: make(map[string]map[string]*ExampleEntry),
		runtimeFile:     runtimeFile,
		cwd:             cwd,
	}, nil
}

// RuntimeFilePath returns the path where runtime data is stored.
func (rt *RuntimeTracker) RuntimeFilePath() string {
	return rt.runtimeFile
}

// LoadedData returns cached file totals for worker grouping.
func (rt *RuntimeTracker) LoadedData() map[string]float64 {
	return rt.cache.FileRuntimes()
}

// Cache returns the loaded runtime cache.
func (rt *RuntimeTracker) Cache() *Cache {
	return rt.cache
}

// AddRuntime accumulates per-file runtime collected during the current run.
func (rt *RuntimeTracker) AddRuntime(filePath string, runtime float64) {
	rt.fileRuntimes[filePath] += runtime
}

// AddTestNotification records timings under the rerunnable owning spec.
func (rt *RuntimeTracker) AddTestNotification(notification types.TestCaseNotification) {
	if notification.FilePath == "" {
		return
	}
	owner, ownerLine := owningFileAndLine(notification)
	if notification.Duration > 0 {
		rt.AddRuntime(owner, notification.Duration.Seconds())
	}
	if notification.TestID != "" && ownerLine > 0 {
		if rt.pendingExamples[owner] == nil {
			rt.pendingExamples[owner] = make(map[string]*ExampleEntry)
		}
		rt.pendingExamples[owner][notification.TestID] = &ExampleEntry{
			LineNumber:            ownerLine,
			LocationRerunArgument: notification.LocationRerunArgument,
			RuntimeSeconds:        notification.Duration.Seconds(),
		}
	}
}

// Shared examples belong to their rerunnable spec, not the support file.
func owningFileAndLine(n types.TestCaseNotification) (string, int) {
	s := strings.TrimPrefix(n.LocationRerunArgument, "./")
	if i := strings.LastIndex(s, ":"); i > 0 {
		if line, err := strconv.Atoi(s[i+1:]); err == nil && line > 0 {
			return s[:i], line
		}
	}
	return n.FilePath, n.LineNumber
}

// SaveToFile replaces full-file totals or merges partial observations.
func (rt *RuntimeTracker) SaveToFile(runKind RunKind) error {
	for filePath, runtime := range rt.fileRuntimes {
		mtime, size, ok := SourceFreshness(filePath)
		if !ok {
			continue
		}
		examples := rt.pendingExamples[filePath]
		if runKind.IsAggregateEligible() {
			rt.cache.MergeAggregateRun(filePath, mtime, size, runtime, examples)
			entry := rt.cache.Files[filePath]
			entry.SourceCwd = rt.cwd
			rt.cache.Files[filePath] = entry
		} else if rt.ExamplesFresh(filePath) {
			rt.cache.MergeObservations(filePath, examples)
		}
	}

	// Include observations with no positive runtime.
	for filePath, examples := range rt.pendingExamples {
		if _, alreadyHandled := rt.fileRuntimes[filePath]; alreadyHandled {
			continue
		}
		if rt.ExamplesFresh(filePath) {
			rt.cache.MergeObservations(filePath, examples)
		}
	}

	return SaveCache(rt.cache, rt.runtimeFile, buildinfo.GetVersionInfo(), rt.cwd, time.Now().UTC())
}

// ExamplesFresh checks selector ownership and source freshness.
func (rt *RuntimeTracker) ExamplesFresh(filePath string) bool {
	entry, ok := rt.cache.Files[filePath]
	return ok && entry.SourceCwd == rt.cwd && rt.cache.IsExamplesFresh(filePath)
}
