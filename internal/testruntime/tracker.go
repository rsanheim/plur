package testruntime

import (
	"crypto/sha256"
	"encoding/hex"
	"maps"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/rsanheim/plur/internal/buildinfo"
	"github.com/rsanheim/plur/types"
)

// RuntimeTracker accumulates runtime data and persists it to the runtime
// cache. Used single-threaded after all workers complete, so no mutex is
// required. Per-example observations are buffered in pendingExamples until
// SaveToFile decides (per RunKind) whether to merge them as an aggregate-
// eligible full run or a partial observation.
type RuntimeTracker struct {
	cache           *Cache
	fileRuntimes    map[string]float64                  // collected this run, by project-relative file path
	pendingExamples map[string]map[string]*ExampleEntry // collected this run, file -> example.id -> entry
	runtimeFile     string
	cwd             string
	printTimings    bool
}

// NewRuntimeTracker creates a tracker, computing the project-specific cache
// file path and loading any existing runtime data. Missing, v1, or corrupt cache
// files are silently replaced by an empty cache.
func NewRuntimeTracker(runtimeDir string) (*RuntimeTracker, error) {
	return newRuntimeTracker(runtimeDir, false)
}

// NewRuntimeTrackerWithTimings creates a tracker that reports cache load/save
// timings to stderr. Use this for real user-facing runs, not dry-run or
// diagnostic paths.
func NewRuntimeTrackerWithTimings(runtimeDir string) (*RuntimeTracker, error) {
	return newRuntimeTracker(runtimeDir, true)
}

func newRuntimeTracker(runtimeDir string, printTimings bool) (*RuntimeTracker, error) {
	runtimeFile, cwd, err := computeRuntimeFilePath(runtimeDir)
	if err != nil {
		return nil, err
	}

	cache := loadCache(runtimeFile, printTimings)

	return &RuntimeTracker{
		cache:           cache,
		fileRuntimes:    make(map[string]float64),
		pendingExamples: make(map[string]map[string]*ExampleEntry),
		runtimeFile:     runtimeFile,
		cwd:             cwd,
		printTimings:    printTimings,
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

// Cache returns the underlying cache. Read-only for callers that need
// per-example data (the splitter).
func (rt *RuntimeTracker) Cache() *Cache {
	return rt.cache
}

// AddRuntime accumulates per-file runtime collected during the current run.
func (rt *RuntimeTracker) AddRuntime(filePath string, runtime float64) {
	rt.fileRuntimes[filePath] += runtime
}

// AddTestNotification accumulates runtime from a test notification. The
// example is attributed to the rerunnable owning spec file derived from
// LocationRerunArgument (which equals file_path:line for plain examples and
// points back to the owning spec for shared examples). Both file-level
// runtime and per-example metadata key by that owner.
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

// owningFileAndLine derives the owning project-relative spec file and line
// from RSpec's location_rerun_argument, the canonical rerunnable target. For
// non-shared examples this is just file_path:line_number; for shared
// examples it points back to the owning spec file. Falls back to the raw
// notification fields when location_rerun_argument is empty or malformed.
func owningFileAndLine(n types.TestCaseNotification) (string, int) {
	s := strings.TrimPrefix(n.LocationRerunArgument, "./")
	if i := strings.LastIndex(s, ":"); i > 0 {
		if line, err := strconv.Atoi(s[i+1:]); err == nil && line > 0 {
			return s[:i], line
		}
	}
	return n.FilePath, n.LineNumber
}

// SaveToFile persists the runtime data to the cache file. runKind dictates
// whether the file-level aggregates are updated (RunKindAggregate) or
// preserved (RunKindPartial). See MergeAggregateRun and MergeObservations
// for the lifecycle.
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

	return saveCache(rt.cache, rt.runtimeFile, buildinfo.GetVersionInfo(), rt.cwd, time.Now().UTC(), rt.printTimings)
}

// PendingFileRuntimes returns a copy of the file-runtime observations
// accumulated this run. Useful for tests and diagnostics.
func (rt *RuntimeTracker) PendingFileRuntimes() map[string]float64 {
	return maps.Clone(rt.fileRuntimes)
}

func computeRuntimeFilePath(runtimeDir string) (string, string, error) {
	projectHash, cwd, err := getProjectHash()
	if err != nil {
		return "", "", err
	}
	return filepath.Join(runtimeDir, projectHash+".json"), cwd, nil
}

func getProjectHash() (string, string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", "", err
	}
	absPath, err := filepath.Abs(cwd)
	if err != nil {
		return "", "", err
	}
	hash := sha256.Sum256([]byte(absPath))
	return hex.EncodeToString(hash[:])[:8], absPath, nil
}
