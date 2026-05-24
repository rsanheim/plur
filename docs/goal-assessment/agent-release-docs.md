# Agent Release Docs Assessment

Reviewer: Faraday
Persona: release manager and docs lead
Model: `gpt-5.5`, reasoning `high`
Mode: read-only assessment

## Scope And Baselines

Faraday compared:

- original release baseline: `v0.56.0`
- current `main`
- RC tag: `v0.60.0-rc.1`

The review focused on release-note-worthy improvements, breaking changes,
migration notes, documentation improvements, remaining documentation risks, and
the goal process from a docs/release perspective.

## Release-Note-Worthy Improvements

### Experimental RSpec splitting and runtime cache

- Added opt-in `--rspec-split` / `PLUR_RSPEC_SPLIT=1`.
- Runtime tracking stores richer per-file and per-example RSpec data.
- Docs cover cache behavior, cold-run fallback, shared-example attribution, and
  pitfalls.
- `doctor` reports runtime cache summary data when available.

Suggested release wording:

> Add experimental RSpec split mode. `plur --rspec-split -n N` can split
> historically slow RSpec files into balanced focused line-target chunks after
> runtime cache data exists. This is opt-in and experimental.

### Clearer help and daily CLI shape

- Top-level help treats commandless `plur` as first-class.
- Watch help leads with `plur watch` and `plur watch find`.
- Irrelevant one-shot worker flags are hidden from watch help.
- `plur test --help` explains that `test` is a target path, not a Plur command.

Suggested release wording:

> Reworked help output around real daily workflows: run tests with `plur`, run
> one target with a path, preview with `plur --dry-run`, and inspect watch
> mappings with `plur watch find`.

### More useful dry-run output

- Human dry-run prints selected job, framework, and reason.
- Human dry-run summarizes the plan and says no commands will run.
- Dry-run warns when explicit excludes match no selected files.
- Dry-run warns when an explicit target does not match the selected job target
  pattern.
- Stable one-shot JSON plan added via `--dry-run-format=json`.

Suggested release wording:

> `plur --dry-run` now explains why a job was selected, summarizes the plan,
> warns about likely no-op filters, and supports a stable JSON plan with
> `--dry-run-format=json`.

### Watch preview is now a real diagnostic surface

- `plur watch find <file>` has human text, JSON output, exit-code semantics,
  ignored-path admission, missing-target reporting, and command plans.
- JSON previews include `job`, `targets`, `argv`, `env`, `cwd`, and `shell`.
- `watch find` and live watch share session/planner/execution-plan paths.
- Watch `--ignore` behavior is validated and reflected in JSON admission output.

Suggested release wording:

> `plur watch find` is now the supported side-effect-free way to preview what a
> file change would do, including stable JSON output and final command plans.

### Stricter configuration API

- TOML config now fails on unknown top-level, job, and watch keys.
- `dry-run` and `dry-run-format` are CLI-only and rejected in TOML.
- Other CLI/session controls are rejected as unknown config keys.
- Run-mode jobs now reject `{{target}}` in `cmd`; run mode appends discovered
  targets automatically.

Suggested release wording:

> Config loading is stricter: typos and CLI-only/session-only settings in TOML
> now fail at startup instead of silently falling back to defaults.

### Output contracts and docs

- Added canonical output contract docs for human output, JSON output, streams,
  and exit codes.
- Public docs front door now links usage, configuration, watch mode, parallel
  execution, and output contracts.
- Usage docs separate human dry-run from JSON plan output and watch preview
  output.
- MkDocs excludes internal goal/planning docs from the published site.

## Breaking Changes And Migration Notes

| Change | User Impact | Migration |
| --- | --- | --- |
| Unknown TOML keys now fail | Existing configs with typos stop at startup. | Fix or remove unknown keys. |
| `dry-run` / `dry-run-format` rejected in TOML | Persisted preview config no longer works. | Use CLI flags: `plur --dry-run` or `plur --dry-run --dry-run-format=json`. |
| Run-mode `cmd` cannot include `{{target}}` | Custom jobs with `{{target}}` in one-shot runs fail. | Remove `{{target}}`; Plur appends targets automatically. |
| `--json` flag removed/rejected | The old JSON-file flag no longer parses. | Use dry-run JSON or watch-find JSON. |
| `plur --dry-run watch` rejected | Dry-run no longer starts watch setup. | Use `plur watch find <changed-file>`. |
| `watch find` no-runnable-target exits 2 | Scripts may need to handle this as a non-error no-op. | Treat exit 2 as "valid preview, nothing would run". |
| Old v1 runtime cache ignored | First run after upgrade may rebalance from file size and regenerate cache. | No manual migration needed. |

## Documentation Improvements To Highlight

- `docs/output-contracts.md` is the biggest release-doc win: it gives script
  authors stable fields and says not to parse human text or `shell`.
- `docs/features/watch-mode.md` now points users to `watch find` instead of
  unsupported `watch --dry-run`.
- `docs/configuration.md` documents strict config keys, CLI-only preview flags,
  target appending, watch target placement, and RSpec-first mixed-framework
  defaults.
- Docs specs protect generated CLI output and specific contracts without
  broadly locking prose.

## Remaining Documentation Risks

1. `CHANGELOG.md` does not yet contain the CLI-UX release contents for
   `v0.60.0`.
2. `README.md` still presents an older quick-start shape and does not mention
   structured dry-run JSON, `watch find`, strict config, or `--rspec-split`.
3. `docs/configuration-test-cases.md` appears stale in places. It is excluded
   from MkDocs, but can mislead contributors.
4. `docs/architecture/runner-jobs-framework.md` has stale dry-run wording.
5. Internal goal docs remain in the public repo under `docs/goal/**`; MkDocs
   excludes them, but it remains hygiene debt.
6. JSON-mode command/config errors are intentionally plain stderr with empty
   stdout; release notes should call this out for automation users.
7. RSpec split is experimental; release notes should repeat the caveats around
   `before(:context)`, dynamic examples, and cold cache fallback.

## Process Assessment

What worked:

- Initial inventory created a concrete baseline.
- T3 made starting problems explicit and measurable.
- Repeated scorecards steered work toward config/API cleanliness and watch
  parity.
- The tracking log is useful as a timeline.
- Phase notes captured red/green evidence.

What was too heavy:

- The process generated a large amount of internal material.
- Commit-per-phase preserved rollback points, but made release-note archaeology
  harder.
- Sub-agent review became operationally fragile.
- Tracking rows used current OID at annotation time, while phase docs carried
  final implementation commits.
- T50/T75 pushed beyond "all categories are strong" into additional parity
  polish.

Recommended process changes:

- Keep baseline inventory, executable examples, scorecards, small DEV loops,
  verification notes, and reflection.
- Keep the Diataxis docs gate.
- Move goal/process docs to `../plur-internal` by default.
- Add a required `Release note / migration impact` field to every DEV phase
  note.
- Require explicit approval to continue once every scorecard category is 4+.
- Store durable evidence under a durable artifact directory, not only `tmp/`.
