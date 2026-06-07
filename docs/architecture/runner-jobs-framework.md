# Runner framework command spec (run vs watch)

This document defines how framework-aware commands are built for run mode
(plur spec) versus watch mode (plur watch). It is the concrete implementation
spec derived from the runner-jobs RFC:

## Goals
- Keep run mode free of command template placeholders.
- Keep watch mode flexible for guard-like mappings.
- Centralize framework-specific behavior (parser + default args) in one place.
- Preserve minitest multi-file execution via ruby -e require list.
- Keep selected-job framework resolution explicit and easy to audit.

## Definitions
- Run mode: `plur spec ...` and default `plur`.
- Watch mode: `plur watch ...` and watch mappings.
- Framework: rspec, minitest, passthrough, go-test (others as needed).
- Resolved job: a `framework.Job` whose `FrameworkName` has been normalized and
  whose internal `Framework` field has been populated from the registry.

## Job schema changes
- Add `framework` to job definitions (string, optional for user jobs).
  - Examples: `framework = "rspec"`, `framework = "minitest"`, `framework = "passthrough"`.
- Job commands are static executable-plus-fixed-args arrays.
- Run mode and watch mode append resolved targets automatically.
- Watch target mappings, not job commands, own path template expansion.

## Framework configurability
- Frameworks are **not user-definable** via config. The registry is code-defined.
- Allowed values: `rspec`, `minitest`, `passthrough`, `go-test`.
- For user-defined jobs, `framework` is optional:
  - If the job name matches a built-in default (e.g., `rspec`, `minitest`, `go-test`), it defaults to that framework.
  - Otherwise it defaults to `passthrough`.
- For built-in defaults (embedded), `framework` is implicit.
- A user can still override the framework explicitly (e.g., `framework = "minitest"` on a job named `rspec`).

## Job args vs framework args
- No separate `args` field is added to Job.
- Users add custom flags directly in `job.cmd`.
- Framework `DefaultArgs(cfg)` are appended after `job.cmd` and before targets.
- If users want full control over args without framework defaults, use `framework = "passthrough"`.

## Framework registry (minimal)
A framework registry provides:
- Parser factory (required).
- DefaultArgs(cfg) (optional).
- DetectPatterns (optional).
- TargetMode (required):
  - append: append files to args
  - ruby-require: build a ruby -e require list from files (minitest)

## Run mode command building
Given: resolved job, files, config.

1) Base args = job.Cmd.
2) Append job.Framework.DefaultArgs(cfg) if any.
3) Append targets according to job.Framework.TargetMode:
   - append: args = append(args, files...)
   - ruby-require: args = buildMinitestRequireList(args, files)

Notes:
- Run mode does not attempt to place args before file tokens; files always trail.
- If a job omits `cmd`, it is invalid (no implicit defaults beyond built-ins).
- In run mode, job.Env is applied (align with watch mode).
- The selected run job resolves its framework once during `runtime.SelectedJob`
  construction. Discovery, command building, parsing, and summaries use the
  resolved `framework.Job`.

## Minitest target mode (ruby-require)
- When files > 1, build:
  - base cmd (from job.Cmd or defaults)
  - "-e", "[list].each { |f| require f }"
- For compatibility, the list should match the existing logic:
  - strip `test/` prefix
  - strip `.rb` suffix
  - do not resolve to absolute paths

## RSpec default args
- DefaultArgs adds formatter + color flags:
  - `-r <formatter>` `--format Plur::JsonRowsFormatter`
  - `--force-color` or `--no-color` based on config
- These are appended before target files.

## Watch mode command building
- Append resolved targets at the end of the command.
- Watch remains responsible for guard-like path substitutions.

## Default jobs vs user jobs
- Built-in defaults are merged with user jobs of the same name (field-by-field overrides).
- User jobs override only the fields they set; missing fields inherit from built-in defaults.
- Auto-detection chooses a canonical job name, then resolves to the merged job.
- Explicit `--use` / `use = "..."` always wins over autodetect.

## Diagnostic output (verbose + dry-run)
- Verbose logs should include:
  - resolved job name
  - resolved framework
  - target_pattern (or framework detect patterns)
  - final command args per worker
- Dry-run output should make the framework visible and easy to verify:
  - a one-line summary includes framework (e.g., "Running 12 specs [rspec] in parallel…")
  - per-worker lines show the exact command (with framework default args applied)

## Validation rules
- Reject template tokens in job commands.
- Validate watch target templates during config load.
- Missing `framework` defaults as described above; unknown values error during config load.

## Cross references
- docs/architecture/runner-jobs-rfc.md
- spec/integration/spec/framework_output_spec.rb (guardrail test for framework output + defaults)

## Guardrail behavior
The integration spec asserts:
- Dry-run summary includes framework tag: `[dry-run] Running 1 spec [rspec] in parallel using 1 workers`
- Debug logs include resolved framework: `framework="rspec"`
- Worker command includes rspec default args (formatter + color) and the target file:
  `bundle exec rspec --fail-fast -r <plur_home>/formatter/json_rows_formatter.rb --format Plur::JsonRowsFormatter --no-color spec/example_spec.rb`

## Implementation status (current)
- Framework registry is implemented in `internal/framework` with TargetMode,
  DefaultArgs, Parser, and DetectPatterns.
- `runtime.SelectedJob.Job` resolves the selected framework once before run-mode
  consumers receive it.
- `framework.Job` owns target pattern lookup and run-argument construction.
- Run mode starts from `job.cmd`, appends framework defaults, then adds targets.
- Minitest uses ruby `-e` require list for multi-file runs (single file appends directly).
- Jobs default framework by name (built-ins) or `passthrough` for custom jobs when omitted.
- Dry-run summary includes `[framework]`; verbose logs include `framework="..."`.
- Job.Env now applies in run mode (aligns with watch mode).
- `plur init` output uses static job command examples.

## Test notes
- `bin/rspec spec/integration/spec/framework_output_spec.rb`
- `bin/rake test:go`
