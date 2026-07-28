package testruntime

import (
	"os"
	"path/filepath"
	"slices"
	"time"

	json "github.com/goccy/go-json"
)

// SchemaVersion is the on-disk schema version Plur reads and writes.
// Older caches with an unsupported schema_version are ignored and replaced on
// the next aggregate-eligible run.
const SchemaVersion = 4

// Cache is the persisted runtime cache. The schema shape and lifecycle rules
// are covered by the cache tests in this package.
type Cache struct {
	Meta  CacheMeta            `json:"meta"`
	Run   CacheRun             `json:"run"`
	Files map[string]FileEntry `json:"files"`
}

// CacheMeta stores cache metadata intended to be readable near the top
// of the JSON file.
type CacheMeta struct {
	SchemaVersion int    `json:"schema_version"`
	PlurVersion   string `json:"plur_version,omitempty"`
}

// CacheRun stores metadata about the most recent cache write.
type CacheRun struct {
	Cwd       string `json:"cwd,omitempty"`
	LastRunAt string `json:"last_run_at,omitempty"`
}

// FileEntry is the per-file record in the runtime cache. RuntimeSeconds is the
// aggregate runtime from the most recent aggregate-eligible full run. Examples
// holds per-example metadata when available.
type FileEntry struct {
	MtimeUnixNano  int64          `json:"mtime_unix_nano"`
	SizeBytes      int64          `json:"size_bytes"`
	RuntimeSeconds float64        `json:"runtime_seconds"`
	Examples       []ExampleEntry `json:"examples,omitempty"`
}

// ExampleEntry is the per-example record.
type ExampleEntry struct {
	ID                    string  `json:"id"`
	LineNumber            int     `json:"line_number"`
	LocationRerunArgument string  `json:"location_rerun_argument,omitempty"`
	RuntimeSeconds        float64 `json:"runtime_seconds"`
}

// NewCache returns an empty runtime cache.
func NewCache() *Cache {
	return &Cache{
		Meta:  CacheMeta{SchemaVersion: SchemaVersion},
		Files: make(map[string]FileEntry),
	}
}

// LoadCache reads a runtime cache from disk. It returns an empty cache for
// missing files, v1 caches (map[string]float64), corrupt JSON, and entries
// with an unsupported schema_version.
func LoadCache(path string) (cache *Cache) {
	defer func() {
		if cache == nil {
			cache = NewCache()
		}
	}()

	data, err := os.ReadFile(path)
	if err != nil {
		return NewCache()
	}

	if err := json.Unmarshal(data, &cache); err != nil {
		return NewCache()
	}
	if cache == nil {
		return NewCache()
	}
	if cache.Meta.SchemaVersion != SchemaVersion {
		return NewCache()
	}
	if cache.Files == nil {
		cache.Files = make(map[string]FileEntry)
	}
	return cache
}

// SaveCache writes the cache atomically: temp file in the same
// directory + rename into place. The caller supplies the current plur version
// to record.
func SaveCache(cache *Cache, path, plurVersion, cwd string, lastRunAt time.Time) error {
	if cache.Meta.SchemaVersion == 0 {
		cache.Meta.SchemaVersion = SchemaVersion
	}
	cache.Meta.PlurVersion = plurVersion
	cache.Run.Cwd = cwd
	cache.Run.LastRunAt = lastRunAt.UTC().Format(time.RFC3339)
	if cache.Files == nil {
		cache.Files = make(map[string]FileEntry)
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

	enc := json.NewEncoder(tmp)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(cache); err != nil {
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
func (c *Cache) FileRuntimes() map[string]float64 {
	out := make(map[string]float64, len(c.Files))
	for path, entry := range c.Files {
		if entry.RuntimeSeconds == 0 {
			continue
		}
		out[path] = entry.RuntimeSeconds
	}
	return out
}

// File returns the entry for a file path (project-relative), or nil if absent.
func (c *Cache) File(path string) *FileEntry {
	entry, ok := c.Files[path]
	if !ok {
		return nil
	}
	return &entry
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

// ExampleLines returns sorted, deduplicated line numbers for the examples
// recorded against a file. Empty if the file is missing from the cache or
// has no recorded examples.
func (c *Cache) ExampleLines(filePath string) []int {
	entry := c.Files[filePath]
	if len(entry.Examples) == 0 {
		return nil
	}
	out := make([]int, 0, len(entry.Examples))
	seen := make(map[int]struct{}, len(entry.Examples))
	for _, ex := range entry.Examples {
		if ex.LineNumber <= 0 {
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

// IsExamplesFresh reports whether cached examples for a file match the
// current source file's mtime/size.
func (c *Cache) IsExamplesFresh(filePath string) bool {
	entry, ok := c.Files[filePath]
	if !ok {
		return false
	}
	mtime, size, ok := SourceFreshness(filePath)
	if !ok {
		return false
	}
	return entry.MtimeUnixNano == mtime && entry.SizeBytes == size
}

// MergeAggregateRun records the result of an aggregate-eligible full-file run.
// It replaces the file's examples and runtime aggregate, and records current
// source freshness. The caller must have already verified this is an
// aggregate-eligible run.
func (c *Cache) MergeAggregateRun(filePath string, mtimeUnixNano, sizeBytes int64, runtimeSeconds float64, examples map[string]*ExampleEntry) {
	c.Files[filePath] = FileEntry{
		MtimeUnixNano:  mtimeUnixNano,
		SizeBytes:      sizeBytes,
		RuntimeSeconds: runtimeSeconds,
		Examples:       exampleSlice(examples),
	}
}

func exampleSlice(examples map[string]*ExampleEntry) []ExampleEntry {
	if len(examples) == 0 {
		return nil
	}
	ids := make([]string, 0, len(examples))
	for id, ex := range examples {
		if ex != nil {
			ids = append(ids, id)
		}
	}
	slices.Sort(ids)

	out := make([]ExampleEntry, 0, len(ids))
	for _, id := range ids {
		entry := *examples[id]
		entry.ID = id
		out = append(out, entry)
	}
	return out
}

// MergeObservations records per-example observations from a partial run for a
// single file. It never modifies RuntimeSeconds and never prunes examples
// missing from this run. If the file has no existing entry, nothing is written:
// partial runs must not create file-level aggregates.
func (c *Cache) MergeObservations(filePath string, examples map[string]*ExampleEntry) {
	existing, ok := c.Files[filePath]
	if !ok {
		return
	}
	idx := existing.ExampleIndex()
	for id, ex := range examples {
		if ex == nil {
			continue
		}
		entry := *ex
		entry.ID = id
		if i, ok := idx[id]; ok {
			existing.Examples[i] = entry
		} else {
			idx[id] = len(existing.Examples)
			existing.Examples = append(existing.Examples, entry)
		}
	}
	c.Files[filePath] = existing
}

func (f *FileEntry) ExampleIndex() map[string]int {
	idx := make(map[string]int, len(f.Examples))
	for i, ex := range f.Examples {
		if ex.ID != "" {
			idx[ex.ID] = i
		}
	}
	return idx
}
