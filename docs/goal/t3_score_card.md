# T3 Score Card - Current Plur CLI Design

Source review: `docs/goal/current_design.md`

## Obviousness

- Score: 3
- Evidence: `plur --dry-run` works as the main happy path, but `plur --help`
  says `Usage: plur <command> [flags]`. `plur test` works by treating `test`
  as a pattern, not a command.
- Main issue: Users can succeed by copying examples, but the CLI grammar does
  not explain what is actually happening.
- Suggested improvement: Make help show commandless use as first-class:
  `plur [patterns...] [flags]`, plus short examples for `plur`, `plur spec`,
  `plur test`, and `plur watch`.
- Risk/tradeoff: More customized Kong help may require code outside the default
  parser output.

## Brevity / Surface Area

- Score: 3
- Evidence: The main daily commands are short (`plur`, `plur spec FILE`,
  `plur watch`), but the visible surface includes `spec`, `watch run`,
  `watch install`, `watch find`, `rails`, `rake`, `doctor`, `config init`,
  `rails:init`, global flags, spec flags, and watch flags.
- Main issue: The common path is brief, but help presents too many equivalent
  or semi-equivalent shapes at once.
- Suggested improvement: Reorder and rewrite help around daily flows first,
  then put maintenance commands below.
- Risk/tradeoff: Existing users may look for command listings in the current
  Kong-generated order.

## Default Quality

- Score: 3
- Evidence: Default RSpec detection works well, and mixed `spec/` plus `test/`
  projects intentionally prefer RSpec. But explicit non-test files such as
  `plur foo/baz/other-file.go` become RSpec targets in an RSpec project.
- Main issue: Defaults are fast and useful for Ruby projects, but hidden
  inference can make surprising commands look valid.
- Suggested improvement: In dry-run, print selected job, framework, reason, and
  whether explicit targets matched the selected framework.
- Risk/tradeoff: Extra dry-run output can be noisy unless kept compact.

## Conceptual Coherence

- Score: 2
- Evidence: `job` and `framework` are separate concepts, but dry-run mostly
  shows only `[rspec]` or `[minitest]`. `spec` is both a command and a generic
  test-running entry point. `{{target}}` is ignored in run mode but honored in
  watch mode.
- Main issue: The implementation has useful concepts, but the user-facing
  vocabulary blends them together.
- Suggested improvement: Standardize visible nouns around `job`, `target`, and
  `watch rule`; make framework a detail of a job.
- Risk/tradeoff: Renaming output and docs may expose breaking config concepts
  that need migration or clean removal.

## Feedback Quality

- Score: 2
- Evidence: Dry-run prints exact worker commands, which is valuable. However,
  it does not explain selection reason, zero-match exclude patterns, or why
  watch no-op changes do nothing. `--dry-run watch` still starts watch setup.
- Main issue: Output shows what will execute, but not enough about why that
  plan was chosen or why user intent produced no action.
- Suggested improvement: Add one selection line to dry-run and warnings for
  zero-match excludes and watch changes with no matching rule/target.
- Risk/tradeoff: Warnings need care so optional broad excludes do not become
  annoying in scripts.

## Composability

- Score: 3
- Evidence: `-C` composes well because config loads from the target directory.
  Dry-run is copyable. `watch find` exits 2 when no runnable target exists.
  But dry-run and watch-find output are different human formats, and there is
  no stable JSON plan for agents.
- Main issue: Plur has good shell primitives, but lacks a stable planning API.
- Suggested improvement: Add `plur plan --json` or
  `plur --dry-run --format=json` with job, reason, targets, excludes, worker
  groups, env, argv, and config files.
- Risk/tradeoff: A JSON contract raises compatibility expectations; keep it
  intentionally small.

## Config/API Cleanliness

- Score: 2
- Evidence: `job.Job` is small, and config precedence is documented. But
  run-mode versus watch-mode target handling differs, watch mappings can be
  valid-looking but no-op, and users must understand both `target_pattern` and
  `[[watch]].targets`.
- Main issue: The config model is close to clean, but target semantics leak
  across too many fields and modes.
- Suggested improvement: Document and then simplify the target model: one-shot
  jobs discover targets, watch rules map changed sources to targets, and dry-run
  exposes both clearly.
- Risk/tradeoff: Simplifying may require breaking old config shapes, which is
  acceptable for this goal but needs clear release notes.

## Top Design Problems

1. Help and usage do not match the real happy path: commandless `plur` and
   `plur watch` are first-class but look secondary or command-required.
2. Job selection is hidden. Users see framework labels and worker commands, not
   selected job, reason, config source, or target-selection explanation.
3. Run and watch modes use different target semantics without enough UI
   explanation.
4. No-op intent is too quiet: bad excludes and unmatched watch changes can look
   successful.
5. Agent/script output is useful but unstable and split across inconsistent
   human formats.

## Top Recommended Changes

1. Fix top-level and watch help copy so the happy paths are obvious.
2. Add a compact dry-run selection line: selected job, framework, reason, and
   config source when available.
3. Warn when `--exclude-pattern` matches no files.
4. Make `--dry-run watch` reject with guidance or print a real watch plan.
5. Add a small structured plan output after the human dry-run path is clearer.

## Big Ticket Ideas

1. Collapse `plur spec` and commandless `plur` into one clearly documented
   "run" model, leaving `spec` only if it still adds real clarity.
2. Build a watch TUI that makes changed file, matched rule, target files, job,
   current run, and last failures visible without requiring debug logs.

## Things That Should Not Change

- Keep commandless `plur` as the core daily command.
- Keep `-C` behavior.
- Keep RSpec-first autodetection when both `spec/` and `test/` exist.
- Keep dry-run as a first-class preview.
- Keep the small TOML `job` model.
- Keep `watch find` as the safe way to inspect a single file-change mapping.
