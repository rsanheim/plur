# Runner jobs RFC

This RFC defines explicit, testable rules for resolving jobs and applying
framework defaults. It focuses on the interaction between user jobs, built-in
jobs, frameworks, and watch mappings.

## Status
Draft

## Normative language
The key words "MUST", "MUST NOT", "SHOULD", "SHOULD NOT", and "MAY" are to be
interpreted as described in RFC 2119.

## Goals
- Make job resolution predictable when user jobs override built-ins.
- Keep one-off jobs easy to define (no extra framework boilerplate).
- Preserve framework-specific behavior (parser, formatter args, target mode).
- Keep autodetection stable: rspec should win in a repo with specs unless
  explicitly overridden.
- Keep logic deterministic and easy to audit in debug output.

## Non-goals
- Redesign watch mappings or add new config formats.
- Make frameworks user-defined.
- Change runtime grouping or output formatting beyond job selection.
- Preserve backward compatibility: this is explicitly out of scope.

## Data model (simplified structure)
This is a logical, TOML-shaped schema. Field order is not significant.

```
config            := { job_def } { watch_def }

job_def           := "[job." job_name "]" NEWLINE { job_field }
job_field         := cmd | env | framework | target_pattern

cmd               := "cmd" "=" string_list
env               := "env" "=" string_list
framework         := "framework" "=" string
target_pattern    := "target_pattern" "=" string

watch_def         := "[[watch]]" NEWLINE { watch_field }
watch_field       := name | source | targets | jobs | ignore | reload

name              := "name" "=" string
source            := "source" "=" string
targets           := "targets" "=" string_list
jobs              := "jobs" "=" string_list
ignore            := "ignore" "=" string_list
reload            := "reload" "=" bool

string_list       := "[" [ string { "," string } ] "]"
job_name          := string
```

## Entities and fields

### Job (resolved)
Fields and meaning:
- `Name` (string, internal): derived from the job key (e.g., `job.rspec`),
  not set via config. MUST be set on resolved jobs for logging.
- `Cmd` ([]string): executable + args. MUST be non-empty for runnable jobs.
- `Env` ([]string): additional environment variables ("KEY=VALUE"). If set,
  it replaces (does not append to) any built-in Env.
- `Framework` (string): framework identity (rspec/minitest/passthrough/go-test).
- `TargetPattern` (string): glob for file discovery (autodetect and directory
  expansion). Uses doublestar semantics.
  - `TargetPattern` is job-specific and can override framework detection
    when set. It does not change the framework identity on its own.

### Framework (registry)
Fields and meaning (code-defined, not user-configurable):
- `Name` (string): registry key.
- `Parser` (func): returns a framework-specific output parser.
- `DefaultArgs(cfg)` (func): returns framework-provided args appended after
  `Cmd` and before targets.
- `DetectPatterns` ([]string): glob patterns used for framework inference and
  autodetection. Patterns use doublestar semantics (e.g., `**/*_spec.rb`).
  - `DetectPatterns` are framework-level defaults used when a job does not
    specify `TargetPattern`. They are not merged with `TargetPattern`.
- `TargetMode` (enum): how to add target files for run mode.
  - `append`: append file paths to args.
  - `ruby-require`: build a Ruby `-e` require list (minitest).

### WatchMapping
Fields and meaning:
- `Name` (string, optional): label for diagnostics.
- `Source` (string, required): glob for watched file paths.
- `Targets` ([]string, optional): mapping tokens for target generation.
- `Jobs` ([]string, required): list of job names to run.
- `Ignore` ([]string, optional): ignore globs.
- `Reload` (bool, optional): reload plur after jobs complete.

Watch mappings apply only in watch mode. They do not affect run mode
autodetection or job resolution.

## Normalization and resolved jobs map

### Inputs
- Built-in jobs (code-embedded defaults).
- User jobs (from config).

### Resolved job construction
A "resolved jobs map" MUST be constructed before resolution. For each
job name in the union of built-ins and user jobs:

1. Initialize `resolved` as the built-in job (if present) or an empty job.
2. If a user job exists for the same name, overlay its fields:
   - `Cmd` overrides when user-provided and non-empty.
   - `Env` overrides when user-provided and non-empty.
   - `Framework` overrides when user-provided and non-empty.
   - `TargetPattern` overrides when user-provided and non-empty.
3. Set `resolved.Name = job_name`.

Notes:
- With current TOML parsing, "unset" and "empty" are indistinguishable for
  slices/strings. This RFC treats zero values as "not provided".

### Framework defaulting rules
After overlaying:
- If `resolved.Framework` is non-empty, normalize and validate it.
- Else if a built-in job exists for the same name, use that framework.
- Else default to `passthrough`.
- Unknown framework values MUST error during config load.

### Target pattern defaulting
After framework defaulting:
- If `resolved.TargetPattern` is empty and a built-in job exists for the same
  name, use the built-in `TargetPattern`.
- Otherwise leave it empty (valid for non-test jobs).

## Resolution order
All resolution steps MUST use the resolved jobs map.

1) Explicit name
- If `--use` / `use = "..."` is provided, select the job by name.
- If the job name is not in the resolved map, error.

2) Explicit file patterns
- If explicit file paths are provided, infer framework using `DetectPatterns`:
  - Each provided pattern is evaluated as a file path, directory path, or glob.
  - A framework matches a pattern if:
    - the pattern is a directory and **at least one** file under it matches any
      `DetectPatterns` entry, OR
    - the pattern is a file and that file matches any `DetectPatterns` entry, OR
    - the pattern is a glob and it expands to **at least one** file that matches
      any `DetectPatterns` entry.
  - A framework matches overall only if **all** provided patterns match.
  - If exactly one framework matches, select the job named after that framework.
  - If zero frameworks match, continue to autodetect.
  - If multiple frameworks match, return an error instructing the user to
    split the command or pass `--use` to force a framework.
  - Explicit pattern inference selects the canonical job for that framework
    (e.g., `rspec`, `minitest`). Custom job names with the same framework are
    only selected via explicit `--use`.
  - After job selection, explicit patterns MUST expand to at least one test
    file; if expansion yields zero matches, return an error (do not fall back
    to autodetect).
  - Pattern expansion rules after selection:
    - Directories expand by appending the job `TargetPattern` (or framework
      `DetectPatterns` if no job `TargetPattern` is set).
    - Files and globs are expanded as provided.

3) Autodetect by priority
- Check jobs in priority order: `rspec` → `minitest` → `go-test`.
- For each job:
  - If `TargetPattern` is set, use it for file discovery.
  - Else use the framework `DetectPatterns` for file discovery.
  - If any file matches, select that job.
  - Autodetect only considers the canonical job names above; jobs with other
    names are not auto-selected even if they share a framework.

If no job is selected, return a "no default spec/test files found" error.

## File discovery (run mode)
- When no explicit patterns are provided:
  - If the resolved job has `TargetPattern`, discover files using it.
  - Otherwise discover files using the framework `DetectPatterns`.
- If discovery yields zero files, return an error.

## Examples (expected behavior)

### Example A: user overrides rspec cmd only
```
[job.rspec]
cmd = ["bin/rspec"]
```
Expected:
- Framework = rspec (from built-in)
- TargetPattern = spec/**/*_spec.rb (from built-in)
- Autodetect picks rspec in this repo

### Example B: one-off job
```
[job.lint]
cmd = ["bundle", "exec", "rubocop"]
```
Expected:
- Framework = passthrough
- TargetPattern = "" (no autodetect impact)

### Example C: custom name, rspec framework
```
[job.fast]
framework = "rspec"
cmd = ["bin/rspec", "--fail-fast"]
```
Expected:
- Framework = rspec
- TargetPattern empty; discovery uses framework `DetectPatterns` when running
  without explicit patterns.
- Selected only via explicit `--use fast`

## Common flows
- `plur`
  - No `--use` and no explicit patterns → autodetect by priority using
    `TargetPattern` (or `DetectPatterns` if unset).
- `plur spec/models`
  - Explicit directory pattern → framework inferred via `DetectPatterns`
    (`**/*_spec.rb`), select `rspec`, expand under `spec/models`.
- `plur test/models`
  - Explicit directory pattern → framework inferred via `DetectPatterns`
    (`**/*_test.rb`), select `minitest`, expand under `test/models`.
- `plur spec/foo_spec.rb test/bar_test.rb`
  - Mixed frameworks in explicit patterns → error, instruct to split or use
    `--use`.
- `plur --use rspec`
  - Explicit name wins; uses resolved `rspec` job and its `TargetPattern`.
- `plur --use fast`
  - Explicit name wins; uses resolved `fast` job even if its framework is rspec.

## Framework detection patterns (current intent)
- rspec: `**/*_spec.rb`
- minitest: `**/*_test.rb`
- go-test: `**/*_test.go`
- passthrough: none
These patterns SHOULD match the built-in `target_pattern` for canonical jobs
unless intentionally diverging.

## Built-in defaults (documented)
These are the canonical defaults that job resolution should preserve unless
explicitly overridden. Keep this in sync with `plur/autodetect/defaults.toml`.

- `rspec`
  - framework: `rspec`
  - cmd: `["bundle", "exec", "rspec", "{{target}}"]`
  - target_pattern: `spec/**/*_spec.rb`
- `minitest`
  - framework: `minitest`
  - cmd: `["bundle", "exec", "ruby", "-Itest", "{{target}}"]`
  - target_pattern: `test/**/*_test.rb`
- `go-test`
  - framework: `go-test`
  - cmd: `["go", "test", "{{target}}"]`
  - target_pattern: `**/*_test.go`

## Compatibility / impact
- User overrides of built-in jobs regain default `TargetPattern` and
  framework behavior unless explicitly overridden.
- One-off jobs no longer require `framework = "passthrough"`.
- Unknown frameworks still fail fast.

## Success criteria
- Full passing build: `bin/rake` (Ruby + Go).
- Manual verification of key flows:
  - `plur` selects rspec in this repo.
  - `plur spec/models` and `plur test/models` select expected frameworks.
  - `plur --use <custom>` respects resolved job + framework defaults.
- Code simplification: remove confusing/duplicated resolver logic where
  possible (documented in the refactor notes).
- Documentation updated: old docs removed/renamed, new RFC and spec reflect
  current behavior.

## Validation rules (fail early)
Validation MUST occur at config load time and apply consistently to both run
and watch modes.

- Jobs:
  - Each resolved job MUST have a non-empty `Cmd`.
  - `Framework` values MUST be known (rspec/minitest/passthrough/go-test).
  - Validation is applied to the resolved job after merging defaults. A user
    job MAY omit `Cmd` if a built-in job with the same name provides it. If a
    resolved job still has an empty `Cmd` (including `cmd = []`), validation
    MUST fail even if the job is never selected during this run.
- Watch mappings:
  - Every entry in `WatchMapping.Jobs` MUST reference a known job name in the
    resolved jobs map.
  - A watch mapping with an unknown job name MUST fail validation, even if
    watch mode is not invoked.

This ensures a single, consistent validation pass for the entire config.

## Testing
- Add integration coverage for:
  - `job.rspec` with only `cmd` still autodetects rspec.
  - One-off job defaults to passthrough and does not affect autodetect.
  - Explicit framework override on a custom name behaves as expected.

## Logging / diagnostics
- Verbose and debug output SHOULD indicate when a job inherits fields from a
  built-in default (e.g., `cmd`, `framework`, `target_pattern`), so users can
  verify the resolved job shape.

## Design criteria
- Clear separation of concerns: resolution vs discovery vs execution.
- Single source of truth for defaults (`defaults.toml` + framework registry),
  no hidden fallbacks.
- Deterministic, testable resolution flow (no implicit side effects).
- Debug output always shows resolved job shape and inheritance sources.
- No backward-compatibility code paths once the refactor lands.

## Implementation guidance
- Complex logic (especially resolved job construction and autodetection) SHOULD
  remain in a single, simple, long function. Avoid early abstraction so the
  full flow is visible in one place and easier to refactor later.

## References
- docs/architecture/runner-jobs-framework.md
