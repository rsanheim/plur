# T18 Score Card - New Design After T13-T17

Source review:
- Original review: `docs/goal/current_design.md`
- Previous reflection: `docs/goal/t12_score_card.md`
- New design notes: `docs/goal/new_design.md`
- Current executable checks: `./plur --help`, `./plur watch --help`,
  `./plur -C fixtures/projects/default-ruby --dry-run`,
  `./plur -C fixtures/projects/default-ruby --dry-run --dry-run-format=json spec/models/user_spec.rb`,
  `./plur -C fixtures/projects/default-ruby watch find spec/spec_helper.rb`,
  `./plur -C fixtures/projects/default-ruby watch find --format=json spec/spec_helper.rb`,
  `./plur -C fixtures/projects/default-ruby watch find --format=json lib/calculator.rb`,
  and `./plur -C fixtures/projects/default-ruby watch find --format=json lib/missing.rb`
- Review inputs: first-time Ruby developer reviewer, automation/CI reviewer
  with xhigh reasoning, docs/UX information-architecture reviewer, and local
  executable review.

## Obviousness

- Score: 4
- Evidence: Top-level help now opens with `Usage: plur [patterns...] [flags]`
  before `plur <command> [flags]`, and the common workflow block starts with
  `plur`, `plur spec/calculator_spec.rb`, `plur test`, `plur --dry-run`,
  `plur watch`, and `plur watch find spec/calculator_spec.rb`. Watch help now
  leads with `plur watch [flags]` and `plur watch find <changed-file> [flags]`.
- Main issue: `plur test` still looks like a command even though it is a
  pattern-driven framework inference path, and the public docs index includes
  broken overview links.
- Suggested improvement: Keep the daily grammar prominent, fix broken public
  docs links, and make `plur test` either a first-class documented workflow or
  less command-looking.
- Risk/tradeoff: More custom help grouping can drift from Kong's generated
  command list if it is not guarded by focused output tests.

## Brevity / Surface Area

- Score: 3
- Evidence: The daily commands are short, but help still gives visible weight
  to `spec`, `watch run`, `watch install`, `watch find`, `rails`, `rake`,
  `doctor`, `config init`, `rails:init`, and `version`. The docs tree also
  exposes public pages such as `docs/configuration-test-cases.md`, `docs/plans/`,
  and `docs/wip/`, which reads more like internal planning and contributor
  material than a small user-facing CLI manual.
- Main issue: The happy path is brief, but the public surface still feels too
  broad and too flat.
- Suggested improvement: Run a [Diataxis](https://diataxis.fr/start-here/)
  public-docs cleanup: make getting started the tutorial, usage a short how-to,
  configuration/output contracts references, and move or de-index planning/WIP
  material.
- Risk/tradeoff: Moving or de-indexing pages can hide useful contributor
  context unless the development docs keep clear entry points.

## Default Quality

- Score: 4
- Evidence: `./plur -C fixtures/projects/default-ruby --dry-run` autodetects
  RSpec, explains `Selected job: rspec (framework: rspec, reason: autodetect)`,
  and plans 13 specs across 4 workers. Suspicious explicit targets now warn
  without blocking advanced use. `watch find spec/spec_helper.rb` exits 2 and
  prints `[watch] No matching rule for spec/spec_helper.rb`, which is a clear
  no-op explanation.
- Main issue: The executable defaults are much safer to inspect, but docs still
  make the tool feel more complex than the main workflow actually is.
- Suggested improvement: Let the next docs pass remove stale or overly broad
  explanations that obscure the default `plur`, `plur --dry-run`, and
  `plur watch find` workflows.
- Risk/tradeoff: Trimming docs too aggressively could remove useful edge-case
  examples for non-standard projects.

## Conceptual Coherence

- Score: 3
- Evidence: T13-T17 improved visible vocabulary: dry-run JSON exposes `job`,
  `framework`, `reason`, `targets`, `warnings`, and `workers`; watch-find JSON
  exposes `matched_rules`, `existing_targets`, `missing_targets`, and
  `exit_code`; `docs/configuration.md` now says run mode appends targets
  automatically and labels `{{target}}` as watch-mode command templating.
- Main issue: The model is clearer but still split across `job`, `framework`,
  `target_pattern`, watch `targets`, watch `jobs`, and watch-only `{{target}}`.
- Suggested improvement: Enforce the public `{{target}}` rule in runtime config:
  reject `{{target}}` in one-shot run-mode job commands with a direct message
  that run mode appends targets automatically.
- Risk/tradeoff: This is intentionally breaking for old configs that relied on
  the tolerated run-mode template path.

## Feedback Quality

- Score: 4
- Evidence: Human feedback is now strong across the reviewed paths:
  dry-run names the selected job and reason; bad CLI excludes warn;
  suspicious explicit targets warn; watch dry-run is rejected with guidance;
  live and preview watch no-ops use `[watch] No matching rule ...`; and watch
  find has human and JSON forms. `./plur watch --help` also explains that
  `--dry-run` is one-shot only and watch mode rejects it.
- Main issue: Some help contexts still inherit global flags that are irrelevant
  or redundant for the subcommand, such as dry-run flags under `watch find`.
- Suggested improvement: Tighten command-specific help so preview commands do
  not advertise unrelated global flags as if they apply equally.
- Risk/tradeoff: Hiding inherited global flags can make Kong behavior harder to
  reason about unless the custom help layer is systematic.

## Composability

- Score: 4
- Evidence: `./plur -C fixtures/projects/default-ruby --dry-run --dry-run-format=json spec/models/user_spec.rb`
  emits JSON with `version`, `mode`, `job`, `targets`, `warnings`, and
  `workers`. `./plur -C fixtures/projects/default-ruby watch find --format=json spec/spec_helper.rb`
  exits 2 with `exit_code: 2` and empty target maps. `lib/calculator.rb` exits
  0 with `existing_targets.rspec = ["spec/calculator_spec.rb"]`, while
  `lib/missing.rb` exits 2 with `missing_targets.rspec = ["spec/missing_spec.rb"]`.
  `docs/output-contracts.md` documents stable JSON and stream roles.
- Main issue: Scripts are much better served, but combined stdout/stderr capture
  can still mix version or warning text with JSON if callers ignore the output
  contract.
- Suggested improvement: Keep structured output narrow and documented, and add
  examples that demonstrate stdout-only JSON capture in shell workflows.
- Risk/tradeoff: Every new structured example increases the chance that docs
  duplicate the canonical output-contract reference.

## Config/API Cleanliness

- Score: 3
- Evidence: The public docs are cleaner after T16, but configuration still
  exposes `cmd`, `framework`, `target_pattern`, `exclude_patterns`, `env`,
  `[[watch]].source`, `[[watch]].targets`, `[[watch]].jobs`, `ignore`,
  `reload`, `{{match}}`, `{{dir_relative}}`, and watch-only `{{target}}`.
  The docs say run-mode job commands should omit `{{target}}`, but runtime
  still tolerates the old shape instead of making the clean rule enforceable.
- Main issue: Config is more accurately documented than before, but it still
  feels like several implementation concepts rather than a minimal user API.
- Suggested improvement: First enforce the `{{target}}` boundary, then continue
  trimming docs so configuration is a compact reference rather than a broad
  implementation guide.
- Risk/tradeoff: Enforcing config rules can break existing local configs, which
  is acceptable for this goal but needs clear errors and release-note quality
  documentation.

## Sub-Agent Reflection

- First-time Ruby developer reviewer: scored obviousness, default quality,
  feedback, and composability at 4 after the help and structured-output work,
  but kept brevity, conceptual coherence, and config/API cleanliness below 4.
  The main course correction was to stop adding output features for a cycle and
  reduce first-page surface area.
- Automation/CI reviewer with xhigh reasoning: scored composability at 4 due to
  dry-run JSON and watch-find JSON, and conceptual coherence at 4 from an
  automation perspective, but kept brevity and config/API cleanliness below 4.
  The main recommended DEV loop was runtime enforcement of the public
  `{{target}}` rule.
- Docs/UX reviewer: scored obviousness, defaults, feedback, and composability
  at 4, but kept brevity, conceptual coherence, and config/API cleanliness at
  3 because public docs still mix how-to, reference, architecture, decision-log,
  planning, and WIP material. The reviewer also found broken public links in
  `docs/index.md` to missing overview pages.
- Local reflection: The next loop should prioritize reducing surface area and
  enforcing concepts rather than adding another output format.

## Are We Moving In The Right Direction?

Yes. T13-T17 closed the biggest T12 composability gap by adding stable JSON for
one-shot dry-run plans and watch previews, documenting stream and exit-code
contracts, and clarifying the run/watch target-template split. The design is
substantially better for agents and shell scripts than it was at T12.

The course correction is that the next work should reduce visible complexity.
The strongest candidates are a Diataxis public-docs cleanup and a config/API
cleanup that rejects `{{target}}` in run-mode job commands.

## Top Design Problems

1. Public docs still expose too much internal planning, WIP, architecture, and
   checklist material in the user-facing surface.
2. Config still has too many overlapping nouns: job, framework,
   target-pattern, watch targets, watch jobs, and command target templates.
3. Runtime still tolerates run-mode `{{target}}` even though public docs now
   say run mode appends targets automatically.
4. Help remains broad: daily workflows are clear, but advanced and maintenance
   commands still appear close to the happy path.
5. `plur test` remains useful but conceptually odd because it looks like a
   command while behaving as a pattern/inference workflow.

## Recommended Next Changes

1. T19-DEV: public docs information-architecture cleanup using Diataxis. Fix
   broken overview links, de-index planning/WIP pages from public navigation,
   and keep Usage, Configuration, Watch Mode, and Output Contracts in distinct
   roles.
2. T20-DEV: enforce the public `{{target}}` rule by rejecting it in one-shot
   run-mode job commands with a direct error.
3. T21-DEV: tighten command-specific help so watch preview commands do not show
   unrelated dry-run options as equally applicable.
4. Later: decide whether `plur test` should become an explicit documented alias
   or be described more clearly as target selection.

## Big Ticket Ideas

1. Build one internal plan model shared by one-shot dry-run, watch-find JSON,
   and any future watch status view so the same nouns appear everywhere.
2. Redesign the public docs as a small product manual plus a separate
   contributor/development area, with internal plans and WIP material outside
   the user docs tree.

## Things That Should Not Change

- Keep commandless `plur` as the primary daily command.
- Keep `-C` behavior and config loading from the target directory.
- Keep RSpec-first autodetection in mixed Ruby projects.
- Keep dry-run text as a human preview and dry-run JSON as the machine API.
- Keep `watch find` as the safe watch diagnostic, with exit code 2 for no
  runnable target.
- Keep warnings non-blocking while commands still have runnable targets.

## Done-Done Status

Not done. Obviousness, default quality, feedback quality, and composability are
now in 4 territory, but brevity, conceptual coherence, and config/API
cleanliness remain below 4. The goal should continue into the next DEV loop.
