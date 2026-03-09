# Exclude pattern CLI design

## Status
Approved

## Summary

Add a repeatable `--exclude-pattern` flag to `plur spec` so plur can remove
test files from its own discovered input set before worker grouping and command
construction. This is a file-level filter only. It does not pass through to the
underlying framework, because plur owns file globbing and worker splitting.

The design mirrors RSpec's `--exclude-pattern` naming while using plur's
existing doublestar-based discovery behavior.

## Goals

- Support file-level exclusion for RSpec, Minitest, and other file-driven jobs.
- Keep file discovery consolidated in `glob.go`.
- Use doublestar semantics for exclude matching.
- Make exclusion visible in both dry-run and real runs through the same logging
  path.
- Preserve existing behavior for explicit files, directories, and glob inputs.

## Non-goals

- Example-level filtering by tag, name, or regex.
- Passing exclude patterns through to RSpec or other frameworks.
- Adding negated positional arguments such as `!spec/foo_spec.rb`.
- Adding config-file support for exclude patterns in this change.

## CLI

Add a repeatable `--exclude-pattern` flag to `SpecCmd`.

Examples:

```bash
plur --exclude-pattern 'spec/system/**/*_spec.rb'
plur spec spec/models --exclude-pattern 'spec/models/system_spec.rb'
plur --use=minitest --exclude-pattern 'test/integration/**/*_test.rb'
```

## Behavior

Resolution order:

1. Resolve the job/framework exactly as today.
2. Build the candidate file list exactly as today:
   - `FindFilesFromJob` for default discovery
   - `ExpandPatternsFromJob` for explicit input patterns
3. Normalize and dedupe the candidate paths.
4. Remove any file whose normalized path matches any exclude pattern.
5. Error if no files remain.
6. Group remaining files and build worker commands.

Exclude semantics:

- Repeated `--exclude-pattern` flags are OR'd together.
- Include resolution always happens before exclude filtering.
- Existing explicit files that match an exclude pattern are removed from the run
  set.
- Explicit nonexistent files still error before exclusion, matching current
  behavior.
- If excludes remove every selected file, return a specific error such as
  `no test files remain after applying exclude patterns`.

## Discovery implementation shape

File discovery remains consolidated in `glob.go`.

This design does not add a second discovery subsystem. Instead:

- Keep `FindFilesFromJob` and `ExpandPatternsFromJob` as the public discovery
  entry points.
- Add exclude support underneath them, not beside them.
- Use a small internal helper in `glob.go` for shared path normalization,
  deduplication, and exclude filtering if that keeps the call sites clear.

`glob.go` stays focused on file selection. It should not own dry-run formatting
or log-level decisions.

## Matching rules

Exclude matching uses doublestar semantics against the already-selected file
list rather than expanding exclude patterns independently and subtracting
results.

Recommended matching flow:

- Normalize candidate file paths.
- Normalize exclude patterns consistently with the same path style.
- For each selected file, test it against each exclude pattern using
  doublestar path matching.
- Drop the file if any pattern matches.

Matching against the selected file list keeps behavior predictable for:

- auto-discovery
- explicit files
- explicit directories
- explicit include globs

## Logging and dry-run visibility

Use one structured logging path for real runs and dry-run runs.

The existing stderr logger is the right primary output path. Discovery and
exclude reporting should avoid adding new ad hoc `fmt.Fprintf` or `toStdErr`
output for this feature.

Recommended logging behavior:

- Emit a summary discovery/exclude event at `DEBUG` by default.
- Include structured fields such as:
  - `job`
  - `framework`
  - `include_patterns`
  - `exclude_patterns`
  - `discovered`
  - `excluded`
  - `remaining`
- Emit a second `DEBUG` event with excluded file paths only when exclusions
  matched at least one file.

Dry-run behavior:

- Reuse the same event shape and output path as normal runs.
- Add a `[dry-run]` prefix to the rendered message in dry-run mode.
- Keep existing dry-run worker command output so users can still see the final
  post-filter split.

This keeps regular runs traceable and makes dry-run an annotated view of the
same execution path rather than a separate reporting mechanism.

## Output expectations

Conceptual example for `--debug --dry-run`:

```text
[dry-run] 12:34:56 - DEBUG - test file discovery framework="rspec" discovered=14 excluded=2 remaining=12 exclude_patterns=[spec/system/**/*_spec.rb]
[dry-run] Running 12 specs [rspec] in parallel using 4 workers
[dry-run] Worker 0: ...
```

Exact wording can follow existing logger conventions, but the important part is
that the exclude effect is visible through the same logger path in both dry-run
and real runs.

## Edge cases

- Auto-discovery with no matches after exclusion should return the specific
  post-exclusion error.
- Explicit include glob with zero pre-exclusion matches should keep the current
  `no test files found matching provided patterns` behavior.
- Repeated exclude patterns should not duplicate work in the final file list.
- Framework behavior should stay consistent across RSpec and Minitest because
  filtering happens before worker grouping and command construction.

## Testing

Add coverage for:

- auto-discovery with one exclude glob
- explicit directory plus exclude
- explicit include glob plus exclude
- repeated exclude patterns
- excluded explicit file leaving zero files
- minitest honoring file exclusion
- dry-run showing the same discovery/exclude log path
- real run with `--debug` showing the same event shape

## Open implementation note

While touching this area, it may be worth refactoring discovery-related
`toStdErr` usage toward the structured stderr logger so discovery diagnostics
share one reporting path. That refactor should stay limited to the paths this
change touches rather than broad output cleanup.
