package testruntime

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
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
