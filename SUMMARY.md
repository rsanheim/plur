# RSpec Split-Specs Implementation Summary

**Branch:** `rspec-split-specs`
**Plan:** `docs/plans/2026-05-12-rspec-split-specs-experimental-plan.md`
**Date:** 2026-05-12

---

## What shipped

Two changes — one default, one opt-in:

1. **Runtime cache v2** (default-on, no flag): The on-disk runtime cache
   moved from `map[string]float64` to a versioned object format that carries
   file aggregates *plus* per-example identity and runtime metadata keyed by
   RSpec's canonical `example.id`. Old v1 caches are ignored and replaced
   on the next aggregate-eligible run.
2. **`--rspec-split` EXPERIMENTAL flag** (opt-in): expands long-pole RSpec
   files into focused `file:line:line:line` targets before file-level
   worker balancing. Gates on `--rspec-split`, RSpec framework, and
   `workers > 1`. Cache-driven; falls back to file-level grouping any time
   the cached example index is stale or absent.

The flag is gated; the cache shape is not. Normal runs always emit and
persist the richer per-example metadata so future splits have data to
work from.

---

## Commits

```
8348a78 Task 9: document v2 runtime cache + experimental --rspec-split
3eed587 Task 8: integration coverage for partial-run classification and --rspec-split
1b6c7dd Task 7: wire splitter into the runtime grouper
ee283ee Task 6: pure RSpec line splitter
8db7f70 Task 5: add --rspec-split EXPERIMENTAL flag
7ac052c Task 4: v2 runtime cache writes end-to-end + file:line CLI support
de13c82 Task 3: parser populates new RSpec identity fields
b699648 Task 2: enrich RSpec formatter output with identity metadata
bca1660 Task 1: runtime cache v2
```

---

## Key design decisions

### `example_index_complete` lifecycle (KISS)

One bit, one meaning: "the `examples` map was populated by a recent
aggregate-eligible full run against a source file whose mtime/size match
what we stored."

- **Aggregate-eligible full run, fresh source** → clear and rewrite
  `examples`, set flag to `true`, record current `mtime_unix_nano` and
  `size_bytes`.
- **Partial run (focused/tagged/fail-fast/aborted/custom-arg), fresh
  source** → merge per-example observations by `example.id`; do not touch
  the flag.
- **Partial run, stale source** → leave `examples` and the flag alone;
  per-example runtime updates only.
- Flag never goes false→true on a partial run.
- Splitter reads `examples` only when flag is `true` **AND** source
  freshness matches at read time.

### Run-kind classification

`classifyRunKind` in `runtime_run_kind.go` demotes a run to "partial"
when any selectivity signal is present:

- `--tag …` (Plur-owned, classified before passthrough sees it)
- positional patterns containing `:` or `[` (file:line, example id)
- any `--` passthrough args (intentionally inclusive — anything custom
  could change selection)
- aborted run (any worker reported `StateError` or fail-fast)

### Split heuristic (KISS for experimental)

```
split iff
  workerCount > 1
  AND runtimeSeconds > totalRuntime / workerCount  (per-worker budget)
  AND len(exampleLines) >= 2
```

Chunks = `min(workerCount, len(exampleLines))`. Examples are round-robin
distributed into chunks so each chunk gets a mix of early and late
examples. No multiplier, no floor, no top-N. The plan explicitly leaves
heuristic tuning to follow-up data.

### File:line CLI support

`fileset.classifyInputs` now passes through positional args whose
substring before the first `:` is an existing regular file. Plur does
not parse the suffix — RSpec interprets line numbers, scoped ids, etc.
This makes `plur spec/foo_spec.rb:42` work end-to-end and supports the
splitter's own generated targets.

---

## Test evidence

### Go suite

```
$ bin/rake test:go
ok  	github.com/rsanheim/plur	1.494s
ok  	github.com/rsanheim/plur/framework	0.014s
ok  	github.com/rsanheim/plur/framework/minitest	0.010s
ok  	github.com/rsanheim/plur/framework/rspec	0.009s
ok  	github.com/rsanheim/plur/internal/buildinfo
ok  	github.com/rsanheim/plur/internal/fileset	0.031s
ok  	github.com/rsanheim/plur/internal/format
ok  	github.com/rsanheim/plur/internal/kongtoml
ok  	github.com/rsanheim/plur/internal/railsinit	0.021s
ok  	github.com/rsanheim/plur/internal/runtime	0.015s
ok  	github.com/rsanheim/plur/logger
ok  	github.com/rsanheim/plur/watch	0.511s
```

New focused unit tests:

- `TestRuntimeCache_*` (13 tests): load/save/v1-ignore/freshness/
  aggregate-vs-partial merging/atomic write/edge cases
- `TestRuntimeTracker` (6 sub-tests): aggregate vs partial save kinds,
  example merge by `TestID`, file aggregate preservation
- `TestClassifyRunKind` (7 sub-tests): each demotion signal
- `TestSplitFile` (7 sub-tests): under-budget, single-worker, single-
  example, above-budget split, example-count bounded chunks,
  deterministic sort on unsorted input, zero-budget defensive behavior
- `TestSplitFile_DoesNotMutateInput`, `TestSplitFile_RepeatedCallsAreStable`
- `TestExpandRspecSplits_*` (3 tests): split path, pass-through under
  budget, pass-through stale cache
- `TestShouldExpandSplits` (4 sub-tests): flag off / single worker /
  non-RSpec / all conditions met
- `TestRspecSplit*` (4 tests): default off, CLI on, env-var on,
  EXPERIMENTAL help marker
- `TestRuntimeCache_ExampleLines`, `TestPerWorkerBudget*`
- `TestDiscover_FileLine*`, `TestDiscover_MultiFileLine*`: fileset
  passthrough

### Ruby integration suite

```
$ bin/rake test
333 examples, 0 failures, 4 pending
```

Including these new specs in `spec/integration/spec/runtime_tracking_spec.rb`:

- v2 schema_version / plur_version / file aggregates / examples index
- second run logs "Using runtime-based grouped execution"
- corrupt cache replaced with valid v2 JSON
- `--dry-run` writes nothing
- runtime-based grouping from v2 aggregates
- focused `file:line` preserves full-file runtime_seconds
- focused runs merge by RSpec `example.id` without dropping others
- `--tag` runs do not update default aggregates
- `--fail-fast` aborted runs do not update default aggregates
- arbitrary `--seed` passthrough does not update default aggregates
- `--rspec-split --dry-run` cold cache passes through file targets
- `--rspec-split --dry-run` warm cache + forcibly-slow file produces
  `file:line` targets and logs "rspec-split applied"
- `--rspec-split` real run of a forced-slow file passes
- Updated `spec/integration/spec/json_rows_formatter_spec.rb` for the
  new identity fields (id / absolute_file_path /
  location_rerun_argument / scoped_id) plus a fallback test for
  metadata-less doubles. 15 examples, 0 failures.

### Full build

```
$ bin/rake
... go fmt + go vet + standardrb + go test + RSpec ...
333 examples, 0 failures, 4 pending
```

---

## Real-world QA + Benchmarks

Hyperfine runs: `--warmup 1 --runs 5 --ignore-failure`, isolated
`PLUR_HOME` per project, warmed cache once before the timing runs.
Same plur binary, same worker count, only `--rspec-split` differs.

### Plur fixture (`fixtures/projects/default-ruby`, 13 specs, -n 4)

| Command | Mean [ms] | Min [ms] | Max [ms] | Relative |
|:---|---:|---:|---:|---:|
| `split-off` | 301.0 ± 5.4 | 292.4 | 307.0 | 1.01 ± 0.02 |
| `split-on`  | 299.5 ± 3.4 | 294.5 | 303.7 | 1.00 |

No splits triggered (no file approaches the per-worker budget). Suite
is too fast and balanced.

### rubocop `spec/rubocop/cop/lint` (154 files, 6087 examples, -n 8)

| Command | Mean [s] | Min [s] | Max [s] | Relative |
|:---|---:|---:|---:|---:|
| `split-off` | 5.503 ± 0.118 | 5.371 | 5.664 | 1.00 ± 0.02 |
| `split-on`  | 5.502 ± 0.069 | 5.435 | 5.578 | 1.00 |

No splits triggered. Slowest file `redundant_type_conversion_spec.rb`
at 0.51s, well under the 0.90s per-worker budget.

### rspec-core `spec/rspec` (66 files, 2092 examples, -n 8)

| Command | Mean [s] | Min [s] | Max [s] | Relative |
|:---|---:|---:|---:|---:|
| `split-off` | 2.368 ± 0.068 | 2.283 | 2.471 | 1.00 |
| `split-on`  | 2.449 ± 0.044 | 2.389 | 2.493 | 1.03 ± 0.04 |

One split triggered: `rake_task_spec.rb` (1.93s) into 8 chunks of
~0.24s each. Net 3% slower because the extra RSpec boot overhead
(~0.3s × 7 chunks) exceeded the parallelism gain on a file already
under 2s. Marginal-split regression — expected and called out in the
docs.

### rubocop full suite (745 files, 31672 examples, -n 8) — **headline result**

| Command | Mean [s] | Min [s] | Max [s] | Relative |
|:---|---:|---:|---:|---:|
| `split-off` | 28.848 ± 0.264 | 28.566 | 29.186 | 1.28 ± 0.06 |
| `split-on`  | 22.491 ± 0.952 | 21.417 | 23.548 | 1.00 |

**~22% faster (6.4 s reduction)** with `--rspec-split` on. Two files
split:

- `spec/rubocop/cli/options_spec.rb` → 8 chunks, 3.21 s each
- `spec/rubocop/server/rubocop_server_spec.rb` → 8 chunks, 2.13 s each

Both substantially exceed the per-worker budget; splitting them pulls
the long pole into line with the rest of the workers' loads.

### Mastodon & Discourse — DEFERRED

Both repos are checked out but not bundle-installed. Mastodon requires
PostgreSQL + Redis setup; Discourse similar. These targets were not
exercised in this session and remain on the list for follow-up QA.
Treat the mastodon/discourse line items from the original plan as
incomplete.

### Pitfalls observed

None encountered in the runs that actually triggered splits. No
`before(:all)` divergence, dynamic-example divergence, or shared-
example surprises showed up in rubocop or rspec-core's affected files,
but the population is small. The pitfalls section in `docs/usage.md`
calls these out so users know what to watch for.

### Headline numbers

- correctness: 333 RSpec integration examples + full Go suite pass
  with the flag both on and off
- speedup observed: 22% on rubocop's full suite (the realistic case
  for this feature)
- regression observed: 3% on rspec-core when only one borderline file
  splits — bounded, predictable, documented
- no-trigger cases (rubocop/cop/lint, plur fixture): identical wall
  time within noise

---

## Open items

1. **Mastodon / Discourse QA** — not run in this session due to setup
   cost. Follow-up should bundle install + DB setup + repeat the
   hyperfine recipe.
2. **Threshold tuning** — the experimental threshold is intentionally
   the simplest possible rule (runtime > budget). The rspec-core
   marginal regression hints that a small multiplier (e.g. 1.5×
   budget) could avoid net-negative splits. Hold until more data lands.
3. **Cache-size bounds** — explicitly deferred per the plan's "Future
   Work (Out Of Scope)" section.
4. **Watch integration spec flakiness** — `spec/integration/watch/
   watch_reload_spec.rb` occasionally times out on `IO#read_nonblock`
   in the broader `bin/rake test` run. Pre-existing flake (verified
   not introduced by this branch); not blocking.

---

## File map

Created:

- `runtime_cache.go` / `runtime_cache_test.go` — v2 data model, atomic
  save, freshness, lifecycle rules, example-line accessor.
- `runtime_run_kind.go` — aggregate vs partial classifier.
- `rspec_line_splitter.go` / `rspec_line_splitter_test.go` — pure
  SplitFile function with table-driven tests.
- `runner_rspec_split_test.go` — runner-level split expansion tests.
- `tmp/bench/run-bench.sh` — hyperfine harness used for the QA above.
- `SUMMARY.md` — this file.

Modified:

- `runtime_tracker.go` / `runtime_tracker_test.go` — wraps the v2
  cache, exposes `LoadedData()` for grouping compat, takes a RunKind on
  save.
- `framework/rspec/formatter.rb` — emits id / absolute_file_path /
  location_rerun_argument / scoped_id; uses `metadata[:line_number]`
  with a location-string fallback.
- `framework/rspec/json_output.go` — `StreamExample` gains the four
  new identity fields.
- `framework/rspec/parser.go` / `parser_test.go` — copies new fields
  into `TestCaseNotification`, prefers `example.id` as TestID.
- `types/notifications.go` — extends `TestCaseNotification`.
- `internal/fileset/fileset.go` / `fileset_test.go` — file:line
  passthrough.
- `runner.go` — `shouldExpandSplits`, `expandRspecSplits`,
  `perWorkerBudget`.
- `cmd_spec.go` — passes RunKind from CLI flags to the tracker.
- `main.go` / `main_test.go` / `config/config.go` — `--rspec-split`
  flag with `PLUR_RSPEC_SPLIT` env binding and EXPERIMENTAL help text.
- `spec/integration/spec/runtime_tracking_spec.rb` — full rewrite to
  v2 schema, plus split + partial-run coverage.
- `spec/integration/spec/json_rows_formatter_spec.rb` — covers the new
  identity fields.
- `docs/usage.md` — runtime cache + experimental flag sections.
- `docs/plans/2026-05-12-rspec-split-specs-experimental-plan.md` —
  earlier-in-session: tightened the threshold, lifecycle rules, QA
  targets, and hyperfine recipe.
