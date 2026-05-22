# Runtime Cache Measurement and Bin-Packing — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking. Pair with the spec at [docs/plans/2026-05-22-runtime-cache-measurement-and-bin-packing-plan.md](2026-05-22-runtime-cache-measurement-and-bin-packing-plan.md) — the spec defines the *what* and *why*; this document covers the *how*, sequencing, and verification.

**Goal:** Move the runtime files into `internal/testruntime/`, trim per-example dead-weight fields, attribute shared examples via `location_rerun_argument`, replace round-robin splitting with bin-packing as a method on `*Cache`, add load/save debug logging, amend `plur doctor`, and run real-project QA to decide whether further format work is needed.

**Architecture:** One branch on top of `rspec-split-specs`. Task 1 is a series of small move + rename commits to keep each step reviewable. Tasks 2–9 build on the new layout. Task 10 is a dedicated simplification pass. Task 11 verifies and finalizes.

**Status snapshot, 2026-05-22:** Tasks 1-7, 9, and 10 are implemented on this branch. Task 8 is partially complete for Plur and RuboCop only. Task 11 remains open because the full verification matrix, benchmark rerun, and performance follow-ups remain.

**Open follow-ups:**
- Re-run the RuboCop `--rspec-split` benchmark after the selector-grouping fix to confirm the example count now matches baseline.
- `LoadCache` needs a nil/wrong-shape guard after JSON decode. A cache file containing valid JSON `null` can leave the decoded `*Cache` nil and panic before the intended "ignore corrupt/unexpected cache" fallback.
- RuboCop's v2 debug measurements exceed the large-project runtime-cache overhead threshold, so the Phase B cache-format follow-up remains open.
- Task 8 still needs the RSpec-core, Mastodon, and Discourse measurements, or an explicit decision to narrow the QA matrix.
- Plur's own suite needs a cleaner benchmark harness before its `plur -n 8` result can be treated as a valid cache comparison.

> **Note on cherry-picking to main:** an earlier draft of this plan claimed Task 1's commits could be cherry-picked back onto `main` as a standalone PR. That premise was wrong — the source files (`runtime_cache.go`, `runtime_run_kind.go`, `rspec_line_splitter.go`) only exist on `rspec-split-specs`, and even `runtime_tracker.go` on `main` is the v1 `map[string]float64` shape that pre-dates the v2 cache work. The package re-org ships as part of `rspec-split-specs`.

**Tech Stack:** Go (existing toolchain), structured `log/slog` via `logger.Logger`, RSpec JSON rows formatter (unchanged), backspin for doctor integration tests.

---

## File Structure

After this plan completes, the runtime concerns live together:

```
internal/testruntime/
  cache.go           — Cache shape, load/save, source freshness, and cache query helpers
  tracker.go         — write-time logic, shared-example attribution
  run_kind.go        — RunKind enum and aggregate/partial classification
  splitter.go        — SplitDecision and bin-packing Cache.SplitFile method
  cache_test.go      — schema, load/save, freshness
  splitter_test.go   — SplitFile bin-packing
  tracker_test.go    — aggregate vs partial runs, shared-example attribution
```

Top-level `runner.go` and `cmd_doctor.go` stay where they are; only their imports and call sites change.

Boundaries:
- `internal/testruntime` owns the cache shape, the splitter logic, and the write-time attribution rules.
- `runner.go` is the only consumer of `SplitDecision` results; the grouper does not need to know splitting happened.
- `cmd_doctor.go` reads the cache once for display stats — it does not import the splitter.

---

## Sequencing Notes

Tasks 1.1 through 1.5 are commit-isolated to stay small and reviewable. Once Task 1 lands, the rest of the plan operates on the new package. Tasks 2–4 are the bulk of the schema and logic work. Tasks 5–7 wire the change through and amend the user-visible surfaces. Tasks 8–11 verify, measure, simplify, and finalize.

Order matters: **build always green** — every commit must pass `bin/rake build`. Don't rename types in commit 1 if commit 2 still references the old names.

---

## Task 1: Package re-organization

**Goal:** Move all runtime-tracking and splitter source into `internal/testruntime/` in a sequence of small, reviewable commits.

**Files affected:** `runtime_cache.go`, `runtime_tracker.go`, `runtime_run_kind.go`, `rspec_line_splitter.go`, all their `_test.go` siblings, plus the import sites in `runner.go`, `cmd_doctor.go`, `main.go`, etc.

### Task 1.1: Move the runtime cache file

- [x] `git mv runtime_cache.go internal/testruntime/cache.go`
- [x] `git mv runtime_cache_test.go internal/testruntime/cache_test.go`
- [x] Change `package main` → `package testruntime` at the top of both files.
- [x] Update every import site to add `"github.com/rsanheim/plur/internal/testruntime"` and qualify references (`testruntime.RuntimeCache`, `testruntime.LoadRuntimeCache`, etc.).
- [x] Do NOT rename types yet — that's Task 1.5.
- [x] Verify: `bin/rake build && bin/rake test:go`
- [x] Commit: `Move runtime cache into internal/testruntime`

### Task 1.2: Move the tracker

- [x] Same pattern: `git mv runtime_tracker.go internal/testruntime/tracker.go` and its test.
- [x] `package main` → `package testruntime`. Import-site updates.
- [x] Verify: `bin/rake build && bin/rake test:go`
- [x] Commit: `Move runtime tracker into internal/testruntime`

### Task 1.3: Move run_kind

- [x] `git mv runtime_run_kind.go internal/testruntime/run_kind.go`. No `_test.go` to move (per the existing file list).
- [x] Same package + import surgery.
- [x] Verify: `bin/rake build && bin/rake test:go`
- [x] Commit: `Move RunKind into internal/testruntime`

### Task 1.4: Move the splitter

- [x] `git mv rspec_line_splitter.go internal/testruntime/splitter.go` and its test.
- [x] Same package + import surgery. `SplitFile` is still a free function at this point — keep it as-is.
- [x] Verify: `bin/rake build && bin/rake test:go`
- [x] Commit: `Move RSpec line splitter into internal/testruntime`

### Task 1.5: Rename types to drop the redundant `Runtime` prefix

Now that the types live in `package testruntime`, the `Runtime` prefix is stuttery (`testruntime.RuntimeCache`). Rename:

```
RuntimeCache       → Cache
RuntimeCacheMeta   → CacheMeta
RuntimeCacheRun    → CacheRun
LoadRuntimeCache   → LoadCache
SaveRuntimeCache   → SaveCache
NewRuntimeCache    → NewCache
RuntimeCacheSchemaVersion → SchemaVersion
```

Keep these as-is (no redundancy):

```
RunKind, RunKindAggregate, RunKindPartial — fine
FileEntry, ExampleEntry — fine
SourceFreshness — fine
SplitDecision, SplitFile — fine
```

- [x] Rename in the source files.
- [x] Update every call site (use `gopls` rename or sed-then-build-loop).
- [x] Verify: `bin/rake build && bin/rake test:go && bin/rake test`
- [x] Commit: `Drop redundant Runtime prefix in runtime package`

**Task 1 success criteria:**
- Build is green after every single commit.
- No type renames happen before the moves; no logic changes happen at all in Task 1.

---

## Task 2: Trim `ExampleEntry` schema

**Goal:** Drop `Status` and `ScopedID` from `ExampleEntry`. Old caches with these fields still load (extra JSON fields are ignored).

**Files:** `internal/testruntime/cache.go`, `internal/testruntime/cache_test.go`

### Requirements

- `ExampleEntry` keeps `LineNumber`, `LocationRerunArgument`, `RuntimeSeconds`.
- Removed: `Status`, `ScopedID`.
- Backward compatibility: loading a cache file that *does* contain `status` and `scoped_id` keys must succeed silently. `encoding/json` already ignores unknown fields, so this is the default behavior — verify with a test.

### Steps

- [x] Update the struct definition in `cache.go`.
- [x] Remove any code in `tracker.go` that populates the dropped fields.
- [x] Add a test that writes an old-shape cache JSON literal to disk, calls `LoadCache`, and asserts the load returns a valid `Cache` without error.
- [x] Update existing tests that asserted on `Status` / `ScopedID` — assert on the remaining three fields instead.
- [x] Verify:

```bash
go test -mod=mod ./internal/testruntime/ -run TestCache -v
```

- [x] Commit: `Trim Status and ScopedID from ExampleEntry`

### Success criteria
- All cache tests pass.
- Loading a JSON file with `status`/`scoped_id` returns a valid `Cache`.
- Save round-trip produces JSON without the removed keys.

---

## Task 3: Shared-example attribution via `location_rerun_argument`

**Goal:** At write time, always use `location_rerun_argument` to derive the owning file. Strip the trailing `:line` portion to get the file path. Fall back to `file_path` only when `location_rerun_argument` is empty or has no `:`.

**Files:** `internal/testruntime/tracker.go`, `internal/testruntime/tracker_test.go`

### Requirements

- The owning file for a recorded example is derived as: take `location_rerun_argument`, find the last `:`, take everything before it. If `location_rerun_argument` is empty or contains no `:`, use `file_path`.
- Normalize the owning file the same way `file_path` is normalized today (strip leading `./` etc., per existing parser code in `framework/rspec/parser.go`).
- Do **not** create a `FileEntry` for the support file when a shared example fires unless the tracker also observes the support file as an aggregate run's primary `file_path`. (In practice: support files never own examples and never get added to `Cache.Files` via shared-example paths.)
- The cache's example record still stores the full `location_rerun_argument` — only the *attribution* changes.

### Tricky bit — parsing

```go
// owningFile derives the owning project-relative spec file for an example.
// Always prefers location_rerun_argument since it's RSpec's canonical
// rerunnable target, including for shared examples.
func owningFile(filePath, locationRerunArgument string) string {
    if i := strings.LastIndex(locationRerunArgument, ":"); i > 0 {
        return normalize(locationRerunArgument[:i])
    }
    return normalize(filePath)
}
```

Use existing `normalize` (or whatever the parser uses today — find it in `framework/rspec/parser.go`).

### Steps

- [x] Find the current write-time site in `tracker.go` where `examples` are bucketed into files. There is likely a loop that already uses `file_path`. Replace with `owningFile(file_path, location_rerun_argument)`.
- [x] Add the `owningFile` helper above (private to the package).
- [x] Tests cover three cases:
  - **Plain example:** `file_path = spec/foo_spec.rb`, `location_rerun_argument = ./spec/foo_spec.rb:42` → owner is `spec/foo_spec.rb`.
  - **Shared example:** `file_path = spec/support/shared_examples/x.rb`, `location_rerun_argument = ./spec/models/user_spec.rb:42` → owner is `spec/models/user_spec.rb`.
  - **Missing rerun arg:** `location_rerun_argument = ""` → owner is `file_path`.
- [x] Verify:

```bash
go test -mod=mod ./internal/testruntime/ -run TestTracker -v
```

- [x] Commit: `Attribute shared examples via location_rerun_argument`

### Success criteria
- All tracker tests pass.
- A shared example whose `location_rerun_argument` points to a real spec file ends up under that spec file's `Examples` map in the cache, not under the support file.
- Plain examples are unchanged.

---

## Task 4: Bin-packing `SplitFile` as a method on `*Cache`

**Goal:** Move splitting logic onto the cache type. Replace round-robin with longest-processing-time bin-packing using cached per-example runtimes. Return a `map[string]float64`.

**Files:** `internal/testruntime/cache.go` (or `splitter.go` if it's cleaner to keep it separate; decide once you see the size), `internal/testruntime/splitter_test.go`

**Status:** Implemented in `internal/testruntime/splitter.go`. The follow-up selector-grouping correction is also implemented: the splitter keeps cache identity by `example.id`, but bin-packs temporary scheduling units grouped by RSpec rerunnable selector so one `file:line` cannot be emitted into multiple split targets.

**Corrective design:** Keep storing examples by `example.id`; that is the right identity for cache merge/update behavior. At split time, derive temporary scheduling units by grouping existing `ExampleEntry` records by rerunnable selector. Prefer `ExampleEntry.LocationRerunArgument` when present, because it is RSpec's canonical rerun target and already handles shared examples. Fall back to `filePath:LineNumber` when the rerun argument is empty. Sum `RuntimeSeconds` across all examples in each selector group, then bin-pack those selector groups. Do not add new cache fields, JSON schema, or persisted types for this.

### Requirements

API:

```go
// SplitDecision maps focused-target spec args (e.g. "spec/foo_spec.rb:12:38")
// to their bin-packed per-target runtime weight in seconds. When the input
// file is not a split candidate, SplitDecision contains exactly one entry:
// the original file path → the file's recorded RuntimeSeconds.
type SplitDecision map[string]float64

func (c *Cache) SplitFile(filePath string, workerCount int, targetPerWorkerRuntime float64) SplitDecision
```

No-split conditions (return single-entry map `{filePath: file.RuntimeSeconds}`):
- `workerCount <= 1`
- `targetPerWorkerRuntime <= 0`
- `c.Files[filePath]` is nil (caller shouldn't call us on unknown files, but guard anyway)
- `c.Files[filePath].RuntimeSeconds <= targetPerWorkerRuntime`
- `len(c.Files[filePath].Examples) < 2`

When splitting applies:
- Build scheduling units from the already-cached examples. The cache key is `example.id`; the scheduling key is the RSpec rerunnable selector.
- Selector derivation:
  - Prefer `ExampleEntry.LocationRerunArgument`, with a leading `./` stripped for consistency.
  - If it is empty, use `filePath:LineNumber`.
  - Ignore examples with no usable selector or non-positive line fallback.
- For each example with `RuntimeSeconds <= 0`, treat it as the file's mean per-example runtime (`file.RuntimeSeconds / len(examples)`). Do not mutate the cached `ExampleEntry`.
- Group examples by selector and sum their effective runtimes. Multiple `example.id` values can contribute to one selector.
- `chunks := min(workerCount, len(selectorGroups))`
- Sort selector groups descending by effective runtime; tiebreak ascending by selector string.
- Initialize `chunks` empty bins; iterate selector groups placing each into the bin with the smallest current sum. Tiebreak by smallest bin index for determinism.
- For each bin, sort its selectors ascending and emit a target string. For selectors belonging to the same file, this remains `filePath:line:line:...`.
- Return `{target1: bin1_sum, target2: bin2_sum, ...}`.

### Pseudo-code

```
func (c *Cache) SplitFile(path, workers, budget) SplitDecision:
    file := c.Files[path]
    if no-split conditions met:
        return {path: file.RuntimeSeconds}

    groups := map selector -> summed effective runtime
    for each exampleID, ex in file.Examples:
        selector := strings.TrimPrefix(ex.LocationRerunArgument, "./")
        if selector == "":
            selector = fmt.Sprintf("%s:%d", path, ex.LineNumber)
        if selector unusable:
            continue
        runtime := ex.RuntimeSeconds
        if runtime <= 0:
            runtime = file.RuntimeSeconds / len(file.Examples)
        groups[selector] += runtime

    units := sorted list of (selector, summedRuntime)
             descending by summedRuntime, tiebreak ascending by selector

    chunks := min(workers, len(units))
    bins := chunks empty (selectors []string, sum float64)
    for each unit:
        b := index of bin with smallest sum (tiebreak: smallest index)
        bins[b].selectors.append(unit.selector)
        bins[b].sum += unit.summedRuntime

    decision := empty map
    for each bin:
        sort bin.selectors ascending
        target := collapse selectors for this file into file:line:line form
        decision[target] = bin.sum
    return decision
```

### Tests

Table-driven, in `splitter_test.go`:

- **Even runtimes, 4 selectors, 2 workers:** two targets, balanced sums.
- **One heavy example dominates:** the heavy one ends up in its own bin (LPT property). Sum disparity is intentional and asserted.
- **Duplicate-line examples:** two or more `example.id` entries with the same `LocationRerunArgument` produce exactly one emitted selector, with summed runtime. The same line must never appear in two targets.
- **Shared examples:** examples whose `FilePath` points at a support file but whose `LocationRerunArgument` points at the owning spec are grouped under the owning spec selector.
- **Missing runtimes:** examples with `RuntimeSeconds == 0` use the mean fallback; total summed runtime across bins ≈ file runtime within float tolerance.
- **More workers than selectors:** number of bins equals number of runnable selectors, not workers.
- **No-split passthrough:** budget exceeds file runtime → single-entry map `{path: file_runtime}`.
- **Determinism:** call `SplitFile` 100 times with the same cache; result map content is identical each time. (Map iteration order is not checked — only content.)
- **Stable target strings:** target string for a given bin uses ascending line order, regardless of input order.

### Steps

- [x] Delete the old free-function `SplitFile` in `splitter.go` (after moving the bin-packing logic to the method).
- [x] Define `SplitDecision` typedef next to `Cache`.
- [x] Implement the method per the pseudo-code.
- [x] Write the test cases above. Use testify `require`/`assert` per project conventions.
- [x] Verify:

```bash
go test -mod=mod ./internal/testruntime/ -run TestSplitFile -v
```

- [x] Commit: `Replace round-robin splitter with bin-packing method on Cache`

### Success criteria
- The old free function `SplitFile` is gone.
- A file with one heavy example and many light ones produces a clearly uneven `SplitDecision` (heaviest bin > average bin).
- Determinism test passes 100 runs.

---

## Task 5: Wire `SplitDecision` into the runner / grouper

**Goal:** Replace the existing call site that builds `exampleLines []int` and consumes the old `SplitDecision` struct with a direct `cache.SplitFile(...)` call returning the map.

**Files:** `runner.go`, `runner_rspec_split_test.go`

**Status:** Implemented. This wiring depends on the Task 4 duplicate-line follow-up before `--rspec-split` is safe for large real suites.

### Requirements

- Wherever the old code produced `SplitDecision{Targets, Chunks, ChunkRuntimeSeconds}`, replace with `decision := cache.SplitFile(file, workers, budget)`.
- Merge `decision` entries directly into the file-runtimes map passed to `GroupSpecFilesByRuntime`. When `len(decision) == 1` and its sole key equals the input `file`, this is a no-op (no split happened); when `len(decision) > 1`, the original file's entry is replaced by the per-target entries.
- Debug log lines need to read from the new map. Existing log at [runner.go:182](../../runner.go) is:

```go
logger.Logger.Debug("rspec-split applied", "file", file, "chunks", decision.Chunks, "chunk_runtime", decision.ChunkRuntimeSeconds)
```

Update to:

```go
logger.Logger.Debug("rspec-split applied", "file", file, "chunks", len(decision))
```

(The per-chunk runtimes are visible in the decision map; if you want them in the log, add `"runtimes", decision` — but slog will format the map sensibly.)

### Steps

- [x] Locate every reference to the old `SplitDecision` struct in `runner.go`. Update.
- [x] Adapt the existing `rspec-split skipped` log calls — most of those don't need to change since they fire before calling `SplitFile`.
- [x] Update existing tests in `runner_rspec_split_test.go` to match the new shape. Watch for assertions on `decision.Chunks` or `decision.ChunkRuntimeSeconds`.
- [x] Add a new test: build a cache with one heavy example (1s) and four light examples (0.1s each), call the split path, and assert the resulting `fileRuntimes` map has uneven values across the generated targets.
- [x] Verify:

```bash
go test -mod=mod . -run 'TestRunner|TestRspecSplit' -v
bin/rspec spec/integration/spec/runtime_tracking_spec.rb
```

- [x] Commit: `Use SplitDecision map and per-target runtimes for grouping`

### Success criteria
- All existing splitter integration tests pass without code changes (the public behavior is the same; only data flow changed).
- The new uneven-runtime test demonstrates per-target bin-packing weights propagate into grouping.

---

## Task 6: Debug logging inside Load/Save

**Goal:** Log structured debug lines on cache load and save, using the existing `logger.Logger`. No signature changes.

**Files:** `internal/testruntime/cache.go`

### Requirements

Exact log shape (per the spec doc):

```go
logger.Logger.Debug("runtimeCache loaded", "duration", dur, "path", path, "files", filesCount, "examples", examplesCount)
logger.Logger.Debug("runtimeCache saved",  "duration", dur, "path", path, "files", filesCount, "examples", examplesCount)
```

- `duration` is the raw `time.Duration` so slog formats it as `1.056ms`, `7.463ms`, `742µs` etc. — `.Milliseconds()` truncates sub-ms loads to a misleading `0`.
- `examplesCount` is the sum of `len(entry.Examples)` across all files in the cache.
- For `LoadCache`: log at the end, after the cache is built. If load fails and a fresh empty cache is returned, log with `files=0, examples=0` and the duration of the failed attempt — this is still useful diagnostic data.
- For `SaveCache`: log after the atomic rename succeeds.

### Steps

- [x] In `LoadCache`, capture `start := time.Now()` at the top. Before returning, compute counts and log.
- [x] In `SaveCache`, capture `start := time.Now()` at the top. After the rename succeeds (success branch only), compute counts and log.
- [x] No public signature changes.
- [x] If there's no existing logger import in `cache.go`, add it. Match how `runner.go` imports the logger.
- [x] Verify by running an integration spec with debug logging enabled and confirming both lines appear:

```bash
PLUR_DEBUG=1 bin/rspec spec/integration/spec/runtime_tracking_spec.rb 2>&1 | grep runtimeCache
```

- [x] Commit: `Log structured debug lines on runtime cache load/save`

### Success criteria
- Debug logs appear with the exact keys above.
- No existing tests break.

---

## Task 7: Amend `plur doctor` runtime data block

**Goal:** Replace the `(file exists)` line in [cmd_doctor.go:113](../../cmd_doctor.go) with a stats summary when the file exists. Keep `(file does not exist)` unchanged.

**Files:** `cmd_doctor.go`, `spec/integration/doctor/doctor_spec.rb`

### Requirements

When the runtime file exists, output (replacing the current `(file exists)` line):

```
                  192K / 28 files / 694 examples
```

- The indent matches the existing `(file exists)` indent.
- Size is human-readable. If a helper exists in the codebase, use it; otherwise use a small inline function:

```go
func humanSize(n int64) string {
    switch {
    case n >= 1<<20:
        return fmt.Sprintf("%.0fM", float64(n)/(1<<20))
    case n >= 1<<10:
        return fmt.Sprintf("%.0fK", float64(n)/(1<<10))
    }
    return fmt.Sprintf("%dB", n)
}
```

- `files` is the count of entries in `cache.Files`.
- `examples` is the sum of `len(entry.Examples)` across all files.
- If the cache file exists but can't be parsed, fall back to `(file exists)` so doctor doesn't crash. Log the parse error at debug level.

### Steps

- [x] In `cmd_doctor.go`, after the existing `os.Stat` check confirms the file exists, attempt to load the cache via `testruntime.LoadCache(runtimePath)`. Compute stats. Emit the new line.
- [x] Keep the `(file does not exist)` branch unchanged.
- [x] Use the cache loader from the runtime package — doctor does not need a separate read path.
- [x] Update the existing backspin snapshot in `spec/integration/doctor/doctor_spec.rb`.
- [x] Verify:

```bash
bin/rspec spec/integration/doctor/doctor_spec.rb
plur doctor   # eyeball check
```

- [x] Commit: `Show cache stats in plur doctor runtime data block`

### Success criteria
- `plur doctor` shows size / files / examples in place of `(file exists)` when a cache exists.
- The integration spec passes with an updated backspin snapshot.
- A missing cache still prints `(file does not exist)`.

---

## Task 8: Real-project QA pass

**Goal:** Measure load/save runtime impact across real projects and decide whether Phase B work is warranted.

**Files:** none for source. Results go in the PR description, or `docs/plans/2026-05-22-runtime-cache-measurement-results.md` if Phase B is triggered.

**Status:** Partially complete. Results and evidence are being recorded in [2026-05-22-runtime-cache-measurement-results.md](2026-05-22-runtime-cache-measurement-results.md). RuboCop exceeds the large-project cache-overhead threshold, so Phase B remains warranted. Plur and RuboCop were measured; RSpec-core, Mastodon, and Discourse are still open.

### Requirements

Measure on:
- [x] Plur (this repo) — attempted and documented; current outer-Plur benchmark is blocked by suite isolation issues.
- [x] RuboCop — full-suite benchmark captured for legacy v1, v2 runtime cache, and v2 `--rspec-split`.
- [ ] RSpec (rspec-core)
- [ ] Mastodon (subset Rob has been using)
- [ ] Discourse (subset Rob has been using)

For each project:

- [x] Warm the cache for Plur and RuboCop so the cache is populated and the format reflects the schema trim.
- [x] Capture runtime-cache load/save timings for RuboCop v2 debug and split-debug runs.
- [x] Note the on-disk cache file size and examples count for RuboCop.
- [x] Spot-check the RuboCop cache JSON: `"status"` and `"scoped_id"` are absent.
- [ ] Repeat the above for RSpec-core, Mastodon, and Discourse if they remain in scope.

Aggregate into:

| Project | examples | median load ms | median save ms | threshold met? |
|---|---|---|---|---|

Decision rule:
- Threshold: small (< 1K examples) > 25 ms, large (≥ 10K examples) > 50 ms, in-between use the small threshold.
- All under threshold → record the table in the PR description, close the size concern, no Phase B.
- Any over threshold → write `docs/plans/2026-05-22-runtime-cache-measurement-results.md` recording the data and proposing a Phase B plan per the menu in the spec.

### Success criteria
- [ ] Numbers table exists for all five projects, or the QA matrix is explicitly narrowed.
- [x] A clear decision (close / Phase B) is recorded for RuboCop: Phase B is still open.

---

## Task 9: Documentation

**Files:** `docs/usage.md`

### Requirements

Two small additions:

- A sentence in the runtime-tracking section noting that load and save emit structured `runtimeCache loaded` / `runtimeCache saved` debug log lines with `duration`, `path`, `files`, `examples` keys (so power users can grep for them).
- A line in the `plur doctor` section noting that the `Runtime Data:` block now shows file size, file count, and example count when the cache exists.

Do **not** document Phase B options as user-facing — gate on Task 8.

### Steps

- [x] Edit `docs/usage.md`. Keep additions minimal — one or two sentences each.
- [x] Verify the doc renders sensibly (mkdocs builds, if applicable).
- [x] Commit: `Document runtime cache debug log and doctor stats`

---

## Task 10: Simplification pass

**Goal:** Re-read every modified file with fresh eyes. Remove what shouldn't have stayed.

**Files:** every file touched by Tasks 1–9.

**Status:** Completed. Cleanup commits consolidated `RunKind` into `run_kind.go`, trimmed stale comments, and kept `SplitDecision` as the public splitter result typedef for readability at the single runner call site.

### Requirements

Read each file. For each, ask:

- **Is there a type I added that wraps a one-line shape?** If yes — consider deleting. `SplitDecision` typedef is a known suspect: if there's only one producer and one consumer, the `map[string]float64` inline may be clearer. Decide and document the choice.
- **Is there a helper that wraps a single function call?** If yes — inline it.
- **Is `RunKind` carrying its weight as a separate file/type?** It's currently in `run_kind.go`. Evaluate: does it have more than one consumer? Does the enum semantic matter outside `tracker.go`? If no, fold into `tracker.go` and delete `run_kind.go`.
- **Is there dead code from before this branch?** Specifically: round-robin chunking logic in `splitter.go`, references to `Status` / `ScopedID` in tests, any leftover top-level `runtime_*.go` files.
- **Are there tests that test the standard library / framework?** A test that proves `encoding/json` ignores extra fields is testing Go, not us — delete it. (The cache test for "old-shape loads silently" stays, because it documents the *cache's* contract.)
- **Are there comments that restate the code or carry temporal context?** Per project CLAUDE.md, strip them.

### Output

Either:
- One or more cleanup commits with descriptive messages (`Remove SplitDecision typedef in favor of inline map`, `Fold RunKind into tracker.go`, etc.); or
- A line in the PR description: "Reviewed for simplification — nothing to remove."

Completed output:
- [x] Reviewed modified files for stale comments and dead round-robin code.
- [x] Removed stale comments and consolidated `RunKind` in `internal/testruntime/run_kind.go`.
- [x] Decided to keep `SplitDecision` as a named map type.

Do not skip this task silently — make the decision explicit.

---

## Task 11: Final verification

**Files:** none modified.

**Status:** Not complete. Targeted checks have passed for the selector-grouping fix, but the full `bin/rake` verification matrix remains open.

Targeted checks run for the selector-grouping fix:
- [x] `go test -mod=mod ./internal/testruntime -run TestSplitFile_GroupsExamplesByRerunnableSelector -v`
- [x] `go test -mod=mod ./internal/testruntime -run TestSplitFile -v`
- [x] `go test -mod=mod ./internal/testruntime`
- [x] `go test -mod=mod . -run 'TestExpandRspecSplits|TestCache_ExampleLines|TestShouldExpandSplits'`
- [x] `PLUR_BINARY=$PWD/plur bin/rspec spec/integration/spec/runtime_tracking_spec.rb:319`
- [x] `PLUR_BINARY=$PWD/plur bin/rspec spec/integration/spec/runtime_tracking_spec.rb`
- [x] `bin/rake test:go`

- [x] `bin/rake test:go` — Go tests green.
- [ ] `bin/rake test` — Ruby integration tests green.
- [ ] `bin/rake standard:fix` — Ruby lint applied.
- [x] `git diff --check` — no trailing whitespace or conflict markers.
- [ ] If anything from Task 8 indicates a threshold breach, confirm the Phase B follow-up plan exists and is linked from the PR description.

### Success criteria
- All verification commands exit zero.
- PR description includes:
  - The Task 8 results table.
  - Either "close the cache size concern" or a link to the Phase B follow-up plan.
  - A note on what Task 10 simplified (or that it found nothing).

---

## Spec Coverage Map

| Spec section | Implementing tasks |
|---|---|
| Schema Changes (drop `status`, `scoped_id`) | Task 2 |
| Shared-Example Ownership | Task 3 |
| Runtime-Weighted Bin-Packing Splitter | Tasks 4, 5 |
| Instrumentation (debug logs) | Task 6 |
| `plur doctor` amendment | Task 7 |
| Real-Project QA | Task 8 |
| Phase B Options | Documented only; gated on Task 8 |
| Package Re-Organization | Task 1 (1.1–1.5) |
| Code-review pass | Task 10 |
| Final Verification | Task 11 |
