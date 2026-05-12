# RSpec Split Specs Experimental Implementation Plan

> **For implementation workers:** Implement this task-by-task. Keep each task small, verify before moving on, and commit after each completed task.

**Goal:** Land the autoresearch RSpec long-spec splitting win behind an explicit experimental flag so it can be tested in mainline without changing default behavior.

**Architecture:** Keep Plur's current file-level runtime grouping as the default. When `--rspec-split` or `PLUR_RSPEC_SPLIT=1` is set for an RSpec job, split historically long-running spec files into focused `file:line:line` targets using an exact example index captured by Plur's own RSpec formatter during prior normal runs. If the index is missing, stale, or incompatible with the current selector args, fall back to file-level grouping and refresh the index as the run executes.

**Tech Stack:** Go CLI with Kong, existing Plur runtime data, Plur's RSpec JSON rows formatter, RSpec focused line execution, `$PLUR_HOME` cache.

---

## Source Material

Use the `autoresearch` branch as reference only. Do not cherry-pick it wholesale.

Adapt the useful ideas:
- Converting a long spec file into multiple RSpec focused targets.
- Expanding long-pole files before existing runtime-based bin packing.
- Caching exact example line metadata.

Do not bring over:
- Worker count default changes.
- Rails setup changes from the separate Rails branch.
- Regex/source-code fallback for example lines.
- Always-on splitting.
- A separate RSpec JSON dry-run discovery command for the first implementation.

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

Use `metadata[:line_number]` for line data, not string parsing. Keep `location` and `location_rerun_argument` in the JSON for debugging and future-proofing.

The tradeoff is intentional: a project may need one normal run to populate the example index before `--rspec-split` can split that file. That is acceptable for the experimental version because it avoids extra RSpec startup and avoids running suite code during Plur `--dry-run`.

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

First iteration should document only the CLI flag and environment variable. If Kong's TOML loader makes a config-file key work automatically, treat that as incidental until we intentionally document it.

## Safety Rules

- Default behavior is unchanged.
- Splitting applies only to selected jobs with `Framework == "rspec"`.
- Splitting is disabled in serial mode or when worker count is 1.
- Splitting requires runtime data and a fresh formatter-captured example index.
- Missing or stale example-index cache means current file-level grouping; the run may refresh the cache for next time.
- No regex/source-code fallback.
- No RSpec subprocess for discovery during `plur --dry-run`.
- No splitting when unsupported passthrough args are present.
- For Plur-owned `--tag` filters, include a selector fingerprint in the cache key so tag-specific runs do not reuse a full-suite example index.
- Runtime tracking stays keyed by original spec file path. RSpec formatter events should continue to report `file_path` without line selectors.

Critical dry-run rule:

```go
if cfg.DryRun {
	// Dry-run plans only from existing cache. It never launches RSpec to discover lines.
	return cachedIndexOnly()
}
```

## File Responsibilities

Create:
- `rspec_example_index.go`: cache model, freshness checks, selector fingerprinting, cache load/save.
- `rspec_example_index_test.go`: parser/cache/freshness/selector tests.
- `rspec_line_splitter.go`: pure split decisions and focused target generation.
- `rspec_line_splitter_test.go`: pure behavior around thresholds, chunking, and fallbacks.

Modify:
- `main.go`: add the experimental CLI/env flag and populate global config.
- `config/config.go`: carry the boolean through `GlobalConfig`.
- `main_test.go`: cover flag metadata and parsing.
- `framework/rspec/formatter.rb`: emit exact selected example index and stop parsing line numbers from location strings.
- `framework/rspec/json_output.go`: add structs for the formatter-emitted example index.
- `framework/rspec/parser.go`: consume example index messages.
- `types/notifications.go`: add a notification or data carrier for RSpec example index rows.
- `runner.go`: collect index rows from workers, save cache, and pass cached index to grouping.
- `grouper.go`: add optional RSpec split expansion before runtime grouping.
- `grouper_test.go`, `runner_test.go`, formatter/parser specs: cover the new data flow.
- `docs/usage.md`: document the experimental flag and fallback rules.

---

## Task 1: Add the Boolean Experimental Flag

**Files:** `main.go`, `config/config.go`, `main_test.go`

- [ ] Add `RspecSplit bool` to `PlurCLI` using the critical field definition above.
- [ ] Add `RspecSplit bool` to `config.GlobalConfig`.
- [ ] Populate `GlobalConfig.RspecSplit` during CLI config construction.
- [ ] Test that the flag is off by default, enabled by `--rspec-split`, enabled by `PLUR_RSPEC_SPLIT=1`, and marked `EXPERIMENTAL` in help metadata.
- [ ] Verify with:

```bash
go test -mod=mod . -run 'TestRspecSplit'
```

## Task 2: Emit Exact Example Index From the Formatter

**Files:** `framework/rspec/formatter.rb`, `spec/integration/spec/json_rows_formatter_spec.rb`

- [ ] Add an `example_index` JSON row during formatter `start`.
- [ ] Build the row from `RSpec.world.example_groups.flat_map(&:descendant_filtered_examples)`.
- [ ] Include `file_path`, `absolute_file_path`, `line_number`, `location`, `location_rerun_argument`, and `scoped_id`.
- [ ] Change existing group/example line extraction to prefer metadata line numbers instead of `location.split(":").last.to_i`.
- [ ] Keep the existing `load_summary` row so current parser behavior remains intact.
- [ ] Verify formatter output with focused Ruby specs:

```bash
mise exec -- bundle exec rspec spec/integration/spec/json_rows_formatter_spec.rb
```

## Task 3: Parse and Cache Example Index Rows

**Files:** `framework/rspec/json_output.go`, `framework/rspec/parser.go`, `types/notifications.go`, `rspec_example_index.go`, `rspec_example_index_test.go`

- [ ] Add Go structs matching the formatter's `example_index` row.
- [ ] Parse `example_index` rows into an internal notification or data carrier.
- [ ] Normalize `./spec/foo_spec.rb` and `spec/foo_spec.rb` to the same project-relative path.
- [ ] Persist index data under `$PLUR_HOME/cache/rspec-example-index`.
- [ ] Include schema version, absolute path, file size, file mtime, selector fingerprint, and the sorted exact line numbers.

Critical cache contract:

```json
{
  "schema_version": 1,
  "abs_path": "/repo/spec/slow_spec.rb",
  "selector_fingerprint": "default",
  "mtime_unix_nano": 1778610000000000000,
  "size_bytes": 12345,
  "lines": [12, 38, 91]
}
```

Cache freshness is based on schema version, selector fingerprint, absolute path, `mtime_unix_nano`, and `size_bytes`. The cache filename should be derived from a stable hash of absolute path plus selector fingerprint.

## Task 4: Add Pure Split Decisions

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

## Task 5: Expand Runtime Grouping When Enabled

**Files:** `grouper.go`, `grouper_test.go`, `runner.go`, `runner_test.go`

- [ ] Add an options struct for runtime grouping so the existing function can remain readable.
- [ ] Keep the old grouping path intact when `RspecSplit` is false.
- [ ] For eligible RSpec runs, load the cached example index before grouping.
- [ ] Expand long-pole files into focused targets before existing longest-processing-time grouping.
- [ ] Assign each generated target an estimated runtime of `original_file_runtime / chunk_count` for grouping balance only.
- [ ] Preserve runtime tracking by original file path; do not persist runtime entries for `file:line` targets.
- [ ] Save formatter-emitted example index rows after workers finish so the next run can split.
- [ ] Log debug reasons when splitting is skipped.
- [ ] Verify:

```bash
go test -mod=mod . -run 'TestGroupSpecFilesByRuntime|TestRunner|TestRSpecExampleIndex'
```

## Task 6: Add Integration Coverage

**Files:** likely `spec/integration/spec/runtime_tracking_spec.rb`, `spec/integration/spec/rspec_args_spec.rb`, fixture specs as needed.

- [ ] Create or reuse a fixture with one historically slow spec file containing multiple examples.
- [ ] Seed runtime data so the file is clearly the long pole.
- [ ] Assert default Plur command keeps file-level targets.
- [ ] Assert `plur --rspec-split -n 4 --dry-run` uses cached focused targets and does not invoke RSpec.
- [ ] Assert a real `plur --rspec-split -n 4` run executes focused `file:line` targets and passes after cache exists.
- [ ] Assert unsupported passthrough args fall back to file-level grouping.
- [ ] Assert Plur-owned `--tag` filters use a distinct selector fingerprint.
- [ ] Verify with:

```bash
bin/rspec spec/integration/spec/runtime_tracking_spec.rb spec/integration/spec/rspec_args_spec.rb
```

## Task 7: Document the Experimental Flag

**Files:** `docs/usage.md`

- [ ] Add a short experimental section for `--rspec-split`.
- [ ] State that the feature is opt-in, RSpec-only, runtime-data-driven, and cache-driven.
- [ ] Mention that a first run may populate the example index before future runs split long files.
- [ ] Include the CLI and environment examples from this plan.
- [ ] Avoid promising stable behavior while it is experimental.

## Task 8: Final Verification

- [ ] Run focused Go tests for the new code.
- [ ] Run focused formatter specs through mise and Bundler.
- [ ] Run focused integration specs for RSpec runtime grouping and args.
- [ ] Run the standard project verification:

```bash
bin/rake test:go
bin/rake test
```

- [ ] Run `git diff --check`.
- [ ] Commit the finished implementation in small logical commits.

## Success Criteria

- `plur` without `--rspec-split` behaves exactly as it does today.
- Normal RSpec runs emit and cache exact selected example line metadata through Plur's formatter.
- `plur --rspec-split` can split a long RSpec file into focused line targets when runtime data and cached exact example lines are available.
- `plur --dry-run --rspec-split` never shells out to RSpec.
- Unsupported args, missing runtime data, stale cache, invalid cache, and incompatible selector fingerprints all fall back to file-level grouping.
- Docs clearly mark the flag experimental.
