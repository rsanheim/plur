# RSpec Line Splitting Alpha Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship the autoresearch RSpec long-spec splitting win behind an explicit alpha feature flag so it can land in mainline and be tested on real projects without changing default behavior.

**Architecture:** Keep the existing file-level runtime grouper as the default. When `--rspec-split=alpha` or `PLUR_RSPEC_SPLIT=alpha` is set, the runtime grouper may split long-running RSpec files into `spec/foo_spec.rb:line:line` chunks using exact example line numbers from `bin/rspec --dry-run --format json`. If line discovery is unavailable or unsafe for the current invocation, the grouper silently falls back to current file-level grouping and logs a debug message.

**Tech Stack:** Go CLI using Kong, existing Plur runtime tracking, RSpec `file:line` focused execution, JSON parsing, best-effort cache under `$PLUR_HOME/cache`.

---

## Source Material

Use the `autoresearch` branch only as reference. Do not cherry-pick the branch wholesale.

Lift and adapt:
- `example_lines.go`: exact RSpec dry-run line discovery and cache idea.
- `long_pole.go`: converting example line numbers into RSpec `file:line:line` targets.
- Selected `grouper.go` hunk: expand long-pole files before LPT grouping.
- Selected `runner.go` hunk: pass RSpec command context into the runtime grouper.

Do not lift:
- `DefaultWorkerCount` changes from 4 to 12.
- Rails setup docs/spec changes. Those live on a separate branch.
- Regex fallback for example lines. Mainline alpha should use exact RSpec dry-run lines or skip splitting.
- Always-on splitting. Alpha must be opt-in.

## Proposed User Interface

CLI:

```bash
plur --rspec-split=alpha -n 8
```

Environment:

```bash
PLUR_RSPEC_SPLIT=alpha plur -n 8
```

Default:

```bash
plur
# equivalent to --rspec-split=off
```

Valid values:
- `off`: default; current behavior.
- `alpha`: enable exact-line RSpec long-spec splitting when safe.

The feature applies only when:
- selected job framework is `rspec`
- runtime data exists
- worker count is greater than 1
- the run is not serial
- the run has no unsupported passthrough args
- exact RSpec example lines are available or discoverable

## Safety Policy

- No default behavior changes.
- No worker count default changes.
- No splitting for non-RSpec jobs.
- No splitting without runtime data.
- No splitting when RSpec line discovery fails.
- No regex/source-code fallback in alpha.
- No shelling out to RSpec during `--dry-run`; dry-run may use fresh cache entries only.
- No splitting when arbitrary passthrough args are present. Support `--tag` because Plur builds those directly and they affect which examples RSpec registers.
- Runtime data remains keyed by original spec file path. RSpec formatter notifications already report `file_path` without `:line` selectors, so the existing runtime tracker should continue aggregating by file.

## File Structure

Create:
- `rspec_example_lines.go`: RSpec dry-run JSON parsing, exact line deduping, line cache load/save.
- `rspec_example_lines_test.go`: parser/cache tests for exact line discovery.
- `rspec_line_splitter.go`: pure functions for deciding whether and how to split one long-running RSpec file.
- `rspec_line_splitter_test.go`: pure splitter tests.

Modify:
- `config/config.go`: add `RspecSplit` to `GlobalConfig`.
- `main.go`: add `RspecSplitMode`, CLI/env flag, and validation.
- `main_test.go`: cover valid and invalid flag values.
- `grouper.go`: add opt-in runtime grouping options and alpha split expansion before sorting/bin packing.
- `grouper_test.go`: cover off-by-default and alpha split behavior.
- `runner.go`: pass alpha split options only for RSpec jobs.
- `runner_test.go`: cover runner option wiring and dry-run/no-shellout behavior.
- `docs/usage.md`: document alpha flag with constraints.

---

## Task 1: Add the Alpha Feature Flag

**Files:**
- Modify: `config/config.go`
- Modify: `main.go`
- Test: `main_test.go`

- [ ] **Step 1: Write failing validation tests**

Add tests near the existing worker count validation tests in `main_test.go`:

```go
func TestRspecSplitModeValidation(t *testing.T) {
	t.Run("accepts off", func(t *testing.T) {
		assert.NoError(t, RspecSplitMode("off").Validate())
	})

	t.Run("accepts alpha", func(t *testing.T) {
		assert.NoError(t, RspecSplitMode("alpha").Validate())
	})

	t.Run("rejects unknown value", func(t *testing.T) {
		err := RspecSplitMode("beta").Validate()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "rspec split mode must be one of: off, alpha")
	})
}
```

- [ ] **Step 2: Run the focused test and verify it fails**

Run:

```bash
go test -mod=mod . -run TestRspecSplitModeValidation
```

Expected: compile failure because `RspecSplitMode` is not defined.

- [ ] **Step 3: Implement the minimal flag type**

Add to `main.go` near `WorkerCount`:

```go
type RspecSplitMode string

const (
	RspecSplitOff   RspecSplitMode = "off"
	RspecSplitAlpha RspecSplitMode = "alpha"
)

func (m RspecSplitMode) Validate() error {
	switch m {
	case RspecSplitOff, RspecSplitAlpha:
		return nil
	default:
		return fmt.Errorf("rspec split mode must be one of: off, alpha")
	}
}
```

Add the CLI field to `PlurCLI`:

```go
RspecSplit RspecSplitMode `help:"Enable alpha RSpec file:line splitting mode (off, alpha)" name:"rspec-split" env:"PLUR_RSPEC_SPLIT" default:"off"`
```

Update `PlurCLI.Validate()`:

```go
if err := cli.RspecSplit.Validate(); err != nil {
	return fmt.Errorf("--rspec-split: %w", err)
}
```

Add to `config.GlobalConfig` in `config/config.go`:

```go
RspecSplit string
```

Populate it in `AfterApply()`:

```go
RspecSplit: string(cli.RspecSplit),
```

- [ ] **Step 4: Run the focused test and verify it passes**

Run:

```bash
go test -mod=mod . -run TestRspecSplitModeValidation
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add main.go main_test.go config/config.go
git commit -m "Add alpha RSpec split feature flag"
```

---

## Task 2: Add Exact RSpec Example Line Discovery

**Files:**
- Create: `rspec_example_lines.go`
- Create: `rspec_example_lines_test.go`

- [ ] **Step 1: Write failing JSON parsing and cache tests**

Create `rspec_example_lines_test.go`:

```go
package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseRSpecDryRunExampleLines(t *testing.T) {
	jsonOutput := []byte(`{
	  "examples": [
	    {"file_path":"./spec/slow_spec.rb","line_number":10},
	    {"file_path":"./spec/slow_spec.rb","line_number":10},
	    {"file_path":"./spec/slow_spec.rb","line_number":40},
	    {"file_path":"./spec/fast_spec.rb","line_number":7}
	  ]
	}`)

	lines, err := parseRSpecDryRunExampleLines(jsonOutput)

	require.NoError(t, err)
	assert.Equal(t, []int{10, 40}, lines["spec/slow_spec.rb"])
	assert.Equal(t, []int{7}, lines["spec/fast_spec.rb"])
}

func TestRSpecExampleLineCacheUsesFileMetadata(t *testing.T) {
	cacheDir := t.TempDir()
	specPath := filepath.Join(t.TempDir(), "slow_spec.rb")
	require.NoError(t, os.WriteFile(specPath, []byte("RSpec.describe('x') { it('a') {} }\n"), 0o644))

	saveCachedRSpecExampleLines(cacheDir, specPath, []int{3, 9})
	lines, ok := loadCachedRSpecExampleLines(cacheDir, specPath)

	require.True(t, ok)
	assert.Equal(t, []int{3, 9}, lines)

	require.NoError(t, os.WriteFile(specPath, []byte("RSpec.describe('x') { it('changed') {} }\n"), 0o644))
	_, ok = loadCachedRSpecExampleLines(cacheDir, specPath)
	assert.False(t, ok)
}
```

- [ ] **Step 2: Run tests and verify they fail**

Run:

```bash
go test -mod=mod . -run 'TestParseRSpecDryRunExampleLines|TestRSpecExampleLineCacheUsesFileMetadata'
```

Expected: compile failure because the parsing/cache functions do not exist.

- [ ] **Step 3: Implement parsing and cache**

Create `rspec_example_lines.go` with these public-to-package functions:

```go
package main

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
)

type rspecDryRunOutput struct {
	Examples []struct {
		FilePath   string `json:"file_path"`
		LineNumber int    `json:"line_number"`
	} `json:"examples"`
}

type rspecExampleLinesCacheEntry struct {
	File  string `json:"file"`
	MTime int64  `json:"mtime_ns"`
	Size  int64  `json:"size"`
	Lines []int  `json:"lines"`
}
```

Implement:
- `parseRSpecDryRunExampleLines(output []byte) (map[string][]int, error)`
- `dedupSortedInts(xs []int) []int`
- `rspecExampleLinesCachePath(cacheDir, sourcePath string) string`
- `loadCachedRSpecExampleLines(cacheDir, sourcePath string) ([]int, bool)`
- `saveCachedRSpecExampleLines(cacheDir, sourcePath string, lines []int)`

Implementation details:
- Strip leading `./` from RSpec JSON file paths.
- Sort line numbers.
- Deduplicate duplicate line numbers.
- Store cache files under `filepath.Join(cacheDir, "rspec-example-lines")`.
- Hash absolute file paths using SHA-1 for cache filenames.
- Cache freshness is based on source file `mtime_ns` and `size`.
- Cache write failures are ignored.

- [ ] **Step 4: Run tests and verify they pass**

Run:

```bash
go test -mod=mod . -run 'TestParseRSpecDryRunExampleLines|TestRSpecExampleLineCacheUsesFileMetadata'
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add rspec_example_lines.go rspec_example_lines_test.go
git commit -m "Add RSpec example line cache"
```

---

## Task 3: Add RSpec Dry-Run Line Resolution

**Files:**
- Modify: `rspec_example_lines.go`
- Modify: `rspec_example_lines_test.go`

- [ ] **Step 1: Write failing resolver tests**

Append to `rspec_example_lines_test.go`:

```go
func TestResolveRSpecExampleLinesUsesCacheWithoutRunner(t *testing.T) {
	cacheDir := t.TempDir()
	specPath := filepath.Join(t.TempDir(), "cached_spec.rb")
	require.NoError(t, os.WriteFile(specPath, []byte("RSpec.describe('x') {}\n"), 0o644))
	saveCachedRSpecExampleLines(cacheDir, specPath, []int{5, 11})

	called := false
	resolved := resolveRSpecExampleLines(RSpecExampleLineResolveOptions{
		CacheDir: cacheDir,
		Files:    []string{specPath},
		RunDryRun: func(args []string) ([]byte, error) {
			called = true
			return nil, nil
		},
	})

	assert.False(t, called)
	assert.Equal(t, []int{5, 11}, resolved[specPath])
}

func TestResolveRSpecExampleLinesRunsBatchedDryRunForUncachedFiles(t *testing.T) {
	cacheDir := t.TempDir()
	specPath := filepath.Join(t.TempDir(), "uncached_spec.rb")
	require.NoError(t, os.WriteFile(specPath, []byte("RSpec.describe('x') {}\n"), 0o644))

	var gotArgs []string
	resolved := resolveRSpecExampleLines(RSpecExampleLineResolveOptions{
		CacheDir:     cacheDir,
		RspecCmd:     []string{"bin/rspec"},
		SelectorArgs: []string{"--tag", "slow"},
		Files:        []string{specPath},
		RunDryRun: func(args []string) ([]byte, error) {
			gotArgs = append([]string{}, args...)
			return []byte(`{"examples":[{"file_path":"` + specPath + `","line_number":12}]}`), nil
		},
	})

	assert.Equal(t, []string{"bin/rspec", "--tag", "slow", "--dry-run", "--format", "json", specPath}, gotArgs)
	assert.Equal(t, []int{12}, resolved[specPath])
}
```

- [ ] **Step 2: Run tests and verify they fail**

Run:

```bash
go test -mod=mod . -run 'TestResolveRSpecExampleLines'
```

Expected: compile failure because `RSpecExampleLineResolveOptions` and `resolveRSpecExampleLines` do not exist.

- [ ] **Step 3: Implement resolver**

Add to `rspec_example_lines.go`:

```go
type RSpecExampleLineResolveOptions struct {
	CacheDir     string
	RspecCmd     []string
	SelectorArgs []string
	Files        []string
	DryRunOnly   bool
	RunDryRun    func(args []string) ([]byte, error)
}
```

Implementation rules:
- Return cached lines first.
- If `DryRunOnly` is true, never call `RunDryRun`.
- If no `RspecCmd` or no `RunDryRun`, return cached-only result.
- Build command args as:
  `RspecCmd + SelectorArgs + ["--dry-run", "--format", "json"] + uncached files`
- Parse JSON output and save cache entries.
- On any dry-run or parse error, return cached results only.

Use `exec.Command` through a helper for production:

```go
func runRSpecDryRunCommand(args []string) ([]byte, error) {
	cmd := exec.Command(args[0], args[1:]...)
	return cmd.Output()
}
```

- [ ] **Step 4: Run tests and verify they pass**

Run:

```bash
go test -mod=mod . -run 'TestResolveRSpecExampleLines'
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add rspec_example_lines.go rspec_example_lines_test.go
git commit -m "Resolve RSpec example lines with dry-run"
```

---

## Task 4: Add Pure RSpec Line Splitting

**Files:**
- Create: `rspec_line_splitter.go`
- Create: `rspec_line_splitter_test.go`

- [ ] **Step 1: Write failing splitter tests**

Create `rspec_line_splitter_test.go`:

```go
package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSplitRSpecFileByLines(t *testing.T) {
	chunks, ok := splitRSpecFileByLines("spec/slow_spec.rb", []int{10, 20, 30, 40, 50, 60, 70, 80}, 2)

	assert.True(t, ok)
	assert.Equal(t, []string{
		"spec/slow_spec.rb:10:30:50:70",
		"spec/slow_spec.rb:20:40:60:80",
	}, chunks)
}

func TestSplitRSpecFileByLinesRequiresEnoughExamplesPerChunk(t *testing.T) {
	chunks, ok := splitRSpecFileByLines("spec/small_spec.rb", []int{10, 20, 30}, 2)

	assert.False(t, ok)
	assert.Equal(t, []string{"spec/small_spec.rb"}, chunks)
}

func TestRSpecChunkCountCapsAtWorkers(t *testing.T) {
	chunks := rspecChunkCount(40.0, 5.0, 4)

	assert.Equal(t, 4, chunks)
}
```

- [ ] **Step 2: Run tests and verify they fail**

Run:

```bash
go test -mod=mod . -run 'TestSplitRSpecFileByLines|TestRSpecChunkCountCapsAtWorkers'
```

Expected: compile failure because splitter functions do not exist.

- [ ] **Step 3: Implement splitter**

Create `rspec_line_splitter.go`:

```go
package main

import (
	"strconv"
	"strings"
)

const minRSpecExamplesPerChunk = 4

func splitRSpecFileByLines(filePath string, lines []int, numChunks int) ([]string, bool) {
	if numChunks <= 1 || len(lines) < numChunks*minRSpecExamplesPerChunk {
		return []string{filePath}, false
	}

	builders := make([]strings.Builder, numChunks)
	for i, line := range lines {
		idx := i % numChunks
		if builders[idx].Len() == 0 {
			builders[idx].WriteString(filePath)
		}
		builders[idx].WriteByte(':')
		builders[idx].WriteString(strconv.Itoa(line))
	}

	chunks := make([]string, 0, numChunks)
	for i := range builders {
		chunk := builders[i].String()
		if chunk != "" {
			chunks = append(chunks, chunk)
		}
	}
	if len(chunks) < 2 {
		return []string{filePath}, false
	}
	return chunks, true
}

func rspecChunkCount(fileRuntime, perWorkerTarget float64, workerCount int) int {
	if workerCount < 2 || perWorkerTarget <= 0 {
		return 1
	}
	chunks := int(fileRuntime/perWorkerTarget + 0.5)
	if chunks < 2 {
		chunks = 2
	}
	if chunks > workerCount {
		chunks = workerCount
	}
	return chunks
}
```

- [ ] **Step 4: Run tests and verify they pass**

Run:

```bash
go test -mod=mod . -run 'TestSplitRSpecFileByLines|TestRSpecChunkCountCapsAtWorkers'
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add rspec_line_splitter.go rspec_line_splitter_test.go
git commit -m "Add pure RSpec line splitter"
```

---

## Task 5: Expand Runtime Groups Behind Alpha

**Files:**
- Modify: `grouper.go`
- Modify: `grouper_test.go`

- [ ] **Step 1: Write failing grouper tests**

Append to `grouper_test.go`:

```go
func TestGroupSpecFilesByRuntimeWithRSpecSplitting(t *testing.T) {
	files := []string{
		"spec/slow_spec.rb",
		"spec/fast_spec.rb",
	}
	runtimeData := map[string]float64{
		"spec/slow_spec.rb": 30.0,
		"spec/fast_spec.rb": 1.0,
	}
	exampleLines := map[string][]int{
		"spec/slow_spec.rb": []int{10, 20, 30, 40, 50, 60, 70, 80},
	}

	groups := GroupSpecFilesByRuntimeWithOpts(files, 2, runtimeData, GroupOpts{
		RspecSplit:   "alpha",
		ExampleLines: exampleLines,
	})

	allFiles := flattenGroupFiles(groups)
	assert.Contains(t, allFiles, "spec/slow_spec.rb:10:30:50:70")
	assert.Contains(t, allFiles, "spec/slow_spec.rb:20:40:60:80")
	assert.NotContains(t, allFiles, "spec/slow_spec.rb")
}

func TestGroupSpecFilesByRuntimeDoesNotSplitByDefault(t *testing.T) {
	files := []string{"spec/slow_spec.rb", "spec/fast_spec.rb"}
	runtimeData := map[string]float64{
		"spec/slow_spec.rb": 30.0,
		"spec/fast_spec.rb": 1.0,
	}

	groups := GroupSpecFilesByRuntime(files, 2, runtimeData)

	allFiles := flattenGroupFiles(groups)
	assert.Contains(t, allFiles, "spec/slow_spec.rb")
	assert.NotContains(t, strings.Join(allFiles, " "), "spec/slow_spec.rb:")
}
```

Add helper:

```go
func flattenGroupFiles(groups []FileGroup) []string {
	var files []string
	for _, group := range groups {
		files = append(files, group.Files...)
	}
	return files
}
```

Import `strings` in `grouper_test.go`.

- [ ] **Step 2: Run tests and verify they fail**

Run:

```bash
go test -mod=mod . -run 'TestGroupSpecFilesByRuntimeWithRSpecSplitting|TestGroupSpecFilesByRuntimeDoesNotSplitByDefault'
```

Expected: compile failure because `GroupSpecFilesByRuntimeWithOpts` and `GroupOpts` do not exist.

- [ ] **Step 3: Implement opt-in group expansion**

In `grouper.go`, add:

```go
type GroupOpts struct {
	RspecSplit   string
	ExampleLines map[string][]int
}
```

Change:

```go
func GroupSpecFilesByRuntime(specFiles []string, numWorkers int, runtimeData map[string]float64) []FileGroup {
	return GroupSpecFilesByRuntimeWithOpts(specFiles, numWorkers, runtimeData, GroupOpts{})
}
```

Add `GroupSpecFilesByRuntimeWithOpts` as the existing body plus a pre-sort expansion step:

```go
if opts.RspecSplit == "alpha" {
	filesWithRuntimes = expandLongRSpecFiles(filesWithRuntimes, numWorkers, opts)
}
```

Implement expansion rules:
- Compute total runtime from `filesWithRuntimes`.
- `perWorkerTarget := totalRuntime / float64(numWorkers)`.
- `splitThreshold := perWorkerTarget * 0.9`.
- For files above threshold, look up exact lines from `opts.ExampleLines`.
- If no lines are present, keep original file.
- Chunk count is `rspecChunkCount(file.runtime, perWorkerTarget, numWorkers)`.
- If `splitRSpecFileByLines` returns chunks, replace original file with chunk entries whose runtime is `file.runtime / float64(len(chunks))`.
- Log debug metadata when at least one file splits.

- [ ] **Step 4: Run grouper tests**

Run:

```bash
go test -mod=mod . -run 'TestGroupSpecFilesByRuntime'
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add grouper.go grouper_test.go
git commit -m "Split long RSpec files in runtime grouper behind alpha"
```

---

## Task 6: Wire Runner Context Into the Grouper

**Files:**
- Modify: `runner.go`
- Modify: `runner_test.go`

- [ ] **Step 1: Write failing runner tests**

Add to `runner_test.go`:

```go
func TestRunnerBuildsRSpecSplitGroupOptionsOnlyForAlphaRSpec(t *testing.T) {
	cfg := &config.GlobalConfig{
		WorkerCount: 4,
		RuntimeDir:  t.TempDir(),
		RspecSplit:  "alpha",
		ConfigPaths: &config.ConfigPaths{CacheDir: t.TempDir()},
	}
	testJob := job.Job{
		Name:      "rspec",
		Cmd:       []string{"bundle", "exec", "rspec", "{{target}}"},
		Framework: "rspec",
	}

	runner, err := NewRunner(cfg, []string{"spec/slow_spec.rb"}, testJob, []string{"--tag", "slow"})
	require.NoError(t, err)

	opts := runner.rspecSplitGroupOpts()

	assert.Equal(t, "alpha", opts.RspecSplit)
	assert.Equal(t, []string{"bundle", "exec", "rspec"}, opts.RspecCmd)
	assert.Equal(t, []string{"--tag", "slow"}, opts.SelectorArgs)
	assert.NotEmpty(t, opts.CacheDir)
}

func TestRunnerDisablesRSpecSplitForUnsupportedExtraArgs(t *testing.T) {
	cfg := &config.GlobalConfig{
		WorkerCount: 4,
		RuntimeDir:  t.TempDir(),
		RspecSplit:  "alpha",
		ConfigPaths: &config.ConfigPaths{CacheDir: t.TempDir()},
	}
	testJob := job.Job{Name: "rspec", Cmd: []string{"bin/rspec"}, Framework: "rspec"}

	runner, err := NewRunner(cfg, []string{"spec/slow_spec.rb"}, testJob, []string{"--format", "documentation"})
	require.NoError(t, err)

	opts := runner.rspecSplitGroupOpts()

	assert.Equal(t, "off", opts.RspecSplit)
}
```

- [ ] **Step 2: Run tests and verify they fail**

Run:

```bash
go test -mod=mod . -run 'TestRunnerBuildsRSpecSplitGroupOptionsOnlyForAlphaRSpec|TestRunnerDisablesRSpecSplitForUnsupportedExtraArgs'
```

Expected: compile failure because `rspecSplitGroupOpts` and GroupOpts fields are missing.

- [ ] **Step 3: Extend `GroupOpts`**

Update `grouper.go`:

```go
type GroupOpts struct {
	RspecSplit   string
	RspecCmd     []string
	SelectorArgs []string
	CacheDir     string
	DryRunOnly   bool
	ExampleLines map[string][]int
}
```

In `GroupSpecFilesByRuntimeWithOpts`, resolve example lines when alpha is enabled and `ExampleLines` was not supplied:

```go
if opts.RspecSplit == "alpha" && opts.ExampleLines == nil {
	opts.ExampleLines = resolveRSpecExampleLines(RSpecExampleLineResolveOptions{
		CacheDir:     opts.CacheDir,
		RspecCmd:     opts.RspecCmd,
		SelectorArgs: opts.SelectorArgs,
		Files:        longPoleFiles,
		DryRunOnly:   opts.DryRunOnly,
		RunDryRun:    runRSpecDryRunCommand,
	})
}
```

- [ ] **Step 4: Implement runner option builder**

Add to `runner.go`:

```go
func (r *Runner) rspecSplitGroupOpts() GroupOpts {
	if r.job.Framework != "rspec" || r.config.RspecSplit != "alpha" || r.config.WorkerCount <= 1 {
		return GroupOpts{RspecSplit: "off"}
	}

	selectorArgs, ok := rspecSplitSelectorArgs(r.extraArgs)
	if !ok {
		logger.Logger.Debug("RSpec split disabled due to unsupported extra args", "extra_args", r.extraArgs)
		return GroupOpts{RspecSplit: "off"}
	}

	cacheDir := ""
	if r.config.ConfigPaths != nil {
		cacheDir = r.config.ConfigPaths.CacheDir
	}

	return GroupOpts{
		RspecSplit:   "alpha",
		RspecCmd:     rspecBaseCmd(r.job),
		SelectorArgs: selectorArgs,
		CacheDir:     cacheDir,
		DryRunOnly:   r.config.DryRun,
	}
}
```

Add helper:

```go
func rspecSplitSelectorArgs(extraArgs []string) ([]string, bool) {
	var selectors []string
	for i := 0; i < len(extraArgs); i++ {
		arg := extraArgs[i]
		if arg == "--tag" {
			if i+1 >= len(extraArgs) {
				return nil, false
			}
			selectors = append(selectors, arg, extraArgs[i+1])
			i++
			continue
		}
		if strings.HasPrefix(arg, "--tag=") {
			selectors = append(selectors, arg)
			continue
		}
		return nil, false
	}
	return selectors, true
}
```

Update `groupFiles()`:

```go
groups = GroupSpecFilesByRuntimeWithOpts(r.files, r.config.WorkerCount, runtimeData, r.rspecSplitGroupOpts())
```

Keep `GroupSpecFilesByRuntime` untouched as the compatibility wrapper.

- [ ] **Step 5: Run runner tests**

Run:

```bash
go test -mod=mod . -run 'TestRunnerBuildsRSpecSplitGroupOptionsOnlyForAlphaRSpec|TestRunnerDisablesRSpecSplitForUnsupportedExtraArgs'
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add runner.go runner_test.go grouper.go
git commit -m "Wire RSpec split options through runner"
```

---

## Task 7: Add Integration Coverage

**Files:**
- Modify: `spec/integration/spec/runtime_tracking_spec.rb`
- Test fixture: `fixtures/projects/default-ruby/spec/calculator_spec.rb`

- [ ] **Step 1: Write failing integration spec for off by default**

Add to `spec/integration/spec/runtime_tracking_spec.rb` near the existing fake runtime data grouping example:

```ruby
it "does not split long RSpec files without the alpha flag" do
  Dir.chdir(default_ruby_dir) do
    project_hash = Digest::SHA256.hexdigest(Dir.pwd)[0, 8]
    runtime_dir = File.join(tmp_plur_home, "runtime")
    FileUtils.mkdir_p(runtime_dir)
    File.write(File.join(runtime_dir, "#{project_hash}.json"), JSON.pretty_generate({
      "spec/calculator_spec.rb" => 30.0,
      "spec/counter_spec.rb" => 1.0
    }))

    result = run_plur("--dry-run", "--debug", "-n", "2")

    worker_lines = result.err.lines.select { |line| line.include?("[dry-run] Worker") }
    expect(worker_lines.join).to include("spec/calculator_spec.rb")
    expect(worker_lines.join).not_to match(/spec\/calculator_spec\.rb:\d+/)
  end
end
```

- [ ] **Step 2: Write failing integration spec for alpha cache-based dry-run**

Add:

```ruby
it "splits cached long RSpec files with the alpha flag" do
  Dir.chdir(default_ruby_dir) do
    project_hash = Digest::SHA256.hexdigest(Dir.pwd)[0, 8]
    runtime_dir = File.join(tmp_plur_home, "runtime")
    FileUtils.mkdir_p(runtime_dir)
    File.write(File.join(runtime_dir, "#{project_hash}.json"), JSON.pretty_generate({
      "spec/calculator_spec.rb" => 30.0,
      "spec/counter_spec.rb" => 1.0
    }))

    cache_dir = File.join(tmp_plur_home, "cache", "rspec-example-lines")
    FileUtils.mkdir_p(cache_dir)
    source = File.expand_path("spec/calculator_spec.rb")
    digest = Digest::SHA1.hexdigest(source)
    stat = File.stat("spec/calculator_spec.rb")
    mtime_ns = (stat.mtime.to_r * 1_000_000_000).to_i
    File.write(File.join(cache_dir, "#{digest}.json"), JSON.generate({
      file: "spec/calculator_spec.rb",
      mtime_ns: mtime_ns,
      size: stat.size,
      lines: [5, 10, 15, 20, 25, 30, 35, 40]
    }))

    result = run_plur("--dry-run", "--debug", "-n", "2", env: {"PLUR_RSPEC_SPLIT" => "alpha"})

    worker_lines = result.err.lines.select { |line| line.include?("[dry-run] Worker") }
    expect(worker_lines.join).to match(/spec\/calculator_spec\.rb:\d+:\d+/)
  end
end
```

The `mtime_ns` value must be an integer matching Go `UnixNano()`.

- [ ] **Step 3: Run the integration file and verify alpha test fails**

Run:

```bash
bin/rspec spec/integration/spec/runtime_tracking_spec.rb
```

Expected: the new alpha test fails before implementation is complete.

- [ ] **Step 4: Complete any missing cache path or dry-run behavior**

Adjust implementation so:
- cache path matches `$PLUR_HOME/cache/rspec-example-lines/<sha1(abs_path)>.json`
- dry-run uses cache only
- worker lines include `spec/calculator_spec.rb:<line>` when alpha and cache are present

- [ ] **Step 5: Run integration file**

Run:

```bash
bin/rspec spec/integration/spec/runtime_tracking_spec.rb
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add spec/integration/spec/runtime_tracking_spec.rb rspec_example_lines.go grouper.go runner.go
git commit -m "Cover alpha RSpec splitting in integration specs"
```

---

## Task 8: Document Alpha Usage

**Files:**
- Modify: `docs/usage.md`
- Modify: `README.md` only if the CLI examples section needs an alpha note after review

- [ ] **Step 1: Add usage docs**

Add a short section to `docs/usage.md` after "Rails And Rake Commands" or near runtime tracking:

````markdown
### Alpha RSpec Line Splitting

Large RSpec files can dominate wall time even when file-level grouping is balanced. The alpha RSpec splitter can break long-running spec files into focused RSpec `file:line` runs:

```bash
PLUR_RSPEC_SPLIT=alpha plur -n 8
plur --rspec-split=alpha -n 8
```

This mode is experimental and opt-in. It only applies to RSpec jobs with runtime data. Plur asks RSpec for exact example line numbers using `rspec --dry-run --format json`, caches those lines under `$PLUR_HOME/cache`, and falls back to normal file-level grouping if discovery fails.
````

- [ ] **Step 2: Run docs grep**

Run:

```bash
rg -n "rspec-split|PLUR_RSPEC_SPLIT|Alpha RSpec" README.md docs
```

Expected: the new docs mention both CLI and env forms.

- [ ] **Step 3: Commit**

```bash
git add docs/usage.md README.md
git commit -m "Document alpha RSpec line splitting"
```

---

## Task 9: Final Verification

**Files:**
- No planned edits.

- [ ] **Step 1: Run Go tests**

Run:

```bash
bin/rake test:go
```

Expected: all Go packages pass.

- [ ] **Step 2: Run focused Ruby integration tests**

Run:

```bash
bin/rspec spec/integration/spec/runtime_tracking_spec.rb spec/integration/spec/rspec_args_spec.rb
```

Expected: all examples pass.

- [ ] **Step 3: Run full Ruby suite if local environment is stable**

Run:

```bash
bin/rake test
```

Expected: all non-pending examples pass. If watch-related specs fail due local filesystem watcher behavior, record the exact failures in the branch notes and run the narrower non-watch suite before opening a PR.

- [ ] **Step 4: Inspect final diff**

Run:

```bash
git diff --stat main..HEAD
git diff --name-status main..HEAD
```

Expected: only files related to RSpec splitting, docs, and tests changed.

- [ ] **Step 5: Commit any final fixes**

If verification required edits:

```bash
git add .
git commit -m "Stabilize alpha RSpec line splitting"
```

If no edits were required, leave the branch at the latest task commit.

## Review Checklist

- The branch is independent of `autoresearch`.
- The branch is independent of `rails-schema-setup`.
- Default behavior is unchanged.
- `DefaultWorkerCount` remains 4.
- Alpha mode is explicit through `--rspec-split=alpha` or `PLUR_RSPEC_SPLIT=alpha`.
- Dry-run does not launch RSpec for discovery.
- Unsupported passthrough args disable splitting.
- Tag filters are passed to RSpec dry-run discovery.
- Runtime tracking still stores original file paths.
- All alpha fallback paths return normal file-level grouping.
