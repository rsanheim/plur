# T12 Score Card - New Design After T8-T11

Source review:
- Original review: `docs/goal/current_design.md`
- Previous reflection: `docs/goal/t7_score_card.md`
- New design notes: `docs/goal/new_design.md`
- Current commits: `b82b977` through `dc9d8bd`
- Executable checks: `./plur --help`,
  `./plur -C fixtures/projects/default-ruby --dry-run spec/spec_helper.rb`,
  `./plur -C fixtures/projects/default-ruby watch find spec/spec_helper.rb`,
  `./plur -C fixtures/projects/rspec-mismatched-dirs watch find lib/example/runner.rb`
- Review inputs: Curie xhigh product-quality review, Herschel high code-path
  review, Darwin high shell-user review, and local executable review.

## Obviousness

- Score: 4
- Evidence: Top-level help now opens with `Usage: plur [patterns...] [flags]`
  and common workflows including `plur`, `plur spec/calculator_spec.rb`,
  `plur --dry-run`, `plur watch`, and `plur watch find ...`. Dry-run also
  explains selected job, framework, and reason.
- Main issue: `spec` is still both a subcommand and a concept, while `plur test`
  looks command-like even though it is pattern-driven job inference.
- Suggested improvement: Keep daily workflows first, then explicitly separate
  daily commands from maintenance/advanced commands in help.
- Risk/tradeoff: More custom help means more snapshot coverage is needed to
  prevent drift from Kong's generated command list.

## Brevity / Surface Area

- Score: 3
- Evidence: T8-T11 did not add new top-level commands, but help still exposes
  `spec`, `watch run`, `watch install`, `watch find`, `rails`, `rake`,
  `doctor`, `config init`, `rails:init`, and `version`.
- Main issue: The daily path is clearer, but advanced and maintenance commands
  still share too much visual weight with core usage.
- Suggested improvement: Rework help grouping around daily, watch, and
  maintenance sections without changing command behavior.
- Risk/tradeoff: Hiding or demoting advanced commands can make discovery harder
  for existing users unless the grouping is clear.

## Default Quality

- Score: 4
- Evidence: Passing a suspicious explicit file now keeps the command
  permissive but warns:
  `[warn] target 'spec/spec_helper.rb' does not match selected job 'rspec' target pattern 'spec/**/*_spec.rb'`.
  RSpec-first autodetect remains intact, and watch previews now explain no-op
  cases directly.
- Main issue: Defaults are now safer to inspect, but the tool can still only
  warn rather than confidently decide whether unusual explicit files are valid.
- Suggested improvement: Add structured plan output so users and agents can see
  selected targets, warnings, and worker commands without parsing prose.
- Risk/tradeoff: A structured format becomes an API contract.

## Conceptual Coherence

- Score: 3
- Evidence: T8-T11 made the visible nouns more consistent: selected job,
  target pattern, watch rule, and watch target all appear in user-facing output.
  But run mode still discovers test targets while watch mode maps source events
  to target templates, and `{{target}}` is still honored differently across
  run/watch behavior.
- Main issue: Feedback is clearer, but the underlying run/watch target model is
  still split.
- Suggested improvement: Pick a consistent `{{target}}` rule for run and watch,
  or reject unsupported template usage with a direct error.
- Risk/tradeoff: Fixing this may require breaking existing custom job configs,
  which is acceptable but needs release-note quality documentation.

## Feedback Quality

- Score: 4
- Evidence: Dry-run now names job selection; mistyped excludes warn; dry-run
  watch refuses misleading behavior; live watch no-ops print
  `[watch] No matching rule ...`; `watch find` now prints
  `[watch] No existing targets ...` instead of logger-shaped `found rules`
  records.
- Main issue: Feedback is much better for human use, but there is not yet a
  machine-readable equivalent.
- Suggested improvement: Add a JSON plan mode for one-shot dry-runs first, then
  consider a matching watch-find JSON shape.
- Risk/tradeoff: JSON output must preserve stable keys and avoid leaking
  incidental formatter details.

## Composability

- Score: 3
- Evidence: Dry-run remains copyable, and `watch find` still exits 2 when no
  runnable target exists. However, scripts and agents still need to parse
  version banners, warnings, human dry-run lines, and worker argv text.
- Main issue: Human composability improved, but agent/script composability is
  still mostly text scraping.
- Suggested improvement: Implement `--dry-run --format=json` or an equivalent
  plan surface with selected job, reason, warnings, targets, excludes, worker
  argv/env, and mode.
- Risk/tradeoff: The JSON plan should be small and versioned enough to avoid
  becoming a second, inconsistent planner.

## Config/API Cleanliness

- Score: 2
- Evidence: Config docs now reflect humanized watch-find output, but the config
  model still exposes `job`, `framework`, `target_pattern`, `[[watch]].targets`,
  `jobs`, ignore lists, and `{{target}}` semantics.
- Main issue: The UX explains the config model better than before, but the model
  itself remains fragmented.
- Suggested improvement: Use a later DEV phase to unify or reject `{{target}}`
  semantics, then simplify one overlapping config concept at a time.
- Risk/tradeoff: Config cleanup is likely to be breaking and should be done
  deliberately rather than hidden inside output polish.

## Sub-Agent Reflection

- Curie, xhigh product-quality reviewer: rated obviousness at 4 after T8/T9,
  but held brevity, defaults, coherence, feedback, composability, and config
  below 4 until explicit target warnings, watch-find output, structured plan
  output, and config cleanup landed.
- Herschel, high code-path reviewer: confirmed T10 belonged beside existing
  `cmd_spec.go` warnings and should reuse fileset/doublestar helpers rather
  than leaking warning logic into runtime selection or runner execution.
- Darwin, high shell-user reviewer: agreed on score movement from T7 to now:
  obviousness, default quality, and feedback quality improved to 4; brevity,
  conceptual coherence, composability, and config/API cleanliness did not move.
  Darwin specifically called out that `watch find` is now more human-readable
  but still not a machine contract, and that `plur watch --help` still lists a
  global `--dry-run` flag that watch mode rejects.
- Local reflection after T10/T11: the two most visible user-facing gaps from
  T7 are now closed. The remaining blockers are not copy polish; they are
  structured output, command-surface grouping, and config semantics.

## Are We Moving In The Right Direction?

Yes. T8-T11 improved the first-use path and the most confusing no-op paths
without adding core concepts. The design now explains key decisions where a
user is looking: help, dry-run, live watch, and watch preview.

The course correction is that the next loop should not keep adding ad hoc
human text. The CLI now has enough human-facing feedback to justify building a
small structured plan output and then cleaning up the conceptual/config model.

## Top Design Problems

1. No structured plan output for dry-run, so scripts and agents still parse
   prose and worker command text.
2. `plur watch --help` still exposes global `--dry-run` even though watch mode
   rejects it at runtime.
3. Help is better but still gives advanced commands nearly equal visual weight.
4. Run/watch target semantics and `{{target}}` behavior remain split.
5. Config exposes too many overlapping nouns for a small test runner.
6. `watch find` is human-friendly now, but it lacks a stable machine format.

## Recommended Next Changes

1. T13-DEV: add structured one-shot plan output for dry-run.
2. T14-DEV: clarify watch help around incompatible global `--dry-run` and
   separate daily flows from maintenance/advanced commands.
3. T15-DEV: document stdout/stderr and exit-code output contracts for human
   output versus future stable machine formats.
4. T16-DEV: make `{{target}}` semantics consistent or reject unsupported run
   usage with a clear error.
5. T17-DEV: add a structured `watch find` output shape or reuse the plan model
   if T13 creates one cleanly.
6. T18-DEV: simplify or rename one config noun that overlaps with another.

## Big Ticket Ideas

1. Build one internal plan model shared by dry-run, JSON output, and watch
   previews.
2. Make watch mode a richer interactive view that shows last changed file,
   matched rule, runnable target, current job, and last result.

## Things That Should Not Change

- Keep commandless `plur` as the primary daily command.
- Keep `-C` behavior and config loading from the target directory.
- Keep RSpec-first autodetection in mixed Ruby projects.
- Keep warnings non-blocking while commands still have runnable targets.
- Keep `watch find` exit code 2 for no runnable target.

## Done-Done Status

Not done. Obviousness, default quality, and feedback quality have moved into
4 territory, but brevity, conceptual coherence, composability, and config/API
cleanliness still need work before the goal can stop early.
