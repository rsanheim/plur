# Runtime Cache Measurement and Bin-Packing Plan

> **For implementation workers:** Implement this task-by-task. Keep each task small, verify before moving on, and commit after each completed task.

**Goal:** Close the [Current Gap from Branch Review](2026-05-12-rspec-split-specs-experimental-plan.md#current-gap-from-branch-review) by building the runtime-weighted bin-packing splitter, trim the obvious dead-weight from the v2 runtime cache per-example schema, and add measurement so we can decide whether further format work is justified by real performance data.

**Architecture:** Phase A in one branch — move the runtime files into `internal/runtime/`, trim derivable per-example fields, attribute shared examples at write time, implement runtime-weighted splitter as a method on the cache, add load/save debug logging, amend the existing `plur doctor` output. Real-project QA from Task 8.1 of the parent plan supplies the data. Phase B is gated on Phase A measurements meeting the decision criteria.

**Tech Stack:** Go CLI, `encoding/json`, structured `log/slog` (existing `logger.Logger`), existing Plur runtime tracker.

---

## Position

The v2 runtime cache works and the experimental `--rspec-split` flag ships line-based round-robin splitting today. Two known gaps remain:

1. The splitter does not use per-example runtimes for chunk balancing, even though the cache records them. The plan's [Current Gap section](2026-05-12-rspec-split-specs-experimental-plan.md#current-gap-from-branch-review) calls this out as the intended follow-up.
2. The v2 cache persists fields the splitter never reads (`status`, `scoped_id`). Dead weight per record.

The runtime cache is also unmeasured. We assume it could become a performance drag on large suites but have no numbers. Before picking a heavier format change (gzip, slow-files-only caching, binary format, SQLite), measure what is actually slow on real projects. Pre-1.0 means we can change the format freely later — but we should change it because data says so, not because the JSON file looks big.

## Decision Criteria

Run combined `LoadRuntimeCache` + `SaveRuntimeCache` time per `plur` invocation, observed on real-project QA after warmup.

| Project size | Threshold | Action if exceeded |
|---|---|---|
| Small (< 1K examples cached) | > 25 ms | Pause, investigate, plan Phase B |
| Large (≥ 10K examples cached) | > 50 ms | Pause, investigate, plan Phase B |
| In between | > 25 ms (use stricter small-project threshold) | Pause, investigate, plan Phase B |

If all observed projects sit under the threshold, declare the cache size concern closed and move on. File size on disk is not the primary metric; runtime impact per invocation is.

Real-project anchors expected during QA:

| Project | Est. spec files | Est. examples cached | Treated as |
|---|---|---|---|
| Plur | ~40 | ~400 | small |
| RuboCop | ~500 | ~10K | borderline (use small threshold) |
| RSpec core | ~200 | ~3K | small |
| Mastodon subset | ~1K | ~15K | large |
| Discourse subset | ~2K | ~30–40K | large |

## Schema Changes

The per-example record loses two fields:

- `scoped_id` removed — already encoded in the example.id key suffix (`./spec/foo_spec.rb[1:2:1]` → `1:2:1`).
- `status` removed — never read by splitter or grouper.

The per-example record keeps:

- `line_number` — splitter target generation
- `location_rerun_argument` — diagnostics and shared-example safety net (also the authoritative source for write-time ownership)
- `runtime_seconds` — bin-packing weight

Result per example:

```json
"./spec/slow_spec.rb[1:1]": {
  "line_number": 12,
  "location_rerun_argument": "./spec/slow_spec.rb:12",
  "runtime_seconds": 0.40
}
```

Smaller per record than today's five-field shape — exact savings to be quantified from real-project QA. No backward-compatibility shim. Old-shape v2 caches with extra `status` / `scoped_id` fields load fine (extra JSON fields are ignored by `json.Unmarshal`); the fields just stop being written on the next save.

## Shared-Example Ownership at Write Time

`location_rerun_argument` is RSpec's canonical rerunnable target — for non-shared examples it's `file_path:line_number`; for shared examples it's the owning spec file. We use it directly as the source of ownership.

Rule: **always use `location_rerun_argument` to determine the owning file.** Strip the trailing `:line` portion to get the file. Fall back to `file_path` only if `location_rerun_argument` is empty or malformed (shouldn't happen with our formatter, but the fallback keeps us safe).

That's the whole rule. No conditional "if it differs from file_path." The cache's `examples` map for the owning file gains the entry. Support files do not get file-level entries unless RSpec also reports them as the rerunnable owner for an aggregate run.

`location_rerun_argument` is preserved per example so we can detect surprises during QA and revert attribution decisions if a real suite exposes a gap.

## Runtime-Weighted Bin-Packing Splitter

Move `SplitFile` from a free function in `rspec_line_splitter.go` to a method on `*RuntimeCache` (after the package re-org, `*runtime.Cache`). The cache already carries everything the splitter needs (`Files[path].RuntimeSeconds`, `Files[path].Examples` with `LineNumber` and `RuntimeSeconds` per entry). No new types — the splitter reads `*ExampleEntry` directly.

```go
// SplitDecision maps focused-target spec args to their bin-packed runtime
// weights. When the input file is not a split candidate, SplitDecision
// contains one entry: the original file path mapped to its file-level runtime.
type SplitDecision map[string]float64

func (c *RuntimeCache) SplitFile(filePath string, workerCount int, targetPerWorkerRuntime float64) SplitDecision
```

`SplitDecision` becomes a typedef alias over `map[string]float64`. Same data shape as `FileRuntimes()`. Chunk count is `len(decision)`. No `SplitDecision` struct, no `ExampleUnit` projection, no `SplitTarget` wrapper.

Algorithm:

1. Gate: return `{filePath: existing runtime}` (no-split) if `workerCount <= 1`, `targetPerWorkerRuntime <= 0`, the file's `runtime_seconds <= targetPerWorkerRuntime`, or the file has fewer than two examples.
2. `chunks := min(workerCount, len(examples))`.
3. For any example with `runtime_seconds <= 0`, treat it as `file_runtime / len(examples)` (mean fallback).
4. Sort examples descending by runtime; tiebreak ascending by line number.
5. Initialize `chunks` empty bins, each tracking summed runtime.
6. For each example, place it into the bin with the smallest current sum. Tiebreak by smallest bin index.
7. Sort each bin's contents ascending by line number before emitting the target string (`file:line:line:...`).
8. Return `SplitDecision{target: bin_sum, ...}`.

Determinism: same cache state → same `SplitDecision` map contents on every call. Map iteration order in Go is randomized, but callers feeding into `GroupSpecFilesByRuntime` already sort by runtime, so iteration order does not affect the worker grouping output.

## Instrumentation

`internal/runtime/cache.go` — log inside `LoadRuntimeCache` and `SaveRuntimeCache`. No signature changes; no caller plumbing.

```go
logger.Logger.Debug("runtimeCache loaded", "duration_ms", ms, "path", path, "files", filesCount, "examples", examplesCount)
logger.Logger.Debug("runtimeCache saved",  "duration_ms", ms, "path", path, "files", filesCount, "examples", examplesCount)
```

Matches the existing `logger.Logger.Debug(msg, key, value, ...)` slog style already used in [runner.go:122](../../runner.go).

`plur doctor` (`cmd_doctor.go`) — amend the existing block, do not duplicate. Today it prints:

```
Cache Directory:  /Users/rsanheim/.plur/cache
Runtime Data:     /Users/rsanheim/.plur/runtime/2053983d.json
                  (file exists)
```

After the change, when the file exists, replace `(file exists)` with the stats line:

```
Cache Directory:  /Users/rsanheim/.plur/cache
Runtime Data:     /Users/rsanheim/.plur/runtime/2053983d.json
                  192K / 28 files / 694 examples
```

When the file does not exist, keep `(file does not exist)` as today. Doctor does not load+save the cache itself, so it cannot report load/save ms — debug logs already cover those. Doctor reports the static facts (size, file count, example count) directly from a one-shot read.

## Real-Project QA

Reuses Task 8.1 of the parent plan. For each project, after warming the cache:

1. Capture cache file size and example count.
2. Run `plur -n 8` three times with debug logging enabled; record the `runtimeCache loaded` and `runtimeCache saved` durations.
3. Tabulate: project / examples cached / median load ms / median save ms / threshold met?
4. Spot-check the on-disk JSON shape against a real Discourse cache to confirm the trimmed schema looks right in the wild (no fields that QA reveals we shouldn't have dropped).

Projects in scope (from parent plan): Plur, RuboCop, RSpec, Mastodon subset, Discourse subset.

Output: a short numbers table in the PR description, or `docs/plans/2026-05-22-runtime-cache-measurement-results.md` created during QA if results warrant Phase B. The table is what triggers Phase B if any row exceeds its threshold.

## Phase B Options (Out of Scope for This Plan)

Documented only so we know what is on the table when Phase A says we need it. The right choice depends on which dimension dominates in the profile:

- **Parse-dominated (json.Unmarshal allocations):** compact JSON (`json.Marshal`, drop `MarshalIndent`) for a small win; for a bigger win, swap encoder (`json-iterator/go` or `bytedance/sonic`) or move to a binary format. Gzip is *not* a parse-speed win — decompression adds CPU on top of unmarshalling.
- **Bytes-on-disk dominated:** gzip the JSON file in place (`.json.gz`, still `zcat | jq`-able). Trades a small CPU cost for ~80% disk savings. Use this when the size complaint is about disk footprint, not latency.
- **Examples-dominated:** cap example tracking to files with `runtime_seconds > N` or top-N slowest. Small files never get split; they never need example entries. Largest absolute byte-and-parse savings on big suites.
- **Last resort:** binary format (gob/msgpack/cbor) or SQLite. Reconsider only if all of the above are insufficient.

No Phase B work happens until Phase A measurements are in.

## Package Re-Organization

Top-level `runtime_*.go` and `rspec_line_splitter.go` move into `internal/runtime/`:

```
internal/runtime/
  cache.go         (was runtime_cache.go; possibly absorbs run_kind.go — see code-review task)
  tracker.go       (was runtime_tracker.go)
  run_kind.go      (was runtime_run_kind.go — keep or fold during code review)
  splitter.go      (was rspec_line_splitter.go; SplitFile becomes method on *Cache)
  cache_test.go
  tracker_test.go
  splitter_test.go
```

**Cherry-pick strategy:** the pure file-rename commits (no content changes other than package declaration and imports) live in their own commits early in the branch so they can be cherry-picked back onto a fresh branch off `main` independently of the bin-packing / schema / shared-example work. Each pure-move commit must contain only:

- `git mv` of the file into `internal/runtime/`
- `package runtime` declaration at the top
- import-path updates in callers
- nothing else — no field renames, no logic changes, no test changes

That makes Task 1 cherry-pickable for an early-merge if reviewers want the re-org without the rest of the plan.

Note: the type names also change to drop the redundant `Runtime` prefix once they are in `package runtime`. That rename is its own commit, *after* the pure move, so the pure moves stay clean.

## File Responsibilities

Modify (after the move):

- `internal/runtime/cache.go`: drop `Status` and `ScopedID` from `ExampleEntry`. Add the `SplitFile` method and `SplitDecision` map typedef. Add debug logging inside `LoadRuntimeCache` and `SaveRuntimeCache`.
- `internal/runtime/tracker.go`: ownership via `location_rerun_argument`, always.
- `internal/runtime/splitter.go`: deleted in favor of the method on `*Cache`, OR retained as a small file holding `SplitDecision`'s typedef and split helpers if the cache file grows too long. Decide during code review.
- `internal/runtime/*_test.go`: updated tests for the trimmed schema, the new ownership rule, and the bin-packing algorithm.
- `runner.go`: consume `SplitDecision` (map) directly; merge into the file-runtimes view passed to `GroupSpecFilesByRuntime`. No `ExampleUnit` construction in the caller.
- `cmd_doctor.go`: amend the runtime data block per the example above.
- `spec/integration/spec/runtime_tracking_spec.rb`: cover shared-example attribution end-to-end; assert old-schema caches are still loadable.
- `spec/integration/plur_doctor/doctor_spec.rb`: assert doctor output includes the amended stats line.
- `docs/usage.md`: brief mention of the load/save debug log keys and amended doctor block.

Do not modify:

- `framework/rspec/formatter.rb` — already emits everything we need.
- The v2 cache top-level structure (`meta`, `run`, `files`) — only `ExampleEntry` changes.

Not creating:

- No `runtime_cache_bench_test.go`. Real-project QA gives us the numbers we need. If we later want a regression guard, add it then.
- No `ExampleUnit`, no `SplitTarget`, no `RuntimeStats` return wrapper.

---

## Task 1: Move runtime files into `internal/runtime/` (pure moves)

**Files:** `runtime_cache.go`, `runtime_tracker.go`, `runtime_run_kind.go`, `rspec_line_splitter.go` and their `_test.go` siblings.

Each move is its own commit so any subset can be cherry-picked back onto `main`.

- [ ] Commit 1: `git mv runtime_cache.go internal/runtime/cache.go` and its test. Change `package main` → `package runtime`. Update import paths in callers. No other changes.
- [ ] Commit 2: same for `runtime_tracker.go` → `internal/runtime/tracker.go`.
- [ ] Commit 3: same for `runtime_run_kind.go` → `internal/runtime/run_kind.go`.
- [ ] Commit 4: same for `rspec_line_splitter.go` → `internal/runtime/splitter.go`.
- [ ] Commit 5: rename types to drop the redundant `Runtime` prefix now that they're in `package runtime` (`RuntimeCache` → `Cache`, `LoadRuntimeCache` → `LoadCache`, `SaveRuntimeCache` → `SaveCache`, `RuntimeCacheMeta` → `CacheMeta`, etc.). Update all references.
- [ ] Verify after each commit:

```bash
bin/rake build && bin/rake test:go
```

## Task 2: Trim `ExampleEntry` schema

**Files:** `internal/runtime/cache.go`, `internal/runtime/cache_test.go`

- [ ] Remove `ScopedID` and `Status` fields from `ExampleEntry`.
- [ ] Remove related test assertions.
- [ ] Add a test that an old-shape cache (with `scoped_id` / `status` fields present) loads successfully — extra JSON fields should be ignored, not fail parsing.
- [ ] Verify:

```bash
go test -mod=mod ./internal/runtime/ -run TestCache
```

## Task 3: Shared-example attribution via `location_rerun_argument`

**Files:** `internal/runtime/tracker.go`, `internal/runtime/tracker_test.go`

- [ ] At write time, always derive the owning file from `location_rerun_argument` by stripping the trailing `:line` portion.
- [ ] Fall back to `file_path` only when `location_rerun_argument` is empty or malformed.
- [ ] Attribute the example to the owning file's `Examples` map; do not create a support-file entry unless RSpec reports it as the rerunnable owner.
- [ ] Tests: ordinary examples (unchanged behavior); shared examples where `location_rerun_argument` points to a real owning file; shared examples where `location_rerun_argument` is empty (fallback to `file_path`).
- [ ] Verify:

```bash
go test -mod=mod ./internal/runtime/ -run TestTracker
```

## Task 4: Bin-packing `SplitFile` as a method on `*Cache`

**Files:** `internal/runtime/cache.go` (or `internal/runtime/splitter.go` if extracted), `internal/runtime/splitter_test.go`

- [ ] Define `type SplitDecision map[string]float64` next to the cache type.
- [ ] Add method `func (c *Cache) SplitFile(filePath string, workerCount int, targetPerWorkerRuntime float64) SplitDecision`.
- [ ] No-split gate returns `{filePath: c.Files[filePath].RuntimeSeconds}` (single-entry map).
- [ ] Bin-packing algorithm per the spec section above. Read `*ExampleEntry` directly from `c.Files[filePath].Examples`; no projection type.
- [ ] Mean-runtime fallback for entries with `RuntimeSeconds <= 0`.
- [ ] Stable, deterministic algorithm: descending-runtime sort with line-number tiebreak; smallest-bin-index tiebreak on placement; line-sorted within each bin for stable target strings.
- [ ] Delete the old free-function `SplitFile`.
- [ ] Table-driven tests: even runtimes, one-large-many-small (LPT puts the big one alone), missing runtimes (fallback applied), more workers than examples (chunks bounded), determinism (same input → same output across calls), no-split passthrough returns single-entry map.
- [ ] Verify:

```bash
go test -mod=mod ./internal/runtime/ -run TestSplitFile
```

## Task 5: Wire `SplitDecision` into the grouper

**Files:** `runner.go`, `runner_rspec_split_test.go`

- [ ] Replace the existing `exampleLines []int` + `SplitDecision{Targets, Chunks, ChunkRuntimeSeconds}` call site with a direct `decision := cache.SplitFile(...)` followed by merging `decision` entries into the file-runtimes map fed to `GroupSpecFilesByRuntime`.
- [ ] Update the existing debug-log lines (e.g. `rspec-split applied`, `rspec-split skipped`) to use `len(decision)` for chunk count and the per-target runtime from the map.
- [ ] Add a test: file with one heavy example and many light ones must produce uneven per-target runtimes (validates we're not using `total / chunks`).
- [ ] Verify:

```bash
go test -mod=mod . -run 'TestRunner|TestRspecSplit'
```

## Task 6: Debug logging inside Load/Save

**Files:** `internal/runtime/cache.go`

- [ ] Capture `time.Now()` at the top of `LoadCache` and `SaveCache`. Compute duration before return.
- [ ] Log via existing `logger.Logger.Debug` with structured keys:

```go
logger.Logger.Debug("runtimeCache loaded", "duration_ms", ms, "path", path, "files", filesCount, "examples", examplesCount)
logger.Logger.Debug("runtimeCache saved",  "duration_ms", ms, "path", path, "files", filesCount, "examples", examplesCount)
```

- [ ] No signature changes. No caller plumbing.
- [ ] Verify by running an integration spec with debug logging on and confirming the lines appear; no behavior change beyond logging.

## Task 7: Amend `plur doctor` runtime data block

**Files:** `cmd_doctor.go`, `spec/integration/plur_doctor/doctor_spec.rb`

- [ ] When the runtime file exists, replace the `(file exists)` line at [cmd_doctor.go:113](../../cmd_doctor.go) with a one-shot read summary:

```
Runtime Data:     /Users/rsanheim/.plur/runtime/2053983d.json
                  192K / 28 files / 694 examples
```

- [ ] Keep `(file does not exist)` as today on miss; do not change behavior when the cache is missing.
- [ ] No load/save ms in this output — runtime impact lives in debug logs. Doctor reads the file once for stats, not via the Cache API's load path.
- [ ] Use existing human-size formatting if present in the codebase; otherwise inline a small helper for KB/MB display.
- [ ] Update existing backspin assertion in `doctor_spec.rb`.
- [ ] Verify:

```bash
bin/rspec spec/integration/plur_doctor/doctor_spec.rb
```

## Task 8: Real-project QA pass

**Files:** results recorded in PR description; create `docs/plans/2026-05-22-runtime-cache-measurement-results.md` only if results warrant Phase B.

- [ ] Warm the cache for each of: Plur, RuboCop, RSpec, Mastodon subset, Discourse subset.
- [ ] For each, capture three `plur -n 8` runs with debug logging on. Record the `runtimeCache loaded` and `runtimeCache saved` `duration_ms` values.
- [ ] Tabulate: project / examples cached / median load ms / median save ms / threshold met?
- [ ] Spot-check the trimmed JSON shape against a Discourse cache to confirm nothing important was lost.
- [ ] If any project exceeds its threshold, write a follow-up Phase B plan.
- [ ] If all projects sit under thresholds, record the table in the PR description and close the size concern.

## Task 9: Documentation

**Files:** `docs/usage.md`

- [ ] One sentence on the new `runtimeCache loaded` / `runtimeCache saved` debug log keys.
- [ ] Update the doctor section to mention the new stats line under `Runtime Data:`.
- [ ] No promises about Phase B work or specific format changes — gate on Task 8 results.

## Task 10: Full code review pass — simplify, dedupe, remove

**Files:** every file touched by Tasks 1–9.

- [ ] Re-read each modified file with a simplification eye. Look for:
  - Types added during implementation that didn't earn their keep (any new struct, alias, or helper that wraps a one-liner is suspect).
  - Helpers that wrap a single call or a one-line slice op.
  - Duplicate concepts. Specifically, evaluate whether `RunKind` is worth a separate file/type or can fold into `tracker.go` as a small enum.
  - Dead code: anything left over from before the schema trim, the round-robin splitter, or the package re-org.
  - Tests that assert framework behavior (`json.Unmarshal` ignores extra fields, etc.) rather than our logic. Delete those.
  - Comments that restate the code or carry temporal context ("recently changed", "now uses") — strip per project CLAUDE.md.
- [ ] If `SplitDecision` ends up only ever returned by `Cache.SplitFile` and consumed by one call site, consider whether the typedef is pulling its weight or whether `map[string]float64` inline is clearer.
- [ ] Commit cleanup changes in small logical units.

## Task 11: Final verification

- [ ] `bin/rake test:go`
- [ ] `bin/rake test`
- [ ] `bin/rake standard:fix`
- [ ] `git diff --check`
- [ ] Confirm cherry-pickability: the Task 1 commits (pure moves + the type-rename) apply cleanly onto a fresh branch off `main` with `git cherry-pick`.

## Success Criteria

- All runtime / splitter code lives in `internal/runtime/`. The pure-move commits are cherry-pickable to `main`.
- Per-example records carry three fields, not five.
- Shared examples are attributed to the rerunnable owning spec file at write time via `location_rerun_argument`, unconditionally.
- The splitter is a method on `*Cache`, uses cached per-example runtimes for longest-processing-time bin-packing, and returns a `SplitDecision` map (target → runtime) with no projection types in between.
- Load and save each emit a structured `logger.Logger.Debug` line with `duration_ms`, `path`, `files`, `examples`.
- `plur doctor`'s existing runtime-data block now shows file size / files / examples in place of `(file exists)`. No new section.
- Real-project QA produces a numbers table and a clear decision: thresholds met → close the concern; thresholds exceeded → follow-up Phase B plan.
- Task 10 cleanup pass produced at least one simplification commit, or explicitly recorded "nothing to simplify" in the PR description.
