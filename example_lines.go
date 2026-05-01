package main

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
)

// rspecDryRunOutput captures the subset of `rspec --dry-run --format json`
// output that we care about: per-example file_path and line_number.
type rspecDryRunOutput struct {
	Examples []struct {
		FilePath   string `json:"file_path"`
		LineNumber int    `json:"line_number"`
	} `json:"examples"`
}

// exampleLinesCacheEntry is what we persist per file.
type exampleLinesCacheEntry struct {
	File  string `json:"file"`
	MTime int64  `json:"mtime_ns"`
	Size  int64  `json:"size"`
	Lines []int  `json:"lines"`
}

// enumerateExampleLines runs a single `<rspecCmd> --dry-run --format json <files...>`
// invocation and returns a map of file_path -> sorted line numbers of every
// example rspec actually registers for that file. This is exact (no fuzzy
// matching), unlike a regex over the source, because conditionally-defined
// examples (`if RuboCop::Platform.windows?`) are skipped here too.
//
// rspecCmd is the base command (e.g. ["bundle", "exec", "rspec"]). filePaths
// are passed to rspec as positional args, relative to cwd.
func enumerateExampleLines(rspecCmd []string, filePaths []string) (map[string][]int, error) {
	if len(rspecCmd) == 0 || len(filePaths) == 0 {
		return nil, fmt.Errorf("enumerateExampleLines: rspecCmd and filePaths must be non-empty")
	}

	args := append([]string{}, rspecCmd[1:]...)
	args = append(args, "--dry-run", "--format", "json")
	args = append(args, filePaths...)

	cmd := exec.Command(rspecCmd[0], args...)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("rspec dry-run failed: %w", err)
	}

	var parsed rspecDryRunOutput
	if err := json.Unmarshal(out, &parsed); err != nil {
		return nil, fmt.Errorf("rspec dry-run JSON parse: %w", err)
	}

	// rspec reports file_path with a "./" prefix; strip it so keys match the
	// caller's relative paths.
	result := make(map[string][]int, len(filePaths))
	for _, ex := range parsed.Examples {
		key := ex.FilePath
		if len(key) > 2 && key[:2] == "./" {
			key = key[2:]
		}
		result[key] = append(result[key], ex.LineNumber)
	}
	for k := range result {
		sort.Ints(result[k])
		result[k] = dedupSortedInts(result[k])
	}
	return result, nil
}

// dedupSortedInts returns xs with adjacent duplicates removed, in place.
// xs must already be sorted. We dedupe because rspec's file:LINE syntax runs
// every example registered at that line, so distributing a duplicated line
// across two chunks would make both chunks run the same examples.
func dedupSortedInts(xs []int) []int {
	if len(xs) < 2 {
		return xs
	}
	out := xs[:1]
	for _, x := range xs[1:] {
		if x != out[len(out)-1] {
			out = append(out, x)
		}
	}
	return out
}

// exampleLinesCacheDir returns the directory we persist per-file caches in.
func exampleLinesCacheDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".plur", "example_lines")
}

// cachePathFor returns the cache file path for a given source file.
func cachePathFor(sourcePath string) string {
	dir := exampleLinesCacheDir()
	if dir == "" {
		return ""
	}
	abs, err := filepath.Abs(sourcePath)
	if err != nil {
		abs = sourcePath
	}
	h := sha1.Sum([]byte(abs))
	return filepath.Join(dir, hex.EncodeToString(h[:])+".json")
}

// loadCachedLines returns the cached example line numbers for sourcePath if
// the cache exists and is fresh (mtime+size match the on-disk file).
func loadCachedLines(sourcePath string) ([]int, bool) {
	cachePath := cachePathFor(sourcePath)
	if cachePath == "" {
		return nil, false
	}
	data, err := os.ReadFile(cachePath)
	if err != nil {
		return nil, false
	}
	var entry exampleLinesCacheEntry
	if err := json.Unmarshal(data, &entry); err != nil {
		return nil, false
	}
	info, err := os.Stat(sourcePath)
	if err != nil {
		return nil, false
	}
	if entry.MTime != info.ModTime().UnixNano() || entry.Size != info.Size() {
		return nil, false
	}
	return entry.Lines, true
}

// saveCachedLines writes the cache entry for sourcePath. Errors are ignored
// (cache is a best-effort optimization).
func saveCachedLines(sourcePath string, lines []int) {
	cachePath := cachePathFor(sourcePath)
	if cachePath == "" {
		return
	}
	info, err := os.Stat(sourcePath)
	if err != nil {
		return
	}
	if err := os.MkdirAll(filepath.Dir(cachePath), 0o755); err != nil {
		return
	}
	entry := exampleLinesCacheEntry{
		File:  sourcePath,
		MTime: info.ModTime().UnixNano(),
		Size:  info.Size(),
		Lines: lines,
	}
	data, err := json.Marshal(entry)
	if err != nil {
		return
	}
	_ = os.WriteFile(cachePath, data, 0o644)
}

// resolveExampleLines returns example lines for each file in filePaths,
// preferring the cache and falling back to a single batched dry-run for
// uncached files. rspecCmd is the base command (e.g. ["bundle", "exec", "rspec"]).
// On any error the returned map only contains entries for files we did
// successfully resolve, so callers should treat missing keys as "fall back to
// the regex-based splitter".
func resolveExampleLines(rspecCmd []string, filePaths []string) map[string][]int {
	result := make(map[string][]int, len(filePaths))
	var needDryRun []string
	for _, f := range filePaths {
		if lines, ok := loadCachedLines(f); ok {
			result[f] = lines
			continue
		}
		needDryRun = append(needDryRun, f)
	}
	if len(needDryRun) == 0 || len(rspecCmd) == 0 {
		return result
	}
	enumerated, err := enumerateExampleLines(rspecCmd, needDryRun)
	if err != nil {
		return result
	}
	for _, f := range needDryRun {
		lines, ok := enumerated[f]
		if !ok {
			continue
		}
		result[f] = lines
		saveCachedLines(f, lines)
	}
	return result
}
