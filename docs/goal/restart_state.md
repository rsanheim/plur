# CLI Goal Restart State

This file is a handoff point for restarting the container/session while the
long-running `docs/goal/cli_goal.md` objective is active.

## Current Position

- Branch: `prep-goal`
- Latest completed DEV phase: `T17-DEV`
- Current phase started but not completed: `T18-REFLECT`
- Latest completed phase commit: `c32e7aa watch: add find json output`
- Docs-process commit just before T17: `c1e4f4e docs: add documentation review gate`
- `tracking.md` contains a `T18-REFLECT` start line at git oid `c32e7aa`.

Do not mark the overall goal complete yet. The endpoint remains either T50 or a
reflection where every scorecard category is 4 or 5 with objective evidence.

## Completed Since T12

- `T13-DEV`: structured one-shot dry-run JSON plan.
- `T14-DEV`: watch help clarifies that global dry-run is one-shot only.
- `T15-DEV`: output-contract reference page for streams, exit codes, and stable
  machine formats.
- `T16-DEV`: public configuration docs now say run-mode job commands omit
  `{{target}}`; watch-mode target replacement is separate.
- `T17-DEV`: `plur watch find --format=json` emits a stable JSON watch preview.

## Verification Evidence

Most recent full gate before this handoff:

```bash
bin/rake
```

Result:

- Go, Ruby, and shell lint ran.
- Go tests passed.
- Default Ruby fixture passed: 68 examples, 0 failures.
- Full Ruby suite passed: 355 examples, 0 failures, 4 pending.

Known pending examples remain the same expected ones:

- watch reload timing-sensitive case
- two database integration cases
- grouped minitest backspin comparison

Focused T17 checks also passed:

```bash
go test -mod=mod ./...
bin/rake build
PLUR_BINARY=$PWD/plur bin/rspec spec/integration/watch/watch_find_spec.rb spec/integration/watch/watch_find_json_spec.rb spec/integration/watch/mismatched_dirs_spec.rb
PLUR_BINARY=$PWD/plur bin/rspec spec/docs/output_contracts_doc_spec.rb
git diff --check
```

## Agent State

At the start of T18, spawning a new objective reviewer failed:

```text
agent thread limit reached
```

Closing old sidecar agents also failed:

```text
thread-store internal error: Permission denied (os error 13)
```

After restart, try objective reviewers again for T18. The user specifically
wants mixed personas and high/xhigh sub-agents where possible. If agents are
still unavailable, stop and tell the user rather than silently continuing the
reflection without them.

## User Instructions To Preserve

- Continue `TX-DEV` loops followed by `TX-REFLECT` until T50 or all scorecard
  categories are 4s/5s with objective evidence.
- Prefer 3-6 small DEV loops per context window when phases are small.
- Commit between phases to avoid a backlog of unrelated changes.
- Do not use `FileUtils.rm_rf` for temporary directories in automated scripts,
  even under tmp dirs; use Ruby `Dir.mktmpdir` when uniqueness/cleanup is
  needed.
- Always use `bin/rake`, not bare `rake`.
- Public help/docs should follow Diataxis: tutorial, how-to, explanation, or
  reference. Search for duplicate knowledge before adding docs.
- Do not add specs that only lock markdown prose unless protecting generated CLI
  output, tricky shell output, or a specific contract.

## Next Action After Restart

1. Run `git status --short`.
2. Resume `T18-REFLECT`.
3. Try to spawn 1-3 objective reviewers with mixed personas and high/xhigh
   reasoning.
4. Gather fresh executable evidence from the current binary:

   ```bash
   ./plur --help
   ./plur watch --help
   ./plur -C fixtures/projects/default-ruby --dry-run
   ./plur -C fixtures/projects/default-ruby --dry-run --dry-run-format=json spec/models/user_spec.rb
   ./plur -C fixtures/projects/default-ruby watch find spec/spec_helper.rb
   ./plur -C fixtures/projects/default-ruby watch find --format=json spec/spec_helper.rb
   ```

5. Write `docs/goal/t18_score_card.md`.
6. Track T18 done via `script/track-goal`.
7. Run a suitable verification gate, then commit T18.
8. Continue into the next DEV loop.

Likely next DEV candidates:

- user-facing docs audit, consolidation, and trimming using Diataxis
- config/API cleanup around target template behavior
- further command-surface simplification for watch/run/config concepts
