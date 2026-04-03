# Runtime Config Boundary Design

## Goal

Create one config-processing path for the current invocation:

1. Parse raw CLI + TOML into `PlurCLI`.
2. Build one runtime-ready config snapshot in `AfterApply()`.
3. Validate that snapshot once, in one place.
4. Hand only that snapshot to downstream command code.

This removes the current divergence where different commands rebuild or validate config differently.

## Scope

### In scope

1. Introduce a broad `RuntimeConfig` owned by `AfterApply()`.
2. Build jobs and watches into that runtime config using the same merge/default rules used by runtime watch execution.
3. Fail fast from `AfterApply()` on invalid config for every non-help command, including `spec`, `watch`, `doctor`, and `db:*`.
4. Remove downstream config resolution and validation paths that duplicate or bypass the new runtime-config boundary.
5. Add an audit script that detects forbidden downstream uses of raw config fields and legacy validation/resolution helpers.

### Out of scope

1. Command-specific config objects.
2. Doctor-specific richer diagnostics beyond what naturally falls out of the shared validation error.
3. New watch merge semantics beyond matching current runtime behavior.

## Current Problem

Config processing currently diverges across multiple code paths:

1. `AfterApply()` validates raw `cli.Job` / `cli.WatchMappings`.
2. Watch commands resolve config again later from a different path.
3. Some command code still reads raw `PlurCLI` config fields directly.
4. `doctor` currently re-derives its own view of config instead of consuming a shared runtime state.

This makes it possible for validation, defaults, and actual runtime behavior to drift apart.

## Desired Execution Flow

The runtime config flow should be:

1. `main()` parses CLI and TOML into raw `PlurCLI`.
2. `PlurCLI.AfterApply()` calls the shared runtime-config path, for example `runtimeconfig.Build(cli)` followed by `runtimeconfig.Validate(rc)`.
3. The builder returns one `RuntimeConfig` snapshot containing merged runtime-ready jobs and watches.
4. `AfterApply()` validates that snapshot once through the shared runtime-config validator.
5. If validation passes, `AfterApply()` stores `cli.runtimeConfig`.
6. Subcommands consume `cli.runtimeConfig` only.

There is no downstream re-merge, re-resolution, or re-validation.

## RuntimeConfig Shape

Start with one broad shared object:

```go
type RuntimeConfig struct {
	Use     string
	Jobs    map[string]job.Job
	Watches []watch.WatchMapping
	Sources []string
}
```

Field contract:

1. `Use` carries the invocation's configured explicit job name, if any, so downstream code does not need to read raw `PlurCLI.Use`.
2. `Jobs` are already merged with defaults and normalized.
3. `Watches` are already merged with defaults into the same runtime watch list this invocation will execute against.
4. `Sources` contains useful config-source metadata for user-facing errors.

This object is intentionally broad. Commands can grab the pieces they need without introducing command-specific config layers yet.

## Ownership Boundary

### Allowed to read raw config fields

Only the config builder path inside `AfterApply()` may read:

1. `cli.Use`
2. `cli.Job`
3. `cli.WatchMappings`
4. raw config-file metadata needed to populate `Sources`

### Forbidden below AfterApply

No downstream production code may:

1. read `cli.Job`
2. read `cli.WatchMappings`
3. read `cli.Use`
4. read `parent.Job`
5. read `parent.WatchMappings`
6. read `parent.Use`
7. merge defaults with user config
8. resolve jobs from raw TOML-backed fields
9. validate config independently

### Allowed below AfterApply

Downstream production code may:

1. read `cli.runtimeConfig`
2. select from `runtimeConfig.Jobs`
3. read `runtimeConfig.Watches`
4. read `runtimeConfig.Use`
5. derive command-local execution state from `runtimeConfig`

## Required Downstream Simplifications

The design is not complete unless these simplifications happen:

1. Remove `loadWatchConfiguration()` from watch command flow.
2. Remove `autodetect.ValidateConfig()` from the runtime command path.
3. Remove `watch.ValidateConfig()` from the runtime command path.
4. Remove downstream calls to `autodetect.ResolveJob(...)` that rebuild config from raw `PlurCLI` fields.
5. Remove downstream fallback logic such as "if no user watches, use resolved/default watches."
6. Reduce helper signatures so they no longer accept `*PlurCLI` only to mine config from it.

Design rule:

- If a production helper below `AfterApply()` accepts `*PlurCLI` for config access, that is presumed wrong unless it only needs non-config command wiring.

## Builder Responsibilities

The new builder is responsible for:

1. reading raw parsed config state from `PlurCLI`
2. merging jobs with defaults
3. producing the same runtime watch list this invocation will execute against
4. returning a built `RuntimeConfig` to the shared runtime-config validation path

It is not responsible for command-specific execution behavior.

For example:

1. watch commands may still decide how to turn `RuntimeConfig.Watches` into watch directories
2. spec commands may still decide how to select the active job

But those decisions must operate only on `RuntimeConfig`, never on raw config fields.

## Invocation-Wide Config vs Command-Local Derived State

`RuntimeConfig` should contain invocation-wide config state only. It should not try to hold every piece of derived execution state.

The following should remain command-local derived state:

1. selected job/result for the current command
2. selection reason for the current command
3. inherited-default logging metadata for the selected job
4. target patterns derived from the selected job
5. discovered `testFiles`
6. watch directories derived from `RuntimeConfig.Watches`
7. runner arguments and other execution-local values

That means the `spec` command flow becomes:

1. select the active job from `runtimeConfig.Jobs` using `runtimeConfig.Use` and `SpecCmd.Patterns`
2. log selection details for that chosen job
3. derive target patterns from the selected job
4. derive `testFiles` from the selected job plus `SpecCmd.Patterns`
5. execute the runner

This preserves the current functionality without pushing filesystem-derived or command-specific state into `RuntimeConfig`.

The current `SpecCmd.Run()` mixes:

1. config resolution
2. job selection
3. execution-local file discovery
4. runner execution

This design moves only step 1 into `AfterApply()`. Steps 2 through 4 remain in command code, but must operate only on `RuntimeConfig`.

If preserving inherited-default logging requires extra metadata, that metadata should travel as part of command-local selection results rather than by reintroducing raw-config access downstream.

## Validation Model

Validation happens once, after the runtime config is built.

The validation target is the built runtime config, not the raw parsed TOML shape. This is critical because the runtime config is the thing commands will actually use.

Validation errors should:

1. fail fast from `AfterApply()`
2. identify the config source when possible
3. identify the failing watch or job when possible
4. explain what rule failed
5. provide the expected correction

The exact error formatting can evolve, but the single validation call path from `AfterApply()` must not.

## Command Handoff

After this change, command code should look like:

1. read `parent.globalConfig` for global runtime flags
2. read `parent.runtimeConfig` for config state
3. derive command-local execution inputs from those two objects
4. execute

Command code should not:

1. resolve config
2. merge config
3. validate config
4. fall back from user config to default config

## Watch Reload Compatibility

Watch reload is a full process replacement today. That is compatible with this design and should remain true.

Requirements:

1. reload must re-enter `main()`
2. reload must rerun `AfterApply()`
3. reload must rebuild and revalidate `RuntimeConfig` through the same shared path as initial startup
4. reload must not have a special config-loading or validation path

One specific edge case must be verified during implementation:

1. relative `-C` behavior on reload, since reload preserves original args and startup re-applies early directory handling

## Audit Script Guardrail

Add a script, for example `script/audit-runtime-config-boundary`, that fails if downstream production code uses forbidden raw-config APIs or legacy helpers.

Initial audit pattern:

```bash
rg -n \
  'autodetect\.(ResolveJob|ValidateConfig)|watch\.ValidateConfig|loadWatchConfiguration|(?:cli|parent)\.(Use|Job|WatchMappings)' \
  cmd_*.go watch_find.go cmd_doctor.go rails_init.go
```

Expected result:

- no matches

This is part of the success criteria, not optional cleanup.

## Constraints

To keep the design honest:

1. There must be exactly one production config build+validate entry point: `AfterApply()`.
2. `RuntimeConfig` must remain broad and shared for now; do not introduce command-specific config layers in this change.
3. No downstream command may perform config fallback or default merging.
4. No downstream command may call a helper that reconstructs runtime config from raw TOML-backed fields.
5. If a command needs a smaller input shape later, that is a follow-up refactor after the shared runtime-config boundary is established.

## Success Criteria

The work is complete only when all of the following are true:

1. `AfterApply()` is the only production path that builds and validates runtime config.
2. All runtime command code consumes `cli.runtimeConfig` instead of raw config fields.
3. Watch commands no longer use `loadWatchConfiguration()`.
4. Runtime command code no longer uses `autodetect.ValidateConfig()` or `watch.ValidateConfig()`.
5. Downstream runtime code no longer reads `Use`, `Job`, or `WatchMappings` from `PlurCLI`.
6. The audit script reports zero matches in downstream production files.
7. The resulting command flow is clearly: parse -> build runtime config -> validate -> execute.
8. Watch reload re-enters the same parse -> build runtime config -> validate -> execute path.

## Status: Complete

All success criteria met as of the `stricter-watch-config` branch.

Key implementation details that evolved from the original design:

* `RuntimeConfig` lives in `internal/runtime` package (not root `main` package as originally planned). A `CLIInput` boundary struct decouples it from `PlurCLI`.
* The `autodetect` package was folded into `internal/runtime` since it had a single consumer and was doing the same job.
* The audit script (`script/audit-runtime-config-boundary`) was removed after confirming zero matches — the patterns it guarded against no longer exist in the codebase.
* `Inherited` field tracking for builtin defaults is part of `RuntimeConfig`, not command-local state.

### Follow-on work

* [#34](https://github.com/rsanheim/plur/issues/34) — Jobs should run without requiring `target_pattern`. The broader question of whether `target_pattern` should be a user-facing config concept at all.
* Watch config name-based override (`MergeWatches`) — see `docs/plans/2026-04-01-watch-config-merge-override.md`.
* `watch find` with no detectable framework errors instead of showing empty watches (reorder empty-watches check before job selection in `watch_find.go`).
* Move pattern-based framework inference out of `SelectJobFromRuntimeConfig`. The `patterns` parameter carries command-local state (spec command's file arguments) into the runtime package, where it drives framework inference via `inferFrameworkFromPatterns`. That logic should live in `cmd_spec.go` — the spec command should resolve the framework from its patterns first, then pass a resolved job name to the runtime package. Not breaking anything today, but it's a responsibility leak.
