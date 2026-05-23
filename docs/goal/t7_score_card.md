# T7 Score Card - New Design After T4-T6

Source review:
- Original review: `docs/goal/current_design.md`
- Original scorecard: `docs/goal/t3_score_card.md`
- New design notes: `docs/goal/new_design.md`
- Current executable checks: `../../../plur --dry-run`, `../../../plur --dry-run watch --timeout 1`, `../../../plur watch find spec/spec_helper.rb`
- Sub-agent reviewers: everyday Ruby/RSpec developer, shell/agent workflow user, watch-mode developer

## Obviousness

- Score: 3
- Evidence: `plur --dry-run spec/calculator_spec.rb` now prints
  `[dry-run] Selected job: rspec (framework: rspec, reason: explicit patterns)`.
  But `plur --help` still says `Usage: plur <command> [flags]`, hiding
  commandless `plur [patterns...]`.
- Main issue: The run output is clearer, but help still teaches the wrong
  grammar first.
- Suggested improvement: Make top-level help lead with `plur [patterns...]`,
  `plur spec FILE`, `plur test`, `plur --dry-run`, and `plur watch`.
- Risk/tradeoff: Custom help may drift from Kong's generated command list if
  not tested.

## Brevity / Surface Area

- Score: 3
- Evidence: T4-T6 added output, not new commands. The visible command surface
  is still `spec`, `watch run`, `watch install`, `watch find`, `rails`,
  `rake`, `doctor`, `config init`, `rails:init`, and `version`.
- Main issue: Daily commands remain short, but help still presents maintenance
  commands at the same level as the happy path.
- Suggested improvement: Reorder or supplement help around daily flows first,
  with maintenance commands below.
- Risk/tradeoff: Existing users may prefer the complete generated command list
  first.

## Default Quality

- Score: 3
- Evidence: Dry-run now explains autodetect versus explicit-pattern selection,
  and a mistyped CLI exclude warns:
  `[warn] --exclude-pattern '*user*/_spec.rb' matched no selected files`.
  But explicit existing non-test files can still become RSpec targets after
  project autodetection.
- Main issue: Defaults are more transparent, but surprising explicit targets
  are still accepted without enough suspicion.
- Suggested improvement: Warn when explicit file targets do not match the
  selected job's normal target pattern.
- Risk/tradeoff: Some users intentionally pass unusual files through RSpec;
  warnings must stay informational rather than blocking.

## Conceptual Coherence

- Score: 3
- Evidence: Dry-run now uses the real `SelectedJob` reason, and
  `plur --dry-run watch` now refuses to pretend it can preview watch mode:
  it points to `plur watch find <changed-file>` instead. However, `spec` is
  still both a command and a generic "run tests" concept, and watch/run target
  semantics remain different.
- Main issue: The CLI now explains key decisions, but the command vocabulary
  still blends jobs, frameworks, targets, and watch rules.
- Suggested improvement: Fix help and docs around the nouns `job`, `target`,
  and `watch rule` before adding more behavior.
- Risk/tradeoff: More precise wording can expose existing inconsistencies that
  require follow-up fixes.

## Feedback Quality

- Score: 3
- Evidence: T4 added selection reason, T5 added unmatched CLI exclude warnings,
  and T6 replaced misleading watch dry-run behavior with direct guidance.
  Live watch still logs no-op changes only at debug level, and
  `plur watch find spec/spec_helper.rb` reports `found rules count=0` while
  `docs/features/watch-mode.md` still says helper files trigger all specs.
- Main issue: Feedback improved for dry-run, but normal watch no-op feedback
  and public docs are now the weakest links.
- Suggested improvement: Update stale watch docs, then surface concise
  normal-level feedback for watched changes that match no rule or no target.
- Risk/tradeoff: Watch no-op feedback can get noisy during editor save storms;
  it needs batching/debouncing.

## Composability

- Score: 3
- Evidence: Dry-run remains copyable, `watch find` is still script-friendly
  and returns exit code 2 when nothing would run, and `--dry-run watch` no
  longer starts a persistent watcher. There is still no stable JSON plan.
- Main issue: Human previews are better, but agents still parse prose, version
  banners, warnings, env, and argv lines.
- Suggested improvement: After help/docs are coherent, add a small structured
  plan output for one-shot runs or `watch find`.
- Risk/tradeoff: JSON output creates a contract that must stay stable.

## Config/API Cleanliness

- Score: 2
- Evidence: T5 deliberately warns only on CLI excludes, because configured
  excludes can be broad defaults that do not apply to focused runs. That keeps
  config less noisy, but run-mode `target_pattern`, watch `targets`, and
  `{{target}}` still differ across modes.
- Main issue: Config behavior is not worse, but the underlying run/watch target
  split remains hard to explain.
- Suggested improvement: Document current target semantics accurately, then
  simplify one duplicated or surprising config shape in a later DEV phase.
- Risk/tradeoff: Cleanups may require breaking config behavior, which is
  acceptable for this goal but needs release-note quality documentation.

## Sub-Agent Reflection

- Everyday Ruby/RSpec reviewer: T4-T6 are clear feedback wins, but help still
  hides the commandless happy path and non-test explicit files remain sharp.
- Shell/agent reviewer: T4-T6 improve stderr readability and script safety,
  but structured output is still the main composability gap.
- Watch-mode reviewer: T6 is the right short-term move, but watch docs are now
  stale and live no-op events remain too quiet.

## Are We Moving In The Right Direction?

Yes. The design is trending toward explicit decisions and safer previews
without adding new core concepts. The first cycle improved feedback quality
from "infer it from worker commands" to direct job/reason/warning/guidance
messages.

The course correction is that we should not keep adding output lines while
help and docs teach stale behavior. The next loop should make the visible
documentation and help match the improved executable behavior.

## Top Design Problems

1. Help still misrepresents the happy path: commandless `plur` and default
   `plur watch` look secondary or command-required.
2. Public watch docs are stale after T6: they still recommend
   `plur watch --dry-run`.
3. Watch no-op events remain quiet in normal output, especially helper files
   or source files that map to no existing target.
4. Explicit non-test file targets are still accepted after autodetect with no
   warning.
5. There is no stable structured plan output for agents or scripts.

## Recommended Next Changes

1. T8-DEV: fix top-level help, watch help, and stale watch docs so the happy
   paths and `watch find` preview model are obvious.
2. T9-DEV: surface concise watch no-op feedback for matched-no-rule and
   matched-no-existing-target events.
3. T10-DEV: warn on explicit targets that do not look compatible with the
   selected job's target pattern.
4. Later: add structured plan output once the human-facing model is coherent.

## Big Ticket Ideas

1. Build a first-class plan model shared by dry-run, JSON output, and watch
   previews so run and watch expose the same nouns.
2. Make watch mode a richer interactive surface that shows watched dirs,
   matched rule, runnable targets, current run, and last result.

## Things That Should Not Change

- Keep commandless `plur` as the core daily command.
- Keep `-C` behavior and config loading from the target directory.
- Keep RSpec-first autodetection when both `spec/` and `test/` exist.
- Keep dry-run as a copyable human preview.
- Keep `watch find` as the safe watch diagnostic.
- Keep warnings non-blocking unless the command already has no runnable files.

## Done-Done Status

Not done. Scores are not all 4s and 5s, help/docs still contradict the new
watch behavior, and important watch/no-op and structured-output gaps remain.
