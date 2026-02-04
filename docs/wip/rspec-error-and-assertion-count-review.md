# RSpec Error Handling + AssertionCount Review (WIP)

## Error vs Failure (RSpec + Minitest)
- **RSpec**
  - **Failure** = failed example (tracked as `failure_count` on `SummaryNotification`).
  - **Error** = error outside of examples (tracked as `errors_outside_of_examples_count`, appended to `totals_line` as “error occurred outside of examples”).
- **Minitest**
  - **Failure** = `Assertion` failures; `StatisticsReporter` counts these as `failures`.
  - **Error** = `UnexpectedError` (uncaught exception); `StatisticsReporter` counts these as `errors`, and `Reportable#error?` checks for `UnexpectedError`.
  - Result codes map to “F” vs “E” based on the failure type.
  - `FailureCount` tracks failures only; `ErrorCount` tracks errors only (no combined count).

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
  - `dump_summary` -> `SuiteNotification{Event: SuiteFinished, TestCount, FailureCount, ErrorCount, PendingCount, Duration}`
  - `ErrorCount` is populated from `errors_outside_of_examples_count`
  - `AssertionCount` is still not used for RSpec

## Error Handling Behavior (Current)
- Worker error vs test failure is decided in `plur/runner.go`:
  - If command error and `ExampleCount == 0` -> `StateError`
  - Else non-zero exit code -> `StateFailed`
- Errors outside examples:
  - RSpec can show: "0 examples, 0 failures, 1 error occurred outside of examples"
  - That string comes from **RSpec’s formatted summary**, captured via `dump_summary`
  - `PrintResults` uses the formatted summary only when **single worker** (`FormattedSummary` present)
  - In multi-worker mode, `parser.FormatSummary` now includes error count text
- `ErrorCount` is populated for RSpec from `errors_outside_of_examples_count`
- Summary formatting for RSpec now includes error counts in multi-worker mode
- `AssertionCount` remains unused for RSpec

## AssertionCount Usage (RSpec)
Verified in code:
- `AssertionCount` is only populated by the **Minitest** parser (`plur/minitest/output_parser.go`)
- RSpec parser never sets it
- RSpec `FormatSummary` ignores it (uses examples/failures/pending plus `ErrorCount`)
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
- **Fixed**: multi-worker summaries now include “errors outside of examples” (see `spec/integration/plur_spec/error_handling_spec.rb`)
- **Fixed**: `runCommand` now preserves `AssertionCount` / `ErrorCount`, so Minitest summaries reflect parsed counts

## Questions for Next Steps
- Should we ever add `AssertionCount` for RSpec, or keep it unset permanently?

## Next Steps (Proposed)
1) **Keep AssertionCount out of RSpec**
   - No formatter or parser support for assertions; keep `AssertionCount` unset for RSpec.
