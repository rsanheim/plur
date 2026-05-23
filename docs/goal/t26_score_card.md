# T26 Score Card

Source review: T23-T25 after `docs/goal/t22_score_card.md`.

Recent changes:
- T23 rejected irrelevant inherited flags on `plur watch find`.
- T24 grouped top-level commands into daily and advanced/setup sections.
- T25 focused the docs landing page and added a Zensical docs-tooling
  follow-up.

## Local Score

| Criteria | Score | Notes |
| --- | ---: | --- |
| Obviousness | 4 | `plur --help` now leads with commandless usage, common workflows, and daily commands. `plur test` still looks like a command even though it is target inference. |
| Brevity / Surface Area | 4 | Grouped help and the focused docs front door reduce first-contact noise. Global flags still make top-level and watch help feel larger than the daily model. |
| Default Quality | 4 | RSpec-first autodetection, dry-run selection reasons, watch-find previews, and direct bad-flag guidance are strong defaults. Human dry-run still exposes worker command internals too early. |
| Conceptual Coherence | 4 | `watch find` help and parser behavior now agree. Remaining coherence issues are `plur test`, one-shot flags visible in watch help, and three related machine-output surfaces: `--json`, `--dry-run-format=json`, and `watch find --format=json`. |
| Feedback Quality | 4 | Bad `watch find` flags now produce direct guidance. Expected runtime errors can still render as timestamped logs, for example invalid `--use` under `watch find`. |
| Composability | 4 | Dry-run JSON and watch-find JSON are stable enough for scripts when stdout is parsed separately from stderr. `workers[].shell` is a convenience string, not a safely quoted execution contract. |
| Config/API Cleanliness | 4 | Run/watch `{{target}}` boundaries are executable and documented, and irrelevant watch-find flags are rejected. `docs/configuration.md` remains broad. |

## External Reviewer Summary

CLI reviewer:
- Scores: all 4s.
- Direction: trending right; no course correction.
- Remaining problems: `plur test` ambiguity, verbose human dry-run output, and
  remaining watch-help flag noise.

Docs reviewer:
- Scores: all 4s.
- Direction: stop docs-focused DEV phases for this cycle.
- Remaining problems: `docs/configuration.md` is still mixed reference/how-to
  material, `.pages` gives architecture/development visible top-level nav
  weight, and watch docs still link to architecture at the end.

Automation reviewer:
- Scores: all 4s.
- Direction: machine output is good enough for scripts on success/no-op paths.
- Remaining problems: JSON error paths are text-only, expected command errors
  can be timestamped log lines, and `workers[].shell` is not a safely quoted
  script contract.

## Evidence

Representative current behavior:

```text
Usage: plur [patterns...] [flags]
       plur <command> [flags]

Daily commands
  spec [<patterns> ...] [flags]
  watch run [flags]
  watch find <file-path> [flags]
```

`plur watch find --help` now shows only watch-find-relevant flags:

```text
Usage: plur watch find <file-path> [flags]

Flags:
  --format="text"    Output format: text or json
```

An irrelevant inherited flag is now rejected with direct guidance:

```text
plur: error: --dry-run does not apply to plur watch find; use `plur watch find --format=json <file>` for a structured watch preview, or `plur --dry-run [patterns...]` for a one-shot test plan
```

Dry-run JSON keeps machine output on stdout and the version banner on stderr:

```text
STDOUT first line: {
STDERR: plur version=v0.56.1-0.20260523131821-3fa7a9b38fe2+dirty
```

Watch-find JSON writes clean JSON to stdout:

```text
exit=0
STDOUT first line: {
STDERR:
```

Remaining error-shape problem:

```text
exit=1
STDERR:
08:31:48 - ERROR - Command failed error=failed to select watch job: job 'does-not-exist' not found. Available jobs: build, go-test, minitest, plur, rails, rake, rspec
```

T25 verification:

```text
script/check-links
script/docs build
bin/rake
```

`bin/rake` passed with 360 examples, 0 failures, and 4 existing pending
examples.

## Are We Moving In The Right Direction?

Yes. T23-T25 closed the gap between help and parser behavior, made the first
help screen scannable, and stopped the docs landing page from treating
architecture/development material as a primary user path. The changes are
incremental, but they remove real confusion rather than adding new concepts.

The next loop should move back to executable behavior. Docs are no longer the
largest blocker, and the clearest remaining issues show up at the CLI boundary:
ambiguous target-vs-command behavior for `plur test`, log-shaped expected user
errors, and the dry-run `shell` field.

## Top Design Problems

1. `plur test` still reads like a command in help but behaves as a target or
   inference pattern. `plur test --help` displays `plur spec` help, and
   `plur test --dry-run` can fail with `file not found: test`.
2. Some expected user errors render as timestamped log lines instead of plain
   `Error: ...` messages.
3. Human dry-run output is still more implementation-heavy than day-one users
   need.
4. `workers[].shell` in dry-run JSON is convenient but weaker than `argv` and
   `env` as a machine contract.
5. Configuration docs remain broad enough to feel like reference plus
   explanation plus troubleshooting.

## Recommended Next Changes

1. T27-DEV: resolve `plur test` ambiguity. Either make `test` an explicit
   documented alias/command with honest help, or remove it from common
   workflows and improve the missing-target error.
2. T28-DEV: make expected runtime/user errors plain and consistent, especially
   selection/config errors after parsing.
3. T29-DEV: tighten the dry-run machine contract around `workers[].shell`.
   Prefer `argv` and `env` as canonical; either demote `shell` in docs or quote
   it correctly with coverage for spaces/metacharacters.
4. Later: slim `docs/configuration.md` into a compact reference with workflow
   material moved to how-to pages.

## Done-Done Status

Not done. The scorecard is all 4s for the first time in this loop, but the
remaining issues are concrete and still visible to users or scripts. Continue
with executable UX refinements before considering the goal complete.
