# Backspin v0.10.0 Migration Review

Review of the `chore/backspin-v0-10-migration` branch. Covers simplicity, correctness, coverage gaps, and upstream feedback for backspin.

## Correctness

No issues found. The migration is mechanically correct:

* `filter:` with default `filter_on: :both` correctly replaces the old `matcher:` callbacks. Normalization now applies to both recorded and actual snapshots before comparison, eliminating the need for separate equality logic.
* `mode: :verify` removal is correct. Backspin v0.10 auto-detects mode based on whether the YAML file exists. The cross-command pattern (record rspec first, then verify plur) works because the first `Backspin.run` call creates the file and the second auto-enters verify mode.
* All 8 normalize helpers are idempotent, which is required because the filter runs twice on recorded data: once at save time (transforming what gets written to YAML), and again at comparison time (normalizing the on-disk data alongside the live run). See "Filter idempotency" below for why this matters upstream.

## Simplicity

### Consolidate `single_failure_golden_spec.rb`

`spec/integration/shared/single_failure_golden_spec.rb` has 3 tests that are near-duplicates:

* "shows filtered backtrace same as rspec using Backspin" (line 30)
* "matches rspec colorized output using Backspin verification" (line 57)
* "uses Backspin snapshot result fields in verify mode" (line 82)

All three run the same commands with the same filter. The first two have nearly identical color-code assertions. The third tests `result.verified?` and `result.expected.stdout` — which is testing backspin's result API, not plur behavior.

Suggest merging the first two into one test, and either removing the third or keeping it only if backspin lacks its own coverage of `verified?`.

### `normalize_doctor_output` is large but correct

`spec/integration/plur_doctor/doctor_spec.rb:14-75` has ~30 gsub calls replacing every dynamic value with a placeholder. The function works, but at that point the golden test is checking structural layout rather than content. The non-backspin test ("includes all expected sections", line 111) with its explicit section list is a better structural guard.

This is a signal for backspin: a template-based or structural comparison mode would help diagnostic commands with all-dynamic output.

### Minor DRY opportunity

Version-line normalization (`.gsub(/^plur version version=.*$/, ...)`) and formatter-path normalization (`.gsub(%r{-r\s+\S+/formatter/json_rows_formatter\.rb}, ...)`) appear in 3 dry-run specs:

* `spec/integration/shared/rspec_args_spec.rb`
* `spec/integration/plur_spec/framework_output_spec.rb`
* `spec/integration/shared/turbo_tests_migration_spec.rb`

Low priority to extract. Per-file helpers are explicit and easy to follow. Only extract if a fourth spec needs the same normalizations.

## Gaps in Coverage

* **No parallel execution snapshots**: Snapshot tests cover dry-run, serial, and error outputs. No snapshot tests exist for actual parallel execution output. This is a pre-existing gap, not introduced by this branch.
* **Minitest cross-command comparison is pending**: `minitest_integration_spec.rb:129` is `pending("plur output does not match raw minitest output yet")`. Intentional — tracked separately.

## Feedback for Backspin Author

These are upstream improvements that would make real-world CLI snapshot testing simpler, ordered by impact.

### 1. Document the filter idempotency requirement

Since `filter:` runs at record time (transforming data before saving to YAML) AND again at verify time on the already-filtered on-disk data, the filter *must* be idempotent. If someone writes a non-idempotent filter (e.g., appending a suffix, incrementing a counter), comparisons silently corrupt.

This should be documented prominently in the filter section of the README, ideally with a "good/bad" example.

### 2. Add `ignore_fields:` / `compare_fields:`

This is the single highest-value improvement for CLI snapshot suites. In plur's cross-command comparisons (record rspec output, verify plur output), we have to normalize `args` and `stderr` to sentinels just to make the comparison focus on `stdout`:

```ruby
def normalize_single_failure_snapshot(snapshot)
  snapshot.merge(
    "args" => ["[SINGLE_FAILURE_COMMAND]"],  # only needed to suppress args diff
    "stdout" => make_summary_line_consistent(snapshot.fetch("stdout", "")).strip,
    "stderr" => ""                            # only needed to suppress stderr diff
  )
end
```

With `ignore_fields:`, this becomes:

```ruby
Backspin.run(command, name: "test",
  filter: ->(s) { s.merge("stdout" => normalize(s["stdout"])) },
  ignore_fields: %w[args stderr]
)
```

### 3. Built-in field-level filter helpers

The most common filter pattern is "gsub a regex in a specific field":

```ruby
# Current: manual hash manipulation
filter: ->(s) { s.merge("stderr" => s.fetch("stderr", "").gsub(/pattern/, "[TOKEN]")) }

# Proposed: declarative helper
filter: Backspin::Filters.gsub("stderr", /pattern/, "[TOKEN]")
```

### 4. Filter composition

When multiple normalizations are needed, a composition helper would be cleaner than a single lambda with multiple transforms:

```ruby
# Proposed
filter: Backspin.chain(
  Backspin::Filters.gsub("stderr", /version=\S+/, "version=[VERSION]"),
  Backspin::Filters.gsub("stderr", %r{-r\s+\S+/formatter\.rb}, "-r [FORMATTER]"),
  Backspin::Filters.sort_lines("stderr", /Worker \d+:/)
)
```

### 5. `sort_lines_matching` helper

`turbo_tests_migration_spec.rb` sorts worker lines because parallel worker ordering is nondeterministic. A built-in `sort_lines_matching(field, pattern)` filter would handle this niche case cleanly.

### 6. Migration guidance: matcher -> filter

The CHANGELOG notes the new `filter_on:` parameter, but a dedicated "Migrating from matcher to filter" section with before/after examples would smooth adoption. The key insight — "if your matcher was just normalizing both sides and comparing, use filter instead" — deserves its own heading.
