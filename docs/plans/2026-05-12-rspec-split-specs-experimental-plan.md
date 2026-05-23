# RSpec Split Specs Experimental Implementation Plan

> **For implementation workers:** Implement this task-by-task. Keep each task small, verify before moving on, and commit after each completed task.

**Status:** Implemented on `rspec-split-specs`. This document is kept as the branch implementation record; the current user-facing behavior is documented in `docs/usage.md`.

**Goal:** Reshape Plur's default runtime tracking data so it supports both today's file-level balancing and RSpec long-file splitting, then expose RSpec splitting behind `--rspec-split`.

**Architecture:** Runtime tracking now writes schema v4 by default, independent of `--rspec-split`. Plur's RSpec formatter emits richer selected-example metadata and per-example runtime details; the runtime tracker persists file-level aggregates for normal worker balancing plus example-level metadata that `--rspec-split` can consume. The experimental flag only changes grouping behavior; it does not control whether runtime metadata is collected.

**Tech Stack:** Go CLI with Kong, Plur runtime tracker, Plur's RSpec JSON rows formatter, RSpec focused line execution, `$PLUR_HOME/runtime`.

---

## Position

The runtime data shape should change first. Plur is pre-1.0, and the current cache is just:

```json
{
  "spec/calculator_spec.rb": 1.234
}
```

That was enough for simple file balancing, but it was the wrong foundation for splitting. A separate `rspec-example-index` cache would duplicate state and create drift. Instead, the default runtime cache now carries the data both features need:

- file-level runtime for normal grouping
- per-example line metadata for splitting
- per-example runtime observations when available
- source freshness metadata so stale line data is ignored
- write-time rules so tag/focused runs do not corrupt full-file data

## RSpec Formatter Baseline

As of the local RSpec checkout, `RSpec::Core::Formatters::JsonFormatter` registers `:stop` and builds its `examples` array from `group_notification.notifications.map { |notification| notification.example }`.

It does not use `RSpec.world.example_groups.flat_map(&:descendant_filtered_examples)` as its JSON output source. That world traversal proves the filtered examples are available after load, but Plur should prefer the same reporter notification path RSpec's JSON formatter uses unless a later requirement needs pre-execution data.

RSpec's `JsonFormatter#format_example` includes:

- `id`
- `description`
- `full_description`
- `status`
- `file_path`
- `line_number`
- `run_time`
- `pending_message`
- `exception` only when present, with class, message, and formatted backtrace

Plur's runtime cache does not need to persist everything RSpec JSON emits. It should persist:

- `id`, because it is RSpec's canonical stable example identifier and is used to merge observations
- source freshness metadata (`mtime_unix_nano`, `size_bytes`) at the file level
- `line_number` and `location_rerun_argument`
- `runtime_seconds`

It should not persist descriptions, full descriptions, exception messages, backtraces, formatter messages, profile output, summaries, seeds, status, scoped IDs, or pending messages in the runtime cache. Those are output/reporting concerns, and storing them would grow the cache without helping file balancing or line splitting.

Use `metadata[:line_number]` for line data, not string parsing. Parser notifications keep RSpec's raw `location`, but the runtime cache persists only the rerunnable target and owner line needed for balancing.

## Existing Type Changes

Modify the structs Plur already has before adding new data carriers.

- Extend `framework/rspec.StreamExample` with `ID`, `AbsoluteFilePath`, `LocationRerunArgument`, and `ScopedID`.
- Extend `framework/rspec.StreamingMessage` only if the formatter emits a batch row for selected examples.
- Extend `types.TestCaseNotification` with the same identity/location fields needed by runtime tracking.
- Keep `TestCollector` as the place that aggregates parsed framework notifications before `RuntimeTracker` persists them.
- Add new runtime-cache structs only for the persisted schema v4 file format.

This keeps the data flow coherent:

```text
RSpec formatter JSON rows
  -> framework/rspec parser structs
  -> types.TestCaseNotification / collector data
  -> RuntimeTracker
  -> runtime cache schema v4
  -> grouper
```

## Runtime Cache Schema V4

The persisted shape is versioned and object-based rather than `map[string]float64`.

Critical shape:

```json
{
  "meta": {
    "schema_version": 4,
    "plur_version": "0.56.0-dev-abc1234"
  },
  "run": {
    "cwd": "/Users/example/src/project",
    "last_run_at": "2026-05-22T15:04:05Z"
  },
  "files": {
    "spec/slow_spec.rb": {
      "mtime_unix_nano": 1778610000000000000,
      "size_bytes": 12345,
      "runtime_seconds": 12.34,
      "examples": [
        {
          "id": "./spec/slow_spec.rb[1:1]",
          "line_number": 12,
          "location_rerun_argument": "./spec/slow_spec.rb:12",
          "runtime_seconds": 0.40
        }
      ]
    }
  }
}
```

Notes:
- `meta.plur_version` should come from `buildinfo.GetVersionInfo()`. Do not persist commit, build date, Go version, OS, architecture, race detector state, job name, framework name, command args, worker count, or historical run metadata.
- `run.cwd` should be the project cwd Plur already uses for the runtime file hash.
- `run.last_run_at` should be overwritten on every cache save in UTC RFC3339 format.
- `files` is keyed by project-relative file path.
- `examples` is an array; each entry stores RSpec's canonical `example.id` as `id`.
- Run selection is a write-time decision, not persisted cache structure. Default/full-file runs may update file-level `runtime_seconds`; focused, tagged, fail-fast, aborted, or custom-arg runs must not.
- Non-aggregate runs may merge individual example observations by `example.id`, but they must not prune examples missing from that run.
- Normal grouping reads file-level `runtime_seconds`.
- RSpec splitting reads each cached example entry's rerunnable selector from `location_rerun_argument`, falling back to `line_number`.
- Non-RSpec runs can store file-level entries without example arrays.
- No backward-compatibility shim is required. If an old cache is present, ignore it and regenerate schema v4 data.

### Freshness Lifecycle

There is no persisted completeness flag. Cached examples are usable for splitting when the file entry exists and current source `mtime_unix_nano` and `size_bytes` match the stored values.

Write rules:

- Aggregate-eligible full run: clear and rewrite `examples`, record current `mtime_unix_nano` and `size_bytes`.
- Partial run (focused/tagged/fail-fast/aborted/custom-arg): merge observed examples by `example.id` into an existing file entry. Do not change file-level `runtime_seconds` and do not prune examples missing from the run.

Read rules (splitter):

- Use `examples` only when current source `mtime_unix_nano` and `size_bytes` equal the stored values.
- Any mismatch means fall back to file-level grouping.

### Future Work (Out Of Scope)

- Cache-size bounds. With very large suites the persisted JSON may grow large. Defer per-file example caps, threshold-based persistence, or eviction until real-world QA shows a problem.

## Known Pitfalls

- `before(:all)` / `before(:context)` state can break when examples from one file are split across workers, because context setup may run once per split process rather than once for the original full file process.
- RSpec suites that define examples dynamically from environment, database state, time, random data, or metaprogramming may produce different example sets between cache generation and split execution.
- Shared examples and custom DSLs can create surprising source locations. Store RSpec's `id` and `location_rerun_argument` so we can debug those cases instead of relying only on line numbers.
- Custom ordering, fail-fast, focus filters, and tag filters can make a run incomplete. Incomplete runs must not overwrite the default full-file aggregate.
- Ruby is dynamic. Expect some meta-heavy suites to need fallback-to-file behavior until we have real-world test coverage.

## User Interface

CLI:

```bash
plur --rspec-split -n 8
```

Environment:

```bash
PLUR_RSPEC_SPLIT=1 plur -n 8
```

Critical CLI field:

```go
RspecSplit bool `help:"EXPERIMENTAL: split long-running RSpec files into focused file:line runs" name:"rspec-split" env:"PLUR_RSPEC_SPLIT" default:"false"`
```

The runtime cache schema and formatter metadata changes are not gated by this flag.

## Safety Rules

- Default grouping behavior remains file-level.
- Runtime cache schema v4 is written for normal runs regardless of `--rspec-split`.
- Splitting applies only when `RspecSplit == true`, the selected job has `Framework == "rspec"`, and worker count is greater than 1.
- Splitting requires runtime cache data with fresh source metadata and rerunnable example selectors for the file.
- Missing or stale example data means current file-level grouping.
- No regex/source-code fallback.
- No RSpec subprocess for discovery during `plur --dry-run`.
- Focused runs and tag runs must not clobber default full-file runtime aggregates.

## Branch Review Follow-Up

Real-project review on Discourse showed that line-only round-robin splitting was not enough. The implemented behavior now:

1. Build split units from cached examples, preferring `location_rerun_argument` or another RSpec-rerunnable target that maps back to the owning spec file.
2. Assign each unit its observed `runtime_seconds`.
3. Bin-pack units into chunks with longest-runtime-first greedy balancing.
4. Feed each chunk's summed runtime into `GroupSpecFilesByRuntime`.
5. Keep deterministic output for repeated runs against the same cache.

Discourse also exposed a related ownership issue: shared examples can report `file_path` as `spec/support/shared_examples/...` while their `example.id` and `location_rerun_argument` point back to a real model spec. Runtime tracking and splitting now key aggregate ownership by the rerunnable/owning spec file when RSpec provides enough metadata.

## File Responsibilities

Create:
- `internal/testruntime/cache.go`: schema v4 persisted data model, load/save, freshness checks, aggregate-eligibility rules.
- `internal/testruntime/cache_test.go`: cache shape, old-cache ignore behavior, aggregate-eligibility behavior, freshness behavior.
- `internal/testruntime/splitter.go`: pure split decisions and focused target generation.
- `internal/testruntime/splitter_test.go`: pure behavior around thresholds, chunking, and fallbacks.

Modify:
- `framework/rspec/formatter.rb`: align example metadata with RSpec's JSON formatter path and stop parsing line numbers from location strings.
- `framework/rspec/json_output.go`: extend existing structs for formatter-emitted selected examples.
- `framework/rspec/parser.go`: consume selected-example and per-example metadata messages.
- `types/notifications.go`: extend existing notification structs with selected example metadata.
- `internal/testruntime/tracker.go`: replace map-only persistence with schema v4 runtime cache and expose file-runtime data for grouping.
- `runner.go`: save runtime data after workers finish and pass cached file runtimes to grouping.
- `grouper.go`: use cached file-level runtimes for existing grouping. Split expansion happens in `runner.go` only when `RspecSplit == true`.
- `main.go`, `config/config.go`, `main_test.go`: add the experimental boolean flag.
- Formatter, parser, runtime tracker, grouper, runner, and integration specs.
- `docs/usage.md`: document runtime cache behavior, pitfalls, and the experimental flag.

---

## Task 1: Define Runtime Cache Schema V4

**Files:** `internal/testruntime/cache.go`, `internal/testruntime/cache_test.go`, `internal/testruntime/tracker.go`, `internal/testruntime/tracker_test.go`

- [ ] Add a versioned runtime cache data model based on the critical shape above.
- [ ] Ignore old `map[string]float64` cache files rather than migrating them.
- [ ] Expose a small method that returns `map[string]float64` for normal file-level grouping.
- [ ] Add source freshness helpers based on project-relative path, `mtime_unix_nano`, and `size_bytes`.
- [ ] Add aggregate-eligibility rules for default/full-file, focused, tagged, fail-fast, aborted, and custom-arg runs.
- [ ] Use `example.id` as the canonical key for persisted RSpec example entries.
- [ ] Persist only `meta.plur_version` from existing build info plus `run.cwd` and `run.last_run_at`; do not persist job/framework/build/platform metadata or historical run metadata.
- [ ] Write cache files atomically by writing a temporary file in the runtime directory and renaming it into place.
- [ ] Ignore invalid or corrupt cache files and regenerate from the next successful run.
- [ ] Verify with focused Go tests:

```bash
go test -mod=mod . -run 'TestRuntimeCache|TestRuntimeTracker'
```

## Task 2: Emit RSpec Metadata by Default

**Files:** `framework/rspec/formatter.rb`, `framework/rspec/json_output.go`, `spec/integration/spec/json_rows_formatter_spec.rb`

- [ ] Register the formatter for `:stop` if it is not already registered.
- [ ] Emit the runtime metadata row from the `stop(group_notification)` path, using `group_notification.notifications.map(&:example)`, matching RSpec's JSON formatter source.
- [ ] Extend existing `StreamExample` fields rather than introducing a parallel example struct.
- [ ] Include `id`, `file_path`, `absolute_file_path`, `line_number`, `location`, `location_rerun_argument`, `scoped_id`, `run_time`, `status`, and `pending_message`.
- [ ] Change existing group/example line extraction to prefer `metadata[:line_number]` instead of `location.split(":").last.to_i`.
- [ ] Keep existing progress, failure, pending, and summary rows compatible.
- [ ] Verify with:

```bash
mise exec -- bundle exec rspec spec/integration/spec/json_rows_formatter_spec.rb
```

## Task 3: Parse and Collect RSpec Metadata

**Files:** `framework/rspec/parser.go`, `types/notifications.go`, `internal/testruntime/tracker.go`

- [ ] Extend existing Go structs to match the formatter's enriched example fields.
- [ ] Normalize `./spec/foo_spec.rb` and `spec/foo_spec.rb` to the same project-relative path.
- [ ] Preserve per-example runtime from existing pass/fail/pending rows.
- [ ] Feed executed test notifications into the runtime tracker after workers finish.
- [ ] Verify parser and collector behavior with focused Go tests.

## Task 4: Save Useful Runtime Data for Normal Runs

**Files:** `internal/testruntime/tracker.go`, `runner.go`, `spec/integration/spec/runtime_tracking_spec.rb`

- [ ] Save schema v4 runtime cache entries after normal runs.
- [ ] Update full-file aggregates only for aggregate-eligible default/full-file runs.
- [ ] Preserve prior default file aggregates when a focused run only covers a subset of a file.
- [ ] Merge selected example observations by RSpec `example.id`; incomplete runs must not prune examples missing from that run.
- [ ] Record source freshness only from aggregate-eligible default/full-file runs.
- [ ] Keep existing runtime-based grouping behavior working through the schema v4 file-level runtime view.

Concrete success criteria:
- A normal `plur spec/calculator_spec.rb` run writes one schema v4 runtime file under `$PLUR_HOME/runtime`.
- The runtime file has `meta.schema_version: 4`.
- The runtime file has `meta.plur_version`.
- The runtime file has `run.cwd`.
- The runtime file has `run.last_run_at` in UTC RFC3339 format.
- The top-level `files["spec/calculator_spec.rb"].runtime_seconds` is greater than 0.
- The same file entry contains an `examples` array with `id`, `line_number`, and `runtime_seconds`.
- A second run logs `Using runtime-based grouped execution`.
- A corrupt runtime file is ignored and replaced by valid schema v4 JSON after a successful run.
- `plur --dry-run` does not create or modify the runtime cache and does not invoke any RSpec discovery command.
- A `--fail-fast` or otherwise aborted run does not overwrite file-level aggregate runtime.
- A focused `spec/calculator_spec.rb:<line>` run does not overwrite the default full-file `runtime_seconds`.
- A focused `spec/calculator_spec.rb:<line>` run may merge the executed example observation by RSpec `example.id`.

Verify with:

```bash
bin/rspec spec/integration/spec/runtime_tracking_spec.rb
```

## Task 5: Add the Boolean Experimental Flag

**Files:** `main.go`, `config/config.go`, `main_test.go`

- [ ] Add `RspecSplit bool` to `PlurCLI` using the critical field definition above.
- [ ] Add `RspecSplit bool` to `config.GlobalConfig`.
- [ ] Populate `GlobalConfig.RspecSplit` during CLI config construction.
- [ ] Test that the flag is off by default, enabled by `--rspec-split`, enabled by `PLUR_RSPEC_SPLIT=1`, and marked `EXPERIMENTAL` in help metadata.
- [ ] Verify with:

```bash
go test -mod=mod . -run 'TestRspecSplit'
```

## Task 6: Add Pure Split Decisions

**Files:** `internal/testruntime/splitter.go`, `internal/testruntime/splitter_test.go`

- [ ] Define a pure function that receives file path, historical runtime, worker count, target group runtime, and exact example units with line/rerun target/runtime data.
- [ ] Return the original file path unchanged when the file is not a long pole or has too few exact lines.
- [ ] Use the simplest possible threshold for the experimental rollout: split files whose historical `runtime_seconds` is greater than the target per-worker runtime budget. No multiplier, no floor, no top-N selection. Tune later based on real-world QA.
- [ ] Produce focused targets like:

```text
spec/slow_spec.rb:12:38:91
```

- [ ] Keep chunk count bounded by worker count and example count.
- [ ] Use cached per-example runtime seconds to bin-pack examples into chunks instead of round-robin splitting by line number.
- [ ] Assign each generated target the summed runtime of its examples, not `original_file_runtime / chunk_count`.
- [ ] Make chunking deterministic so repeated runs produce stable commands.
- [ ] Preserve a simple fallback for examples with missing runtime, such as a per-file average or small default.
- [ ] Verify the pure splitter with table-driven tests.

## Task 7: Expand Runtime Grouping When Enabled

**Files:** `grouper.go`, `grouper_test.go`, `runner.go`, `runner_test.go`

- [ ] Keep the old grouping behavior when `RspecSplit` is false.
- [ ] When `RspecSplit == true` for an RSpec job, read file runtimes and example runtime units from the runtime cache.
- [ ] Expand long-pole files into focused targets before existing longest-processing-time grouping.
- [ ] Assign each generated target the summed runtime of its examples for grouping balance.
- [ ] Do not persist runtime entries for generated `file:line` targets.
- [ ] Log debug reasons when splitting is skipped.
- [ ] Treat shared examples carefully: the cache should preserve diagnostic source file information, but grouping/splitting ownership should follow the rerunnable owning spec file when available.
- [ ] Verify:

```bash
go test -mod=mod . -run 'TestGroupSpecFilesByRuntime|TestRunner|TestRspecSplit'
```

## Task 8: Add Integration Coverage

**Files:** likely `spec/integration/spec/runtime_tracking_spec.rb`, `spec/integration/spec/rspec_args_spec.rb`, fixture specs as needed.

- [ ] Assert normal RSpec runs write schema v4 runtime data with file aggregates and selected example lines.
- [ ] Assert existing runtime grouping still balances slow files from schema v4 file aggregates.
- [ ] Assert cache writes are atomic with a temp-file-and-rename path.
- [ ] Assert invalid or corrupt cache files are ignored and regenerated.
- [ ] Assert `plur --dry-run` never writes runtime cache and never shells out for discovery.
- [ ] Assert `--fail-fast` and aborted runs do not overwrite file-level aggregate runtime.
- [ ] Assert focused file:line runs do not overwrite default full-file runtime aggregates.
- [ ] Assert focused file:line runs merge observations by RSpec `example.id`.
- [ ] Assert `plur --rspec-split -n 4 --dry-run` uses cached focused targets and does not invoke RSpec.
- [ ] Assert a real `plur --rspec-split -n 4` run executes focused `file:line` targets and passes after cache exists.
- [ ] Assert split chunks are balanced from cached per-example runtime data, not just from line count.
- [ ] Assert shared examples are attributed to the rerunnable owning spec file for aggregate grouping when `location_rerun_argument` points there.
- [ ] Assert unsupported passthrough args fall back to file-level grouping.
- [ ] Assert Plur-owned `--tag` filters are classified as tagged and do not update default full-file aggregates.
- [ ] Verify with:

```bash
bin/rspec spec/integration/spec/runtime_tracking_spec.rb spec/integration/spec/rspec_args_spec.rb
```

## Task 8.1: Full Build, Real-Project QA, And Benchmarks

**Files:** no planned source edits; record findings in the PR or a short docs note if needed.

- [ ] Run the full build:

```bash
bin/rake
```

- [ ] Run agent-driven QA against the following real-world RSpec projects:
  - Plur itself
  - RuboCop
  - RSpec (rspec-core or the rspec meta-repo)
  - A large subset of Mastodon
  - A subset of Discourse
- [ ] For each project, warm the runtime cache, then benchmark `--rspec-split` off vs on with the same worker count using `hyperfine` (already installed). Use enough runs to get stable numbers — at minimum:

```bash
hyperfine --warmup 1 --runs 5 \
  --export-json tmp/bench-<project>.json \
  -n 'split-off' 'plur -n 8' \
  -n 'split-on'  'plur -n 8 --rspec-split'
```

- [ ] Record wall-time mean/stddev, slowest worker, and command count from each run set. Per-worker timing is not in `hyperfine` output; capture it from plur's own summary or logs alongside the hyperfine numbers.
- [ ] Record whether any suites hit known pitfalls such as `before(:all)`, dynamic examples, or custom RSpec DSLs.
- [ ] Treat correctness failures (different test counts, new failures attributable to splitting) as blockers; treat benchmark regressions as reasons to keep the feature behind the experimental flag.

## Task 9: Document Runtime Cache Schema V4 and Experimental Split

**Files:** `docs/usage.md`

- [ ] Update runtime tracking docs to describe the schema v4 cache at a high level.
- [ ] Add a short experimental section for `--rspec-split`.
- [ ] Document known pitfalls: `before(:all)`, dynamic RSpec suites, shared examples, custom DSLs, and Ruby metaprogramming.
- [ ] State that the feature is opt-in, RSpec-only, runtime-data-driven, and cache-driven.
- [ ] Mention that normal runs populate the metadata future split runs need.
- [ ] Include the CLI and environment examples from this plan.
- [ ] Avoid promising stable split behavior while it is experimental.

## Task 10: Final Verification

- [ ] Run focused Go tests for the new code.
- [ ] Run focused formatter specs through mise and Bundler.
- [ ] Run focused integration specs for runtime tracking and RSpec args.
- [ ] Run the full build and real-project QA from Task 8.1.
- [ ] Run the standard project verification:

```bash
bin/rake test:go
bin/rake test
```

- [ ] Run `git diff --check`.
- [ ] Commit the finished implementation in small logical commits.

## Success Criteria

- Normal runtime tracking writes schema v4 data by default with no `--rspec-split` flag.
- Existing runtime-based file grouping still works from schema v4 file aggregates.
- RSpec formatter output aligns with RSpec's JSON formatter data source and no longer depends on parsing line numbers out of location strings.
- RSpec selected example metadata and per-example runtime details live in the main runtime cache.
- Focused or tag-filtered runs do not corrupt default full-file runtime aggregates.
- `plur --rspec-split` can split a long RSpec file into focused line targets when runtime cache data has fresh example lines.
- Split chunks use cached per-example runtimes for balancing when those runtimes are available.
- `plur --dry-run --rspec-split` never shells out to RSpec.
- Real-project QA has covered Plur, RuboCop, and at least one other RSpec project.
- Benchmarks compare `--rspec-split` off vs on after cache warmup.
- Docs clearly mark the split flag experimental and document known pitfalls.
