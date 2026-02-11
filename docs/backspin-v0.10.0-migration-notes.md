# Backspin v0.10.0 Migration Notes

Updated on: 2026-02-11

## Dependency Updates

- Updated `Gemfile` to `gem "backspin", "~> 0.10.0", require: false`.
- Updated `Gemfile.lock` to `backspin (0.10.0)`.
- Updated appraisal lockfiles:
  - `gemfiles/rspec_3.13.1.gemfile.lock`
  - `gemfiles/rspec_3.13.2.gemfile.lock`

## Upstream Review (v0.9.0 -> v0.10.0)

Reviewed:
- Changelog: https://github.com/rsanheim/backspin/blob/main/CHANGELOG.md
- Compare: https://github.com/rsanheim/backspin/compare/v0.9.0...v0.10.0

Key changes in `0.10.0`:

1. `filter_on` added to `Backspin.run`/`Backspin.capture`.
   - Default is `:both`.
   - Optional `:record` preserves old record-only filtering behavior.
2. `filter` behavior changed:
   - Filter now runs during verify comparisons/diffs by default (`filter_on: :both`).
3. Matcher safety improvements:
   - Matcher callbacks receive mutable copies, so in-place mutations do not mutate snapshots.
4. Snapshot serialization is immutable:
   - `Snapshot#to_h` now returns frozen data created at initialization.

## Plur Usage Review and Simplification

### What we changed

We moved most normalization logic from `matcher:` lambdas into `filter:` snapshot canonicalization.

This removes duplicated `record`/`verify` normalization code and uses the new default behavior (`filter_on: :both`) to normalize both recorded and actual snapshots before matching/diffing.

### Simplification pattern now used

1. Define a snapshot filter helper in each spec:
   - normalize volatile stdout/stderr data
   - optionally normalize `args` when comparing different commands with the same record name
2. Call `Backspin.run(..., filter: ->(snapshot) { ... })`
3. Keep explicit assertions on `result.actual` for behavior we still want to check directly.

### Updated call sites

1. `spec/integration/plur_doctor/doctor_spec.rb`
   - Replaced stdout normalization matcher with `filter`.
2. `spec/integration/plur_spec/change_dir_config_spec.rb`
   - Replaced stderr matcher with `filter` that normalizes tmp paths.
3. `spec/integration/plur_spec/framework_output_spec.rb`
   - Replaced dry-run stderr matcher with `filter`.
4. `spec/integration/shared/rspec_args_spec.rb`
   - Replaced dry-run stderr matcher with `filter`.
5. `spec/integration/shared/turbo_tests_migration_spec.rb`
   - Replaced dry-run stderr matcher with `filter`.
6. `spec/integration/plur_spec/pending_output_spec.rb`
   - Replaced dual matcher setup with `filter`.
   - Normalized `args`/`stderr` so RSpec baseline vs plur comparison focuses on normalized stdout contract.
7. `spec/integration/shared/single_failure_golden_spec.rb`
   - Replaced matchers with `filter`.
   - Normalized `args`/`stderr` for RSpec baseline vs plur comparison.
8. `spec/integration/plur_spec/minitest_integration_spec.rb`
   - Replaced matchers with `filter` in grouped minitest snapshot flows.
   - Normalized `args` for cross-command comparison path.

## Verification Run

Executed after migration:

- `bin/rspec spec/integration/plur_doctor/doctor_spec.rb spec/integration/plur_spec/change_dir_config_spec.rb spec/integration/plur_spec/framework_output_spec.rb spec/integration/plur_spec/minitest_integration_spec.rb spec/integration/plur_spec/pending_output_spec.rb spec/integration/shared/rspec_args_spec.rb spec/integration/shared/single_failure_golden_spec.rb spec/integration/shared/turbo_tests_migration_spec.rb`
- `bin/rake test`
- `bin/rake`

Result: passing (with existing expected pending examples only).

## Notes

- We intentionally kept cross-command golden comparisons where they provide value (RSpec baseline vs plur behavior checks).
- No snapshot format migration was required for this update (still `format_version: 4.0`).
- If record-only filtering is needed in future, pass `filter_on: :record` explicitly.

## Suggested Backspin Improvements

These are upstream improvements that would simplify real-world CLI snapshot suites further:

1. Built-in field-level filter helpers
   - Example: `Backspin::Filters.gsub("stdout", /regex/, "[TOKEN]")`
   - Value: less custom filter boilerplate in test suites.
2. Filter chaining/composition helpers
   - Example: `filter: Backspin.filter_chain(normalize_paths, normalize_timing, strip_banner)`
   - Value: cleaner reuse across many specs without ad-hoc helper methods.
3. Verify-scope comparison configuration
   - Example: `compare_fields: %w[stdout status]` or `ignore_fields: %w[args stderr]`
   - Value: avoids mutating/normalizing unrelated fields just to compare cross-command behavior.
4. Better diff ergonomics for large stdout snapshots
   - Show context windows and a summarized field change table before full diff.
   - Value: easier debugging for long CLI outputs.
5. Migration guidance docs for matcher -> filter
   - Dedicated section with before/after examples for `filter_on: :both`.
   - Value: smoother adoption of `0.10.x` defaults.

## Recommended Team Conventions (Plur)

To keep Backspin usage consistent in this repo:

1. Prefer `filter:` for canonicalization.
2. Use `matcher:` only for truly semantic comparisons that cannot be represented as normalization.
3. Keep one small `normalize_*_snapshot` helper per spec file.
4. In cross-command comparisons, normalize `args` explicitly so contracts focus on output semantics.
5. Keep direct assertions on `result.actual` for critical user-visible guarantees.
