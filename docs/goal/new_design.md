## Plur - new design specs, plans, ideas

_for TX-DEV phases as part of `cli_goal.md`_

## T4-DEV - Explain Dry-Run Job Selection

Pain point: `plur --dry-run` shows exact worker commands, but it does not say
which job was selected or why. Users have to infer the selection from the
framework label and command argv. This is especially confusing in mixed
RSpec/Minitest projects and for custom jobs where the job name is not visible.

Change: add one compact dry-run line after file discovery and before worker
commands:

```text
[dry-run] Selected job: rspec (framework: rspec, reason: autodetect)
```

The line should use the existing `runtime.SelectedJob` data rather than adding
a new selection abstraction. It should appear only in dry-run mode, because
normal test output should stay focused on execution and failures.

Acceptance criteria:
- `plur --dry-run` shows selected job, framework, and reason.
- Passing an explicit target such as `test/` shows the explicit-pattern reason.
- Passing `--use custom-job` shows the configured job name and explicit-name
  reason.
- Existing worker dry-run output remains copyable.
- Focused integration tests and Ruby lint pass.

Before evidence:
- T1 transcript `tmp/cli_inventory_demo.txt` showed dry-run jumping from
  `plur version=...` directly to `[dry-run] Running ...`, so job selection had
  to be inferred from the worker command.

After evidence:

```text
plur version=v0.56.1-...
[dry-run] Selected job: rspec (framework: rspec, reason: autodetect)
[dry-run] Running 1 spec [rspec] in parallel using 1 worker
```

```text
plur version=v0.56.1-...
[dry-run] Selected job: minitest (framework: minitest, reason: explicit patterns)
[dry-run] Running 1 test [minitest] in parallel using 1 worker
```

Tradeoff: dry-run gains one line of output. That is deliberate for human
clarity, and the worker command lines remain unchanged and copyable.

## T5-DEV - Warn When CLI Excludes Match Nothing

Pain point: `--exclude-pattern` is powerful but easy to mistype. A plausible
pattern such as `*user*/_spec.rb` can match no selected files while the command
still succeeds, so the user thinks a file was excluded when it was not.

Change: when an explicit CLI `--exclude-pattern` matches none of the files in
the selected test plan, print a compact warning before the run or dry-run
worker plan:

```text
[warn] --exclude-pattern '*user*/_spec.rb' matched no selected files
```

The warning is limited to CLI-provided excludes. Configured job excludes may be
broad defaults that naturally do not match a focused target, so warning on those
would make normal focused runs noisy.

Acceptance criteria:
- A dry-run with a non-matching CLI `--exclude-pattern` exits successfully and
  prints a warning.
- A matching CLI `--exclude-pattern` does not warn.
- Exclude behavior itself is unchanged: matching excludes still remove files,
  and excluding every candidate still returns the existing error.
- Focused integration tests, Go tests for file discovery details, Ruby lint,
  and the full build pass.

Before evidence:
- T1 transcript `tmp/cli_inventory_demo.txt` showed
  `plur spec --exclude-pattern '*user*/_spec.rb'` keeping
  `spec/models/user_spec.rb` with no warning.

After evidence:

```text
plur version=v0.56.1-...
[warn] --exclude-pattern '*user*/_spec.rb' matched no selected files
[dry-run] Selected job: rspec (framework: rspec, reason: autodetect)
[dry-run] Running 3 specs [rspec] in parallel using 3 workers
```

The refreshed transcript is available at `tmp/cli_inventory_demo.txt`.

Tradeoff: a successful command can now print a warning to stderr. That is
intentional for explicit CLI intent; it is not applied to config excludes.

## T6-DEV - Reject Dry-Run Watch With Guidance

Pain point: `plur --dry-run watch` currently starts watch mode. That is
surprising because dry-run means "show what would happen without doing it" for
normal test runs, while watch mode attaches a watcher and waits for input.

Change: when global `--dry-run` is combined with `watch run` or the default
`watch` command, exit quickly with a focused message:

```text
Error: plur watch does not support --dry-run yet.
Use `plur watch find <changed-file>` to preview which tests a file change would run.
Use `plur --dry-run [patterns...]` to preview a one-shot test run.
```

Acceptance criteria:
- `plur --dry-run watch` exits non-zero without installing or starting the
  watcher.
- The message points to `plur watch find <changed-file>` for watch previews.
- Normal `plur watch --timeout 1` behavior is unchanged.
- Focused watch dry-run tests and the full build pass.

Before evidence:
- `plur --dry-run watch --timeout 1` printed `ready and watching ...` and
  exited 0 after the timeout, so it did real watch setup despite the dry-run
  flag.

After evidence:

```text
Error: plur watch does not support --dry-run yet.
Use `plur watch find <changed-file>` to preview which tests a file change would run.
Use `plur --dry-run [patterns...]` to preview a one-shot test run.
```

Tradeoff: this does not implement a full watch plan. It chooses honest
guidance over a misleading partial dry-run.

## T8-DEV - Make Help And Watch Docs Match The Real Happy Paths

Pain point: `plur --help` still says `Usage: plur <command> [flags]`, even
though commandless `plur [patterns...]` is the daily run command. `plur watch
--help` likewise presents `watch <command>` first, and public watch docs still
recommend `plur watch --dry-run` even though T6 now rejects it.

Change: customize help output just enough to lead with real workflows:

```text
Usage: plur [patterns...] [flags]
       plur <command> [flags]

Common workflows:
  plur
  plur spec/calculator_spec.rb
  plur --dry-run
  plur watch
  plur watch find spec/calculator_spec.rb
```

For watch help, lead with `plur watch [flags]`, `plur watch find
<changed-file>`, and only then the subcommand form. Update
`docs/features/watch-mode.md` to remove `watch --dry-run` and point users to
`watch find`.

Acceptance criteria:
- `plur --help` shows commandless usage first and includes common workflow
  examples.
- `plur watch --help` shows `plur watch [flags]` and `plur watch find
  <changed-file>` before subcommand details.
- Watch docs no longer recommend `plur watch --dry-run`.
- Focused help/doc tests and the full build pass.

Before evidence:
- `plur --help` printed `Usage: plur <command> [flags]`.
- `plur watch --help` printed `Usage: plur watch <command> [flags]`.
- `docs/features/watch-mode.md` recommended `plur watch --dry-run`.

After evidence:

```text
Usage: plur [patterns...] [flags]
       plur <command> [flags]

Common workflows:
  plur
  plur spec/calculator_spec.rb
  plur test
  plur --dry-run
  plur watch
  plur watch find spec/calculator_spec.rb
```

```text
Usage: plur watch [flags]
       plur watch find <changed-file> [flags]
       plur watch <command> [flags]
```

`docs/features/watch-mode.md` now uses `plur watch find` for preview examples.

Tradeoff: help gets a thin custom layer over Kong output. The detailed flags
and command lists still come from Kong so drift stays limited.

## T9-DEV - Surface Live Watch No-Op Feedback

Pain point: watch mode can observe a file change and then appear to do nothing.
For example, `spec/spec_helper.rb` is under a watched directory, but the
built-in spec rule does not map it to a runnable target. Today that is only
visible in debug logs or via a separate `watch find` command.

Change: after a debounced live watch batch, print concise normal-output
messages when a changed path has no runnable result:

```text
[watch] No matching rule for spec/spec_helper.rb
[watch] No existing targets for lib/missing.rb (missing: spec/missing_spec.rb)
```

The event handler should return no-op details rather than printing directly,
so watch mode remains responsible for user-facing output and tests can inspect
the planning result.

Acceptance criteria:
- A live watched change with no matching rule prints a normal-output no-op
  message.
- A matched watch rule with only missing targets prints the changed path and
  missing target names.
- Changes that run jobs do not print no-op messages.
- Focused watch tests and the full build pass.

Before evidence:
- `plur watch find spec/spec_helper.rb` reported `found rules count=0`, but
  live watch did not give the user a normal-output explanation for that no-op.

After evidence:

```text
[watch] No matching rule for spec/spec_helper.rb
```

The focused integration check is
`PLUR_BINARY=$PWD/plur bin/rspec spec/integration/watch/watch_integration_spec.rb:75`.

Tradeoff: watch mode may print more messages during noisy save bursts. Keep the
messages per debounced batch, concise, and only for paths that produce no run.

## T10-DEV - Warn When Explicit Targets Do Not Match The Selected Job

Pain point: Plur intentionally passes explicit existing files through to the
selected framework, but that can make a likely mistake look valid. For example,
`plur --dry-run lib/calculator.rb` in an RSpec project selects the `rspec` job
and plans to run `bundle exec rspec lib/calculator.rb`, even though the normal
target pattern is `spec/**/*_spec.rb`.

Change: after discovery succeeds, warn when a CLI positional argument is an
explicit existing file target that does not match the selected job's target
pattern:

```text
[warn] target 'lib/calculator.rb' does not match selected job 'rspec' target pattern 'spec/**/*_spec.rb'
```

The warning should not reject the command. RSpec and other jobs may accept
non-standard explicit files, so the safest UX improvement is visibility rather
than a hard failure.

Acceptance criteria:
- A dry-run with an explicit non-matching file exits successfully and prints a
  warning.
- A matching explicit spec file does not warn.
- Glob inputs and directory inputs do not warn per individual expanded file.
- Focused integration tests, file-set unit tests, and the full build pass.

Before evidence:
- Earlier inventory and review phases showed explicit non-test files becoming
  RSpec targets with no hint that they were outside the selected job's normal
  target pattern.

After evidence:

```text
[warn] target 'lib/calculator.rb' does not match selected job 'rspec' target pattern 'spec/**/*_spec.rb'
[dry-run] Selected job: rspec (framework: rspec, reason: autodetect after patterns)
[dry-run] Running 1 spec [rspec] in parallel using 1 worker
```

Tradeoff: the command remains permissive. This preserves advanced framework
behavior while making likely mistakes visible.

## T11-DEV - Humanize Watch Find Output

Pain point: T8 and T6 point users to `plur watch find <changed-file>` as the
safe watch preview command, but its output is still logger-shaped:

```text
level=INFO msg="found rules" name=lib-to-spec source=lib/**/*.rb
```

That is useful to scripts, but weaker for the human workflow now promoted by
help and docs. It also does not match T9's live watch no-op wording.

Change: print plain watch-preview lines from `watch find`:

```text
[watch] Checking lib/example.rb
[watch] Matched rule lib-to-spec (source: lib/**/*.rb, jobs: rspec, target: spec/{{match}}_spec.rb)
[watch] Would run job rspec with spec/example_spec.rb
```

For no-op previews, reuse the live-watch language:

```text
[watch] No matching rule for spec/spec_helper.rb
[watch] No existing targets for lib/example/runner.rb (missing: spec/example/runner_spec.rb)
```

Acceptance criteria:
- `watch find` prints human-readable matched-rule and runnable-target lines.
- A no-rule result exits 2 and prints `No matching rule`.
- A matched rule with missing targets exits 2 and prints `No existing targets`.
- Existing watch-find exit codes remain unchanged.
- Focused watch-find specs and the full build pass.

Before evidence:
- T7 scorecard and Curie's review both identified `found rules count=0` and
  other logger-shaped records as a remaining feedback-quality gap.

After evidence:

```text
[watch] No matching rule for spec/spec_helper.rb
```

Tradeoff: `watch find` becomes more human-first. A structured plan format is
still a separate phase; this phase should not pretend the human text is a
stable machine API.

## T13-DEV - Add Structured Dry-Run Plan Output

Pain point: `plur --dry-run` is copyable and useful for humans, but agents and
shell scripts still have to parse prose, version banners, warning lines, and
worker command strings. T12 and the high-reasoning shell review both identified
this as the main blocker for composability.

Change: add an explicit dry-run format flag:

```bash
plur --dry-run --dry-run-format=json spec/calculator_spec.rb
```

The default remains `text`. JSON output should go to stdout and include a small
stable plan:

```json
{
  "version": 1,
  "mode": "spec",
  "job": {"name": "rspec", "framework": "rspec", "reason": "explicit_patterns"},
  "targets": ["spec/calculator_spec.rb"],
  "warnings": [],
  "workers": [
    {"index": 0, "targets": ["spec/calculator_spec.rb"], "argv": ["bundle", "exec", "rspec"], "env": ["PARALLEL_TEST_GROUPS=1"], "shell": "..."}
  ]
}
```

This phase is intentionally one-shot only. `watch find` structured output can
reuse the same ideas later, but it should not make this first plan format too
large.

Acceptance criteria:
- `--dry-run --dry-run-format=json` emits parseable JSON on stdout.
- The JSON includes selected job, framework, reason, targets, warnings, and
  worker commands.
- Text dry-run output remains unchanged by default.
- `--dry-run-format=json` without `--dry-run` errors clearly.
- Focused integration tests, Go tests, and the full build pass.

Before evidence:
- T12 scorecard kept composability at 3 because scripts parse human lines such
  as `[dry-run] Selected job...` and `[dry-run] Worker 0...`.

After evidence:

```text
plur --dry-run --dry-run-format=json spec/calculator_spec.rb
```

produces a JSON plan on stdout with `job`, `targets`, and `workers` keys.

Tradeoff: this introduces a small output contract. Keep it narrow and versioned
instead of treating existing human text as a stable API.

## T14-DEV - Clarify Watch Help For Dry-Run Flags

Pain point: `plur watch --dry-run` correctly exits with guidance, but
`plur watch --help` still lists the global `--dry-run` and
`--dry-run-format` flags without any watch-specific warning. That makes help
and runtime behavior contradict each other.

Change: keep the global flags visible, but annotate them in watch help:

```text
--dry-run                  One-shot run preview only; watch mode rejects it
--dry-run-format="text"    One-shot dry-run output format: text or json
```

The common workflow block already points to `plur watch find <file>` for watch
previews, so this phase should only close the remaining flag-list ambiguity.

Acceptance criteria:
- `plur watch --help` explains that `--dry-run` is a one-shot run preview flag,
  not a live-watch preview flag.
- `plur watch --help` keeps pointing to `plur watch find` for watch previews.
- Top-level help remains unchanged except for any necessary formatting.
- Focused help specs and the full build pass.

Before evidence:
- Darwin's T12 review noted that `plur watch --help` listed `--dry-run` even
  though live watch rejects it.

After evidence:

```text
--dry-run                  One-shot run preview only; watch mode rejects it
```

Tradeoff: this is still a custom help overlay. It is lower risk than trying to
hide global Kong flags only for one subcommand.

## T15-DEV - Document Output Contracts

Pain point: Plur now has clearer human output and a JSON dry-run plan, but the
contract is implicit. Shell users and agents need to know which streams,
formats, and exit codes are stable, and which output is meant for humans only.

Change: add `docs/output-contracts.md` and link it from the docs index. The
doc should define:

- one-shot run stdout/stderr roles
- dry-run text versus `--dry-run-format=json`
- warning behavior with exit 0
- `watch find` stdout and exit code 2 for no runnable target
- debug output as unstable diagnostic text

Acceptance criteria:
- The docs index links to output contracts.
- The output contract doc names `--dry-run-format=json`, `warnings`,
  `workers`, `watch find`, and exit code 2.
- The doc clearly says human text is not the machine API.
- Focused doc specs and the full build pass.

Before evidence:
- T12/Darwin kept composability at 3 partly because stdout/stderr, warnings,
  exit codes, and stable versus unstable output were not documented.

After evidence:
- `docs/output-contracts.md` documents the current behavior and the JSON plan
  contract.

Tradeoff: this is documentation, not a new guarantee for every human line.
Keep the machine contract limited to JSON plan keys and documented exit codes.

## T16-DEV - Make `{{target}}` A Watch-Only Public Concept

Pain point: The implementation still has a split: one-shot run mode builds
framework-aware commands and appends discovered targets, while watch mode
honors `{{target}}` in job commands. The old configuration docs described that
split but still made `{{target}}` feel like something users should reason
about in run-mode job commands.

Change: tighten the public configuration docs:

- describe run mode as appending discovered targets automatically
- tell users not to put `{{target}}` in run-mode job commands
- describe `{{target}}` as a watch-mode job command template only
- keep `{{match}}` and `{{dir_relative}}` as watch target templates

This is a documentation/coherence phase, not an execution change. The runtime
still tolerates old configs, but public docs should teach the cleaner model.

Acceptance criteria:
- `docs/configuration.md` says run-mode `cmd` should omit `{{target}}`.
- `docs/configuration.md` labels `{{target}}` as watch-mode job-command
  behavior.
- Public config examples continue to omit `{{target}}` from job commands.
- Focused doc specs and the full build pass.

Before evidence:
- T12 kept conceptual coherence at 3 because `{{target}}` behavior differed
  across run and watch and remained hard to explain.

After evidence:
- The configuration docs present one simple run-mode rule: targets are appended
  automatically.

Tradeoff: this does not remove the internal compatibility path yet. A later
phase can reject or migrate unsupported `{{target}}` usage once the public docs
have stopped teaching it.

## T17-DEV - Add Structured `watch find` Output

Pain point: T11 made `watch find` much better for humans, but T12 and Darwin's
review correctly kept composability at 3 because scripts still have to parse
watch prose and exit codes. T13 added one-shot JSON; watch previews should have
the same escape hatch.

Change: add a command-local format flag:

```bash
plur watch find --format=json spec/spec_helper.rb
```

The JSON should go to stdout and include:

- output `version`
- `mode = "watch_find"`
- changed `file`
- matched rules
- existing targets by job
- missing targets by job
- the exit code Plur will use

Human `watch find` output stays the default. Exit code 2 still means no
runnable target.

Acceptance criteria:
- `watch find --format=json` emits parseable JSON on stdout.
- No-rule and missing-target cases still exit 2 and include structured details.
- Text `watch find` output remains unchanged by default.
- Output contract docs mention the structured watch format.
- Public docs change is treated as reference material, with an overlap check
  against existing user-facing docs before adding new prose.
- Focused integration tests, Go tests, and the full build pass.

Before evidence:
- `docs/output-contracts.md` explicitly said watch-find human text was not yet
  a machine API.

After evidence:
- `watch find --format=json` provides a stable machine-readable watch preview.
- Documentation check: the only public-doc change is in
  `docs/output-contracts.md`, which is the canonical reference for streams,
  exit codes, and machine formats. `docs/features/watch-mode.md` and
  `docs/usage.md` remain how-to oriented and still link users to `watch find`
  without duplicating JSON key lists.

Tradeoff: this creates a second structured output contract. Keep it aligned
with the dry-run plan shape where practical and do not add a full shared plan
abstraction yet.

Follow-up candidate: run a dedicated user-facing documentation audit phase.
Use Diátaxis to classify `docs/getting-started.md`, `docs/usage.md`,
`docs/configuration.md`, `docs/output-contracts.md`, and feature docs; then
trim duplicated command explanations and move detailed facts to canonical
reference pages.

## T19-DEV - Clean Up Public Docs IA

Pain point: T18 showed that the executable CLI is clearer than the public docs
surface. The docs index links to missing overview pages, `.pages` still allows
internal planning/WIP material into public navigation, `docs/usage.md` mixes
common workflow how-to material with runtime-cache and experimental split
explanations, and `docs/features/watch-mode.md` mixes user how-to content with
architecture and decision-log details.

Change: do a scoped Diataxis cleanup of the public docs:

- make docs navigation explicit so user docs do not automatically include
  `docs/goal/`, `docs/plans/`, `docs/wip/`, or superpowers material
- fix broken overview links without adding future-plan placeholders
- keep `docs/usage.md` focused on common workflows and link to canonical
  references for output contracts, configuration, and feature explanations
- move advanced parallel/runtime/split explanation to the parallel execution
  feature page
- trim watch mode to user-facing how-to/troubleshooting and link architecture
  details to the architecture page

Acceptance criteria:
- `docs/index.md` and `docs/overview/index.md` no longer link to missing pages.
- `docs/.pages` explicitly lists public sections and does not rely on `...`
  to include internal planning directories.
- `docs/usage.md` is a concise how-to page for common workflows.
- `docs/features/watch-mode.md` no longer contains architecture diagrams,
  implementation details, or decision-log sections.
- Advanced runtime-cache and RSpec split explanation still exists in a more
  appropriate feature/explanation page.
- Focused docs checks and a suitable build gate pass.

Before evidence:
- `docs/index.md` links to missing `overview/project-status.md` and
  `overview/roadmap.md`.
- `docs/overview/index.md` repeats the same missing links.
- `docs/.pages` ends with `...`, which lets internal docs appear in generated
  navigation.
- `docs/usage.md` has an `Output Formats` section about internal dual
  formatters even though `docs/output-contracts.md` is now the canonical
  reference.
- `docs/features/watch-mode.md` includes architecture, implementation details,
  known output limitations, and technical decision-log content in the user
  feature page.

After evidence:
- `docs/index.md` and `docs/overview/index.md` link only to existing public
  pages.
- `docs/.pages` lists explicit public pages/sections, and `mkdocs.yml` excludes
  `README.md`, `configuration-test-cases.md`, `goal/**`, `plans/**`,
  `superpowers/**`, and `wip/**` from the generated public site.
- `docs/generate_pages_list.py` uses the same skip policy for generated docs
  listings.
- `docs/usage.md` is now a common-workflows how-to that links to
  `docs/configuration.md`, `docs/output-contracts.md`,
  `docs/features/watch-mode.md`, and `docs/features/parallel-execution.md`
  instead of duplicating reference details.
- `docs/features/watch-mode.md` is now a user-facing watch how-to and links to
  `docs/architecture/plur-watch-architecture.md` for implementation details.
- Runtime-cache and experimental RSpec split explanation moved to
  `docs/features/parallel-execution.md`.
- Verification passed:
  - `script/docs build`
  - `script/check-links`
  - `bin/rake`

Tradeoff: public navigation now hides internal planning/WIP docs from the
generated site, but the source files remain in the repo. MkDocs revision-date
parallel processing is disabled because the plugin hit a git-object memory
failure in this container; serial processing is slower but reliable for this
docs size.

## T20-DEV - Reject Run-Mode `{{target}}` Job Commands

Pain point: T16 made the public docs teach a cleaner rule: one-shot run mode
appends discovered targets automatically, while `{{target}}` is a watch-mode
job command template. Runtime still accepts `{{target}}` in run-mode job
commands and silently strips it, so a config can violate the public model and
appear to work.

Change: when `plur` / `plur spec` selects a job whose `cmd` contains
`{{target}}`, fail before planning or running tests with a direct error:

```text
job "custom" command uses {{target}}, but run mode appends targets automatically; remove {{target}} from job cmd
```

Watch mode keeps accepting `{{target}}` in job commands. This phase should not
change `job.BuildJobCmd` or watch execution semantics.

Acceptance criteria:
- Run mode rejects the selected job when `cmd` contains `{{target}}`.
- The error explains that run mode appends targets automatically and tells the
  user to remove `{{target}}`.
- Watch mode still supports `{{target}}` in job commands.
- Existing tests that used run-mode `{{target}}` fixtures are updated to the
  documented config shape.
- Focused configuration/watch tests and the full build pass.

Before evidence:
- `runner.go` currently logs `ignoring {{target}} tokens in run mode`.
- `framework/run_args.go` strips `{{target}}` and appends targets.
- Integration fixtures and specs still contain run-mode job commands such as
  `cmd = ["bundle", "exec", "rspec", "--fail-fast", "{{target}}"]`.

After evidence:
- Added an outside-in integration spec proving run mode rejects a selected
  user job whose `cmd` contains `{{target}}`.
- Run mode still works with inherited built-in jobs that internally share
  `{{target}}` commands with watch mode:
  `./plur -C fixtures/projects/default-ruby --dry-run spec/models/user_spec.rb`
  succeeds and appends the target once.
- Watch mode still supports `{{target}}`: `./plur -C fixtures/projects/default-ruby watch find --format=json lib/calculator.rb`
  exits 0 with `existing_targets.rspec = ["spec/calculator_spec.rb"]`.
- Updated old run-mode fixtures and docs to omit `{{target}}` from user job
  commands.
- Verification passed:
  - red: `PLUR_BINARY=$PWD/plur bin/rspec spec/integration/spec/configuration_spec.rb:352`
  - green: same focused spec after implementation
  - `PLUR_BINARY=$PWD/plur bin/rspec spec/docs/configuration_target_doc_spec.rb spec/integration/spec/configuration_spec.rb spec/integration/spec/framework_output_spec.rb spec/integration/spec/change_dir_config_spec.rb`
  - `PLUR_BINARY=$PWD/plur bin/rspec spec/integration/watch/watch_find_spec.rb spec/integration/watch/watch_find_json_spec.rb spec/integration/watch/watch_config_spec.rb`
  - `go test -mod=mod ./...`
  - `bin/rake`

Tradeoff: built-in default jobs still carry internal `{{target}}` tokens so the
same job definitions continue to work in watch mode. The user-facing rule is
enforced at the selected run-mode command boundary instead of global config
validation, because global validation would reject valid watch configs.

## T21-DEV - Tighten `watch find` Help

Pain point: `plur watch find --help` is a focused preview command, but Kong's
inherited global/parent flags make it look like a one-shot runner and live
watch command too. It lists `--dry-run`, `--dry-run-format`, `--json`,
`--first-is-1`, `--workers`, `--rspec-split`, and `--ignore` even though
`watch find` has its own `--format` flag and does not execute tests or filter
live file events.

Change: keep the existing Kong-backed help shape, but hide command-irrelevant
inherited flags from `plur watch find --help`. Leave universal/debugging
controls such as `-C`, `--debug`, `--verbose`, and `--version` visible, and do
not change parsing behavior in this phase.

Acceptance criteria:
- `plur watch find --help` still shows usage, `<file-path>`, and `--format`.
- `plur watch find --help` does not show one-shot run flags:
  `--dry-run`, `--dry-run-format`, `--json`, `--first-is-1`, `--workers`, or
  `--rspec-split`.
- `plur watch find --help` does not show live-watch-only `--ignore`.
- `plur watch --help` and `plur watch run --help` still show `--ignore`.
- Focused help specs, Go tests, and the full build pass.

Before evidence:
- `./plur watch find --help` shows the command-specific `--format` flag, but
  also shows inherited run/watch-run flags that do not affect the preview.
- `./plur -C fixtures/projects/default-ruby watch find --ignore='lib/**' lib/calculator.rb`
  still reports `spec/calculator_spec.rb`, proving `--ignore` is live-watch
  event filtering rather than `watch find` filtering.
- Duplication check: existing public docs mention `watch find` in watch-mode
  and output-contract pages, but this phase changes generated CLI help only.

After evidence:
- Added a generated-help integration spec for `plur watch find --help`.
- `./plur watch find --help` shows usage, `<file-path>`, common diagnostic
  flags, and `--format="text"`, without the inherited run/live-watch-only
  flags.
- `./plur watch --help` and `./plur watch run --help` still show `--ignore`.
- Verification passed:
  - red: `PLUR_BINARY=$PWD/plur bin/rspec spec/integration/spec/help_spec.rb`
  - `bin/rake build`
  - green: `PLUR_BINARY=$PWD/plur bin/rspec spec/integration/spec/help_spec.rb`
  - `PLUR_BINARY=$PWD/plur bin/rspec spec/integration/spec/help_spec.rb spec/integration/watch/watch_ignore_spec.rb`
  - `go test -mod=mod ./...`
  - `bin/rake`

Tradeoff: this is still a presentation-only cleanup. The hidden inherited
flags remain accepted by Kong for `watch find`; a future cleanup can reject or
ignore no-op flag combinations consistently across all non-run commands.

## T23-DEV - Reject No-Op `watch find` Flags

Pain point: T21 made `plur watch find --help` focused, but the parser still
accepts the inherited flags hidden from help. For example, `plur watch find
--dry-run lib/calculator.rb`, `--workers=99`, and `--ignore='lib/**'` all exit
successfully while producing the same preview. That makes the command look
cleaner than it behaves.

Change: when `watch find` is invoked with command-irrelevant CLI flags, fail
before calculating targets with a contextual error. This phase only rejects
explicit CLI flags; it should not reject default global values or configuration
loaded for other commands.

Reject these `watch find` flags:

- `--dry-run`
- `--dry-run-format` when paired with `--dry-run`
- `--json`
- `--first-is-1` / `--no-first-is-1`
- `--workers` / `-n`
- `--rspec-split`
- `--ignore`

Acceptance criteria:
- `plur watch find --dry-run FILE` exits non-zero and points to
  `watch find --format=json` for structured watch preview and `plur --dry-run`
  for one-shot run preview.
- `plur watch find --workers=99 FILE`, `-n 2`, `--json=FILE`,
  `--rspec-split`, `--no-first-is-1`, and `--ignore=PATTERN` exit non-zero
  with a message that the flag does not apply to `plur watch find`.
- Valid `watch find` text and JSON output still work.
- Focused watch-find specs, Go tests, and the full build pass.

Before evidence:
- `./plur -C fixtures/projects/default-ruby watch find --dry-run lib/calculator.rb`
  exits 0 and previews `spec/calculator_spec.rb`.
- `./plur -C fixtures/projects/default-ruby watch find --dry-run --dry-run-format=json lib/calculator.rb`
  exits 0 and still prints human text.
- `./plur -C fixtures/projects/default-ruby watch find --workers=99 lib/calculator.rb`
  exits 0 and previews the same target.

Tradeoff: this turns previously harmless no-op flags into errors. That is a
clean break in service of making command-specific help and behavior match.

After evidence:
- Added integration coverage that proves `watch find` rejects explicit no-op
  flags hidden from its help: `--dry-run`, `--dry-run-format`, `--json`,
  `--first-is-1`, `--no-first-is-1`, `--workers`, `-n`, `--rspec-split`, and
  `--ignore`.
- The guard uses Kong's parsed path and ignores config/env-resolved values, so
  `.plur.toml` or environment settings for run-mode flags do not break a plain
  `watch find`.
- `./plur -C fixtures/projects/default-ruby watch find --dry-run-format=json lib/calculator.rb`
  now exits 1 with guidance to use `watch find --format=json` for structured
  watch previews or `plur --dry-run` for one-shot test plans.
- `./plur -C fixtures/projects/default-ruby watch find --workers=99 lib/calculator.rb`
  now exits 1 with `--workers does not apply to plur watch find`.
- `./plur -C fixtures/projects/default-ruby watch find --format=json lib/calculator.rb`
  still emits the stable watch-find JSON plan.
- Verification passed:
  - red: `PLUR_BINARY=$PWD/plur bin/rspec spec/integration/watch/watch_find_spec.rb`
  - `bin/rake build`
  - green: `PLUR_BINARY=$PWD/plur bin/rspec spec/integration/watch/watch_find_spec.rb`
  - `PLUR_BINARY=$PWD/plur bin/rspec spec/integration/watch/watch_find_spec.rb spec/integration/watch/watch_find_json_spec.rb spec/integration/spec/help_spec.rb`
  - `go test -mod=mod ./...`
  - `git diff --check`
  - `bin/rake`

## T24-DEV - Group Top-Level Help Commands

Pain point: T22 kept Brevity / Surface Area below 4 because `plur --help`
still gives day-one commands and setup/maintenance commands equal weight.
`spec`, `watch run`, `watch find`, `watch install`, `rails:init`,
`config init`, `doctor`, and `version` all appear in one flat command list.

Change: keep all commands discoverable, but group the generated top-level
command list so daily commands appear first and advanced/setup commands appear
second. This is a presentation change only; command names, parsing, aliases,
and behavior stay the same.

Acceptance criteria:
- `plur --help` includes a `Daily commands` group before advanced/setup
  commands.
- Daily commands include `spec`, `watch run`, and `watch find`.
- Advanced/setup commands include `watch install`, `rails`, `doctor`,
  `config init`, `rails:init`, and `version`.
- Existing common workflow help remains at the top.
- Focused help specs, Go tests, and the full build pass.

Before evidence:
- `./plur --help` shows one flat `Commands:` list where `watch install` appears
  between `watch run` and `watch find`, and setup commands are visually equal
  to daily commands.

Tradeoff: help gains group headings. This adds presentation structure without
removing commands or adding another help mode.

After evidence:
- `./plur --help` now lists `Daily commands` first:
  `spec`, `watch run`, and `watch find`.
- The same help output then lists `Advanced and setup commands`:
  `watch install`, `rails`, `doctor`, `config init`, `rails:init`, and
  `version`.
- `./plur watch --help` inherits the same useful grouping for watch
  subcommands: `watch run` and `watch find` before `watch install`.
- The commands remain visible and unchanged; this is command metadata and help
  presentation only.
- Verification passed:
  - red: `PLUR_BINARY=$PWD/plur bin/rspec spec/integration/spec/help_spec.rb:3`
  - `bin/rake build`
  - green: `PLUR_BINARY=$PWD/plur bin/rspec spec/integration/spec/help_spec.rb`
  - `PLUR_BINARY=$PWD/plur bin/rspec spec/integration/spec/help_spec.rb spec/integration/watch/watch_find_spec.rb`
  - `go test -mod=mod ./...`
  - `bin/rake`

## T25-DEV - Focus The Docs Front Door

Pain point: T22's docs reviewer kept the public docs below 5/5 because
`docs/index.md` still promotes architecture and development material as large
first-page sections. Those pages are useful, but ordinary users landing on the
docs need the fast path: install, run, configure, watch, and understand output.

Change: rewrite the docs landing page as a user-oriented guide page. Keep
architecture, development, overview, and benchmark material discoverable, but
demote it to a short "Deeper material" section after the user-facing path.
Do not change generated nav in this phase.

Diataxis role: `docs/index.md` is a map/orientation page. It should point to
tutorial/how-to/reference pages without duplicating their details.

Duplication check:
- `docs/getting-started.md` owns installation and first run.
- `docs/usage.md` owns common workflows.
- `docs/configuration.md` owns TOML reference.
- `docs/output-contracts.md` owns stable machine output.
- `docs/features/watch-mode.md` owns watch how-to/troubleshooting.

Acceptance criteria:
- `docs/index.md` starts with a clear heading and user-focused next steps.
- Architecture/development links are still present but no longer large
  first-page sections.
- No new markdown prose specs are added.
- `script/check-links`, `script/docs build`, and the full build pass.

Before evidence:
- `docs/index.md` has large `Architecture` and `Development` sections with
  detailed contributor links directly after the feature list.

Tradeoff: contributors have one less prominent landing-page list, but the nav
and deeper links still expose architecture and development material.

## Follow-Up - Evaluate Zensical Docs Migration

Source: Material for MkDocs 9.7.6 now prints an upstream MkDocs 2.0 warning
during `script/docs build`. The local environment is still pinned to MkDocs
1.6.1 and Material 9.7.6, and Material itself requires `mkdocs<2`, but the
warning is a real long-term docs tooling signal.

References:
- [Material for MkDocs: What MkDocs 2.0 means for your documentation projects](https://squidfunk.github.io/mkdocs-material/blog/2026/02/18/mkdocs-2.0/)
- [Material for MkDocs changelog for 9.7.2-9.7.6](https://squidfunk.github.io/mkdocs-material/changelog/)

Follow-up item: evaluate migrating the docs build from MkDocs to
[Zensical](https://zensical.org/) once the CLI UX loop is complete or when a
docs-tooling phase is selected.

Acceptance criteria:
- Inventory the current docs stack: Material theme options, `social`,
  `awesome-pages`, `git-revision-date-localized`, and `panzoom` plugins.
- Spike a Zensical build against this repo without changing public docs
  content.
- Record unsupported plugin/theme behavior and the migration path.
- If viable, update docs tooling and verify `script/check-links`,
  `script/docs build`, and `bin/rake`.
- If not viable yet, decide explicitly whether `script/docs` should set
  `NO_MKDOCS_2_WARNING=1` while staying pinned to MkDocs 1.x.

## T27-DEV - Stop Presenting `plur test` As A Command

Pain point: `plur test` is useful when a project has a `test/` directory, but
it is not a command. It is a positional target pattern handled by the default
`spec` command path. Current top-level help shows `plur test` in common
workflows, which makes `plur test --help` especially confusing: it renders
`plur spec` help.

Change: make top-level help show an explicit Minitest target path instead of
bare `plur test`, and intercept `plur test --help` / `plur test -h` with a
plain explanation that `test` is a target path, not a Plur command. Do not add
a new `test` command in this phase; the goal is to remove the false command
signal.

Acceptance criteria:
- `plur --help` no longer lists bare `plur test`.
- `plur --help` still shows how to run a Minitest target, for example
  `plur test/calculator_test.rb`.
- `plur test --help` exits non-zero with a direct explanation instead of
  rendering `plur spec` help.
- Existing target inference for real `test/` paths is unchanged.
- Focused help/error specs and the full build pass.

Before evidence:
- `./plur --help` lists `plur test                           Run Minitest
  targets`.
- `./plur test --help` prints `Usage: plur spec [<patterns> ...] [flags]`.

Tradeoff: users who already know `plur test` works can still use it in
projects with a `test/` directory, but new users no longer see a bare target
path presented like a command.

After evidence:
- `./plur --help` now lists `plur test/calculator_test.rb        Run one
  Minitest target`.
- `./plur test --help` exits 1 with:

```text
Error: `test` is a target path, not a Plur command.
Use `plur test/calculator_test.rb` to run a Minitest target.
Use `plur --help` to list Plur commands.
```

- `./plur -C fixtures/projects/minitest-success test --dry-run` still selects
  the `minitest` job by explicit patterns and expands the real `test/`
  directory.

## T28-DEV - Print Expected Command Errors Plainly

Pain point: several expected user errors still look like internal logs:

```text
08:31:48 - ERROR - Command failed error=failed to select watch job: job 'does-not-exist' not found. Available jobs: ...
```

That shape is noisy for humans and brittle for automation. It also conflicts
with newer parser errors that already use direct `plur: error:` or `Error:`
messages.

Change: when a command returns a normal error from `ctx.Run`, print a plain
stderr line:

```text
Error: failed to select watch job: job 'does-not-exist' not found. Available jobs: ...
```

Keep custom `ExitCode` behavior unchanged, because those paths already print
their own output or intentionally exit silently.

Acceptance criteria:
- Invalid `watch find --use=...` exits 1 with `Error: ...`, not timestamped
  `ERROR - Command failed ...`.
- Missing explicit targets exit 1 with `Error: file not found: ...`, not
  timestamped log output.
- Test failure execution behavior is unchanged.
- Focused error specs, Go tests, and the full build pass.

Before evidence:
- `./plur watch find --format=json --use=does-not-exist spec/integration/spec/help_spec.rb`
  exits 1 with a timestamped `ERROR - Command failed` stderr line.

After evidence:
- Invalid watch job selection now exits 1 with no stdout and plain stderr:

```text
Error: failed to select watch job: job 'does-not-exist' not found. Available jobs: build, go-test, minitest, plur, rails, rake, rspec
```

- Missing explicit targets still print the version banner first, then a plain
  error:

```text
plur version=v0.56.1-0.20260523134152-3a34c4192f1f+dirty
Error: file not found: spec/nonexistent_spec.rb
```

## T29-DEV - Quote Dry-Run Shell Strings

Pain point: dry-run JSON already exposes `argv` and `env`, but it also includes
`workers[].shell` as a convenience string. Today that string is built with a
plain space join, so targets containing spaces or shell metacharacters are not
copyable and can mislead scripts into parsing the wrong field.

Change: quote command argv when building dry-run shell strings, and update the
output-contract docs to state that `argv` and `env` are the canonical machine
contract. Keep `shell` as a copyable human convenience field.

Acceptance criteria:
- `dryRunString` quotes args with spaces and single quotes.
- Existing no-special-character dry-run strings remain unchanged.
- `docs/output-contracts.md` says scripts should prefer `argv` and `env`.
- Focused Go tests, docs checks, and the full build pass.

Before evidence:
- `dryRunString(exec.Command("bundle", "exec", "rspec", "spec/my spec_spec.rb"))`
  produced `bundle exec rspec spec/my spec_spec.rb`, which is not a faithful
  shell command for the single target path.

After evidence:
- `dryRunString(exec.Command("bundle", "exec", "rspec", "spec/my spec_spec.rb",
  "spec/quote's_spec.rb"))` now returns:

```text
bundle exec rspec 'spec/my spec_spec.rb' 'spec/quote'\''s_spec.rb'
```

- Existing simple dry-run strings such as `bundle exec rspec spec/foo_spec.rb`
  remain unchanged.
- `docs/output-contracts.md` now identifies `argv` and `env` as the canonical
  script fields and `shell` as a quoted human convenience string.
