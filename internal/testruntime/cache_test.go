package testruntime

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCache_LoadEmptyOrMissing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "missing.json")

	cache := LoadCache(path)
	require.NotNil(t, cache)
	assert.Equal(t, SchemaVersion, cache.Meta.SchemaVersion)
	assert.Empty(t, cache.Files)
}

func TestCache_IgnoresV1Cache(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cache.json")

	v1 := map[string]float64{
		"spec/foo_spec.rb": 1.234,
		"spec/bar_spec.rb": 2.5,
	}
	data, err := json.Marshal(v1)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, data, 0644))

	cache := LoadCache(path)
	assert.Equal(t, SchemaVersion, cache.Meta.SchemaVersion)
	assert.Empty(t, cache.Files, "v1 caches must be ignored, not migrated")
}

func TestCache_IgnoresCorruptCache(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cache.json")
	require.NoError(t, os.WriteFile(path, []byte("not json {{{"), 0644))

	cache := LoadCache(path)
	assert.Equal(t, SchemaVersion, cache.Meta.SchemaVersion)
	assert.Empty(t, cache.Files)
}

func TestCache_IgnoresNullCache(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cache.json")
	require.NoError(t, os.WriteFile(path, []byte("null"), 0644))

	cache := LoadCache(path)
	require.NotNil(t, cache)
	assert.Equal(t, SchemaVersion, cache.Meta.SchemaVersion)
	assert.Empty(t, cache.Files)
}

func TestCache_IgnoresUnknownSchemaVersion(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cache.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"meta": {"schema_version": 99}, "files": {"spec/x.rb": {"runtime_seconds": 1.0}}}`), 0644))

	cache := LoadCache(path)
	assert.Equal(t, SchemaVersion, cache.Meta.SchemaVersion)
	assert.Empty(t, cache.Files)
}

func TestCache_SaveAndReload(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cache.json")

	cache := NewCache()
	cache.MergeAggregateRun("spec/foo_spec.rb", 123, 456, 2.5, map[string]*ExampleEntry{
		"./spec/foo_spec.rb[1:1]": {
			LineNumber:            12,
			LocationRerunArgument: "./spec/foo_spec.rb:12",
			RuntimeSeconds:        1.0,
		},
	})

	savedAt := time.Date(2026, 5, 22, 15, 4, 5, 0, time.UTC)
	require.NoError(t, SaveCache(cache, path, "plur-test", "/Users/example/project", savedAt))

	reloaded := LoadCache(path)
	assert.Equal(t, SchemaVersion, reloaded.Meta.SchemaVersion)
	assert.Equal(t, "plur-test", reloaded.Meta.PlurVersion)
	assert.Equal(t, "/Users/example/project", reloaded.Run.Cwd)
	assert.Equal(t, "2026-05-22T15:04:05Z", reloaded.Run.LastRunAt)

	entry := reloaded.File("spec/foo_spec.rb")
	require.NotNil(t, entry)
	assert.Equal(t, int64(123), entry.MtimeUnixNano)
	assert.Equal(t, int64(456), entry.SizeBytes)
	assert.Equal(t, 2.5, entry.RuntimeSeconds)
	assert.Len(t, entry.Examples, 1)

	ex := requireExample(t, entry, "./spec/foo_spec.rb[1:1]")
	assert.Equal(t, 12, ex.LineNumber)
	assert.Equal(t, "./spec/foo_spec.rb:12", ex.LocationRerunArgument)
	assert.Equal(t, 1.0, ex.RuntimeSeconds)
}

// TestCache_IgnoresUnknownExampleFields ensures that cache JSON with obsolete
// per-example keys still loads. These extra fields should be silently dropped
// on the next save.
func TestCache_IgnoresUnknownExampleFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cache.json")

	oldShape := `{
	  "meta": {"schema_version": 4, "plur_version": "v0"},
	  "run": {"cwd": "/tmp/project", "last_run_at": "2026-05-22T01:02:03Z"},
	  "files": {
	    "spec/foo_spec.rb": {
	      "mtime_unix_nano": 1,
	      "size_bytes": 2,
	      "runtime_seconds": 1.0,
	      "examples": [
	        {
	          "id": "./spec/foo_spec.rb[1:1]",
	          "line_number": 12,
	          "location_rerun_argument": "./spec/foo_spec.rb:12",
	          "scoped_id": "1:1",
	          "runtime_seconds": 0.5,
	          "status": "passed"
	        }
	      ]
	    }
	  }
	}`
	require.NoError(t, os.WriteFile(path, []byte(oldShape), 0644))

	cache := LoadCache(path)
	entry := cache.File("spec/foo_spec.rb")
	require.NotNil(t, entry)
	ex := requireExample(t, entry, "./spec/foo_spec.rb[1:1]")
	assert.Equal(t, 12, ex.LineNumber)
	assert.Equal(t, 0.5, ex.RuntimeSeconds)
}

func TestCache_SaveWritesCompactJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cache.json")

	cache := NewCache()
	cache.MergeAggregateRun("spec/foo_spec.rb", 1, 2, 1.0, nil)

	require.NoError(t, SaveCache(cache, path, "plur-test", "/tmp/project", time.Date(2026, 5, 22, 1, 2, 3, 0, time.UTC)))

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.NotContains(t, string(data), "\n  \"", "runtime cache should avoid pretty-print indentation")
	assert.Contains(t, string(data), `"meta":{"schema_version":4,"plur_version":"plur-test"}`)
	assert.Contains(t, string(data), `"run":{"cwd":"/tmp/project","last_run_at":"2026-05-22T01:02:03Z"}`)
}

func TestCache_SaveSerializesExamplesAsArrayWithID(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cache.json")

	cache := NewCache()
	cache.MergeAggregateRun("spec/foo_spec.rb", 1, 2, 1.0, map[string]*ExampleEntry{
		"./spec/foo_spec.rb[1:1]": {
			LineNumber:            12,
			LocationRerunArgument: "./spec/foo_spec.rb:12",
			RuntimeSeconds:        1.0,
		},
	})

	require.NoError(t, SaveCache(cache, path, "plur-test", "/tmp/project", time.Date(2026, 5, 22, 1, 2, 3, 0, time.UTC)))

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.NotContains(t, string(data), `"example_count"`)
	assert.NotContains(t, string(data), `"example_index_complete"`)

	var raw struct {
		Files map[string]struct {
			Examples any `json:"examples"`
		} `json:"files"`
	}
	require.NoError(t, json.Unmarshal(data, &raw))
	examples, ok := raw.Files["spec/foo_spec.rb"].Examples.([]any)
	require.True(t, ok, "examples should serialize as an array, not an object keyed by example ID")
	require.Len(t, examples, 1)
	example, ok := examples[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "./spec/foo_spec.rb[1:1]", example["id"])
	assert.Equal(t, float64(12), example["line_number"])
}

func TestCache_SaveDoesNotEscapeHTML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cache.json")

	cache := NewCache()
	cache.MergeAggregateRun("spec/html_escape_spec.rb", 1, 2, 1.0, map[string]*ExampleEntry{
		`./spec/html_escape_spec.rb[1:1]`: {
			LineNumber:            1,
			LocationRerunArgument: `./spec/html_escape_spec.rb:1?value=<tag>&other=>`,
			RuntimeSeconds:        1.0,
		},
	})

	require.NoError(t, SaveCache(cache, path, "plur-test", "/tmp/project", time.Date(2026, 5, 22, 1, 2, 3, 0, time.UTC)))

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(data), `<tag>&other=>`)
	assert.NotContains(t, string(data), `\u003c`)
	assert.NotContains(t, string(data), `\u0026`)
	assert.NotContains(t, string(data), `\u003e`)
}

func TestCache_SaveIsAtomic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cache.json")

	cache := NewCache()
	cache.MergeAggregateRun("spec/foo_spec.rb", 1, 2, 1.0, nil)
	require.NoError(t, SaveCache(cache, path, "v1", "/tmp/project", time.Now()))

	// No leftover temp files in the directory.
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	for _, e := range entries {
		assert.NotContains(t, e.Name(), ".tmp", "atomic write must clean up temp files")
	}
}

func TestCache_FileRuntimesSkipsZero(t *testing.T) {
	cache := NewCache()
	cache.MergeAggregateRun("spec/has_runtime.rb", 0, 0, 3.14, nil)
	cache.Files["spec/no_runtime.rb"] = FileEntry{}

	rt := cache.FileRuntimes()
	assert.Equal(t, 3.14, rt["spec/has_runtime.rb"])
	_, present := rt["spec/no_runtime.rb"]
	assert.False(t, present, "files with no recorded runtime should be omitted")
}

func TestCache_MergeAggregateRunReplacesExamples(t *testing.T) {
	cache := NewCache()
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
	assert.Len(t, entry.Examples, 1, "aggregate run prunes examples missing from the run")
	assert.Equal(t, "./spec/x_spec.rb[1:1]", entry.Examples[0].ID)
}

func TestCache_MergeObservationsPreservesAggregate(t *testing.T) {
	cache := NewCache()
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
	assert.Len(t, entry.Examples, 2, "partial run must not prune missing examples")
	assert.Equal(t, 1.5, requireExample(t, entry, "./spec/x_spec.rb[1:1]").RuntimeSeconds)
	assert.Equal(t, 4.0, requireExample(t, entry, "./spec/x_spec.rb[1:2]").RuntimeSeconds)
}

func TestCache_MergeObservationsSkipsUnseenFiles(t *testing.T) {
	cache := NewCache()
	cache.MergeObservations("spec/never_seen.rb", map[string]*ExampleEntry{
		"./spec/never_seen.rb[1:1]": {LineNumber: 1, RuntimeSeconds: 0.1},
	})

	assert.Nil(t, cache.File("spec/never_seen.rb"), "partial runs must not create file-level entries")
}

func TestCache_IsExamplesFresh(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "foo_spec.rb")
	require.NoError(t, os.WriteFile(source, []byte("# spec\n"), 0644))

	info, err := os.Stat(source)
	require.NoError(t, err)

	cache := NewCache()
	cache.MergeAggregateRun(source, info.ModTime().UnixNano(), info.Size(), 1.0, map[string]*ExampleEntry{
		"id": {LineNumber: 1, RuntimeSeconds: 0.1},
	})

	assert.True(t, cache.IsExamplesFresh(source), "fresh source should report fresh")

	// Change the file: bump mtime + size.
	time.Sleep(10 * time.Millisecond)
	require.NoError(t, os.WriteFile(source, []byte("# spec changed\n"), 0644))
	assert.False(t, cache.IsExamplesFresh(source), "modified source should report stale")
}

func TestCache_IsExamplesFreshUsesSourceFreshnessOnly(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(dir, "foo_spec.rb")
	require.NoError(t, os.WriteFile(source, []byte("# spec\n"), 0644))

	cache := NewCache()
	mtime, size, ok := SourceFreshness(source)
	require.True(t, ok)
	cache.Files[source] = FileEntry{
		MtimeUnixNano:  mtime,
		SizeBytes:      size,
		RuntimeSeconds: 1.0,
	}

	assert.True(t, cache.IsExamplesFresh(source), "freshness is derived from source metadata")
}

func requireExample(t *testing.T, entry *FileEntry, id string) ExampleEntry {
	t.Helper()
	idx := entry.ExampleIndex()
	i, ok := idx[id]
	require.True(t, ok, "expected example %s", id)
	return entry.Examples[i]
}
