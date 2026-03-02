# Runtime Tracker Overhaul Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace the opaque hash-based runtime file format with a rich, versioned JSON schema that uses human-readable filenames and stores per-file metadata (example_count, last_seen, etc.) to enable smarter test distribution.

**Architecture:** The RuntimeTracker struct gets new fields matching the schema. The file is named `{sanitized-project-path}.json` instead of `{sha256-prefix}.json`. Loading ignores any file without `schema_version: 2` (hard cutover, no legacy loading). The grouper reads `total_seconds` from the new per-file records. A `RuntimeData` struct models the top-level JSON; a `FileRuntime` struct models per-file entries.

**Tech Stack:** Go stdlib (`encoding/json`, `crypto/sha256`, `time`, `path/filepath`), testify for tests

---

## Key Design Decisions

**File naming:** `{dir-basename}-{8-char-hash}.json` where the basename is the last directory component of the absolute project path and the hash is the first 8 chars of SHA-256 of the full absolute path. Example: `/Users/rob/src/myapp` → `myapp-a1b2c3d4.json`. This is human-readable and unique per absolute path. We keep the `project_hash` inside the JSON as metadata as well.

**Schema version:** Integer field `schema_version`. Current version is `2` (we skip `1` since the old flat format was effectively v1). On load, if `schema_version` is missing or not `2`, we treat the file as empty (overwrite on next save).

**Merge strategy:** Same as before — loaded data is the base, new run data overwrites matching keys. But now each file entry tracks `example_count` and `last_seen` in addition to `total_seconds`.

**What we trim from Rob's example:** We drop `runner` block (ruby/rspec versions — not available from plur's perspective without shelling out), `mean_seconds` (computable from total_seconds/example_count), and `last_seconds` (same as total_seconds for current implementation since we overwrite). We keep: `schema_version`, `project_root`, `project_hash`, `generated_at`, `plur_version`, `framework`, and the `files` map with `total_seconds`, `example_count`, `last_seen`.

**Consumers:** `LoadedData()` still returns `map[string]float64` for the grouper — it extracts `total_seconds` from each `FileRuntime`. This means the grouper is unaffected.

---

### Task 1: Define new data structures

**Files:**
* Modify: `runtime_tracker.go`

**Step 1: Add the new structs at the top of runtime_tracker.go**

Add after the imports, before the existing `RuntimeTracker` struct:

```go
const RuntimeSchemaVersion = 2

// RuntimeData is the top-level schema for the runtime JSON file.
type RuntimeData struct {
	SchemaVersion int                    `json:"schema_version"`
	ProjectRoot   string                 `json:"project_root"`
	ProjectHash   string                 `json:"project_hash"`
	GeneratedAt   string                 `json:"generated_at"`
	PlurVersion   string                 `json:"plur_version"`
	Framework     string                 `json:"framework"`
	Files         map[string]FileRuntime `json:"files"`
}

// FileRuntime stores runtime data for a single test file.
type FileRuntime struct {
	TotalSeconds float64 `json:"total_seconds"`
	ExampleCount int     `json:"example_count"`
	LastSeen     string  `json:"last_seen"`
}
```

**Step 2: Update RuntimeTracker struct fields**

Change `runtimes` from `map[string]float64` to `map[string]FileRuntime` and `loadedData` similarly. Add fields for framework and project metadata:

```go
type RuntimeTracker struct {
	runtimes    map[string]FileRuntime // collected this run
	loadedData  map[string]FileRuntime // loaded from file at construction
	runtimeFile string                 // computed once at creation
	projectRoot string                 // absolute path of project
	projectHash string                 // 8-char SHA-256 hash of projectRoot
	framework   string                 // set before save (e.g. "rspec", "minitest")
}
```

**Step 3: Commit**

```bash
git add runtime_tracker.go
git commit -m "Define new RuntimeData and FileRuntime structs for schema v2"
```

---

### Task 2: Rewrite file naming to use sanitized project path

**Files:**
* Modify: `runtime_tracker.go`
* Test: `runtime_tracker_test.go`

**Step 1: Write tests for the new naming scheme**

Add a test for `sanitizeProjectPath`:

```go
t.Run("sanitizeProjectPath produces readable filenames", func(t *testing.T) {
    tests := []struct {
        input    string
        expected string
    }{
        {"/Users/rob/src/myapp", "Users_rob_src_myapp"},
        {"/home/user/project", "home_user_project"},
        {"/tmp/a", "tmp_a"},
    }
    for _, tc := range tests {
        assert.Equal(t, tc.expected, sanitizeProjectPath(tc.input))
    }
})
```

**Step 2: Run test to verify it fails**

Run: `go test -run "TestRuntimeTracker/sanitizeProjectPath" -v`
Expected: FAIL — function doesn't exist yet

**Step 3: Implement `sanitizeProjectPath` and update `computeRuntimeFilePath`**

```go
func sanitizeProjectPath(absPath string) string {
    // Replace path separators with underscores, strip leading underscore
    sanitized := strings.ReplaceAll(absPath, string(filepath.Separator), "_")
    sanitized = strings.TrimLeft(sanitized, "_")
    return sanitized
}
```

Update `computeRuntimeFilePath` to use the new naming and return projectRoot + projectHash alongside:

```go
func computeRuntimeFilePath(runtimeDir string) (filePath, projectRoot, projectHash string, err error) {
    cwd, err := os.Getwd()
    if err != nil {
        return "", "", "", err
    }
    absPath, err := filepath.Abs(cwd)
    if err != nil {
        return "", "", "", err
    }

    hash := sha256.Sum256([]byte(absPath))
    projectHash = hex.EncodeToString(hash[:])[:8]
    sanitized := sanitizeProjectPath(absPath)
    filePath = filepath.Join(runtimeDir, sanitized+".json")
    return filePath, absPath, projectHash, nil
}
```

**Step 4: Update `NewRuntimeTracker` to use the new signature**

**Step 5: Run tests to verify they pass**

Run: `go test -run "TestRuntimeTracker" -v`

**Step 6: Commit**

```bash
git add runtime_tracker.go runtime_tracker_test.go
git commit -m "Use sanitized project path for runtime filenames instead of hash"
```

---

### Task 3: Rewrite load/save to use the new schema

**Files:**
* Modify: `runtime_tracker.go`
* Test: `runtime_tracker_test.go`

**Step 1: Write failing test for new save format**

```go
t.Run("SaveToFile writes schema v2 format", func(t *testing.T) {
    tempDir := t.TempDir()
    rt, err := NewRuntimeTracker(tempDir)
    require.NoError(t, err)
    rt.SetFramework("rspec")

    rt.AddRuntime("spec/foo_spec.rb", 1.5, 3)
    err = rt.SaveToFile()
    require.NoError(t, err)

    // Read and parse the file
    data, err := os.ReadFile(rt.RuntimeFilePath())
    require.NoError(t, err)

    var runtimeData RuntimeData
    err = json.Unmarshal(data, &runtimeData)
    require.NoError(t, err)

    assert.Equal(t, RuntimeSchemaVersion, runtimeData.SchemaVersion)
    assert.NotEmpty(t, runtimeData.ProjectRoot)
    assert.Len(t, runtimeData.ProjectHash, 8)
    assert.NotEmpty(t, runtimeData.GeneratedAt)
    assert.Equal(t, "rspec", runtimeData.Framework)

    fileRT := runtimeData.Files["spec/foo_spec.rb"]
    assert.Equal(t, 1.5, fileRT.TotalSeconds)
    assert.Equal(t, 3, fileRT.ExampleCount)
    assert.NotEmpty(t, fileRT.LastSeen)
})
```

**Step 2: Run test to verify it fails**

**Step 3: Rewrite `AddRuntime` to accept example count**

```go
func (rt *RuntimeTracker) AddRuntime(filePath string, seconds float64, exampleCount int) {
    existing := rt.runtimes[filePath]
    existing.TotalSeconds += seconds
    existing.ExampleCount += exampleCount
    existing.LastSeen = time.Now().UTC().Format(time.RFC3339)
    rt.runtimes[filePath] = existing
}
```

**Step 4: Update `AddTestNotification` to pass example count of 1**

```go
func (rt *RuntimeTracker) AddTestNotification(notification types.TestCaseNotification) {
    if notification.FilePath != "" && notification.Duration > 0 {
        rt.AddRuntime(notification.FilePath, notification.Duration.Seconds(), 1)
    }
}
```

**Step 5: Rewrite `SaveToFile`**

```go
func (rt *RuntimeTracker) SaveToFile() error {
    merged := make(map[string]FileRuntime)
    for k, v := range rt.loadedData {
        merged[k] = v
    }
    for k, v := range rt.runtimes {
        merged[k] = v
    }

    data := RuntimeData{
        SchemaVersion: RuntimeSchemaVersion,
        ProjectRoot:   rt.projectRoot,
        ProjectHash:   rt.projectHash,
        GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
        PlurVersion:   GetVersionInfo(),
        Framework:     rt.framework,
        Files:         merged,
    }

    file, err := os.Create(rt.runtimeFile)
    if err != nil {
        return err
    }
    defer file.Close()

    encoder := json.NewEncoder(file)
    encoder.SetIndent("", "  ")
    return encoder.Encode(data)
}
```

**Step 6: Rewrite `loadExistingData` to handle schema versioning**

```go
func loadExistingData(runtimeFile string) map[string]FileRuntime {
    rawBytes, err := os.ReadFile(runtimeFile)
    if err != nil {
        return make(map[string]FileRuntime)
    }

    var data RuntimeData
    if err := json.Unmarshal(rawBytes, &data); err != nil {
        return make(map[string]FileRuntime)
    }

    // Only load data with matching schema version — hard cutover, no legacy
    if data.SchemaVersion != RuntimeSchemaVersion {
        return make(map[string]FileRuntime)
    }

    if data.Files == nil {
        return make(map[string]FileRuntime)
    }
    return data.Files
}
```

**Step 7: Add `SetFramework` method and `LoadedData` adapter**

`LoadedData()` must still return `map[string]float64` for the grouper:

```go
func (rt *RuntimeTracker) SetFramework(fw string) {
    rt.framework = fw
}

func (rt *RuntimeTracker) LoadedData() map[string]float64 {
    result := make(map[string]float64, len(rt.loadedData))
    for k, v := range rt.loadedData {
        result[k] = v.TotalSeconds
    }
    return result
}
```

**Step 8: Remove `getProjectHash` (now inlined in `computeRuntimeFilePath`)**

**Step 9: Update all existing tests for the new API**

Notably `AddRuntime` now takes 3 args, and `LoadedData` returns float64 values extracted from `FileRuntime.TotalSeconds`.

**Step 10: Run tests**

Run: `go test -run "TestRuntimeTracker" -v`
Expected: all pass

**Step 11: Commit**

```bash
git add runtime_tracker.go runtime_tracker_test.go
git commit -m "Implement schema v2 runtime format with rich metadata"
```

---

### Task 4: Add legacy file handling test

**Files:**
* Test: `runtime_tracker_test.go`

**Step 1: Write test that legacy format is ignored**

```go
t.Run("ignores legacy flat format files", func(t *testing.T) {
    tempDir := t.TempDir()

    // Write a legacy-format file (just key:value floats, no schema_version)
    legacyData := map[string]float64{
        "spec/foo_spec.rb": 1.5,
        "spec/bar_spec.rb": 2.0,
    }
    // We need to write it to the path the tracker would use
    rt, err := NewRuntimeTracker(tempDir)
    require.NoError(t, err)

    legacyBytes, _ := json.Marshal(legacyData)
    os.WriteFile(rt.RuntimeFilePath(), legacyBytes, 0644)

    // Re-create tracker to load
    rt2, err := NewRuntimeTracker(tempDir)
    require.NoError(t, err)

    // Should have empty loaded data since legacy format has no schema_version
    assert.Empty(t, rt2.LoadedData())
})
```

**Step 2: Run test — should pass (already handled by schema check)**

**Step 3: Commit**

```bash
git add runtime_tracker_test.go
git commit -m "Add test confirming legacy runtime format is ignored"
```

---

### Task 5: Wire framework into RuntimeTracker from Runner

**Files:**
* Modify: `runner.go`
* Modify: `main.go` (if needed)

**Step 1: Set framework on the tracker when Runner is created**

In `NewRunner`, after creating the tracker:

```go
tracker.SetFramework(j.Framework)
```

**Step 2: Run Go tests**

Run: `go test ./... -v -count=1`
Expected: all pass

**Step 3: Commit**

```bash
git add runner.go
git commit -m "Wire framework name into RuntimeTracker from Runner"
```

---

### Task 6: Update integration tests

**Files:**
* Modify: `spec/integration/plur_spec/runtime_tracking_spec.rb`

**Step 1: Update runtime file expectations**

The integration tests currently expect `{hash}.json` filenames. Update to expect `{sanitized_path}.json` filenames:

```ruby
# Old:
expect(matches.first).to match(%r{#{temp_runtime_dir}/[a-f0-9]{8}\.json$})

# New — the filename should be the sanitized absolute path:
project_path = File.expand_path(".")
sanitized = project_path.gsub("/", "_").sub(/\A_/, "")
expect(matches.first).to end_with("#{sanitized}.json")
```

**Step 2: Update runtime data parsing to expect new schema**

```ruby
runtime_data = JSON.parse(File.read(runtime_file))
expect(runtime_data["schema_version"]).to eq(2)
expect(runtime_data["project_root"]).to eq(File.expand_path("."))
expect(runtime_data["files"]).to be_a(Hash)
expect(runtime_data["files"].keys).to include("spec/calculator_spec.rb")

file_entry = runtime_data["files"]["spec/calculator_spec.rb"]
expect(file_entry["total_seconds"]).to be > 0
expect(file_entry["example_count"]).to be > 0
expect(file_entry["last_seen"]).to match(/\d{4}-\d{2}-\d{2}T/)
```

**Step 3: Update the "runtime-based grouping" test that writes fake runtime data**

The fake data must now be in schema v2 format:

```ruby
runtime_data = {
  "schema_version" => 2,
  "project_root" => File.expand_path("."),
  "project_hash" => project_hash,
  "generated_at" => Time.now.utc.iso8601,
  "plur_version" => "test",
  "framework" => "rspec",
  "files" => {
    "spec/calculator_spec.rb" => { "total_seconds" => 5.0, "example_count" => 10, "last_seen" => Time.now.utc.iso8601 },
    # ... other files ...
  }
}
```

Also update the filename used:
```ruby
sanitized = File.expand_path(".").gsub("/", "_").sub(/\A_/, "")
File.write(File.join(runtime_dir, "#{sanitized}.json"), JSON.pretty_generate(runtime_data))
```

**Step 4: Update the "merge behavior" test**

```ruby
updated_data = JSON.parse(File.read(runtime_file))
expect(updated_data["schema_version"]).to eq(2)
expect(updated_data["files"].keys.size).to eq(initial_data["files"].keys.size)
```

**Step 5: Run integration tests**

Run: `bin/rspec spec/integration/plur_spec/runtime_tracking_spec.rb`
Expected: all pass

**Step 6: Commit**

```bash
git add spec/integration/plur_spec/runtime_tracking_spec.rb
git commit -m "Update integration tests for schema v2 runtime format"
```

---

### Task 7: Run full build and verify

**Step 1: Run full build**

Run: `bin/rake`
Expected: all lint, build, tests pass

**Step 2: Manual smoke test**

```bash
bin/rake install
cd fixtures/default-ruby
plur -n 2
cat ~/.plur/runtime/*.json  # Should see new schema v2 format with readable filename
```

**Step 3: Final commit if any fixups needed**

---

## Summary of Changes

| File | Change |
|------|--------|
| `runtime_tracker.go` | New `RuntimeData`/`FileRuntime` structs, sanitized filenames, schema v2 load/save, `SetFramework()`, `LoadedData()` adapter |
| `runtime_tracker_test.go` | Updated for new API, new tests for sanitized paths, schema v2, legacy rejection |
| `runner.go` | Wire `SetFramework` call in `NewRunner` |
| `spec/integration/plur_spec/runtime_tracking_spec.rb` | Updated expectations for new filename format and JSON schema |

**Files NOT changed:** `grouper.go`, `grouper_test.go` — the `LoadedData()` adapter preserves the `map[string]float64` interface.
