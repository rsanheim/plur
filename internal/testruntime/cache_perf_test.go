package testruntime

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/rsanheim/plur/internal/buildinfo"
)

// largeSuiteThreshold is the combined LoadCache+SaveCache budget for a
// large suite (>= 10K cached examples), per the decision criteria in
// docs/plans/2026-05-22-runtime-cache-measurement-results.md. The cache is
// loaded on every plur run (including dry runs), so regressions here are
// paid by every invocation against a big project.
const largeSuiteThreshold = 50 * time.Millisecond

// TestCacheLoadSavePerformanceThreshold fails if the runtime-cache hot path
// exceeds the documented large-suite budget on a RuboCop-shaped cache
// (745 files / 31,672 examples, ~6MB JSON).
//
// goccy/go-json was adopted specifically to meet this budget after compact
// JSON and schema-shape cleanups still left stdlib encoding/json ~4.8x too
// slow on load. Swapping the JSON implementation (e.g. back to stdlib, or
// forward to encoding/json/v2) must keep this test green; see the
// "go-json Parser Adoption" section of the measurement doc and
// BenchmarkCache_{Load,Save}LargeRspecCache for the full methodology.
func TestCacheLoadSavePerformanceThreshold(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping timing-sensitive cache threshold test in short mode")
	}
	if buildinfo.RaceEnabled {
		t.Skip("skipping cache threshold test under race detector (2-20x slowdown)")
	}

	cache := buildLargeRspecCache()
	path := filepath.Join(t.TempDir(), "runtime.json")
	savedAt := time.Date(2026, 5, 22, 20, 48, 43, 0, time.UTC)
	if err := SaveCache(cache, path, "perf-test", "/Users/example/rubocop", savedAt); err != nil {
		t.Fatal(err)
	}

	// Best-of-N wall time: the minimum is the least noise-sensitive
	// estimate of intrinsic speed on a shared/loaded machine.
	const runs = 8

	minLoad := time.Duration(1<<63 - 1)
	for i := 0; i < runs; i++ {
		start := time.Now()
		loaded := LoadCache(path)
		elapsed := time.Since(start)
		if len(loaded.Files) != benchFileCount {
			t.Fatalf("loaded %d files, want %d", len(loaded.Files), benchFileCount)
		}
		minLoad = min(minLoad, elapsed)
	}

	minSave := time.Duration(1<<63 - 1)
	for i := 0; i < runs; i++ {
		start := time.Now()
		if err := SaveCache(cache, path, "perf-test", "/Users/example/rubocop", savedAt); err != nil {
			t.Fatal(err)
		}
		minSave = min(minSave, time.Since(start))
	}

	combined := minLoad + minSave
	t.Logf("runtime cache load=%v save=%v combined=%v (budget %v)", minLoad, minSave, combined, largeSuiteThreshold)

	if combined > largeSuiteThreshold {
		t.Errorf("runtime cache load+save = %v, exceeds the %v large-suite budget\n"+
			"load=%v save=%v on a %d-file / %d-example cache\n"+
			"This path runs on every plur invocation. goccy/go-json was adopted to meet this budget\n"+
			"(stdlib encoding/json load is ~4.8x slower; see docs/plans/2026-05-22-runtime-cache-measurement-results.md).\n"+
			"If you changed the JSON implementation or cache schema, re-run\n"+
			"BenchmarkCache_{Load,Save}LargeRspecCache and update the measurement doc before relaxing this budget.",
			combined, largeSuiteThreshold, minLoad, minSave, benchFileCount, benchExampleCount)
	}
}
