package main

import (
	"encoding/json"
	"maps"
	"os"
	"path/filepath"
)

// RuntimeCacheSchemaVersion is the on-disk schema version Plur reads and writes.
// Older v1 caches (plain map[string]float64) are ignored and replaced on the
// next aggregate-eligible run.
const RuntimeCacheSchemaVersion = 2

// RuntimeCache is the persisted v2 runtime cache.
// See docs/plans/2026-05-12-rspec-split-specs-experimental-plan.md for the
// shape and lifecycle rules.
type RuntimeCache struct {
	SchemaVersion int                   `json:"schema_version"`
	PlurVersion   string                `json:"plur_version,omitempty"`
	Files         map[string]*FileEntry `json:"files"`
}

// FileEntry is the per-file record in the v2 cache. RuntimeSeconds is the
// aggregate runtime from the most recent aggregate-eligible full run. Examples
// holds per-example metadata (keyed by RSpec example.id) when available.
type FileEntry struct {
	MtimeUnixNano        int64                    `json:"mtime_unix_nano"`
	SizeBytes            int64                    `json:"size_bytes"`
	RuntimeSeconds       float64                  `json:"runtime_seconds"`
	ExampleCount         int                      `json:"example_count,omitempty"`
	ExampleIndexComplete bool                     `json:"example_index_complete"`
	Examples             map[string]*ExampleEntry `json:"examples,omitempty"`
}

// ExampleEntry is the per-example record. Keyed by RSpec's canonical
// example.id in FileEntry.Examples.
type ExampleEntry struct {
	LineNumber            int     `json:"line_number"`
	LocationRerunArgument string  `json:"location_rerun_argument,omitempty"`
	ScopedID              string  `json:"scoped_id,omitempty"`
	RuntimeSeconds        float64 `json:"runtime_seconds"`
	Status                string  `json:"status,omitempty"`
}

// NewRuntimeCache returns an empty v2 cache.
func NewRuntimeCache() *RuntimeCache {
	return &RuntimeCache{
		SchemaVersion: RuntimeCacheSchemaVersion,
		Files:         make(map[string]*FileEntry),
	}
}

// LoadRuntimeCache reads a v2 cache from disk. It returns an empty cache for
// missing files, v1 caches (map[string]float64), corrupt JSON, and entries
// with a non-v2 schema_version.
func LoadRuntimeCache(path string) *RuntimeCache {
	data, err := os.ReadFile(path)
	if err != nil {
		return NewRuntimeCache()
	}

	var cache RuntimeCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return NewRuntimeCache()
	}
	if cache.SchemaVersion != RuntimeCacheSchemaVersion {
		return NewRuntimeCache()
	}
	if cache.Files == nil {
		cache.Files = make(map[string]*FileEntry)
	}
	return &cache
}

// SaveRuntimeCache writes the cache atomically: temp file in the same
// directory + rename into place. The caller supplies the current plur version
// to record.
func SaveRuntimeCache(cache *RuntimeCache, path, plurVersion string) error {
	if cache.SchemaVersion == 0 {
		cache.SchemaVersion = RuntimeCacheSchemaVersion
	}
	cache.PlurVersion = plurVersion
	if cache.Files == nil {
		cache.Files = make(map[string]*FileEntry)
	}

	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".runtime-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	success := false
	defer func() {
		if !success {
			os.Remove(tmpPath)
		}
	}()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return err
	}
	success = true
	return nil
}

// FileRuntimes returns a map of project-relative file paths to total file
// runtime in seconds. Used by file-level worker grouping. Entries with no
// recorded aggregate runtime (RuntimeSeconds == 0) are skipped so the grouper
// can apply its default-runtime fallback.
func (c *RuntimeCache) FileRuntimes() map[string]float64 {
	out := make(map[string]float64, len(c.Files))
	for path, entry := range c.Files {
		if entry == nil || entry.RuntimeSeconds == 0 {
			continue
		}
		out[path] = entry.RuntimeSeconds
	}
	return out
}

// File returns the entry for a file path (project-relative), or nil if absent.
func (c *RuntimeCache) File(path string) *FileEntry {
	return c.Files[path]
}

// SourceFreshness reads mtime and size of a source file. ok is false if the
// file cannot be stat'd.
func SourceFreshness(path string) (mtimeUnixNano, sizeBytes int64, ok bool) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, 0, false
	}
	return info.ModTime().UnixNano(), info.Size(), true
}

// exampleLines returns sorted, deduplicated line numbers for the examples
// recorded against a file. Empty if the file is missing from the cache or
// has no recorded examples.
func (c *RuntimeCache) exampleLines(filePath string) []int {
	entry := c.Files[filePath]
	if entry == nil || len(entry.Examples) == 0 {
		return nil
	}
	out := make([]int, 0, len(entry.Examples))
	seen := make(map[int]struct{}, len(entry.Examples))
	for _, ex := range entry.Examples {
		if ex == nil || ex.LineNumber <= 0 {
			continue
		}
		if _, dup := seen[ex.LineNumber]; dup {
			continue
		}
		seen[ex.LineNumber] = struct{}{}
		out = append(out, ex.LineNumber)
	}
	return out
}

// IsExamplesFresh reports whether the cached examples for a file can be
// trusted by the splitter: example_index_complete is true AND the recorded
// mtime/size still match the current source file.
func (c *RuntimeCache) IsExamplesFresh(filePath string) bool {
	entry := c.Files[filePath]
	if entry == nil || !entry.ExampleIndexComplete {
		return false
	}
	mtime, size, ok := SourceFreshness(filePath)
	if !ok {
		return false
	}
	return entry.MtimeUnixNano == mtime && entry.SizeBytes == size
}

// RunKind describes the selection a Plur invocation made. The cache uses this
// to decide whether to update file-level aggregates and the
// example_index_complete flag.
type RunKind int

const (
	// RunKindAggregate is a default/full-file run. It may rewrite file-level
	// aggregates and set example_index_complete.
	RunKindAggregate RunKind = iota
	// RunKindPartial is any non-aggregate run: focused (file:line), tag
	// filtered, fail-fast, aborted, or custom-arg. It may merge per-example
	// observations but must not touch file-level aggregates or the flag.
	RunKindPartial
)

// IsAggregateEligible reports whether a run of the given kind may update
// file-level aggregates and mark example_index_complete.
func (k RunKind) IsAggregateEligible() bool {
	return k == RunKindAggregate
}

// MergeAggregateRun records the result of an aggregate-eligible full-file run.
// It replaces the file's examples map and runtime aggregate, sets
// example_index_complete, and records current source freshness. The caller
// must have already verified this is an aggregate-eligible run.
func (c *RuntimeCache) MergeAggregateRun(filePath string, mtimeUnixNano, sizeBytes int64, runtimeSeconds float64, examples map[string]*ExampleEntry) {
	if examples == nil {
		examples = make(map[string]*ExampleEntry)
	}
	c.Files[filePath] = &FileEntry{
		MtimeUnixNano:        mtimeUnixNano,
		SizeBytes:            sizeBytes,
		RuntimeSeconds:       runtimeSeconds,
		ExampleCount:         len(examples),
		ExampleIndexComplete: true,
		Examples:             examples,
	}
}

// MergeObservations records per-example observations from a partial run for a
// single file. It never modifies RuntimeSeconds, never flips
// ExampleIndexComplete from false to true, and never prunes examples missing
// from this run. If the file has no existing entry, nothing is written:
// partial runs must not create file-level aggregates.
func (c *RuntimeCache) MergeObservations(filePath string, examples map[string]*ExampleEntry) {
	existing := c.Files[filePath]
	if existing == nil {
		return
	}
	if existing.Examples == nil {
		existing.Examples = make(map[string]*ExampleEntry)
	}
	maps.Copy(existing.Examples, examples)
	existing.ExampleCount = len(existing.Examples)
}
