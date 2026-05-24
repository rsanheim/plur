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

## T31-DEV - Explain Missing Bare `test` Targets

Pain point: T27 made `plur test --help` explain that `test` is a target path,
not a Plur command, but the more likely mistake still falls through:

```text
plur test --dry-run
Error: file not found: test
```

That is technically correct in a repo without a `test/` directory, but it does
not connect the failure to the target-path mental model.

Change: keep target discovery behavior unchanged, but specialize the missing
bare `test` target error:

```text
Error: file not found: test; `test` is a target path, not a Plur command. Create a test/ directory or pass a Minitest target like test/calculator_test.rb
```

Acceptance criteria:
- `plur test --dry-run` in a repo without `test/` exits 1 with the explanatory
  target-path message.
- Other missing target errors remain unchanged.
- A real `test/` directory still selects the Minitest job by explicit patterns.
- Focused error specs, Go tests, and the full build pass.

Before evidence:
- `./plur test --dry-run` exits 1 with `Error: file not found: test`.

After evidence:
- In this repo, which has no top-level `test/` directory,
  `./plur test --dry-run` exits 1 with:

```text
Error: file not found: test; `test` is a target path, not a Plur command. Create a test/ directory or pass a Minitest target like test/calculator_test.rb
```

- In `fixtures/projects/minitest-success`, `./plur -C fixtures/projects/minitest-success test --dry-run`
  still selects `minitest` by explicit patterns and expands `test/` to the two
  fixture test files.

## T32-DEV - Clarify Output Error Contracts

Pain point: the output contract docs still implied that exit code 1 only means
selected work ran and failed. Recent CLI cleanup made command errors plain and
less log-like, but those errors still exit 1 and may happen before JSON output
exists. That made the docs overpromise for automation.

Change: clarify that exit code 1 can mean either selected work failed or Plur
could not plan/run the command. Document that JSON modes emit structured stdout
only after Plur successfully builds a dry-run/watch plan; command and
configuration errors remain plain stderr. Also state directly that scripts
should execute from `argv` and `env`, not by parsing the human `shell` string.

Acceptance criteria:
- `docs/output-contracts.md` says exit code 1 can cover failed selected work or
  planning/running errors.
- Dry-run JSON and watch-find JSON sections document stderr-only command/config
  errors.
- The dry-run worker schema says `shell` is for humans and scripts should use
  `argv` and `env`.
- Focused docs spec, link check, and full build pass.

Before evidence:
- `docs/output-contracts.md` said: "Exit code 1 means selected work ran and
  failed."
- `./plur watch find --format=json --use=does-not-exist spec/integration/spec/help_spec.rb`
  exits 1 with empty stdout and a plain stderr `Error: ...` line.

## T33-DEV - Remove Unused `--json` Flag

Pain point: `plur --help`, `plur spec --help`, and watch help still advertise
`--json="" Save detailed test results as JSON to the specified file`, but the
flag is not implemented. That creates a third machine-output surface next to
the real contracts: `--dry-run-format=json` and `watch find --format=json`.

Change: remove the unused global `--json` flag from the CLI and config object.
This is a clean break: users who try `--json` should get an unknown-flag error
instead of a silently ignored option. Keep the existing structured APIs as the
canonical JSON surfaces.

Diataxis / duplication check:
- This is generated CLI reference/help, not a tutorial or how-to guide.
- `rg -- "--json|JSON output|Save detailed test results"` shows no active
  public user docs for the flag; references are in historical goal notes,
  release tooling `gh --json` calls, tests, and code.

Acceptance criteria:
- Top-level, `spec`, `watch`, and `watch run` help no longer show `--json`.
- `plur --json=tmp/results.json --dry-run` exits non-zero as an unknown flag.
- `watch find` no longer needs a bespoke rejection path for `--json`.
- Focused help/watch specs, Go tests, and the full build pass.

Before evidence:
- `./plur --help`, `./plur spec --help`, `./plur watch --help`, and
  `./plur watch run --help` all list `--json="" Save detailed test results as
  JSON to the specified file`.

After evidence:
- Top-level, `spec`, `watch`, and `watch run` help no longer list `--json` or
  `Save detailed test results as JSON`.
- Removed `JSON` from `PlurCLI` and `config.GlobalConfig`; `watch find` no
  longer carries a special `--json` no-op rejection path.
- `./plur --json=tmp/results.json --dry-run` exits non-zero with:

```text
plur: error: unknown flag --json, did you mean "--job"?
```

- Verification:
  - red: `PLUR_BINARY=$PWD/plur bin/rspec spec/integration/spec/help_spec.rb spec/integration/watch/watch_find_spec.rb`
  - green: `PLUR_BINARY=$PWD/plur bin/rspec spec/integration/spec/help_spec.rb spec/integration/watch/watch_find_spec.rb`
  - `go test -mod=mod ./...`
  - `script/check-links`
  - `bin/rake`

## T35-DEV - Explain Removed `--json`

Pain point: T33 removed the unused global `--json` file-output flag, but the
raw Kong parser error now says:

```text
plur: error: unknown flag --json, did you mean "--job"?
```

That is technically correct but unhelpful: the closest useful alternatives are
the two real structured preview APIs, not `--job`.

Change: pre-parse `--json` and `--json=...` before Kong, and print direct
guidance:

```text
Error: --json is not a Plur flag.
Use `plur --dry-run --dry-run-format=json [patterns...]` for a structured one-shot plan.
Use `plur watch find --format=json <file>` for a structured watch preview.
```

Acceptance criteria:
- `plur --json=tmp/results.json --dry-run` exits 1 with the direct guidance.
- `plur watch find --json=tmp/watch-find.json FILE` exits 1 with the same
  guidance, not a generic unknown-flag suggestion.
- `--json` after a passthrough `--` remains passthrough input for commands that
  support passthrough args.
- Focused help/watch specs, Go tests, and the full build pass.

Before evidence:
- `./plur --json=tmp/results.json --dry-run` exits 80 with
  `unknown flag --json, did you mean "--job"?`.

After evidence:
- `./plur --json=tmp/results.json --dry-run` and
  `./plur watch find --json=tmp/watch-find.json spec/spec_helper.rb` both exit
  1 with:

```text
Error: --json is not a Plur flag.
Use `plur --dry-run --dry-run-format=json [patterns...]` for a structured one-shot plan.
Use `plur watch find --format=json <file>` for a structured watch preview.
```

- `./plur -C fixtures/projects/default-ruby --dry-run spec/calculator_spec.rb -- --json`
  still treats `--json` as an RSpec passthrough arg after `--`.
- Verification:
  - red: `PLUR_BINARY=$PWD/plur bin/rspec spec/integration/spec/help_spec.rb spec/integration/watch/watch_find_spec.rb`
  - green: `PLUR_BINARY=$PWD/plur bin/rspec spec/integration/spec/help_spec.rb spec/integration/watch/watch_find_spec.rb spec/integration/spec/rspec_args_spec.rb`
  - `script/check-links`
  - `bin/rake`

## T36-DEV - Align `watch run` Dry-Run Help

Pain point: `plur watch --help` already explains that `--dry-run` is a
one-shot preview flag and watch mode rejects it. But `plur watch run --help`
still inherits the generic run wording:

```text
--dry-run                  Print what would be executed without running
--dry-run-format="text"    Dry-run output format: text or json
```

That creates a small but real mismatch: the command-specific help for the
actual watch runner is less accurate than the parent watch help.

Change: apply the same dry-run help wording to `plur watch run --help`:

```text
--dry-run                  One-shot run preview only; watch mode rejects it
--dry-run-format="text"    One-shot dry-run output format: text or json
```

Acceptance criteria:
- `plur watch run --help` shows the watch-specific dry-run wording.
- `plur watch --help` keeps the same wording.
- `plur spec --help` keeps generic one-shot dry-run wording.
- Focused help specs, watch ignore help specs, and the full build pass.

Before evidence:
- `./plur watch run --help` shows generic `--dry-run` help even though
  `plur watch run --dry-run` exits with watch dry-run guidance.

After evidence:
- `./plur watch run --help` now shows:

```text
--dry-run                  One-shot run preview only; watch mode rejects it
--dry-run-format="text"    One-shot dry-run output format: text or json
```

- `./plur spec --help` still keeps one-shot run wording:

```text
--dry-run                  Print what would be executed without running
--dry-run-format="text"    Dry-run output format: text or json
```

- Verification:
  - red: `PLUR_BINARY=$PWD/plur bin/rspec spec/integration/spec/help_spec.rb`
  - green: `PLUR_BINARY=$PWD/plur bin/rspec spec/integration/spec/help_spec.rb spec/integration/watch/watch_ignore_spec.rb`
  - `go test -mod=mod ./...`
  - `script/check-links`
  - `bin/rake`

## T37-DEV - Add A Skimmable Dry-Run Summary

Pain point: human dry-run starts well with selected job and run count, then
immediately prints long worker commands. For larger suites, the user's first
skim has to parse command strings to understand the plan shape.

Change: keep the copyable worker commands, but add a compact human-only summary
and a divider before them:

```text
[dry-run] Plan: 13 targets across 4 workers; no tests will run
[dry-run] Commands:
```

JSON dry-run is unchanged; scripts already use `targets` and `workers`.

Acceptance criteria:
- Text dry-run prints the plan summary before worker commands.
- Text dry-run prints a `Commands:` divider before the first worker.
- Dry-run JSON output still does not include human worker-command lines on
  stderr.
- Focused dry-run specs, Go tests, and the full build pass.

Before evidence:
- `./plur --dry-run spec/integration/spec/help_spec.rb spec/integration/watch/watch_find_spec.rb`
  jumps from `Running 2 specs ...` directly to `[dry-run] Worker 0: ...`.

After evidence:
- `./plur --dry-run spec/integration/spec/help_spec.rb spec/integration/watch/watch_find_spec.rb`
  now prints:

```text
[dry-run] Running 2 specs [rspec] in parallel using 2 workers
[dry-run] Plan: 2 targets across 2 workers; no tests will run
[dry-run] Commands:
[dry-run] Worker 0: PARALLEL_TEST_GROUPS=2 TEST_ENV_NUMBER=1 bin/rspec ...
```

- `./plur --dry-run --dry-run-format=json spec/integration/spec/help_spec.rb`
  still prints only the version line to stderr and keeps the plan on stdout.
- Verification:
  - red: `PLUR_BINARY=$PWD/plur bin/rspec spec/integration/spec/general_integration_spec.rb spec/integration/spec/dry_run_plan_spec.rb`
  - green: `PLUR_BINARY=$PWD/plur bin/rspec spec/integration/spec/general_integration_spec.rb spec/integration/spec/dry_run_plan_spec.rb`
  - green after snapshot updates: `PLUR_BINARY=$PWD/plur bin/rspec spec/integration/spec/general_integration_spec.rb spec/integration/spec/runtime_tracking_spec.rb spec/integration/spec/turbo_tests_migration_spec.rb spec/integration/spec/rspec_args_spec.rb spec/integration/spec/framework_output_spec.rb spec/integration/spec/dry_run_plan_spec.rb`
  - `go test -mod=mod ./...`
  - `script/check-links`
  - `bin/rake`

## T39-DEV - Keep Worker Command Errors Off Stdout

Pain point: agent and CI workflows need stdout/stderr separation to be
predictable. A worker command that writes stderr and exits before producing test
events currently prints the same stderr twice: once on stderr while the worker
runs, then again on stdout when Plur renders errored worker output.

Root cause: `runCommand` appends captured stderr into `WorkerResult.Output`.
`PrintResults` later prints errored `WorkerResult.Output` with `fmt.Print`, so
command-level stderr is replayed on stdout for `StateError` results.

Change: preserve live stderr streaming, but do not replay captured stderr as
stdout for worker command errors. Normal test stdout, progress, summaries, and
test failure details should keep their current streams.

Acceptance criteria:
- A custom job that writes `WORKER_STDERR_MARKER` to stderr and exits non-zero
  includes that marker on stderr.
- The same marker does not appear on stdout.
- The command still exits 1 and prints the usual Plur summary on stdout.
- Focused output specs, Go tests, and the full build pass.

Before evidence:

```text
status=1
--- stdout ---

Finished in 0.00030 seconds (files took 0 seconds to load)
0 examples, 0 failures
WORKER_STDERR_MARKER
--- stderr ---
plur version=v0.56.1-0.20260523143117-7401c595d485+dirty
Running 1 spec [rspec] serially
WORKER_STDERR_MARKER
```

After evidence:

```text
status=1
--- stdout ---

Finished in 0.00054 seconds (files took 0 seconds to load)
0 examples, 0 failures
--- stderr ---
plur version=v0.56.1-0.20260523143740-3bfcf99080ae+dirty
Running 1 spec [rspec] serially
WORKER_STDERR_MARKER
```

- Verification:
  - red: `PLUR_BINARY=$PWD/plur bin/rspec spec/integration/spec/output_spec.rb`
  - green: `bin/rake build`
  - green: `PLUR_BINARY=$PWD/plur bin/rspec spec/integration/spec/output_spec.rb`
  - `go test -mod=mod ./...`
  - `script/check-links`
  - `bin/rake`

## T40-DEV - Refresh Output Contracts Reference

Pain point: `docs/output-contracts.md` is the right public reference for
scripts and agents, but it still shows old dry-run text that jumps straight
from selected job to worker commands. It also does not show concrete
stdout/stderr examples for JSON previews, JSON-mode errors, or watch-find
no-op previews.

Diátaxis role: reference. Keep this page focused on stable contracts and
examples, not a tutorial or workflow guide.

Duplication check: `README.md`, `docs/features/watch-mode.md`, and
`docs/usage.md` mention dry-run or `watch find`, but none are the canonical
stdout/stderr contract. Goal docs and specs contain historical examples only.

Change: update `docs/output-contracts.md` so it names the current dry-run text
shape, explains stdout/stderr roles after T39, and includes concise command
examples for:
- text dry-run stderr
- JSON dry-run stdout plus version stderr
- JSON-mode parser error with empty stdout
- `watch find --format=json` no-op exit code 2

Acceptance criteria:
- Output contracts show the new `Plan` and `Commands` lines.
- JSON sections clearly state that successful machine JSON goes to stdout.
- Error examples show that parser/config errors can write plain stderr and no
  stdout.
- Link checks and the full build pass.

After evidence:
- `docs/output-contracts.md` now documents worker stderr streaming, the text
  dry-run `Plan` and `Commands` lines, JSON dry-run stdout/stderr separation,
  parser error `exit=80` with empty stdout, and watch-find JSON `exit=2`.
- Verification:
  - `script/check-links`
  - red/green docs contract spec:
    `bin/rspec spec/docs/output_contracts_doc_spec.rb`
  - `bin/rake`

## T41-DEV - Make Dry-Run Summary Job-Neutral

Pain point: T37 added a useful plan summary, but it says `no tests will run`.
That is clear for RSpec and Minitest, but Plur jobs can also run linters,
custom checks, Rails/Rake tasks, or other commands. The dry-run guarantee is
that Plur will not execute commands.

Change: update the human dry-run summary from:

```text
[dry-run] Plan: 1 target across 1 worker; no tests will run
```

to:

```text
[dry-run] Plan: 1 target across 1 worker; no commands will run
```

Acceptance criteria:
- Text dry-run uses `no commands will run`.
- Text dry-run no longer emits `no tests will run`.
- Backspin dry-run snapshots and output contract docs match the new wording.
- Focused specs, link checks, and the full build pass.

After evidence:
- `./plur --dry-run ...` now prints
  `[dry-run] Plan: ...; no commands will run`.
- Current user-facing docs and dry-run Backspin snapshots use the new wording.
- Historical T37 notes still show the old wording as phase history; T41 records
  the replacement.
- Verification:
  - red: `PLUR_BINARY=$PWD/plur bin/rspec spec/integration/spec/general_integration_spec.rb`
  - green: `bin/rake build`
  - green: `PLUR_BINARY=$PWD/plur bin/rspec spec/integration/spec/general_integration_spec.rb spec/integration/spec/framework_output_spec.rb spec/integration/spec/rspec_args_spec.rb spec/integration/spec/turbo_tests_migration_spec.rb spec/docs/output_contracts_doc_spec.rb`
  - `script/check-links`
  - `bin/rake`

## T43-DEV - Route Worker Startup Errors To Stderr

Pain point: T39 stopped subprocess stderr from replaying on stdout, but a
worker command that cannot start still puts Plur's own runtime error on stdout:

```text
Finished in 0.00021 seconds (files took 0 seconds to load)
0 examples, 0 failures
Error: failed to start command: exec: "definitely-not-a-real-plur-command": executable file not found in $PATH
```

That breaks the output contract for scripts. Framework test output and summaries
belong on stdout; Plur runtime/startup errors belong on stderr.

Root cause: `errorResult()` stores startup errors in `WorkerResult.Output`.
`PrintResults()` prints errored worker output with stdout-oriented `fmt.Print`.

Change: keep framework errored output on stdout, but do not use
`WorkerResult.Output` for Plur startup errors. If an errored worker has no
captured output and carries a non-exit runtime error, print that error to
stderr.

Acceptance criteria:
- A missing worker executable exits 1.
- `failed to start command` appears on stderr.
- `failed to start command` does not appear on stdout.
- Existing RSpec syntax/error-output behavior remains on stdout.
- Focused output/error specs and the full build pass.

After evidence:
- Red: `PLUR_BINARY=$PWD/plur bin/rspec spec/integration/spec/output_spec.rb`
  failed because stderr did not include `failed to start command`.
- Green: `PLUR_BINARY=$PWD/plur bin/rspec spec/integration/spec/output_spec.rb spec/integration/spec/error_handling_spec.rb`
  passed with 13 examples, 0 failures.
- `go test -mod=mod ./...` passed.
- `script/check-links` passed.
- `bin/rake` passed with 372 examples, 0 failures, and 4 existing pending
  examples.

## T44-DEV - Include Job Env In Dry-Run JSON

Pain point: `docs/output-contracts.md` tells scripts to use `workers[].argv`
and `workers[].env` as the executable dry-run plan. But `workers[].env`
currently omits configured job env entries from `.plur.toml`, even though
`buildEnv()` appends them to the actual command environment.

Change: make `dryRunEnv()` include environment entries Plur adds beyond the
inherited process environment. That includes `PARALLEL_TEST_GROUPS`,
`TEST_ENV_NUMBER`, and configured `job.Env`, without dumping the full inherited
shell environment.

Acceptance criteria:
- Dry-run JSON for a job with `env = ["CUSTOM_TOKEN=secret"]` includes that
  entry in `workers[].env`.
- Dry-run text worker commands include the same configured env entry.
- Dry-run JSON still does not include unrelated inherited env such as `PATH`.
- Focused dry-run JSON specs, Go tests, and the full build pass.

After evidence:
- Red: `PLUR_BINARY=$PWD/plur bin/rspec spec/integration/spec/dry_run_plan_spec.rb`
  failed because `workers[].env` only had `PARALLEL_TEST_GROUPS` and
  `TEST_ENV_NUMBER`.
- Green: `PLUR_BINARY=$PWD/plur bin/rspec spec/integration/spec/dry_run_plan_spec.rb`
  passed after `dryRunEnv()` started including Plur-added env entries.
- `go test -mod=mod ./...` passed.
- `PLUR_BINARY=$PWD/plur bin/rspec spec/integration/spec/dry_run_plan_spec.rb spec/integration/spec/general_integration_spec.rb`
  passed with 29 examples, 0 failures, and 2 existing pending examples.
- `bin/rspec spec/docs/output_contracts_doc_spec.rb` passed.
- `script/check-links` passed.
- First `bin/rake` attempt caught a Rails/Rake regression: caller-provided
  `RAILS_ENV=test` disappeared from dry-run and verbose command strings. The
  fix restores `RAILS_ENV` as an inherited env entry that Plur intentionally
  displays.
- Green after regression fix:
  `PLUR_BINARY=$PWD/plur bin/rspec spec/integration/spec/dry_run_plan_spec.rb spec/integration/spec/rails_rake_spec.rb`
  passed with 22 examples, 0 failures.
- `bin/rake` passed with 373 examples, 0 failures, and 4 existing pending
  examples.

## T45-DEV - Keep Empty Watch JSON Structured

Status: committed
Commit: 334650255e90428929dde0742a7428451c7c5e42

Pain point: `plur watch find --format=json` is the stable machine-readable
watch preview, but the no-watch-mapping path returned human prose on stdout:

```text
No watch mappings configured.
Either add job/watch configuration to .plur.toml or ensure your project structure
matches a supported framework (Ruby with Gemfile, Go with go.mod).
```

That breaks scripts and agents that choose JSON mode before they know whether a
project has watch mappings.

Change: after normalizing the changed file path, handle an empty watch mapping
set inside the same JSON formatter used by other `watch find` no-op previews.
JSON mode emits an empty `watch_find` plan and exits 2. Text mode keeps the
existing human guidance.

Diataxis role: this updates `docs/output-contracts.md` as reference material,
not a tutorial or how-to guide.

Duplication check:
- `docs/output-contracts.md` owns the stable JSON fields and exit codes.
- `docs/features/watch-mode.md` owns human watch-mode usage examples.
- `docs/usage.md` only points to `watch find` as a workflow.

Acceptance criteria:
- A project with a selected job but no configured watch mappings prints valid
  JSON for `watch find --format=json FILE`.
- The JSON shape matches the existing empty no-rule preview:
  `matched_rules: []`, `existing_targets: {}`, `missing_targets: {}`,
  `exit_code: 2`.
- Stdout contains no human prose in JSON mode.
- Text mode with no watch mappings still prints the existing guidance.
- Focused watch JSON specs, output-contract docs spec, Go tests, link check,
  and the full build pass.

After evidence:
- Red: `PLUR_BINARY=$PWD/plur bin/rspec spec/integration/watch/watch_find_json_spec.rb`
  failed because JSON mode tried to parse stdout beginning with
  `No watch mappings configured.`
- A sidecar reviewer caught a precedence risk: moving job selection after the
  empty-watch branch could mask an explicit invalid `--use`. Added a regression
  for `--use=missing` in a project with no watch mappings; it failed with exit
  2 before the fix.
- Green focused checks:
  `PLUR_BINARY=$PWD/plur bin/rspec spec/integration/watch/watch_find_json_spec.rb spec/integration/watch/watch_find_spec.rb`
  passed with 10 examples, 0 failures.
- `bin/rspec spec/docs/output_contracts_doc_spec.rb` passed with 2 examples,
  0 failures.
- `go test -mod=mod ./...` passed.
- `script/check-links` passed.
- `bin/rake` passed with 376 examples, 0 failures, and 4 existing pending
  examples.

## T46-DEV - Focus `watch run` Flags On Live Watching

Status: committed
Commit: 02fd8b80fcc6d06bc29bb18a96be8388c59c30c0

Pain point: `plur watch run --help` still advertises inherited one-shot run
flags such as `--workers`, `--first-is-1`, and `--rspec-split`, even though
live watch execution currently runs configured watch jobs directly and does not
use Plur's parallel one-shot runner. That makes the watch surface look more
powerful than it is and invites no-op flags.

Change: keep live-watch controls visible (`--ignore`, `--timeout`,
`--debounce`, `--use`, `-C`, debug/verbose), but hide one-shot runner controls
from watch help and reject explicit no-op runner flags on `watch run`.
`--dry-run` keeps the existing watch-preview guidance.

Diataxis role: this is generated CLI help and behavior. Public docs should link
to existing watch and output-contract references rather than duplicate a new
flag matrix.

Duplication check:
- `spec/integration/spec/help_spec.rb` owns generated help expectations.
- `spec/integration/watch/watch_find_spec.rb` already rejects no-op flags for
  `watch find`.
- `docs/features/watch-mode.md` owns watch usage and already points previews to
  `watch find`.

Acceptance criteria:
- `plur watch --help` and `plur watch run --help` do not show flag rows for
  `--workers`, `--first-is-1`, `--rspec-split`, `--dry-run`, or
  `--dry-run-format`.
- `plur watch --help` may still mention `plur --dry-run [patterns...]` as a
  separate one-shot workflow.
- `plur watch run --workers=99`, `-n 2`, `--rspec-split`,
  `--first-is-1`, `--no-first-is-1`, and `--dry-run-format=json` exit before
  starting watch mode with direct guidance.
- `plur watch --dry-run` still exits with the existing preview guidance.
- Focused help/watch specs, Go tests, and the full build pass.

After evidence:
- Red: `PLUR_BINARY=$PWD/plur bin/rspec spec/integration/spec/help_spec.rb spec/integration/watch/watch_run_flags_spec.rb`
  failed because watch help still showed inherited one-shot flags and
  `watch run --workers=99 --timeout 1` started watch mode.
- Green focused checks:
  `PLUR_BINARY=$PWD/plur bin/rspec spec/integration/spec/help_spec.rb spec/integration/watch/watch_run_flags_spec.rb spec/integration/watch/watch_dry_run_spec.rb spec/integration/watch/watch_ignore_spec.rb`
  passed with 15 examples, 0 failures.
- `go test -mod=mod ./...` initially failed once in
  `TestTestCollectorComplexity` and then passed on rerun, confirming the
  repeated full-gate failure was timing noise in the smallest baseline. The
  complexity helper now skips scaling-ratio checks when the previous size ran
  below 500 microseconds.
- `go test -mod=mod ./...` passed after that stabilization.
- `script/check-links` passed.
- `bin/rake` passed with 377 examples, 0 failures, and 4 existing pending
  examples.

## T48-DEV - Use One Target-Passing Rule In Watch Jobs

Status: committed
Commit: 2da1ea0eb6b54be142da15743dd586fb7b06391e

Pain point: `watch find` can report that a changed file maps to a target, but
actual watch execution can drop that target if the selected job command does
not contain `{{target}}`. That is especially bad because generated config
templates define watch jobs like:

```toml
[job.rspec]
cmd = ["bundle", "exec", "rspec"]
```

In one-shot run mode, Plur appends targets to that command automatically. In
watch mode, the same command currently runs without targets.

Change: make watch execution use the same target command builder as one-shot
execution for changed targets. If `cmd` contains `{{target}}`, Plur uses that
placement. If it does not, Plur appends the resolved targets to the command.
Running all tests from the interactive watch prompt still uses the no-target
command path.

Diataxis role: update `docs/configuration.md` as reference material; do not add
a broad tutorial.

Duplication check:
- `docs/configuration.md` owns the public `cmd` and `{{target}}` semantics.
- `docs/architecture/runner-jobs-framework.md` has a short architecture note
  that should match current command building.
- `job.BuildJobCmd` already documents the intended target appending behavior.

Acceptance criteria:
- Watch execution appends targets when the job command has no `{{target}}`.
- Watch execution still honors `{{target}}` placement when present.
- Watch execution with no targets still does not run a target command.
- Interactive watch "run all tests" remains a no-target command.
- Configuration docs no longer say watch jobs without `{{target}}` run without
  targets.
- Focused watch/job tests, configuration docs spec, Go tests, link check, and
  the full build pass.

Before evidence:
- `go test -mod=mod ./watch -run TestExecuteJob_WithoutTargetPlaceholder -count=1`
  failed because `$@` was empty: expected `"file1.rb file2.rb\n"`, actual
  `"\n"`. That showed watch execution dropped targets when `cmd` had no
  `{{target}}`.

After evidence:
- `go test -mod=mod ./watch -run 'TestExecuteJob_(WithoutTargetPlaceholder|BatchesMultipleTargets|NoTargets|SingleTarget)' -count=1`
  passed.
- `go test -mod=mod ./watch` passed.
- `bin/rspec spec/docs/configuration_target_doc_spec.rb` passed.
- `go test -mod=mod ./...` passed.
- `script/check-links` passed.
- `bin/rake` passed with 377 examples, 0 failures, and 4 existing pending
  examples.

## T49-DEV - Normalize Empty Watch-Find Text Exit Code

Status: committed
Commit: 0652eebccf70b969a5e7a402662d570b28cc6875

Pain point: `plur watch find --format=json FILE` exits 2 when no watch mappings
are configured, but text mode prints the same no-mapping condition and exits 0.
That makes the preview less predictable for shells and agents: exit 2 should
mean "the command ran, but nothing would run for this file".

Change: keep the existing human guidance text, but return exit code 2 for text
mode when no watch mappings are configured. Invalid explicit job selection keeps
exiting 1 before the no-mapping preview.

Diataxis role: update `docs/output-contracts.md` as reference material because
this is an exit-code contract, not a tutorial.

Duplication check:
- `docs/output-contracts.md` owns stable streams, formats, and exit codes.
- `docs/features/watch-mode.md` and `docs/usage.md` mention `watch find` as
  workflow docs and do not need the low-level no-mapping exit detail.
- `watch_find_json_spec.rb` already covers the JSON no-mapping contract.

Acceptance criteria:
- Text `watch find` with no configured watch mappings exits 2.
- The human guidance text for no configured watch mappings remains unchanged.
- JSON no-mapping output still emits structured JSON and exits 2.
- Invalid explicit `--use` with no watch mappings still exits 1.
- Focused watch specs, output-contract docs checks, Go tests, link check, and
  the full build pass.

Before evidence:
- Red: `PLUR_BINARY=$PWD/plur bin/rspec spec/integration/watch/watch_find_spec.rb --example "keeps human guidance"`
  failed with expected exit status 2 and actual 0.

After evidence:
- `bin/rake build` passed and rebuilt `./plur`.
- `PLUR_BINARY=$PWD/plur bin/rspec spec/integration/watch/watch_find_spec.rb --example "keeps human guidance"`
  passed.
- `PLUR_BINARY=$PWD/plur bin/rspec spec/integration/watch/watch_find_spec.rb spec/integration/watch/watch_find_json_spec.rb`
  passed with 10 examples, 0 failures.
- `bin/rspec spec/docs/output_contracts_doc_spec.rb` passed with 2 examples,
  0 failures.
- `go test -mod=mod ./...` passed.
- `script/check-links` passed.
- `git diff --check` passed.
- `bin/rake` passed with 377 examples, 0 failures, and 4 existing pending
  examples.

## T50-DEV - Deduplicate Dry-Run JSON Environment Entries

Status: committed
Commit: aab8066118cf0839ec1bb00cb4081270a1698660

Pain point: dry-run JSON is the script-friendly one-shot plan, but a configured
job can make `workers[].env` ambiguous by repeating an environment key. The
actual process environment uses the last value for a duplicated key, while the
JSON plan can currently show both entries.

Change: deduplicate dry-run environment entries by key before rendering JSON or
the human shell string. Keep the final effective value for each key, matching
`exec.Cmd.Environ()` semantics.

Diataxis role: update `docs/output-contracts.md` as reference material because
this tightens the `workers[].env` contract.

Duplication check:
- `docs/output-contracts.md` owns dry-run JSON fields and shell guidance.
- `docs/configuration.md` explains job `env` configuration and does not need
  dry-run rendering details.
- `utils.go` already centralizes dry-run env extraction for text and JSON, so
  the change should stay there rather than adding formatter-specific cleanup.

Acceptance criteria:
- Dry-run JSON `workers[].env` contains at most one entry per key.
- Duplicate configured job env entries keep the last value.
- Duplicate configured job env can override managed env keys in the rendered
  plan, matching execution semantics.
- Dry-run human `shell` uses the same deduplicated entries.
- Focused dry-run JSON specs, Go tests, output-contract docs checks, link
  check, and the full build pass.

Before evidence:
- Red: `PLUR_BINARY=$PWD/plur bin/rspec spec/integration/spec/dry_run_plan_spec.rb --example "deduplicates worker env"`
  failed because `workers[].env` contained both `CUSTOM_TOKEN=old` and
  `CUSTOM_TOKEN=new`.
- Red: `go test -mod=mod . -run 'TestDryRunString/duplicate_env_vars_keep_final_value' -count=1`
  failed because dry-run env preserved duplicate `CUSTOM_TOKEN` and
  `TEST_ENV_NUMBER` entries.

After evidence:
- `go test -mod=mod . -run 'TestDryRunString/duplicate_env_vars_keep_final_value' -count=1`
  passed.
- `bin/rake build` passed and rebuilt `./plur`.
- `PLUR_BINARY=$PWD/plur bin/rspec spec/integration/spec/dry_run_plan_spec.rb --example "deduplicates worker env"`
  passed.
- `PLUR_BINARY=$PWD/plur bin/rspec spec/integration/spec/dry_run_plan_spec.rb`
  passed with 5 examples, 0 failures.
- `bin/rspec spec/docs/output_contracts_doc_spec.rb` passed with 2 examples,
  0 failures.
- `go test -mod=mod . -run TestDryRunString -count=1` passed.
- `go test -mod=mod ./...` passed.
- `script/check-links` passed.
- `git diff --check` passed.
- `bin/rake` passed with 378 examples, 0 failures, and 4 existing pending
  examples.

## T51-ARCH - Review Watch Find And Live Watch Planning

Status: verified
Commit: 13ebbf1e1113c62396603eba27bb274c8d8d3b96

Pain point: `plur watch find FILE` and live `plur watch` share
`watch.FindTargetsForFile`, but they do not share the full watch planning path.
The result is structural drift around selected job lookup, watch directory
planning, global ignores, event admission, reload behavior, cwd normalization,
and final command planning.

Change: pause edge-case patching and record the architecture review for a
multi-phase refactor. The target shape is one shared core path from runtime
config/session setup through file-event planning. `watch find` presents the
plan; live watch executes the plan and remains persistent.

Advisor review:
- Herschel traced live watch from CLI/config through watcher event processing
  and execution.
- Hubble traced `watch find` and classified output-only differences versus
  semantic drift.
- Locke proposed the implementation approach: extract a pure `watch.Planner`,
  then add a shared watch session facade.

Decision:
- T52 starts with characterization tests.
- Then extract event admission, pure planning, `watch find` rendering from the
  same plan, and shared session setup in separate DEV phases.
- Avoid a big-bang event-pipeline rewrite.

Artifacts:
- Review note: `docs/goal/t51_watch_planning_review.md`
- Implementation plan:
  `docs/superpowers/plans/2026-05-23-watch-plan-parity.md`

After evidence:
- Local flow map reviewed `cmd_watch.go`, `watch_find.go`,
  `watch/file_event_handler.go`, `watch/find.go`, `watch/processor.go`, and
  `watch/watcher.go`.
- Three sub-agent reviewers independently agreed that `FindTargetsForFile` is
  shared but below the correct abstraction boundary.

## T52-DEV - Characterize Watch Planning Before Extraction

Status: verified
Commit: b14da74d071f6eb0c58dd9fc3ad19e8199f2ff29

Pain point: the watch parity refactor needs to move planning responsibilities
out of `FileEventHandler` and `watch_find.go`, but the current behavior is only
partly described by focused tests. Moving code without better characterization
would make it easy to change reload, no-runnable, missing-target, or job-order
semantics accidentally.

Change: add behavior-preserving Go tests around the current watch planning
surface. Cover `FindTargetsForFile` as the existing single-path preview helper
and `FileEventHandler.HandleBatch` as the existing live-watch planning plus
execution wrapper.

Acceptance criteria:
- `FindTargetsForFile` tests cover runnable targets, no matching rule, missing
  targets, per-watch ignore, no-target source-file behavior, and multiple jobs.
- `FileEventHandler.HandleBatch` tests cover reload-only mappings and mixed
  runnable/no-rule batches.
- No production behavior changes are required in this phase.
- Focused watch tests, full Go tests, and the full build pass.

After evidence:
- `go test -mod=mod ./watch -run 'Test(FindTargetsForFile_CurrentPlanningCases|FileEventHandler_HandleBatch_(ShouldReload|MixedRunnableAndNoRule))' -count=1`
  passed.
- `go test -mod=mod ./watch` passed.
- `go test -mod=mod ./...` passed.
- `script/check-links` passed.
- `bin/rake` passed with 378 examples, 0 failures, and 4 existing pending
  examples.

## T57-DEV - Lock Live And Find Planning Parity

Status: verified
Commit: 1c11aa0d3fa4bb2d5d2f4c6416f9e35f74287d27

Pain point: after T56, both paths share the same session and planner setup, but
there is no direct regression test proving that a `watch find` preview and a
live watch batch produce the same runnable target list for the same changed
file.

Change: add parity coverage at both boundaries. The Go session test compares
`Session.PlanPath` with the `FileEventHandler` produced by the same session.
The Ruby integration spec compares `plur watch find --format=json` with the
target list logged by a live `plur watch` run after changing the same file.

Acceptance criteria:
- A session-level test proves preview planning and live batch execution use the
  same job name, targets, cwd, reload flag, and no-runnable feedback.
- A CLI integration spec proves `watch find --format=json lib/calculator.rb`
  previews the same target that live watch executes for a `lib/calculator.rb`
  modification.
- Focused parity tests, full Go tests, link check, and the full build pass.

Before evidence:
- No test directly compared `watch find` preview planning with the live watch
  batch produced from the same changed path.

After evidence:
- `go test -mod=mod ./internal/watchsession -run TestSessionPlanPathMatchesLiveHandlerBatch -count=1`
  passed.
- `bin/rake build && PLUR_BINARY=$PWD/plur bin/rspec spec/integration/watch/watch_find_live_parity_spec.rb`
  passed with 1 example and 0 failures.
- `go test -mod=mod ./...` passed.
- `script/check-links` passed.
- `bin/rake` passed with 379 examples, 0 failures, and 4 existing pending
  examples.

## T59-DEV - Show Watch Find Command Plans

Status: verified
Commit: b519569d6017c01f456a2cd4465ee6d57ac51840

Pain point: T57 proves `watch find` and live watch agree on targets, but the
preview still exposes only rules and target lists. For custom jobs, users and
agents cannot see the final argv/env/cwd that live watch will execute.

Change: render `watch.Plan.JobPlans` in `watch find` output. Text mode should
keep the existing target line and add a copyable command line per runnable job.
JSON mode should add a stable `job_plans` array containing job name, targets,
argv, env, cwd, and shell string. Keep no-rule and no-watch-mapping output
unchanged except for an empty `job_plans` array.

Acceptance criteria:
- `plur watch find lib/calculator.rb` shows the final command that would run.
- `plur watch find --format=json lib/calculator.rb` includes `job_plans` with
  command argv, env, cwd, shell, and targets.
- Existing no-rule and no-watch JSON contracts remain structured and exit 2.
- Output contract docs describe the new field.
- Focused watch find specs, docs link check, and the full build pass.

Before evidence:
- Red: `PLUR_BINARY=$PWD/plur bin/rspec spec/integration/watch/watch_find_spec.rb spec/integration/watch/watch_find_json_spec.rb`
  failed because text output had no `[watch] Command:` line and JSON had no
  `job_plans` field.

After evidence:
- `plur -C fixtures/projects/default-ruby watch find lib/calculator.rb` exits 0
  and includes `[watch] Command: bundle exec rspec spec/calculator_spec.rb`.
- `plur -C fixtures/projects/default-ruby watch find --format=json lib/calculator.rb`
  exits 0 and includes `job_plans[0].argv`, `job_plans[0].env`,
  `job_plans[0].cwd`, `job_plans[0].shell`, and targets.
- `PLUR_BINARY=$PWD/plur bin/rspec spec/integration/watch/watch_find_spec.rb spec/integration/watch/watch_find_json_spec.rb spec/docs/output_contracts_doc_spec.rb`
  passed with 12 examples and 0 failures.
- `go test -mod=mod ./...` passed.
- `script/check-links` passed.
- `bin/rake` passed with 379 examples, 0 failures, and 4 existing pending
  examples.

## T60-DEV - Hint On Shared Helper Watch No-Rules

Status: verified
Commit: 779f04b19e093f66cb72285b22f01df25885e420

Pain point: `spec/spec_helper.rb`, `spec/support/*.rb`, and similar shared test
helper files often matter, but the built-in watch rules intentionally do not
guess which tests to run for them. Today the user only sees `No matching rule`,
which is accurate but not actionable.

Change: when a Ruby shared helper-style path under `spec/` or `test/` has no
matching watch rule, add one concise hint: add a `[[watch]]` mapping for shared
files if that change should run tests. Keep generic no-rule files terse to
avoid noisy watch output.

Acceptance criteria:
- `plur watch find spec/spec_helper.rb` still exits 2 and prints the no-rule
  line plus a helper-file hint.
- Live watch prints the same helper-file hint for `spec/spec_helper.rb`.
- Unrelated no-rule files keep the existing terse output.
- Focused watch specs, link check, and the full build pass.

Before evidence:
- `PLUR_BINARY=$PWD/plur bin/rspec spec/integration/watch/watch_find_spec.rb spec/integration/watch/watch_integration_spec.rb`
  failed with the expected missing helper-hint assertions for both `watch find`
  and live watch.

After evidence:
- `./plur -C fixtures/projects/default-ruby watch find spec/spec_helper.rb`
  exits 2 and prints the no-rule line plus the helper-file hint.
- `./plur -C fixtures/projects/default-ruby watch find README.md` exits 2 and
  stays terse with no hint.
- `PLUR_BINARY=$PWD/plur bin/rspec spec/integration/watch/watch_find_spec.rb spec/integration/watch/watch_integration_spec.rb spec/docs/output_contracts_doc_spec.rb`
  passed with 17 examples and 0 failures.
- `go test -mod=mod ./...` passed.
- `script/check-links` passed.
- `bin/rake` passed with 380 examples, 0 failures, and 4 existing pending
  examples.

## T61-DEV - Reject Unknown Config Keys

Status: verified
Commit: a4a75148df253effbce5f663f56a7ce968420c5f

Pain point: `.plur.toml` typos are currently only debug-logged. A misspelled
key like `wokers` or `job.rspec.cmdd` can make Plur ignore the user's intended
configuration while continuing with defaults or inherited job settings.

Change: make TOML config keys strict. If any loaded config contains unknown
leaf keys, fail during configuration loading with a direct error that names the
config file and the unknown keys. Keep valid top-level, job, and watch keys
accepted.

Acceptance criteria:
- Unknown top-level config keys fail before command execution.
- Unknown nested `job.*` and `watch.*` keys fail before command execution.
- The error names the config file and unknown key path.
- Valid config files and TOML 1.1 compatibility cases still pass.
- Focused config specs, Go tests, link check, and the full build pass.

Before evidence:
- `go test -mod=mod ./internal/kongtoml` failed because
  `TestValidateRejectsUnknownKeys` still received nil.
- `PLUR_BINARY=$PWD/plur bin/rspec spec/integration/spec/configuration_spec.rb`
  failed the new unknown-key cases because configs with `wokers`,
  `job.rspec.cmdd`, and `watch.soruce` still executed.

After evidence:
- Unknown config keys now fail with `Configuration error:` plus the config path
  and unknown key paths.
- `plur config init` skips loading the existing project config, so it can report
  or replace an invalid existing `.plur.toml`.
- `fixtures/projects/default-ruby/.plur.toml` now uses a valid comment-only
  config so strict validation still exercises config loading without changing
  built-in detection.
- `PLUR_BINARY=$PWD/plur bin/rspec spec/integration/spec/configuration_spec.rb spec/integration/init/config_init_spec.rb spec/integration/spec/dry_run_plan_spec.rb spec/integration/watch/watch_find_spec.rb spec/integration/watch/watch_find_json_spec.rb spec/integration/watch/watch_spec.rb spec/docs/configuration_target_doc_spec.rb`
  passed with 71 examples and 0 failures.
- `go test -mod=mod ./...` passed.
- `script/check-links` passed.
- `bin/rake` passed with 381 examples, 0 failures, and 4 existing pending
  examples.

## T62-DEV - Reject Persisted Dry-Run Config

Status: verified
Commit: c069a720034c37672afa94540e313a98e35affd2

Pain point: `dry-run = true` and `dry-run-format = "json"` are preview
controls, but TOML currently accepts them as persistent project configuration.
That can make ordinary `plur` invocations silently stop executing tests or force
machine-output mode outside the command where the user intended it.

Change: reject `dry-run` and `dry-run-format` in config files with a
configuration error that says those keys are CLI-only. Keep normal persistent
settings such as `workers`, `color`, `verbose`, and `use` valid.

Acceptance criteria:
- Config files containing `dry-run` or `dry-run-format` fail before command
  execution.
- The error names the config file and CLI-only key paths.
- CLI `--dry-run` and `--dry-run-format=json` behavior remains unchanged.
- Focused config/watch-find specs, Go tests, link check, and full build pass.

Before evidence:
- `go test -mod=mod ./internal/kongtoml` failed because
  `TestValidateRejectsCLIOnlyConfigKeys` still received nil.
- `PLUR_BINARY=$PWD/plur bin/rspec spec/integration/spec/configuration_spec.rb spec/integration/watch/watch_find_spec.rb`
  failed the new persisted dry-run config case because TOML `dry-run` still
  executed.

After evidence:
- Config files with `dry-run` or `dry-run-format` now fail with
  `Configuration error:` and name those keys as CLI-only.
- `script/cli-inventory` now keeps the default `plur` binary as a PATH lookup
  instead of expanding it to a potentially stale checkout binary.
- `PLUR_BINARY=$PWD/plur bin/rspec spec/integration/spec/configuration_spec.rb spec/integration/watch/watch_find_spec.rb spec/docs/configuration_target_doc_spec.rb`
  passed with 48 examples and 0 failures.
- `bin/rspec spec/integration/spec/cli_inventory_spec.rb` and
  `PLUR_BINARY=$PWD/plur bin/rspec spec/integration/spec/cli_inventory_spec.rb`
  both passed with 3 examples and 0 failures.
- `go test -mod=mod ./...` passed.
- `script/check-links` passed.
- `bin/rake` passed with 382 examples, 0 failures, and 4 existing pending
  examples.

## T64-DEV - Use A Durable TOML Config Schema

Status: verified
Commit: 19e1fd5ce49abdcc80783e3acb5d1d1b4b3a6284

Pain point: T61/T62 made TOML strict, but the key allowlist was still derived
from the full Kong CLI tree. That meant adding a transient CLI flag could
silently make it a valid persistent config key, even when it was only meant for
one invocation or one watch session.

Change: validate TOML against a small config schema owned by the config loader:
global persistent keys, `[job.<name>]` fields from the job config type, and
`[[watch]]` fields from the watch mapping type. Keep `dry-run` and
`dry-run-format` recognized only so they can produce the clearer CLI-only
error. Everything else from the CLI surface is rejected as an unknown config
key.

Acceptance criteria:
- Config validation no longer walks the Kong application tree to decide which
  keys are accepted.
- Persistent keys such as `workers`, `color`, `verbose`, `use`, `[job.*]`, and
  `[[watch]]` remain valid.
- Transient/session controls such as `debug`, `auto`, `first-is-1`,
  `rspec-split`, `watch-ignore`, and `watch-run-timeout` fail before command
  execution.
- Configuration docs say TOML accepts only documented persistent settings and
  that operational controls belong on the CLI or in environment variables.
- Focused config specs, Go tests, link check, and full build pass.

Before evidence:
- `allowedConfigKeys` walked Kong command and flag nodes, so flags such as
  `debug`, `rspec-split`, `watch-run-timeout`, and `config-init-force` were
  treated as known TOML keys.
- The T64 unit/integration tests failed before the schema change because those
  CLI/session controls were accepted as config.

After evidence:
- The config schema now comes from persistent global keys plus `job.Job` and
  `watch.WatchMapping` TOML fields, not the Kong application tree.
- `go test -mod=mod ./internal/kongtoml -count=1` passed.
- `PLUR_BINARY=$PWD/tmp/plur-t64 bin/rspec spec/integration/spec/configuration_spec.rb spec/integration/watch/watch_find_spec.rb spec/docs/configuration_target_doc_spec.rb`
  passed with 49 examples and 0 failures.
- `go test -mod=mod ./...` passed.
- `script/check-links` passed.
- `bin/rake` passed with 383 examples, 0 failures, and 4 existing pending
  examples.

## T65-DEV - Show Successful Watch Find JSON Contract

Status: verified
Commit: 710ead53f93870d53bb7ab240cf9aebcfefebfe7

Pain point: `watch find --format=json` now exposes `job_plans`, but the output
contract reference only shows the no-runnable exit 2 JSON shape. Users and
agents still have to infer the successful `exit_code: 0` shape from the field
list or tests.

Change: add a successful runnable `watch find --format=json lib/calculator.rb`
example to `docs/output-contracts.md`, showing matched rule, existing target,
`job_plans`, argv/env/cwd/shell, and empty stderr. Keep this in the output
contract reference instead of duplicating it in watch-mode how-to docs.

Acceptance criteria:
- `docs/output-contracts.md` includes a successful `watch_find` JSON example
  with `exit_code` 0.
- The example includes `job_plans[].argv`, `job_plans[].env`,
  `job_plans[].cwd`, and `job_plans[].shell`.
- The docs spec protects the machine-contract fields without pinning unrelated
  prose.
- Docs check and focused docs spec pass.

Duplication check:
- `docs/output-contracts.md` already owns stable dry-run and watch-find output
  shapes.
- `docs/features/watch-mode.md` and `docs/usage.md` are workflow/how-to docs
  that should link to the output contract rather than duplicating JSON shape
  examples.

Before evidence:
- The watch-find JSON section documented `job_plans` in the field list and
  showed only the no-runnable `exit_code: 2` shape.

After evidence:
- `docs/output-contracts.md` now shows a successful `watch_find` JSON example
  with `exit_code` 0 and a runnable `job_plans` entry.
- `bin/rspec spec/docs/output_contracts_doc_spec.rb` passed with 2 examples
  and 0 failures.
- `script/check-links` passed.
- `bin/rake` passed with 383 examples, 0 failures, and 4 existing pending
  examples.

## T66-DEV - Surface Watch Planning Errors

Status: verified
Commit: cdc241456f86b33dae823ed468fb49cabc2ad47c

Pain point: `watch.Planner` called `FindTargetsForFile` for each changed path
and silently skipped any returned error. That made invalid watch source globs,
ignore globs, or future target-resolution failures indistinguishable from "no
runnable change" at the shared planning layer.

Change: make planning errors first-class. The runtime config validator rejects
invalid watch source and ignore glob patterns before commands run. The shared
planner records per-path planning errors instead of dropping them. Live watch
logs those errors, and `watch find --format=json` can include planning errors
with `exit_code` 1 when a plan cannot be built.

Acceptance criteria:
- Invalid `[[watch]].source` and `[[watch]].ignore` glob patterns fail during
  config validation with the watch name and pattern kind.
- `watch.Planner` records planning errors on the returned plan instead of
  silently continuing.
- `watch find` treats planning errors as exit code 1 and includes them in its
  JSON plan shape.
- Focused Go tests, focused config/docs specs, full Go tests, link check, and
  full build pass.

Before evidence:
- Red: `go test -mod=mod ./watch ./internal/runtime -run 'TestPlanner_PlanPathRecordsPlanningErrors|TestValidateRuntimeConfigRejectsInvalidWatchGlobPatterns' -count=1`
  failed because `watch.Plan` had no `Errors` field and invalid watch glob
  patterns were accepted.

After evidence:
- Invalid user watch `source` and `ignore` glob patterns now fail config
  validation with the watch name and pattern kind.
- `watch.Planner` returns per-path planning errors instead of silently dropping
  them.
- `watch find` exits 1 for planning errors and includes `errors` in the JSON
  plan shape when planning fails.
- `go test -mod=mod ./watch ./internal/runtime -run 'TestPlanner_PlanPathRecordsPlanningErrors|TestValidateRuntimeConfigRejectsInvalidWatchGlobPatterns' -count=1`
  passed.
- `go test -mod=mod . -run 'TestWatchFind(ExitCodeReportsPlanningErrors|BuildWatchFindPlanIncludesPlanningErrors)' -count=1`
  passed.
- `PLUR_BINARY=$PWD/tmp/plur-t66 bin/rspec spec/integration/watch/watch_config_spec.rb spec/integration/spec/configuration_spec.rb spec/docs/output_contracts_doc_spec.rb`
  passed with 51 examples and 0 failures.
- `go test -mod=mod ./...` passed.
- `script/check-links` passed.
- `bin/rake` passed with 384 examples, 0 failures, and 4 existing pending
  examples.

## T68-DEV - Share Watch Find Event Admission

Status: verified
Commit: cb15ed8f592602ecd9c6f1e1321b51f13473cc8d

Pain point: T56 made `watch find` and live watch share session setup and
planning, but `watch find` still jumped directly to `PlanPath`. Live watch
first runs file events through `session.AdmitEvent`, which applies default and
custom ignore patterns before planning. That meant a path ignored by live watch
could still produce a runnable `watch find` preview.

Change: add a session preview-admission helper that synthesizes the same
file-modify admission live watch receives from the watcher. `watch find` uses
that helper before planning, accepts the watch-level `--ignore` flag, and
renders ignored previews as exit 2 with an optional JSON `admission` object.

Acceptance criteria:
- `watch find` uses the same event-admission function as live watch before
  planning.
- `plur watch --ignore=... find --format=json FILE` exits 2 with no runnable
  plan when the preview path is ignored.
- `watch find --help` shows `--ignore` because it now affects previews.
- Output contract docs describe the optional ignored-preview `admission`
  object.
- Focused Go tests, focused Ruby/docs specs, full Go tests, link check, and
  full build pass.

Before evidence:
- Red: `go test -mod=mod ./internal/watchsession -run TestSessionAdmitPathForPreviewUsesLiveAdmission -count=1`
  failed because `Session.AdmitPathForPreview` did not exist.
- Red: `PLUR_BINARY=$PWD/plur bin/rspec spec/integration/watch/watch_find_json_spec.rb spec/integration/watch/watch_find_spec.rb spec/integration/spec/help_spec.rb --example 'ignore'`
  failed because `watch find --ignore` was still rejected and produced no JSON
  preview.

After evidence:
- `Session.AdmitPathForPreview` now synthesizes the same file-modify event
  admission that live watch uses.
- `watch find` applies default/custom watch ignore patterns before planning.
- `plur watch --ignore=lib/** find --format=json lib/calculator.rb` exits 2
  with no runnable plan and includes `admission.reason = "ignored"`.
- `watch find --help` now includes `--ignore`.
- `go test -mod=mod ./internal/watchsession . -run 'TestSessionAdmitPathForPreviewUsesLiveAdmission|TestWatchFind' -count=1`
  passed.
- `PLUR_BINARY=$PWD/tmp/plur-t68 bin/rspec spec/integration/watch/watch_find_json_spec.rb spec/integration/watch/watch_find_spec.rb spec/integration/spec/help_spec.rb spec/docs/output_contracts_doc_spec.rb`
  passed with 22 examples and 0 failures.
- `go test -mod=mod ./...` passed.
- `script/check-links` passed.
- `bin/rake` passed with 385 examples, 0 failures, and 4 existing pending
  examples.

## T69-DEV - Share Watch Execution Plans

Status: verified
Commit: pending

Pain point: T59/T65 made `watch find` print `argv`, `env`, `cwd`, and shell
commands, but that rendering still rebuilt the command shape in `watch_find.go`.
Live watch passed only job/targets/cwd to its executor, so preview command
plans and executed command inputs could drift even though target selection was
shared.

Change: add a `watch.ExecutionPlan` type built from planner job plans and cwd.
The live file-event handler now builds those execution plans before calling the
executor, and `watch find` renders JSON/text command previews from the same
builder.

Acceptance criteria:
- Session parity tests compare preview execution plans with live handler
  `ExecutedPlans`.
- `watch find` command JSON/text output is built from `watch.ExecutionPlan`,
  not a separate command-plan reconstruction.
- Live watch execution uses the same argv/env/cwd fields carried by
  `watch.ExecutionPlan`.
- Focused Go tests, focused watch JSON/parity specs, full Go tests, link
  check, and full build pass.

Before evidence:
- Red: `go test -mod=mod ./internal/watchsession -run TestSessionPlanPathMatchesLiveHandlerBatch -count=1`
  failed because `watch.BuildExecutionPlans`, `watch.ExecutionPlan`, and
  `HandleResult.ExecutedPlans` did not exist.

After evidence:
- `watch.ExecutionPlan` now carries job name, job, targets, argv, normalized
  env, cwd, and is built from planner job plans.
- Live watch handler results now include `ExecutedPlans`, and executors receive
  `ExecutionPlan` directly.
- `watch find` renders text/JSON command previews from `watch.ExecutionPlan`
  instead of rebuilding argv/env in `watch_find.go`.
- `go test -mod=mod ./internal/watchsession ./watch -run 'TestSessionPlanPathMatchesLiveHandlerBatch|TestFileEventHandler|TestExecuteJob' -count=1`
  passed.
- `PLUR_BINARY=$PWD/tmp/plur-t69 bin/rspec spec/integration/watch/watch_find_json_spec.rb spec/integration/watch/watch_find_live_parity_spec.rb spec/docs/output_contracts_doc_spec.rb`
  passed with 8 examples and 0 failures.
- `go test -mod=mod ./...` passed.
- `script/check-links` passed.
- `bin/rake` passed with 385 examples, 0 failures, and 4 existing pending
  examples.

## T56-DEV - Share Watch Session Setup

Status: verified
Commit: 7afbd8c447d1dcc062ba7847e774b47511ac67f5

Pain point: T55 made `watch find` render from `watch.Planner`, but live watch
and watch preview still assembled the session inputs independently. Job
selection, cwd normalization, global ignore defaults, watch directory
filtering, planner construction, and live handler construction still lived in
the outer command paths.

Change: introduce `internal/watchsession` as the command-facing session facade.
Both `plur watch` and `plur watch find` use it to build the selected job,
normalized cwd, watch mappings, filtered watch directories, default/custom
ignore patterns, planner, event admission, and live handler. Output and
persistence remain at the CLI edges.

Acceptance criteria:
- `watchsession.New` selects the watch job, normalizes cwd, resolves watch
  directories, applies default/custom ignore patterns, and builds the shared
  planner.
- Live watch uses the session for watch dirs, event admission, and
  `FileEventHandler` setup.
- `watch find` uses the session for path normalization and planning when watch
  mappings exist while preserving the no-watch mapping output contract.
- Focused session tests, watch package tests, watch find/run integration specs,
  full Go tests, link check, and the full build pass.

Before evidence:
- Red: `go test -mod=mod ./internal/watchsession -run TestNew -count=1`
  failed because `New` and `Options` were undefined.

After evidence:
- `go test -mod=mod ./internal/watchsession -count=1` passed.
- `go test -mod=mod ./watch -count=1` passed.
- `bin/rake build` passed.
- `PLUR_BINARY=$PWD/plur bin/rspec spec/integration/watch/watch_config_spec.rb spec/integration/watch/watch_find_spec.rb spec/integration/watch/watch_find_json_spec.rb spec/integration/watch/watch_run_flags_spec.rb`
  passed with 18 examples and 0 failures.
- `go test -mod=mod ./...` passed.
- `script/check-links` passed.
- `bin/rake` passed with 378 examples, 0 failures, and 4 existing pending
  examples.

## T55-DEV - Render Watch Find From The Planner

Status: verified
Commit: a43369b81f5724109df0a806988bdb9be8cec876

Pain point: T54 extracted `watch.Planner`, but `watch find` still calls
`watch.FindTargetsForFile` directly. That leaves the preview command below the
new shared planning boundary.

Change: make `watch find` build a single-path `watch.Plan` and render text/JSON
from that plan. Keep the existing output contract stable in this phase.

Acceptance criteria:
- `watch find` text output is unchanged for runnable, no-rule, missing-target,
  and no-watch-mapping cases.
- `watch find --format=json` keeps the same stable JSON shape and exit codes.
- `watch_find.go` no longer calls `watch.FindTargetsForFile` directly.
- Focused watch find specs, watch package tests, full Go tests, link check, and
  the full build pass.

Before evidence:
- `watch_find.go` called `watch.FindTargetsForFile` directly, so preview
  rendering bypassed the new `watch.Planner` boundary.

After evidence:
- `rg -n "FindTargetsForFile" watch_find.go` returned no matches.
- `bin/rake build` passed.
- `PLUR_BINARY=$PWD/plur bin/rspec spec/integration/watch/watch_find_spec.rb spec/integration/watch/watch_find_json_spec.rb`
  passed with 10 examples and 0 failures.
- `go test -mod=mod ./watch` passed.
- `go test -mod=mod ./...` passed.
- `script/check-links` passed.
- `bin/rake` passed with 378 examples, 0 failures, and 4 existing pending
  examples.

## T54-DEV - Extract A Pure Watch Planner

Status: verified
Commit: 6fdd40d498464af4ae9adee4ab59a989f43788a9

Pain point: `FileEventHandler.HandleBatch` currently plans and executes live
watch work in one method. That keeps `watch find` from reusing the same batch
planning result without inheriting live side effects, and it makes the next
parity steps harder to reason about.

Change: introduce a side-effect-free `watch.Planner` that turns changed paths
into a `watch.Plan`. The plan owns matched rules, existing/missing targets,
reload intent, no-runnable feedback, and ordered job plans. `FileEventHandler`
then executes the job plans.

Acceptance criteria:
- `Planner.PlanPath` and `Planner.PlanBatch` describe the same planning
  behavior that `FileEventHandler` used before.
- `FileEventHandler.HandleBatch` delegates planning to `Planner` and only keeps
  execution/error handling.
- Existing live watch behavior is preserved by characterization tests.
- Focused planner tests, watch package tests, full Go tests, link check, and
  the full build pass.

Before evidence:
- Red: `go test -mod=mod ./watch -run TestPlanner -count=1` failed because
  `Planner` was undefined.

After evidence:
- `go test -mod=mod ./watch -run TestPlanner -count=1` passed.
- `go test -mod=mod ./watch -run 'Test(FileEventHandler|FindTargetsForFile|Planner)' -count=1`
  passed.
- `go test -mod=mod ./watch` passed.
- `go test -mod=mod ./...` passed.
- `script/check-links` passed.
- `bin/rake` passed with 378 examples, 0 failures, and 4 existing pending
  examples.

## T53-DEV - Extract Live Watch Event Admission

Status: verified
Commit: 3d0505017ff32b03a17a522917ced30ddff0567b

Pain point: live watch decides whether a watcher event is meaningful inside
`cmd_watch.go`: skip watcher lifecycle events, convert absolute paths to
project-relative paths, apply global ignores, and accept only create/modify.
`watch find` cannot share that behavior while it is embedded in the live loop.

Change: extract event admission into a pure `watch.AdmitEvent` helper. Live
watch keeps the same behavior, but the event-to-relative-path decision becomes
testable and reusable by the later watch session/planner work.

Acceptance criteria:
- `watch.AdmitEvent` admits create and modify events as cwd-relative paths.
- It rejects watcher path events, non-create/modify effects, relative-path
  failures, and paths matching global ignores.
- `cmd_watch.go` uses the helper instead of carrying the admission checks
  inline.
- Focused admission tests, watch package tests, full Go tests, link check, and
  the full build pass.

Before evidence:
- Red: `go test -mod=mod ./watch -run TestAdmitEvent -count=1` failed with
  undefined `AdmissionResult` and `AdmitEvent`.

After evidence:
- `go test -mod=mod ./watch -run TestAdmitEvent -count=1` passed.
- `go test -mod=mod ./watch` passed.
- `go test -mod=mod ./...` passed.
- `script/check-links` passed.
- `bin/rake` passed with 378 examples, 0 failures, and 4 existing pending
  examples.
