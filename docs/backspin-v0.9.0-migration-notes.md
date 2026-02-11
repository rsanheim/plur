# Backspin v0.9.0 Migration Notes

## What Changed in Plur

Plur now uses `backspin` `~> 0.9.0` in the RSpec suite.

Updated all existing Backspin usage for the new API:

- Replaced `Backspin.run!` with `Backspin.run(command, name: ...)`
- Removed `:playback` usage (no longer supported)
- Switched result access from removed convenience accessors (`result.stdout`) to snapshot fields (`result.actual.stdout`)
- Rewrote old block-returning-`Open3.capture3` usage into explicit command snapshots
- Regenerated records to `format_version: 4.0` with single `snapshot` payloads
- Kept single-call usage for same-command snapshots; retained two-call flows only where we intentionally compare different commands (RSpec baseline vs plur output)

## Backspin Feedback / Snags

### Tricky parts encountered

1. API shape changed in a meaningful way:
   - Old block style in plur tests returned `Open3.capture3` tuples.
   - New block capture mode captures process stdout/stderr, not returned tuples.
   - Fix required moving to explicit command arrays in `Backspin.run`.

2. Record format migration is strict:
   - v2/v3 records are rejected by v0.9.0.
   - Existing `fixtures/backspin/*.yml` had to be regenerated.

3. Removed playback mode required test redesign:
   - Prior test demonstrating `mode: :playback` was replaced with explicit `:verify` + snapshot contract assertions.

4. Snapshot volatility in CLI tests:
   - Dry-run output includes version/path data.
   - Needed targeted normalizers in matcher lambdas to avoid noisy failures.

### Feedback for backspin itself (for later)

1. A migration helper for old record formats would reduce adoption friction (v2/v3 -> v4).
2. A first-class "ignore fields" matcher helper (e.g. ignore timestamp/path patterns) would reduce custom matcher boilerplate.
3. A replacement for removed playback mode (or an explicit rationale section in docs) would help users migrating test intent around speed-focused fixtures.

## New Backspin Call Sites Added (5)

1. `spec/integration/shared/rspec_args_spec.rb`
   - Snapshot dry-run passthrough formatter command construction.
2. `spec/integration/plur_spec/framework_output_spec.rb`
   - Snapshot dry-run output for non-standard RSpec job config.
3. `spec/integration/plur_spec/change_dir_config_spec.rb`
   - Snapshot `-C nonexistent` error output.
4. `spec/integration/plur_spec/change_dir_config_spec.rb`
   - Snapshot missing `-C` argument error output.
5. `spec/integration/shared/turbo_tests_migration_spec.rb`
   - Snapshot turbo_tests-style tag filtering dry-run output.

## Impact / Usefulness Analysis

1. Better regression detection for CLI command assembly:
   - Worker command shape, flag ordering, and formatter wiring are now checked as full contracts, not just fragments.

2. Better error UX coverage:
   - `-C` failure messaging is now golden-tested, making accidental wording or structure regressions easier to catch.

3. Better migration confidence from turbo_tests semantics:
   - Tag + directory expansion behavior is now captured in one snapshot contract.

4. Tradeoff:
   - Snapshot tests can be brittle with dynamic metadata.
   - Mitigation added: custom normalizers for version and formatter-path churn in relevant specs.

## Follow-up Issues (Backspin)

- `#27` Proposal: extend filter to support verification-time canonicalization
  - https://github.com/rsanheim/backspin/issues/27
- `#28` Preserve `first_recorded_at` metadata for auto re-record workflows
  - https://github.com/rsanheim/backspin/issues/28
- `#29` Support `BACKSPIN_MODE` / `RECORD_MODE` environment overrides for run mode
  - https://github.com/rsanheim/backspin/issues/29
- `#30` Improve verification diff UX (field summary, changed-fields-only, truncation/full mode)
  - https://github.com/rsanheim/backspin/issues/30
