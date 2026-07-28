package testruntime

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/rsanheim/plur/internal/buildinfo"
	"github.com/rsanheim/plur/internal/testutil"
)

const (
	benchFileCount    = 745
	benchExampleCount = 31672
)

func BenchmarkCache_SaveLargeRspecCache(b *testing.B) {
	cache := buildLargeRspecCache()
	dir := b.TempDir()
	path := filepath.Join(dir, "runtime.json")
	savedAt := time.Date(2026, 5, 22, 20, 48, 43, 0, time.UTC)

	requireBenchmarkNoError(b, SaveCache(cache, path, "bench", "/Users/example/rubocop", savedAt))
	stat, err := os.Stat(path)
	requireBenchmarkNoError(b, err)
	b.SetBytes(stat.Size())
	b.ReportAllocs()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		requireBenchmarkNoError(b, SaveCache(cache, path, "bench", "/Users/example/rubocop", savedAt))
	}
}

func BenchmarkCache_LoadLargeRspecCache(b *testing.B) {
	cache := buildLargeRspecCache()
	dir := b.TempDir()
	path := filepath.Join(dir, "runtime.json")
	savedAt := time.Date(2026, 5, 22, 20, 48, 43, 0, time.UTC)

	requireBenchmarkNoError(b, SaveCache(cache, path, "bench", "/Users/example/rubocop", savedAt))
	stat, err := os.Stat(path)
	requireBenchmarkNoError(b, err)
	b.SetBytes(stat.Size())
	b.ReportAllocs()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		loaded := LoadCache(path)
		if len(loaded.Files) != benchFileCount {
			b.Fatalf("loaded %d files, want %d", len(loaded.Files), benchFileCount)
		}
	}
}

// TestCacheLoadBudget enforces the runtime-cache load budget. The cache is read
// on every plur run against a large suite, so a regression here slows everyone.
func TestCacheLoadBudget(t *testing.T) {
	if buildinfo.RaceEnabled {
		t.Skip("skipping cache load budget under race detector")
	}

	budgetMS := 30.0
	if testutil.IsCI() {
		budgetMS = 45.0
	}
	if v, ok := os.LookupEnv("PLUR_CACHE_LOAD_BUDGET_MS"); ok {
		parsed, err := strconv.ParseFloat(v, 64)
		if err != nil {
			t.Fatalf("invalid PLUR_CACHE_LOAD_BUDGET_MS=%q: %v", v, err)
		}
		budgetMS = parsed
	}

	// best of 3 runs
	bestMS := 0.0
	for i := 0; i < 3; i++ {
		res := testing.Benchmark(BenchmarkCache_LoadLargeRspecCache)
		ms := float64(res.NsPerOp()) / 1e6
		if i == 0 || ms < bestMS {
			bestMS = ms
		}
	}

	t.Logf("runtime cache load = %.1fms (budget %.0fms)", bestMS, budgetMS)
	if bestMS > budgetMS {
		t.Fatalf("runtime cache load %.1fms exceeds %.0fms budget", bestMS, budgetMS)
	}
}

func buildLargeRspecCache() *Cache {
	cache := NewCache()
	examplesLeft := benchExampleCount
	for fileIndex := 0; fileIndex < benchFileCount; fileIndex++ {
		filesLeft := benchFileCount - fileIndex
		examplesForFile := examplesLeft / filesLeft
		if examplesLeft%filesLeft != 0 {
			examplesForFile++
		}
		examplesLeft -= examplesForFile

		filePath := fmt.Sprintf("spec/rubocop/cop/generated_%03d_spec.rb", fileIndex)
		examples := make(map[string]*ExampleEntry, examplesForFile)
		var runtime float64
		for exampleIndex := 0; exampleIndex < examplesForFile; exampleIndex++ {
			line := 10 + exampleIndex*3
			exampleID := fmt.Sprintf("./%s[1:%d:%d]", filePath, fileIndex+1, exampleIndex+1)
			exampleRuntime := 0.001 + float64((fileIndex+exampleIndex)%97)/10000
			runtime += exampleRuntime
			examples[exampleID] = &ExampleEntry{
				LineNumber:            line,
				LocationRerunArgument: fmt.Sprintf("./%s:%d", filePath, line),
				RuntimeSeconds:        exampleRuntime,
			}
		}
		cache.MergeAggregateRun(filePath, int64(1_700_000_000_000_000_000+fileIndex), int64(1_000+fileIndex), runtime, examples)
	}
	return cache
}

func requireBenchmarkNoError(b *testing.B, err error) {
	b.Helper()
	if err != nil {
		b.Fatal(err)
	}
}
