# T22 Score Card

Reflection after T19-T21:

- T19 cleaned public docs navigation and Diataxis roles.
- T20 rejected user run-mode job commands containing `{{target}}`.
- T21 tightened `plur watch find --help` so the preview command no longer
  advertises unrelated run/live-watch flags.

Current executable evidence:

```bash
./plur --help
./plur watch --help
./plur watch find --help
./plur -C fixtures/projects/default-ruby --dry-run spec/models/user_spec.rb
./plur -C fixtures/projects/default-ruby --dry-run --dry-run-format=json spec/models/user_spec.rb
./plur -C fixtures/projects/default-ruby watch find --format=json spec/spec_helper.rb
```

Verification evidence for this reflection:

```bash
bin/rake
script/check-links
script/docs build
```

`bin/rake` passed with 357 Ruby examples, 0 failures, 4 expected pending
examples. `script/check-links` and `script/docs build` passed when run
sequentially. A parallel attempt raced on MkDocs `site/` cleanup and was rerun
sequentially.

## Scorecard

### Obviousness

- Score: 4
- Evidence: `./plur --help` starts with `Usage: plur [patterns...] [flags]`
  and lists common workflows before the command list. `./plur watch find
  --help` now focuses on preview-specific usage and `--format`. Dry-run text
  says which job and framework were selected.
- Main issue: top-level help still makes advanced/setup commands visually close
  to day-one workflows.
- Suggested improvement: split the first help screen into daily commands first
  and advanced/setup commands second.
- Risk/tradeoff: hiding too much from default help could make maintenance
  commands harder to discover.

### Brevity / Surface Area

- Score: 3
- Evidence: the daily commands are short, but top-level help still exposes
  `watch install`, `rails:init`, `config init`, `--json`, `--workers`,
  `--rspec-split`, and `plur test`. The docs index still foregrounds
  architecture and development material for ordinary readers.
- Main issue: the first page remains busy for a new Ruby/RSpec user who only
  needs `plur`, `plur FILE`, `plur --dry-run`, and `plur watch`.
- Suggested improvement: make default help and docs landing pages prioritize
  common workflows, with advanced commands grouped or demoted.
- Risk/tradeoff: advanced users may need one extra step to discover setup and
  Rails maintenance commands.

### Default Quality

- Score: 4
- Evidence: `./plur -C fixtures/projects/default-ruby --dry-run
  spec/models/user_spec.rb` selects the RSpec job and plans one target without
  configuration. Mixed Ruby projects default to RSpec. Watch preview works with
  `watch find`, and the docs now keep watch behavior user-facing.
- Main issue: human dry-run still prints implementation-heavy formatter and
  bootstrap arguments by default.
- Suggested improvement: make dry-run human output easier to skim while keeping
  verbose/debug output copyable for low-level command details.
- Risk/tradeoff: reducing command detail in default dry-run could make exact
  worker command debugging less immediate.

### Conceptual Coherence

- Score: 3
- Evidence: T20 made the run/watch `{{target}}` boundary real, and T21 made
  `watch find` help match the preview concept. However, hidden inherited flags
  still parse on `watch find`: combinations like `--dry-run`, `--workers`,
  `--json`, `--rspec-split`, and `--no-first-is-1` are accepted but have no
  useful watch-preview effect.
- Main issue: help now says the right thing, but command parsing still permits
  some no-op cross-mode flag combinations.
- Suggested improvement: reject irrelevant inherited flags for `watch find`
  with contextual guidance, especially `--dry-run` versus `--format=json`.
- Risk/tradeoff: stricter parsing could break scripts that passed harmless
  global flags accidentally.

### Feedback Quality

- Score: 4
- Evidence: invalid run-mode `{{target}}` errors directly: `job "custom"
  command uses {{target}}, but run mode appends targets automatically; remove
  {{target}} from job cmd`. Watch find text says when no rule or no runnable
  target exists. JSON outputs include `warnings` or `exit_code`.
- Main issue: command failures can still be wrapped in log-style output such as
  timestamped `ERROR` lines instead of plain user-facing `Error: ...`.
- Suggested improvement: make expected user errors render as direct plain text
  while keeping debug logging for diagnostics.
- Risk/tradeoff: changing error formatting touches many commands and may affect
  existing snapshots or scripts.

### Composability

- Score: 4
- Evidence: dry-run JSON emits `job`, `targets`, `workers`, `argv`, `env`, and
  `warnings`. Watch-find JSON emits `matched_rules`, `existing_targets`,
  `missing_targets`, and `exit_code`. Output contracts document stdout,
  stderr, and stable machine formats.
- Main issue: dry-run JSON includes a `shell` string built from joining argv,
  which is not a fully quoted shell command contract.
- Suggested improvement: either quote `shell` safely with tests for spaces and
  metacharacters, or demote `shell` below `argv`/`env` in the machine contract.
- Risk/tradeoff: changing `shell` may affect scripts that copied it despite
  `argv` and `env` being the safer structured fields.

### Config/API Cleanliness

- Score: 4
- Evidence: public docs now say run mode appends targets automatically and
  rejects user `{{target}}`; watch mode owns `{{target}}` command templates.
  The runtime enforces that split for selected run-mode jobs.
- Main issue: the configuration reference is still broad and introduces many
  nouns at once: jobs, frameworks, target patterns, watch mappings, watch
  targets, watch jobs, env, workers, and troubleshooting.
- Suggested improvement: slim `docs/configuration.md` into a true reference,
  with starter examples first and workflow explanation linked elsewhere.
- Risk/tradeoff: moving explanatory material out of the reference can fragment
  learning if links are not clear.

## Sub-Agent Reflection

- First-time Ruby/RSpec reviewer: scored Obviousness 4, Default Quality 4,
  Conceptual Coherence 4, Feedback Quality 4, Composability 4, Config/API
  Cleanliness 4, and Brevity 3. The main blocker is first-page surface area:
  help and docs still expose advanced machinery too early.
- Automation/CI reviewer with xhigh reasoning: scored Obviousness 4, Brevity
  4, Default Quality 4, Feedback Quality 4, Composability 4, Config/API
  Cleanliness 4, and Conceptual Coherence 3. The main blocker is hidden
  inherited flags that still parse as no-ops on `watch find`.
- Docs/Diataxis reviewer: scored every category at 4 for T19-T21 scope, but
  did not recommend a 5/5 done state. Remaining docs work is to slim
  configuration reference material and demote architecture/development from
  the public front door.

## Are We Moving In The Right Direction?

Yes. T19-T21 were a useful course correction after the structured-output loop:
they reduced docs sprawl, made a public config rule executable, and tightened a
command-specific help surface. This improved config/API cleanliness and
obviousness without adding a new user concept.

The next course correction should make relevance executable, not just visible.
T21 removed irrelevant flags from `watch find --help`, but the parser still
accepts several of those flags in no-op combinations. That gap is now the most
concrete conceptual-coherence issue.

## Top Design Problems

1. Some command-specific help cleanup is presentation-only; hidden inherited
   flags can still parse and silently do nothing on `watch find`.
2. The first help screen and docs index still expose advanced/setup material
   close to the everyday workflow.
3. Dry-run human output is useful but implementation-heavy for day-one use.
4. Configuration docs are much cleaner than before, but still broad enough to
   feel like explanation plus reference plus troubleshooting in one page.
5. `plur test` remains useful but conceptually odd because it appears like a
   command while behaving as target selection/inference.

## Recommended Next Changes

1. T23-DEV: reject irrelevant inherited flags on `watch find` with direct
   guidance, especially `--dry-run` and `--dry-run-format` versus
   `--format=json`.
2. T24-DEV: reduce first-screen help surface by grouping or demoting advanced
   setup/maintenance commands without removing them.
3. T25-DEV: slim the configuration reference around starter examples and
   canonical tables, moving workflow/troubleshooting explanations elsewhere.
4. Later: decide how to explain or replace `plur test` so it no longer looks
   like an undocumented command.

## Big Ticket Ideas

1. Build one internal plan model shared by one-shot dry-run, watch-find JSON,
   and future watch status output so the same nouns and validation apply
   everywhere.
2. Add a help/profile model with default, advanced, and machine-oriented views
   instead of ad hoc post-processing of Kong help output.

## Things That Should Not Change

- Keep commandless `plur` as the primary daily command.
- Keep `-C` behavior and config loading from the target directory.
- Keep RSpec-first autodetection in mixed Ruby projects.
- Keep dry-run text and dry-run JSON as separate human and machine surfaces.
- Keep `watch find` as the safe watch diagnostic, with JSON and exit code 2
  for no runnable target.
- Keep no-backward-compat cleanup posture for internal CLI/config concepts.

## Done-Done Status

Not done. The scorecard is better than T18 and most categories are in 4
territory, but Brevity / Surface Area and Conceptual Coherence remain below 4
under objective review. Continue into the next DEV loop.
