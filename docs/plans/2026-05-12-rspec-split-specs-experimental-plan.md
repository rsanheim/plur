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
- selector fingerprints so tag/focused runs do not corrupt full-file data

## Key RSpec Finding

Plur's formatter already runs after RSpec has loaded spec files and applied filters. At formatter `start`, `RSpec.world.example_groups.flat_map(&:descendant_filtered_examples)` exposes the exact examples selected for the current process.

The useful fields are richer than the current `location.split(":").last.to_i` approach:

```ruby
{
  file_path: example.metadata[:file_path],
  absolute_file_path: example.metadata[:absolute_file_path],
  line_number: example.metadata[:line_number],
  location: example.location,
  location_rerun_argument: example.location_rerun_argument,
  scoped_id: example.metadata[:scoped_id]
}
```

Use `metadata[:line_number]` for line data, not string parsing. Keep `location`, `location_rerun_argument`, and `scoped_id` in the JSON for diagnostics and future grouping choices.

## Runtime Cache V2

The exact Go structs can be refined during implementation, but the persisted shape should be versioned and object-based rather than `map[string]float64`.

Critical shape:

```json
{
  "schema_version": 2,
  "generated_at": "2026-05-12T18:30:00Z",
  "project_root": "/repo",
  "jobs": {
    "rspec": {
      "framework": "rspec",
      "selections": {
        "default": {
          "files": {
            "spec/slow_spec.rb": {
              "path": "spec/slow_spec.rb",
              "abs_path": "/repo/spec/slow_spec.rb",
              "mtime_unix_nano": 1778610000000000000,
              "size_bytes": 12345,
              "runtime_seconds": 12.34,
              "example_count": 3,
              "examples": [
                {
                  "line_number": 12,
                  "location": "./spec/slow_spec.rb:12",
                  "location_rerun_argument": "./spec/slow_spec.rb:12",
                  "scoped_id": "1:1",
                  "runtime_seconds": 0.40,
                  "status": "passed"
                }
              ]
            }
          }
        }
      }
    }
  }
}
```

Notes:
- `selections.default` means no Plur-owned selector args and no focused line selection.
- Tag runs should use a selector fingerprint such as `tag:<hash>`.
- Focused file:line runs may update example observations, but they must not overwrite the default full-file aggregate.
- Normal grouping reads file-level `runtime_seconds`.
- RSpec splitting reads the same file entry's `examples[].line_number`.
- Minitest and passthrough jobs can store file-level entries without example arrays.
- No backward-compatibility shim is required. If an old v1 cache is present, ignore it and regenerate v2 data.

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
- Splitting applies only to selected jobs with `Framework == "rspec"`.
- Splitting is disabled in serial mode or when worker count is 1.
- Splitting requires v2 runtime data with fresh source metadata and example lines for the file.
- Missing or stale example data means current file-level grouping.
- No regex/source-code fallback.
- No RSpec subprocess for discovery during `plur --dry-run`.
- Focused runs and tag runs must not clobber default full-file runtime aggregates.

## File Responsibilities

Create:
- `runtime_cache.go`: v2 persisted data model, load/save, freshness checks, selector fingerprinting.
- `runtime_cache_test.go`: cache shape, v1 ignore behavior, selector behavior, freshness behavior.
- `rspec_line_splitter.go`: pure split decisions and focused target generation.
- `rspec_line_splitter_test.go`: pure behavior around thresholds, chunking, and fallbacks.

Modify:
- `framework/rspec/formatter.rb`: always emit selected example metadata and stop parsing line numbers from location strings.
- `framework/rspec/json_output.go`: add structs for formatter-emitted selected examples.
- `framework/rspec/parser.go`: consume selected-example and per-example metadata messages.
- `types/notifications.go`: carry selected example metadata through parser/collector.
- `test_collector.go`: retain selected example metadata alongside test notifications.
- `runtime_tracker.go`: replace map-only persistence with runtime cache v2 and expose file-runtime data for grouping.
- `runner.go`: save v2 runtime data after workers finish and pass v2 data to grouping.
- `grouper.go`: use file-level v2 runtimes for existing grouping and example lines for optional splitting.
- `main.go`, `config/config.go`, `main_test.go`: add the experimental boolean flag.
- Formatter, parser, runtime tracker, grouper, runner, and integration specs.
- `docs/usage.md`: document runtime cache behavior and the experimental flag.

---

## Task 1: Define Runtime Cache V2

**Files:** `runtime_cache.go`, `runtime_cache_test.go`, `runtime_tracker.go`, `runtime_tracker_test.go`

- [ ] Add a versioned runtime cache data model based on the critical shape above.
- [ ] Ignore old `map[string]float64` cache files rather than migrating them.
- [ ] Expose a small method that returns `map[string]float64` for normal file-level grouping.
- [ ] Add source freshness helpers based on absolute path, `mtime_unix_nano`, and `size_bytes`.
- [ ] Add selector fingerprinting for default runs and Plur-owned tag runs.
- [ ] Verify with focused Go tests:

```bash
go test -mod=mod . -run 'TestRuntimeCache|TestRuntimeTracker'
```

## Task 2: Emit RSpec Metadata by Default

**Files:** `framework/rspec/formatter.rb`, `spec/integration/spec/json_rows_formatter_spec.rb`

- [ ] Add a formatter row for selected examples during `start`.
- [ ] Build the row from `RSpec.world.example_groups.flat_map(&:descendant_filtered_examples)`.
- [ ] Include `file_path`, `absolute_file_path`, `line_number`, `location`, `location_rerun_argument`, and `scoped_id`.
- [ ] Change existing group/example line extraction to prefer `metadata[:line_number]` instead of `location.split(":").last.to_i`.
- [ ] Keep the existing progress, failure, pending, and summary rows compatible.
- [ ] Verify with:

```bash
mise exec -- bundle exec rspec spec/integration/spec/json_rows_formatter_spec.rb
```

## Task 3: Parse and Collect RSpec Metadata

**Files:** `framework/rspec/json_output.go`, `framework/rspec/parser.go`, `types/notifications.go`, `test_collector.go`

- [ ] Add Go structs matching the formatter's selected-example row.
- [ ] Normalize `./spec/foo_spec.rb` and `spec/foo_spec.rb` to the same project-relative path.
- [ ] Preserve per-example runtime from existing pass/fail/pending rows.
- [ ] Make `TestCollector` expose selected examples and executed test notifications to the runtime tracker.
- [ ] Verify parser and collector behavior with focused Go tests.

## Task 4: Save Useful Runtime Data for Normal Runs

**Files:** `runtime_tracker.go`, `runner.go`, `spec/integration/spec/runtime_tracking_spec.rb`

- [ ] Save v2 runtime cache entries after normal runs.
- [ ] Update default full-file aggregates only for runs that selected whole files under the default selector.
- [ ] Preserve prior default file aggregates when a focused run only covers a subset of a file.
- [ ] Store selected example lines even when runtimes are incomplete, as long as the source metadata is fresh.
- [ ] Keep existing runtime-based grouping behavior working through the v2 file-level runtime view.
- [ ] Verify with:

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
- [ ] For eligible RSpec runs, read file runtimes and example lines from runtime cache v2.
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
- [ ] Assert Plur-owned `--tag` filters use a distinct selector fingerprint.
- [ ] Verify with:

```bash
bin/rspec spec/integration/spec/runtime_tracking_spec.rb spec/integration/spec/rspec_args_spec.rb
```

## Task 9: Document Runtime Cache V2 and Experimental Split

**Files:** `docs/usage.md`

- [ ] Update runtime tracking docs to describe the v2 cache at a high level.
- [ ] Add a short experimental section for `--rspec-split`.
- [ ] State that the feature is opt-in, RSpec-only, runtime-data-driven, and cache-driven.
- [ ] Mention that normal runs populate the metadata future split runs need.
- [ ] Include the CLI and environment examples from this plan.
- [ ] Avoid promising stable split behavior while it is experimental.

## Task 10: Final Verification

- [ ] Run focused Go tests for the new code.
- [ ] Run focused formatter specs through mise and Bundler.
- [ ] Run focused integration specs for runtime tracking and RSpec args.
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
- RSpec formatter output no longer depends on parsing line numbers out of location strings.
- RSpec selected example metadata and per-example runtime details live in the main runtime cache.
- Focused or tag-filtered runs do not corrupt default full-file runtime aggregates.
- `plur --rspec-split` can split a long RSpec file into focused line targets when v2 runtime data has fresh example lines.
- `plur --dry-run --rspec-split` never shells out to RSpec.
- Docs clearly mark the split flag experimental.
