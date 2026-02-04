# RSpec Error Handling + AssertionCount Review (WIP)

## Error vs Failure (RSpec + Minitest)
- **RSpec**
  - **Failure** = failed example (tracked as `failure_count` on `SummaryNotification`).
  - **Error** = error outside of examples (tracked as `errors_outside_of_examples_count`, appended to `totals_line` as “error occurred outside of examples”).
- **Minitest**
  - **Failure** = `Assertion` failures; `StatisticsReporter` counts these as `failures`.
  - **Error** = `UnexpectedError` (uncaught exception); `StatisticsReporter` counts these as `errors`, and `Reportable#error?` checks for `UnexpectedError`.
  - Result codes map to “F” vs “E” based on the failure type.

## Scope
Review how RSpec parsing and error handling works today, and confirm how the new `AssertionCount` / `ErrorCount` fields interact with RSpec. Identify gaps and tests/docs that cover error handling.

## RSpec Parsing Path (Current)
- Formatter: `plur/rspec/formatter.rb` emits JSON rows with:
  - `load_summary` (count + load_time)
  - per-example events (`example_passed`, `example_failed`, `example_pending`)
  - `dump_failures`, `dump_pending`, `dump_summary` (formatted output + summary counts)
- Parser: `plur/rspec/parser.go`
  - `load_summary` -> `SuiteNotification{Event: SuiteStarted, TestCount, LoadTime}`
  - per-example -> `TestCaseNotification` with `TestPassed`, `TestFailed`, or `TestPending`
  - `dump_summary` -> `SuiteNotification{Event: SuiteFinished, TestCount, FailureCount, PendingCount, Duration}`
  - No `AssertionCount` or `ErrorCount` emitted for RSpec
  - No `TestError` events emitted (the enum exists but is unused)

## Error Handling Behavior (Current)
- Worker error vs test failure is decided in `plur/runner.go`:
  - If command error and `ExampleCount == 0` -> `StateError`
  - Else non-zero exit code -> `StateFailed`
- Errors outside examples:
  - RSpec can show: "0 examples, 0 failures, 1 error occurred outside of examples"
  - That string comes from **RSpec’s formatted summary**, captured via `dump_summary`
  - `PrintResults` uses the formatted summary only when **single worker** (`FormattedSummary` present)
  - In multi-worker mode, we fall back to `parser.FormatSummary` which has no error count
- `ErrorCount` is **not** populated anywhere for RSpec today:
  - Formatter emits only `failure_count` and `pending_count`
  - Parser sets only `FailureCount` and `PendingCount`
  - Summary formatting for RSpec ignores `AssertionCount` and `ErrorCount`

## AssertionCount Usage (RSpec)
Verified in code:
- `AssertionCount` is only populated by the **Minitest** parser (`plur/minitest/output_parser.go`)
- RSpec parser never sets it
- RSpec `FormatSummary` ignores it (uses only total examples/failures/pending)
- `AssertionCount` therefore stays zero for RSpec runs

## Tests Covering Error Handling
- `spec/integration/plur_spec/error_handling_spec.rb`
  - Syntax errors and exceptions from the `rspec-errors` fixture
  - Ensures "error occurred outside of examples" is present (single-worker path)
  - Ensures non-zero exit status for failures / exceptions
  - Ensures empty or malformed JSON paths show user-friendly errors

## Docs on Execution Flow
- `docs/architecture/test-processing-flow.md`
  - Describes runner -> parser -> collector -> summary flow
  - Mentions Minitest raw stdout is captured but not streamed

## Gaps / Follow-ups
- RSpec error count is not represented in notifications:
  - No `ErrorCount` in formatter payload or parser notifications
  - Multi-worker summaries lose “errors outside of examples”
- `TestError` event exists but is never emitted
- `runCommand` rebuilds `WorkerResult` and currently drops `AssertionCount` / `ErrorCount`
  - This affects minitest counts in real runs and should be fixed separately

## Questions for Next Steps
- Should the RSpec formatter emit a separate `error_count` (if RSpec exposes it)?
- If not, should we continue treating errors as part of `FailureCount`?
- Should we add a `TestError` event for RSpec (if we can distinguish it)?

## Next Steps (Proposed)
1) **Emit and parse RSpec error counts explicitly**
   - Extend `plur/rspec/formatter.rb` to include `errors_outside_of_examples_count` (from `SummaryNotification`) in the JSON payload for `dump_summary`.
   - Parse it in `plur/rspec/parser.go` and set `SuiteNotification.ErrorCount` for RSpec.
2) **Separate “errors outside of examples” in summaries**
   - Update RSpec summary formatting for multi-worker mode to include the error count (mirroring RSpec’s totals line).
3) **Keep AssertionCount out of RSpec**
   - No formatter or parser support for assertions; keep `AssertionCount` unset for RSpec.
4) **Fix count propagation in `runCommand`**
   - `runCommand` currently rebuilds `WorkerResult` and drops `AssertionCount` / `ErrorCount`; carry those fields through.
5) **Add/adjust tests**
   - Add a test that validates error counts for RSpec in multi-worker mode.
   - Ensure existing error-handling specs still cover syntax errors and load errors as errors.
