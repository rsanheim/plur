package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRuntimeCache_LoadEmptyOrMissing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "missing.json")

	cache := LoadRuntimeCache(path)
	require.NotNil(t, cache)
	assert.Equal(t, RuntimeCacheSchemaVersion, cache.Meta.SchemaVersion)
	assert.Empty(t, cache.Files)
}

func TestRuntimeCache_IgnoresV1Cache(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cache.json")

	v1 := map[string]float64{
		"spec/foo_spec.rb": 1.234,
		"spec/bar_spec.rb": 2.5,
	}
	data, err := json.Marshal(v1)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, data, 0644))

	cache := LoadRuntimeCache(path)
	assert.Equal(t, RuntimeCacheSchemaVersion, cache.Meta.SchemaVersion)
	assert.Empty(t, cache.Files, "v1 caches must be ignored, not migrated")
}

func TestRuntimeCache_IgnoresCorruptCache(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cache.json")
	require.NoError(t, os.WriteFile(path, []byte("not json {{{"), 0644))

	cache := LoadRuntimeCache(path)
	assert.Equal(t, RuntimeCacheSchemaVersion, cache.Meta.SchemaVersion)
	assert.Empty(t, cache.Files)
}

func TestRuntimeCache_IgnoresUnknownSchemaVersion(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cache.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"meta": {"schema_version": 99}, "files": {"spec/x.rb": {"runtime_seconds": 1.0}}}`), 0644))

	cache := LoadRuntimeCache(path)
	assert.Equal(t, RuntimeCacheSchemaVersion, cache.Meta.SchemaVersion)
	assert.Empty(t, cache.Files)
}

func TestRuntimeCache_SaveAndReload(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cache.json")

	cache := NewRuntimeCache()
	cache.MergeAggregateRun("spec/foo_spec.rb", 123, 456, 2.5, map[string]*ExampleEntry{
		"./spec/foo_spec.rb[1:1]": {
			LineNumber:            12,
			LocationRerunArgument: "./spec/foo_spec.rb:12",
			ScopedID:              "1:1",
			RuntimeSeconds:        1.0,
			Status:                "passed",
		},
	})

	savedAt := time.Date(2026, 5, 22, 15, 4, 5, 0, time.UTC)
	require.NoError(t, SaveRuntimeCache(cache, path, "plur-test", "/Users/example/project", savedAt))

	reloaded := LoadRuntimeCache(path)
	assert.Equal(t, RuntimeCacheSchemaVersion, reloaded.Meta.SchemaVersion)
	assert.Equal(t, "plur-test", reloaded.Meta.PlurVersion)
	assert.Equal(t, "/Users/example/project", reloaded.Run.Cwd)
	assert.Equal(t, "2026-05-22T15:04:05Z", reloaded.Run.LastRunAt)

	entry := reloaded.File("spec/foo_spec.rb")
	require.NotNil(t, entry)
	assert.Equal(t, int64(123), entry.MtimeUnixNano)
	assert.Equal(t, int64(456), entry.SizeBytes)
	assert.Equal(t, 2.5, entry.RuntimeSeconds)
	assert.True(t, entry.ExampleIndexComplete)
	assert.Len(t, entry.Examples, 1)

	ex := entry.Examples["./spec/foo_spec.rb[1:1]"]
	require.NotNil(t, ex)
	assert.Equal(t, 12, ex.LineNumber)
	assert.Equal(t, "1:1", ex.ScopedID)
	assert.Equal(t, 1.0, ex.RuntimeSeconds)
}

func TestRuntimeCache_SaveWritesHumanReadableHeaderOrder(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cache.json")

	cache := NewRuntimeCache()
	cache.MergeAggregateRun("spec/foo_spec.rb", 1, 2, 1.0, nil)

	require.NoError(t, SaveRuntimeCache(cache, path, "plur-test", "/tmp/project", time.Date(2026, 5, 22, 1, 2, 3, 0, time.UTC)))

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(data), "{\n  \"meta\": {\n    \"schema_version\": 2,\n    \"plur_version\": \"plur-test\"\n  },\n  \"run\": {\n    \"cwd\": \"/tmp/project\",\n    \"last_run_at\": \"2026-05-22T01:02:03Z\"\n  },\n  \"files\": {")
}

func TestRuntimeCache_SaveIsAtomic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cache.json")

	cache := NewRuntimeCache()
	cache.MergeAggregateRun("spec/foo_spec.rb", 1, 2, 1.0, nil)
	require.NoError(t, SaveRuntimeCache(cache, path, "v1", "/tmp/project", time.Now()))

	// No leftover temp files in the directory.
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	for _, e := range entries {
		assert.NotContains(t, e.Name(), ".tmp", "atomic write must clean up temp files")
	}
}

func TestRuntimeCache_FileRuntimesSkipsZero(t *testing.T) {
	cache := NewRuntimeCache()
	cache.MergeAggregateRun("spec/has_runtime.rb", 0, 0, 3.14, nil)
	cache.Files["spec/no_runtime.rb"] = &FileEntry{ExampleIndexComplete: false}

	rt := cache.FileRuntimes()
	assert.Equal(t, 3.14, rt["spec/has_runtime.rb"])
	_, present := rt["spec/no_runtime.rb"]
	assert.False(t, present, "files with no recorded runtime should be omitted")
}

func TestRuntimeCache_MergeAggregateRunReplacesExamples(t *testing.T) {
	cache := NewRuntimeCache()
	cache.MergeAggregateRun("spec/x_spec.rb", 1, 2, 1.0, map[string]*ExampleEntry{
		"./spec/x_spec.rb[1:1]": {LineNumber: 5, RuntimeSeconds: 0.5},
		"./spec/x_spec.rb[1:2]": {LineNumber: 10, RuntimeSeconds: 0.5},
	})

	// Second aggregate run drops the [1:2] example (e.g., it was removed).
	cache.MergeAggregateRun("spec/x_spec.rb", 3, 4, 2.0, map[string]*ExampleEntry{
		"./spec/x_spec.rb[1:1]": {LineNumber: 5, RuntimeSeconds: 0.6},
	})

	entry := cache.File("spec/x_spec.rb")
	require.NotNil(t, entry)
	assert.Equal(t, int64(3), entry.MtimeUnixNano)
	assert.Equal(t, 2.0, entry.RuntimeSeconds)
	assert.True(t, entry.ExampleIndexComplete)
	assert.Len(t, entry.Examples, 1, "aggregate run prunes examples missing from the run")
}

func TestRuntimeCache_MergeObservationsPreservesAggregate(t *testing.T) {
	cache := NewRuntimeCache()
	cache.MergeAggregateRun("spec/x_spec.rb", 1, 2, 5.0, map[string]*ExampleEntry{
		"./spec/x_spec.rb[1:1]": {LineNumber: 5, RuntimeSeconds: 1.0},
		"./spec/x_spec.rb[1:2]": {LineNumber: 10, RuntimeSeconds: 4.0},
	})

	// Partial run only observes [1:1].
	cache.MergeObservations("spec/x_spec.rb", map[string]*ExampleEntry{
		"./spec/x_spec.rb[1:1]": {LineNumber: 5, RuntimeSeconds: 1.5},
	})

	entry := cache.File("spec/x_spec.rb")
	require.NotNil(t, entry)
	assert.Equal(t, 5.0, entry.RuntimeSeconds, "partial run must not touch file-level runtime")
	assert.True(t, entry.ExampleIndexComplete, "partial run must not flip the flag")
	assert.Len(t, entry.Examples, 2, "partial run must not prune missing examples")
	assert.Equal(t, 1.5, entry.Examples["./spec/x_spec.rb[1:1]"].RuntimeSeconds)
	assert.Equal(t, 4.0, entry.Examples["./spec/x_spec.rb[1:2]"].RuntimeSeconds)
}

func TestRuntimeCache_MergeObservationsSkipsUnseenFiles(t *testing.T) {
	cache := NewRuntimeCache()
	cache.MergeObservations("spec/never_seen.rb", map[string]*ExampleEntry{
		"./spec/never_seen.rb[1:1]": {LineNumber: 1, RuntimeSeconds: 0.1},
	})

	assert.Nil(t, cache.File("spec/never_seen.rb"), "partial runs must not create file-level entries")
}

func TestRuntimeCache_IsExamplesFresh(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "foo_spec.rb")
	require.NoError(t, os.WriteFile(source, []byte("# spec\n"), 0644))

	info, err := os.Stat(source)
	require.NoError(t, err)

	cache := NewRuntimeCache()
	cache.MergeAggregateRun(source, info.ModTime().UnixNano(), info.Size(), 1.0, map[string]*ExampleEntry{
		"id": {LineNumber: 1, RuntimeSeconds: 0.1},
	})

	assert.True(t, cache.IsExamplesFresh(source), "fresh source should report fresh")

	// Change the file: bump mtime + size.
	time.Sleep(10 * time.Millisecond)
	require.NoError(t, os.WriteFile(source, []byte("# spec changed\n"), 0644))
	assert.False(t, cache.IsExamplesFresh(source), "modified source should report stale")
}

func TestRuntimeCache_IsExamplesFreshFalseWhenIncomplete(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "foo_spec.rb")
	require.NoError(t, os.WriteFile(source, []byte("# spec\n"), 0644))

	cache := NewRuntimeCache()
	mtime, size, ok := SourceFreshness(source)
	require.True(t, ok)
	cache.Files[source] = &FileEntry{
		MtimeUnixNano:        mtime,
		SizeBytes:            size,
		RuntimeSeconds:       1.0,
		ExampleIndexComplete: false,
		Examples:             map[string]*ExampleEntry{"id": {LineNumber: 1}},
	}

	assert.False(t, cache.IsExamplesFresh(source), "incomplete index must report not-fresh")
}

func TestRuntimeCache_RunKindEligibility(t *testing.T) {
	assert.True(t, RunKindAggregate.IsAggregateEligible())
	assert.False(t, RunKindPartial.IsAggregateEligible())
}
