# RSpec Split Specs Experimental Implementation Plan

> **For implementation workers:** Implement this task-by-task. Keep each task small, verify before moving on, and commit after each completed task.

**Goal:** Reshape Plur's default runtime tracking data so it supports both today's file-level balancing and future RSpec long-file splitting, then expose RSpec splitting behind `--rspec-split`.

**Architecture:** Runtime tracking gets a v2 cache format that is written by default, independent of `--rspec-split`. Plur's RSpec formatter always emits richer selected-example metadata and per-example runtime details; the runtime tracker persists file-level aggregates for normal worker balancing plus example-level metadata that `--rspec-split` can consume. The experimental flag only changes grouping behavior; it does not control whether runtime metadata is collected.

**Tech Stack:** Go CLI with Kong, Plur runtime tracker, Plur's RSpec JSON rows formatter, RSpec focused line execution, `$PLUR_HOME/runtime`.

---

## Position

The runtime data shape should change first. Plur is pre-1.0, and the current cache is just:

```json
{
  "spec/calculator_spec.rb": 1.234
}
```

That is enough for simple file balancing, but it is the wrong foundation for splitting. A separate `rspec-example-index` cache would duplicate state and create drift. Instead, make the default runtime cache carry the data both features need:

- file-level runtime for normal grouping
- per-example line metadata for splitting
- per-example runtime observations when available
- source freshness metadata so stale line data is ignored
- run-selection metadata so tag/focused runs do not corrupt full-file data

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

- `id`, because it is RSpec's canonical stable example identifier and should be the map key for examples
- `file_path`, `absolute_file_path`, and source freshness metadata
- `line_number`, `location`, and `location_rerun_argument`
- `scoped_id` if available
- `runtime_seconds` and `status`
- `pending_message` only if useful for diagnostics

It should not persist descriptions, full descriptions, exception messages, backtraces, formatter messages, profile output, summaries, or seeds in the runtime cache. Those are output/reporting concerns, and storing them would grow the cache without helping file balancing or line splitting.

Use `metadata[:line_number]` for line data, not string parsing. Keep `location` and `location_rerun_argument` for diagnostics and future grouping choices.

## Existing Type Changes

Modify the structs Plur already has before adding new data carriers.

- Extend `framework/rspec.StreamExample` with `ID`, `AbsoluteFilePath`, `LocationRerunArgument`, and `ScopedID`.
- Extend `framework/rspec.StreamingMessage` only if the formatter emits a batch row for selected examples.
- Extend `types.TestCaseNotification` with the same identity/location fields needed by runtime tracking.
- Keep `TestCollector` as the place that aggregates parsed framework notifications before `RuntimeTracker` persists them.
- Add new runtime-cache structs only for the persisted v2 file format.

This keeps the data flow coherent:

```text
RSpec formatter JSON rows
  -> framework/rspec parser structs
  -> types.TestCaseNotification / collector data
  -> RuntimeTracker
  -> runtime cache v2
  -> grouper
```

## Runtime Cache V2

The exact Go structs can be refined during implementation, but the persisted shape should be versioned and object-based rather than `map[string]float64`.

Critical shape:

```json
{
  "schema_version": 2,
  "generated_at": "2026-05-12T18:30:00Z",
  "project_root": "/repo",
  "producer": {
    "name": "plur",
    "version": "dev-abc1234",
    "commit": "abc1234",
    "built_at": "2026-05-12T18:00:00Z",
    "built_by": "goreleaser",
    "race_enabled": false,
    "go_version": "go1.25.8",
    "goos": "darwin",
    "goarch": "arm64"
  },
  "jobs": {
    "rspec": {
      "framework": "rspec",
      "last_run": {
        "started_at": "2026-05-12T18:29:30Z",
        "worker_count": 4,
        "test_env_number_first_is_1": false,
        "command": ["bundle", "exec", "rspec"],
        "extra_args": [],
        "rspec_split": false,
        "dry_run": false,
        "selection": {
          "kind": "default",
          "aggregate_eligible": true,
          "args": []
        }
      },
      "files": {
        "spec/slow_spec.rb": {
          "path": "spec/slow_spec.rb",
          "abs_path": "/repo/spec/slow_spec.rb",
          "mtime_unix_nano": 1778610000000000000,
          "size_bytes": 12345,
          "runtime_seconds": 12.34,
          "example_count": 3,
          "example_index_complete": true,
          "examples": {
            "./spec/slow_spec.rb[1:1]": {
              "id": "./spec/slow_spec.rb[1:1]",
              "line_number": 12,
              "location": "./spec/slow_spec.rb:12",
              "location_rerun_argument": "./spec/slow_spec.rb:12",
              "scoped_id": "1:1",
              "runtime_seconds": 0.40,
              "status": "passed"
            }
          }
        }
      }
    }
  }
}
```

Notes:
- `examples` is keyed by RSpec's canonical `example.id`; do not invent a separate example key.
- `last_run.selection` is metadata about the run that produced the newest observations. It is not a cache partition key.
- `selection.kind == "default"` means no Plur-owned tag filters, no focused file:line selection, and no unsupported custom args.
- `selection.aggregate_eligible == true` means this run is allowed to update file-level `runtime_seconds` and mark `example_index_complete`.
- Tag, focused, and custom-arg runs should set `aggregate_eligible: false`; they may merge individual example observations by `id`, but they must not overwrite full-file aggregates or mark an example index complete.
- `producer` should come from existing build/runtime data where possible: `buildinfo.GetVersionInfo()`, `buildinfo.Commit`, `buildinfo.Date`, `buildinfo.BuiltBy`, `buildinfo.RaceEnabled`, `runtime.Version()`, `runtime.GOOS`, and `runtime.GOARCH`.
- `last_run` should capture the execution context that generated or last refreshed observations: command, extra args, worker count, `first-is-1`, `--rspec-split`, `--dry-run`, selection metadata, and start time.
- Focused file:line runs may update example observations, but they must not overwrite the default full-file aggregate.
- Normal grouping reads file-level `runtime_seconds`.
- RSpec splitting reads each cached example entry's `line_number`.
- Minitest and passthrough jobs can store file-level entries without example arrays.
- No backward-compatibility shim is required. If an old v1 cache is present, ignore it and regenerate v2 data.

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

The runtime cache v2 and formatter metadata changes are not gated by this flag.

## Safety Rules

- Default grouping behavior remains file-level.
- Runtime cache v2 is written for normal runs regardless of `--rspec-split`.
- Splitting applies only when `RspecSplit == true`, the selected job has `Framework == "rspec"`, and worker count is greater than 1.
- Splitting requires v2 runtime data with fresh source metadata and example lines for the file.
- Missing or stale example data means current file-level grouping.
- No regex/source-code fallback.
- No RSpec subprocess for discovery during `plur --dry-run`.
- Focused runs and tag runs must not clobber default full-file runtime aggregates.

## File Responsibilities

Create:
- `runtime_cache.go`: v2 persisted data model, load/save, freshness checks, run selection classification.
- `runtime_cache_test.go`: cache shape, v1 ignore behavior, selection behavior, freshness behavior.
- `rspec_line_splitter.go`: pure split decisions and focused target generation.
- `rspec_line_splitter_test.go`: pure behavior around thresholds, chunking, and fallbacks.

Modify:
- `framework/rspec/formatter.rb`: align example metadata with RSpec's JSON formatter path and stop parsing line numbers from location strings.
- `framework/rspec/json_output.go`: extend existing structs for formatter-emitted selected examples.
- `framework/rspec/parser.go`: consume selected-example and per-example metadata messages.
- `types/notifications.go`: extend existing notification structs with selected example metadata.
- `test_collector.go`: retain selected example metadata alongside test notifications.
- `runtime_tracker.go`: replace map-only persistence with runtime cache v2 and expose file-runtime data for grouping.
- `runner.go`: save v2 runtime data after workers finish and pass v2 data to grouping.
- `grouper.go`: use file-level v2 runtimes for existing grouping and example lines only when `RspecSplit == true`.
- `main.go`, `config/config.go`, `main_test.go`: add the experimental boolean flag.
- Formatter, parser, runtime tracker, grouper, runner, and integration specs.
- `docs/usage.md`: document runtime cache behavior, pitfalls, and the experimental flag.

---

## Task 1: Define Runtime Cache V2

**Files:** `runtime_cache.go`, `runtime_cache_test.go`, `runtime_tracker.go`, `runtime_tracker_test.go`

- [ ] Add a versioned runtime cache data model based on the critical shape above.
- [ ] Ignore old `map[string]float64` cache files rather than migrating them.
- [ ] Expose a small method that returns `map[string]float64` for normal file-level grouping.
- [ ] Add source freshness helpers based on absolute path, `mtime_unix_nano`, and `size_bytes`.
- [ ] Add run selection classification for `default`, `focused`, `tagged`, and `custom_args` runs.
- [ ] Use `example.id` as the canonical key for persisted RSpec example entries.
- [ ] Persist producer metadata from existing build info and Go runtime info.
- [ ] Persist `last_run.selection` metadata from the actual Plur invocation, including whether the run is allowed to update full-file aggregates.
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

**Files:** `framework/rspec/parser.go`, `types/notifications.go`, `test_collector.go`

- [ ] Extend existing Go structs to match the formatter's enriched example fields.
- [ ] Normalize `./spec/foo_spec.rb` and `spec/foo_spec.rb` to the same project-relative path.
- [ ] Preserve per-example runtime from existing pass/fail/pending rows.
- [ ] Make `TestCollector` expose selected examples and executed test notifications to the runtime tracker.
- [ ] Verify parser and collector behavior with focused Go tests.

## Task 4: Save Useful Runtime Data for Normal Runs

**Files:** `runtime_tracker.go`, `runner.go`, `spec/integration/spec/runtime_tracking_spec.rb`

- [ ] Save v2 runtime cache entries after normal runs.
- [ ] Update full-file aggregates only for aggregate-eligible default/full-file runs.
- [ ] Preserve prior default file aggregates when a focused run only covers a subset of a file.
- [ ] Merge selected example observations by RSpec `example.id`; incomplete runs must not prune examples missing from that run.
- [ ] Mark `example_index_complete` only after aggregate-eligible default/full-file runs with fresh source metadata.
- [ ] Keep existing runtime-based grouping behavior working through the v2 file-level runtime view.

Concrete success criteria:
- A normal `plur spec/calculator_spec.rb` run writes one v2 runtime file under `$PLUR_HOME/runtime`.
- The v2 file has `schema_version: 2`.
- The RSpec job entry contains `files["spec/calculator_spec.rb"].runtime_seconds > 0`.
- The cache includes `producer.version`, `producer.goos`, `producer.goarch`, and `producer.race_enabled`.
- The RSpec job entry includes `last_run.worker_count`, `last_run.command`, `last_run.extra_args`, `last_run.rspec_split`, and `last_run.selection.kind`.
- The same file entry contains an `examples` object keyed by RSpec `example.id`, with at least one entry containing `line_number` and `runtime_seconds`.
- A second run logs `Using runtime-based grouped execution`.
- A focused `spec/calculator_spec.rb:<line>` run does not overwrite the default full-file `runtime_seconds`.

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

**Files:** `rspec_line_splitter.go`, `rspec_line_splitter_test.go`

- [ ] Define a pure function that receives file path, historical runtime, worker count, target group runtime, and exact example lines.
- [ ] Return the original file path unchanged when the file is not a long pole or has too few exact lines.
- [ ] Split only files whose historical runtime is large enough to distort worker balance. A reasonable first rule is: split files whose runtime is greater than the target per-worker runtime budget.
- [ ] Produce focused targets like:

```text
spec/slow_spec.rb:12:38:91
```

- [ ] Keep chunk count bounded by worker count and example count.
- [ ] Make chunking deterministic so repeated runs produce stable commands.
- [ ] Verify the pure splitter with table-driven tests.

## Task 7: Expand Runtime Grouping When Enabled

**Files:** `grouper.go`, `grouper_test.go`, `runner.go`, `runner_test.go`

- [ ] Keep the old grouping behavior when `RspecSplit` is false.
- [ ] When `RspecSplit == true` for an RSpec job, read file runtimes and example lines from runtime cache v2.
- [ ] Expand long-pole files into focused targets before existing longest-processing-time grouping.
- [ ] Assign each generated target an estimated runtime of `original_file_runtime / chunk_count` for grouping balance only.
- [ ] Do not persist runtime entries for generated `file:line` targets.
- [ ] Log debug reasons when splitting is skipped.
- [ ] Verify:

```bash
go test -mod=mod . -run 'TestGroupSpecFilesByRuntime|TestRunner|TestRspecSplit'
```

## Task 8: Add Integration Coverage

**Files:** likely `spec/integration/spec/runtime_tracking_spec.rb`, `spec/integration/spec/rspec_args_spec.rb`, fixture specs as needed.

- [ ] Assert normal RSpec runs write v2 runtime data with file aggregates and selected example lines.
- [ ] Assert existing runtime grouping still balances slow files from v2 file aggregates.
- [ ] Assert focused file:line runs do not overwrite default full-file runtime aggregates.
- [ ] Assert `plur --rspec-split -n 4 --dry-run` uses cached focused targets and does not invoke RSpec.
- [ ] Assert a real `plur --rspec-split -n 4` run executes focused `file:line` targets and passes after cache exists.
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

- [ ] Run agent-driven QA against Plur itself, RuboCop, and at least one other real-world RSpec project.
- [ ] For each real-world project, warm the v2 runtime cache, run with `--rspec-split` off, then run with `--rspec-split` on using the same worker count.
- [ ] Record whether any suites hit known pitfalls such as `before(:all)`, dynamic examples, or custom RSpec DSLs.
- [ ] Benchmark `--rspec-split` off vs on after cache warmup, and record wall-time, slowest worker, and command count.
- [ ] Treat correctness failures as blockers; treat benchmark regressions as reasons to keep the feature behind the experimental flag.

## Task 9: Document Runtime Cache V2 and Experimental Split

**Files:** `docs/usage.md`

- [ ] Update runtime tracking docs to describe the v2 cache at a high level.
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

- Normal runtime tracking writes v2 data by default with no `--rspec-split` flag.
- Existing runtime-based file grouping still works from v2 file aggregates.
- RSpec formatter output aligns with RSpec's JSON formatter data source and no longer depends on parsing line numbers out of location strings.
- RSpec selected example metadata and per-example runtime details live in the main runtime cache.
- Focused or tag-filtered runs do not corrupt default full-file runtime aggregates.
- `plur --rspec-split` can split a long RSpec file into focused line targets when v2 runtime data has fresh example lines.
- `plur --dry-run --rspec-split` never shells out to RSpec.
- Real-project QA has covered Plur, RuboCop, and at least one other RSpec project.
- Benchmarks compare `--rspec-split` off vs on after cache warmup.
- Docs clearly mark the split flag experimental and document known pitfalls.
