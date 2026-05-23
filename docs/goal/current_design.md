# Current Plur CLI Design

## T1-INV Inventory

Status: T1 inventory harness added and exercised.

Evidence:
- Harness: `script/cli-inventory`
- Guard spec: `spec/integration/spec/cli_inventory_spec.rb`
- Demo transcript: `tmp/cli_inventory_demo.txt`
- Focused verification: `bin/rspec spec/integration/spec/cli_inventory_spec.rb`

The harness builds an isolated fixture under repo-local `tmp/`, clears global
config influence with isolated `HOME` and `PLUR_HOME`, and supports:

```bash
script/cli-inventory
PLUR_DEMO=1 script/cli-inventory > tmp/cli_inventory_demo.txt
```

`PLUR_DEMO=1` prints every inventory case with `INTENT`, `COMMAND`, `CWD`,
exit status, stdout, and stderr. Summary mode prints a compact status table.

## Inventory Results

| Inventory label | Exercised command | Current result |
| --- | --- | --- |
| `plur` | `plur --dry-run` | Autodetects RSpec and runs `spec/**/*_spec.rb`; default run is commandless even though help says `plur <command>`. |
| `plur spec` | `plur spec --dry-run` | Same selected RSpec files as commandless default. |
| `plur spec foo_spec.rb` | `plur spec --dry-run spec/calculator_spec.rb` | Narrows to one RSpec target and one worker. |
| `plur test` | `plur --dry-run test` | A `test/` directory selects Minitest by explicit pattern inference. |
| `plur test foo_test.rb` | `plur --dry-run test/calculator_test.rb` | A single `_test.rb` file selects Minitest. |
| `plur spec/**/*.rb` | `plur --dry-run 'spec/**/*.rb'` | Quoted recursive globs are expanded by Plur and stay RSpec. |
| `plur --use custom-job` | `plur --dry-run --use custom-job` with a temporary TOML job | Uses the configured job, but dry-run output only says `[rspec]`, not the selected job name. |
| `plur foo/baz/other-file.go` | `plur --dry-run foo/baz/other-file.go` | Existing non-test Go source is accepted as a single RSpec target after RSpec autodetection. |
| `plur foo/baz/other_test.rs` | `plur --dry-run foo/baz/other_test.rs` | Rust-looking test file is also accepted as a single RSpec target. |
| `plur foo_spec.rb bar_spec.rb` | `plur --dry-run foo_spec.rb bar_spec.rb` | Multiple explicit RSpec files are split across workers. |
| `plur -C ~/src/oss/rubocoop spec` | `plur -C <isolated rubocoop fixture> spec --dry-run` | `-C` changes target discovery context before running; shell-expanded home paths work when passed as real paths. |
| `plur spec --exclude-pattern '*user*/_spec.rb'` | exact exclude argument | Command succeeds, but the demo's `spec/models/user_spec.rb` remains because the pattern shape does not match that path. |
| `plur foo/**/*_spec.rb other/**/*_spec.rb` | quoted multi-glob command | Combines recursive globs and splits the matched files across workers. |
| `plur --help` | `plur --help` | Help lists commands and global flags, but presents usage as command-required despite commandless default. |
| `plur help spec` | `plur help spec` | Help-as-command works and shows spec flags. |
| `plur "foo/(1|2|3|)_spec.rb"` | closest supported `foo/{1,2,3,}_spec.rb` | Regex-style grouping is not the model; brace glob works for `1`, `2`, `3`, but the empty alternative did not match `_spec.rb`. |
| `plur watch` | `plur watch --help` and `plur watch find spec/calculator_spec.rb` | Watch help says `plur watch <command>` and lists `watch run`; `watch find` gives a safe preview of a file change. |

## Current Target Selection Model

Plur currently resolves targets in this order:

1. `--use <job>` selects a configured or built-in job.
2. Explicit file, directory, or glob arguments can infer a framework from the path shape.
3. Without explicit framework evidence, Plur autodetects from files on disk, preferring RSpec before Minitest and Go.
4. After a job is selected, explicit existing files are passed through even when they do not look like that framework's normal test files.
5. Directory arguments are expanded using the selected framework's target pattern.
6. Exclude patterns are applied after discovery using doublestar-style path matching.

## UX Gaps Exposed By T1

- Help says `Usage: plur <command> [flags]`, but `plur` without a command is a primary happy path.
- Dry-run output explains the framework (`[rspec]`, `[minitest]`) but not the selected job name or why it was selected.
- Explicit files that do not look like tests can silently become RSpec targets if the project autodetects RSpec.
- Exclude pattern syntax is powerful but easy to misuse; a plausible-looking pattern can leave the intended file in the run.
- Watch has three shapes (`plur watch`, `plur watch run`, `plur watch find`), but help emphasizes subcommands and does not make the default persistent behavior obvious.
- Regex-like user intent is not supported, but the error/translation path is not discoverable from the CLI.

## Desired Design Comparison

The desired CLI should make the selected job, selection reason, target set, and
safe next action obvious. The current behavior is often capable and composable,
but the CLI asks users to infer too much from worker commands. The next phases
should focus on reducing hidden inference, aligning help with the real happy
paths, and improving dry-run feedback before adding new concepts.

## T2-CURR-REVIEW

Status: UX, design, and architecture review completed against the T1
inventory, current docs/code, external CLI references, and three persona
reviews.

Evidence:
- T1 transcript: `tmp/cli_inventory_demo.txt`
- Current help: `./plur --help`, `./plur help spec`, `./plur watch --help`
- Current config docs: `docs/configuration.md`
- CLI entrypoint and parsing: `main.go`, `cmd_spec.go`, `cmd_watch.go`, `watch_find.go`
- Target selection and defaults: `internal/runtime/config.go`, `internal/runtime/defaults.go`, `internal/fileset/fileset.go`
- Config model: `job/job.go`
- Persona reviews from sub-agents:
  - Everyday Ruby/RSpec developer
  - Shell power user / agent workflow user
  - Watch-mode developer

### External Design References

- [watchexec](https://github.com/watchexec/watchexec) is a clear reference for
  the watch mental model: watch paths, run a command on modifications, coalesce
  noisy filesystem events, and respect ignore files.
- [Vitest watch config](https://main.vitest.dev/config/watch) has a crisp
  distinction between interactive watch and one-shot run: watch is default in
  interactive TTYs, while CI and non-interactive shells do not watch unless
  explicitly asked.
- [ripgrep](https://github.com/BurntSushi/ripgrep) is a model for strong
  defaults with explicit escape hatches: it searches recursively, respects
  gitignore, skips hidden/binary files by default, and makes the "turn all
  filtering off" escape hatch discoverable.
- [fd](https://github.com/sharkdp/fd) is a model for humane replacement syntax:
  `fd PATTERN` instead of the more verbose `find -iname '*PATTERN*'`, while
  keeping regex and glob paths available.
- [GitHub CLI](https://cli.github.com/manual/gh) is a model for domain nouns as
  subcommands: `gh pr`, `gh issue`, `gh repo`, `gh run`. That structure helps
  users guess where behavior belongs.

### What Works Well

- The core happy path is good. In a normal RSpec project, `plur --dry-run`
  immediately discovers specs and prints exact worker commands.
- `plur spec path/to/file_spec.rb` behaves as daily RSpec users expect: it
  narrows the run to that file and avoids extra workers.
- Mixed RSpec/Minitest behavior is useful: commandless `plur` prefers RSpec,
  while passing `test` or a `_test.rb` path selects Minitest.
- `-C` is implemented at the right point. `handleChangeDir` in `main.go`
  changes the working directory before config loading, so project-local config
  resolves from the target project.
- The `job.Job` model is small enough to understand: command, env, framework,
  target pattern, and exclude patterns.
- `watch find` is a strong diagnostic affordance. It previews the same mapping
  logic used by watch mode, returns exit code 2 when nothing would run, and is
  safer than launching a persistent watch process during investigation.

### Main Confusing Parts

- Top-level help says `Usage: plur <command> [flags]`, but commandless `plur`
  is a primary workflow through `Spec default:"withargs"` in `main.go`.
- `spec` is both a command name and a conceptual stand-in for "run tests".
  `plur test` works, but not because there is a `test` command. It works
  because `test` is parsed as a pattern and then framework inference selects
  Minitest.
- Dry-run output names the framework, not the selected job or selection reason.
  A custom RSpec job still reads like `[rspec]`, and a custom passthrough job
  can read like `[passthrough]`, hiding the user-facing job name.
- Explicit existing files are accepted after framework selection even when the
  path does not match the selected framework. In an RSpec project,
  `plur foo/baz/other-file.go` becomes an RSpec target because the project
  autodetects RSpec and `fileset.Discover` passes existing files through.
- Exclude patterns have powerful doublestar semantics, but the output gives no
  feedback when an exclude pattern matches zero files. The inventory case
  `--exclude-pattern '*user*/_spec.rb'` looks plausible but does not exclude
  `spec/models/user_spec.rb`.
- Watch help exposes the global `--dry-run` flag, but watch mode does not honor
  dry-run semantics. `plur --dry-run watch run --timeout 1` still starts watcher
  setup instead of previewing a plan.
- Watch mode can silently do nothing for important files. A helper such as
  `spec/spec_helper.rb` sits under a watched directory but does not match the
  built-in `spec/**/*_spec.rb` rule, so no normal output explains why no tests
  ran.

### Current Architecture Review

Plur has two related but different planning flows:

1. One-shot run mode is target-set first. It selects a job, discovers files,
   filters excludes, groups files by historical runtime, then executes worker
   commands.
2. Watch mode is event-mapping first. It selects one job, loads watch mappings,
   watches source directories, turns changed paths into target files, then runs
   one job command for those targets.

That split is sensible, but the CLI does not explain it. Users see one `plur`
tool with `spec`, `watch`, `--use`, `target_pattern`, `{{target}}`, and
`[[watch]]`, but the same words have slightly different meanings across modes.
The config docs call out one important split: in run mode `{{target}}` tokens
are ignored and targets are appended, while watch mode honors `{{target}}`.
That is a surprising difference for a declarative config API.

The code already has useful internal concepts that are not surfaced. For
example, `runtime.SelectedJob.Reason` records whether selection came from an
explicit name, explicit patterns, autodetect, or autodetect after patterns.
Dry-run output could use that directly instead of asking users to infer the
decision from worker commands.

### Persona Findings

Everyday Ruby/RSpec developer:
- Strength: the default RSpec happy path is fast and familiar.
- Confusion: commandless `plur` is real but hidden by help; RSpec/Minitest
  selection is invisible; non-test files can become RSpec targets.
- Smallest useful changes: fix top-level help usage, add one selection
  explanation line in dry-run, warn when excludes match zero files.

Shell power user / agent workflow user:
- Strength: dry-run is the right primitive and `-C` composes well.
- Confusion: output is not stable or structured enough for agents; job and
  framework are separate concepts but output mostly shows framework.
- Smallest useful changes: add a machine-readable plan format, include job and
  reason in dry-run summaries, label or restrict explicit non-matching files.

Watch-mode developer:
- Strength: `watch find` is a strong preview tool and the internal event to
  mapping to target model is clear.
- Confusion: `--dry-run` appears for watch but does not preview watch; no-op
  changes are quiet; help emphasizes `watch run` rather than `plur watch`.
- Smallest useful changes: reject or redirect `--dry-run watch` to
  `watch find`, lead watch help with the happy path, and surface no-rule
  changes in normal output.

### Design Direction

The current CLI is already capable enough to build on. The highest-leverage
direction is not a new abstraction; it is making existing decisions explicit:

- show selected job, framework, and selection reason in dry-run;
- align help with actual happy paths (`plur`, `plur <patterns>`, `plur watch`);
- make "preview what will happen" consistent across run and watch;
- warn on likely no-op user intent, especially zero-match excludes and
  no-rule watch changes;
- add a stable plan output for scripts and agents after the human dry-run
  surface is coherent.

### Recommended First Improvements

1. **Fix help and usage copy.** Show `plur [patterns...] [flags]` or examples
   that make commandless operation first-class. For watch, lead with
   `plur watch` and `plur watch find FILE`.
2. **Explain job selection in dry-run.** One line is enough:
   `Selected job: rspec (reason: autodetect, framework: rspec)`.
3. **Make no-op intent visible.** Warn when excludes match no files and when a
   watched change matches no rule or no existing target.
4. **Clarify watch dry-run.** Either make `--dry-run watch` print a watch plan
   or reject it with a direct pointer to `plur watch find FILE`.
5. **Plan for structured output.** A future `plur plan --json` or
   `plur --dry-run --format=json` should expose config files, selected job,
   reason, targets, excluded files, worker groups, env, and argv.

### Things To Preserve

- Commandless `plur` as the main run command.
- `-C` loading config from the target directory.
- The small `job` config shape.
- Dry-run as a first-class, copyable plan view.
- RSpec-first autodetect when both `spec/` and `test/` exist.
- `watch find` as a safe diagnostic command.
