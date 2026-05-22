# Runtime Cache Measurement and Bin-Packing — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking. Pair with the spec at [docs/plans/2026-05-22-runtime-cache-measurement-and-bin-packing-plan.md](2026-05-22-runtime-cache-measurement-and-bin-packing-plan.md) — the spec defines the *what* and *why*; this document covers the *how*, sequencing, and verification.

**Goal:** Move the runtime files into `internal/testruntime/`, trim per-example dead-weight fields, attribute shared examples via `location_rerun_argument`, replace round-robin splitting with bin-packing as a method on `*Cache`, add load/save debug logging, amend `plur doctor`, and run real-project QA to decide whether further format work is needed.

**Architecture:** One branch on top of `rspec-split-specs`. Task 1 is a series of small move + rename commits to keep each step reviewable. Tasks 2–9 build on the new layout. Task 10 is a dedicated simplification pass. Task 11 verifies and finalizes.

> **Note on cherry-picking to main:** an earlier draft of this plan claimed Task 1's commits could be cherry-picked back onto `main` as a standalone PR. That premise was wrong — the source files (`runtime_cache.go`, `runtime_run_kind.go`, `rspec_line_splitter.go`) only exist on `rspec-split-specs`, and even `runtime_tracker.go` on `main` is the v1 `map[string]float64` shape that pre-dates the v2 cache work. The package re-org ships as part of `rspec-split-specs`.

**Tech Stack:** Go (existing toolchain), structured `log/slog` via `logger.Logger`, RSpec JSON rows formatter (unchanged), backspin for doctor integration tests.

---

## File Structure

After this plan completes, the runtime concerns live together:

```
internal/testruntime/
  cache.go           — RuntimeCache renamed to Cache; SplitFile method lives here
  tracker.go         — write-time logic, shared-example attribution
  run_kind.go        — RunKind enum (may fold into cache.go during Task 10)
  splitter.go        — bin-packing helpers if cache.go grows unwieldy; otherwise empty / removed
  cache_test.go      — schema, load/save, freshness, SplitFile bin-packing
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

- [ ] `git mv runtime_cache.go internal/testruntime/cache.go`
- [ ] `git mv runtime_cache_test.go internal/testruntime/cache_test.go`
- [ ] Change `package main` → `package testruntime` at the top of both files.
- [ ] Update every import site to add `"github.com/rsanheim/plur/internal/testruntime"` and qualify references (`testruntime.RuntimeCache`, `testruntime.LoadRuntimeCache`, etc.).
- [ ] Do NOT rename types yet — that's Task 1.5.
- [ ] Verify: `bin/rake build && bin/rake test:go`
- [ ] Commit: `Move runtime cache into internal/testruntime`

### Task 1.2: Move the tracker

- [ ] Same pattern: `git mv runtime_tracker.go internal/testruntime/tracker.go` and its test.
- [ ] `package main` → `package testruntime`. Import-site updates.
- [ ] Verify: `bin/rake build && bin/rake test:go`
- [ ] Commit: `Move runtime tracker into internal/testruntime`

### Task 1.3: Move run_kind

- [ ] `git mv runtime_run_kind.go internal/testruntime/run_kind.go`. No `_test.go` to move (per the existing file list).
- [ ] Same package + import surgery.
- [ ] Verify: `bin/rake build && bin/rake test:go`
- [ ] Commit: `Move RunKind into internal/testruntime`

### Task 1.4: Move the splitter

- [ ] `git mv rspec_line_splitter.go internal/testruntime/splitter.go` and its test.
- [ ] Same package + import surgery. `SplitFile` is still a free function at this point — keep it as-is.
- [ ] Verify: `bin/rake build && bin/rake test:go`
- [ ] Commit: `Move RSpec line splitter into internal/testruntime`

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

- [ ] Rename in the source files.
- [ ] Update every call site (use `gopls` rename or sed-then-build-loop).
- [ ] Verify: `bin/rake build && bin/rake test:go && bin/rake test`
- [ ] Commit: `Drop redundant Runtime prefix in runtime package`

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

- [ ] Update the struct definition in `cache.go`.
- [ ] Remove any code in `tracker.go` that populates the dropped fields.
- [ ] Add a test that writes an old-shape cache JSON literal to disk, calls `LoadCache`, and asserts the load returns a valid `Cache` without error.
- [ ] Update existing tests that asserted on `Status` / `ScopedID` — assert on the remaining three fields instead.
- [ ] Verify:

```bash
go test -mod=mod ./internal/testruntime/ -run TestCache -v
```

- [ ] Commit: `Trim Status and ScopedID from ExampleEntry`

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

- [ ] Find the current write-time site in `tracker.go` where `examples` are bucketed into files. There is likely a loop that already uses `file_path`. Replace with `owningFile(file_path, location_rerun_argument)`.
- [ ] Add the `owningFile` helper above (private to the package).
- [ ] Tests cover three cases:
  - **Plain example:** `file_path = spec/foo_spec.rb`, `location_rerun_argument = ./spec/foo_spec.rb:42` → owner is `spec/foo_spec.rb`.
  - **Shared example:** `file_path = spec/support/shared_examples/x.rb`, `location_rerun_argument = ./spec/models/user_spec.rb:42` → owner is `spec/models/user_spec.rb`.
  - **Missing rerun arg:** `location_rerun_argument = ""` → owner is `file_path`.
- [ ] Verify:

```bash
go test -mod=mod ./internal/testruntime/ -run TestTracker -v
```

- [ ] Commit: `Attribute shared examples via location_rerun_argument`

### Success criteria
- All tracker tests pass.
- A shared example whose `location_rerun_argument` points to a real spec file ends up under that spec file's `Examples` map in the cache, not under the support file.
- Plain examples are unchanged.

---

## Task 4: Bin-packing `SplitFile` as a method on `*Cache`

**Goal:** Move splitting logic onto the cache type. Replace round-robin with longest-processing-time bin-packing using cached per-example runtimes. Return a `map[string]float64`.

**Files:** `internal/testruntime/cache.go` (or `splitter.go` if it's cleaner to keep it separate; decide once you see the size), `internal/testruntime/splitter_test.go`

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
- `chunks := min(workerCount, len(examples))`
- For each example with `RuntimeSeconds <= 0`, treat it as the file's mean per-example runtime (`file.RuntimeSeconds / len(examples)`). Do not mutate the cached `ExampleEntry`.
- Sort examples descending by effective runtime; tiebreak ascending by line number.
- Initialize `chunks` empty bins; iterate examples placing each into the bin with the smallest current sum. Tiebreak by smallest bin index for determinism.
- For each bin, sort its example line numbers ascending and emit a target string `filePath:line:line:...`.
- Return `{target1: bin1_sum, target2: bin2_sum, ...}`.

### Pseudo-code

```
func (c *Cache) SplitFile(path, workers, budget) SplitDecision:
    file := c.Files[path]
    if no-split conditions met:
        return {path: file.RuntimeSeconds}

    examples := list of (line, effectiveRuntime) from file.Examples
                with fallback applied
    sort examples descending by effectiveRuntime, tiebreak ascending by line

    chunks := min(workers, len(examples))
    bins := chunks empty (lines []int, sum float64)
    for each example:
        b := index of bin with smallest sum (tiebreak: smallest index)
        bins[b].lines.append(example.line)
        bins[b].sum += example.effectiveRuntime

    decision := empty map
    for each bin:
        sort bin.lines ascending
        target := path + ":" + join(bin.lines, ":")
        decision[target] = bin.sum
    return decision
```

### Tests

Table-driven, in `splitter_test.go`:

- **Even runtimes, 4 examples, 2 workers:** two targets, balanced sums.
- **One heavy example dominates:** the heavy one ends up in its own bin (LPT property). Sum disparity is intentional and asserted.
- **Missing runtimes:** examples with `RuntimeSeconds == 0` use the mean fallback; total summed runtime across bins ≈ file runtime within float tolerance.
- **More workers than examples:** number of bins equals number of examples, not workers.
- **No-split passthrough:** budget exceeds file runtime → single-entry map `{path: file_runtime}`.
- **Determinism:** call `SplitFile` 100 times with the same cache; result map content is identical each time. (Map iteration order is not checked — only content.)
- **Stable target strings:** target string for a given bin uses ascending line order, regardless of input order.

### Steps

- [ ] Delete the old free-function `SplitFile` in `splitter.go` (after moving the bin-packing logic to the method).
- [ ] Define `SplitDecision` typedef next to `Cache`.
- [ ] Implement the method per the pseudo-code.
- [ ] Write the test cases above. Use testify `require`/`assert` per project conventions.
- [ ] Verify:

```bash
go test -mod=mod ./internal/testruntime/ -run TestSplitFile -v
```

- [ ] Commit: `Replace round-robin splitter with bin-packing method on Cache`

### Success criteria
- The old free function `SplitFile` is gone.
- A file with one heavy example and many light ones produces a clearly uneven `SplitDecision` (heaviest bin > average bin).
- Determinism test passes 100 runs.

---

## Task 5: Wire `SplitDecision` into the runner / grouper

**Goal:** Replace the existing call site that builds `exampleLines []int` and consumes the old `SplitDecision` struct with a direct `cache.SplitFile(...)` call returning the map.

**Files:** `runner.go`, `runner_rspec_split_test.go`

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

- [ ] Locate every reference to the old `SplitDecision` struct in `runner.go`. Update.
- [ ] Adapt the existing `rspec-split skipped` log calls — most of those don't need to change since they fire before calling `SplitFile`.
- [ ] Update existing tests in `runner_rspec_split_test.go` to match the new shape. Watch for assertions on `decision.Chunks` or `decision.ChunkRuntimeSeconds`.
- [ ] Add a new test: build a cache with one heavy example (1s) and four light examples (0.1s each), call the split path, and assert the resulting `fileRuntimes` map has uneven values across the generated targets.
- [ ] Verify:

```bash
go test -mod=mod . -run 'TestRunner|TestRspecSplit' -v
bin/rspec spec/integration/spec/runtime_tracking_spec.rb
```

- [ ] Commit: `Use SplitDecision map and per-target runtimes for grouping`

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

- [ ] In `LoadCache`, capture `start := time.Now()` at the top. Before returning, compute counts and log.
- [ ] In `SaveCache`, capture `start := time.Now()` at the top. After the rename succeeds (success branch only), compute counts and log.
- [ ] No public signature changes.
- [ ] If there's no existing logger import in `cache.go`, add it. Match how `runner.go` imports the logger.
- [ ] Verify by running an integration spec with debug logging enabled and confirming both lines appear:

```bash
PLUR_DEBUG=1 bin/rspec spec/integration/spec/runtime_tracking_spec.rb 2>&1 | grep runtimeCache
```

- [ ] Commit: `Log structured debug lines on runtime cache load/save`

### Success criteria
- Debug logs appear with the exact keys above.
- No existing tests break.

---

## Task 7: Amend `plur doctor` runtime data block

**Goal:** Replace the `(file exists)` line in [cmd_doctor.go:113](../../cmd_doctor.go) with a stats summary when the file exists. Keep `(file does not exist)` unchanged.

**Files:** `cmd_doctor.go`, `spec/integration/plur_doctor/doctor_spec.rb`

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

- [ ] In `cmd_doctor.go`, after the existing `os.Stat` check confirms the file exists, attempt to load the cache via `testruntime.LoadCache(runtimePath)`. Compute stats. Emit the new line.
- [ ] Keep the `(file does not exist)` branch unchanged.
- [ ] Use the cache loader from the runtime package — doctor does not need a separate read path.
- [ ] Update the existing backspin snapshot in `spec/integration/plur_doctor/doctor_spec.rb`. The snapshot file likely lives under `spec/integration/plur_doctor/backspin/` — re-record it with `BACKSPIN_RECORD=1 bin/rspec ...` per existing project conventions.
- [ ] Verify:

```bash
bin/rspec spec/integration/plur_doctor/doctor_spec.rb
plur doctor   # eyeball check
```

- [ ] Commit: `Show cache stats in plur doctor runtime data block`

### Success criteria
- `plur doctor` shows size / files / examples in place of `(file exists)` when a cache exists.
- The integration spec passes with an updated backspin snapshot.
- A missing cache still prints `(file does not exist)`.

---

## Task 8: Real-project QA pass

**Goal:** Measure load/save runtime impact across real projects and decide whether Phase B work is warranted.

**Files:** none for source. Results go in the PR description, or `docs/plans/2026-05-22-runtime-cache-measurement-results.md` if Phase B is triggered.

### Requirements

Measure on:
- Plur (this repo)
- RuboCop
- RSpec (rspec-core)
- Mastodon (subset Rob has been using)
- Discourse (subset Rob has been using)

For each project:

- [ ] Warm the cache: run `plur -n 8` once with the new binary so the cache is populated and the format reflects the schema trim.
- [ ] Take three measurement runs with `PLUR_DEBUG=1 plur -n 8 2>&1 | grep runtimeCache`. Record the `duration` values for both load and save.
- [ ] Note the on-disk cache file size and the `examples` count from the doctor output.
- [ ] Spot-check the cache JSON: grep for `"status"` and `"scoped_id"` — should be absent. Sanity-check that `examples` map entries have exactly the three expected fields.

Aggregate into:

| Project | examples | median load ms | median save ms | threshold met? |
|---|---|---|---|---|

Decision rule:
- Threshold: small (< 1K examples) > 25 ms, large (≥ 10K examples) > 50 ms, in-between use the small threshold.
- All under threshold → record the table in the PR description, close the size concern, no Phase B.
- Any over threshold → write `docs/plans/2026-05-22-runtime-cache-measurement-results.md` recording the data and proposing a Phase B plan per the menu in the spec.

### Success criteria
- Numbers table exists for all five projects.
- A clear decision (close / Phase B) is recorded.

---

## Task 9: Documentation

**Files:** `docs/usage.md`

### Requirements

Two small additions:

- A sentence in the runtime-tracking section noting that load and save emit structured `runtimeCache loaded` / `runtimeCache saved` debug log lines with `duration`, `path`, `files`, `examples` keys (so power users can grep for them).
- A line in the `plur doctor` section noting that the `Runtime Data:` block now shows file size, file count, and example count when the cache exists.

Do **not** document Phase B options as user-facing — gate on Task 8.

### Steps

- [ ] Edit `docs/usage.md`. Keep additions minimal — one or two sentences each.
- [ ] Verify the doc renders sensibly (mkdocs builds, if applicable).
- [ ] Commit: `Document runtime cache debug log and doctor stats`

---

## Task 10: Simplification pass

**Goal:** Re-read every modified file with fresh eyes. Remove what shouldn't have stayed.

**Files:** every file touched by Tasks 1–9.

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

Do not skip this task silently — make the decision explicit.

---

## Task 11: Final verification

**Files:** none modified.

- [ ] `bin/rake test:go` — Go tests green.
- [ ] `bin/rake test` — Ruby integration tests green.
- [ ] `bin/rake standard:fix` — Ruby lint applied.
- [ ] `git diff --check` — no trailing whitespace or conflict markers.
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
