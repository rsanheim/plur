# Plur v0.60.0 Release Notes Draft

## Summary

Plur v0.60.0 focuses on making daily CLI use more obvious, making preview modes
more useful for humans and scripts, tightening TOML configuration, and improving
watch-mode planning consistency.

This draft includes changes visible between `v0.56.0` and `v0.60.0-rc.1`.
Some runtime-cache/RSpec-split work predates the focused CLI-UX loop, but it is
part of the release train and should be represented in release notes.

## Added

- Experimental RSpec split mode via `--rspec-split` or
  `PLUR_RSPEC_SPLIT=1`.
  This can split historically slow RSpec files into focused `file:line` targets
  after runtime cache data exists.

- Stable dry-run JSON:

  ```bash
  plur --dry-run --dry-run-format=json [patterns...]
  ```

  The JSON plan includes version, mode, selected job, targets, warnings, and
  per-worker `argv`, `env`, and display `shell`.

- Stable watch preview JSON:

  ```bash
  plur watch find --format=json <changed-file>
  ```

  The JSON preview includes matched watch rules, existing/missing targets,
  optional admission information, final command `job_plans`, and exit code.

- Watch command plans in text and JSON preview output. `plur watch find FILE`
  now shows the final command Plur would run for the mapped test target.

- Canonical output contract docs covering human output, JSON output, stdout,
  stderr, and exit codes.

## Changed

- Top-level help now leads with daily workflows:

  ```text
  plur
  plur spec/calculator_spec.rb
  plur test/calculator_test.rb
  plur --dry-run
  plur watch
  plur watch find spec/calculator_spec.rb
  ```

- Watch help now focuses on watch workflows and hides one-shot run flags that do
  not apply to watch mode.

- `plur --dry-run` now explains selected job, framework, and reason before
  showing worker commands.

- Dry-run text now includes a compact plan summary and explicitly says no
  commands will run.

- Dry-run warns when an explicit exclude pattern matches no selected targets.

- Dry-run warns when an explicit target does not match the selected job target
  pattern.

- `plur watch find` is now the supported side-effect-free way to preview what a
  file change would do.

- `plur watch find` and live `plur watch` now share the core path for runtime
  config, job selection, event admission, watch planning, and execution-plan
  construction.

- Runtime tracking stores richer RSpec metadata for balancing and experimental
  split planning.

- `plur doctor` reports runtime cache summary details when available.

## Breaking Changes And Migration Notes

### Unknown TOML keys now fail

Before, typoed TOML keys could be ignored or only debug-logged. Now config
loading fails fast for unknown top-level, job, and watch keys.

Migration: fix or remove unknown keys.

### Dry-run settings are CLI-only

These settings are no longer valid in TOML:

```toml
dry-run = true
dry-run-format = "json"
```

Migration: pass preview controls on the command line:

```bash
plur --dry-run
plur --dry-run --dry-run-format=json
```

### Run-mode job commands cannot include `{{target}}`

Run mode appends discovered targets automatically. A one-shot job command like
this is no longer valid:

```toml
[job.rspec]
cmd = ["bundle", "exec", "rspec", "{{target}}"]
```

Migration:

```toml
[job.rspec]
cmd = ["bundle", "exec", "rspec"]
```

Keep `{{target}}` for watch mappings when you need target placement inside a
watch command.

### Old `--json` flag removed

The old top-level `--json=FILE` flag is no longer the supported machine-output
surface.

Migration:

- Use `plur --dry-run --dry-run-format=json` for one-shot command plans.
- Use `plur watch find --format=json <changed-file>` for watch previews.
- Use normal test output for executed run results.

### `plur --dry-run watch` is rejected

Dry-run is a one-shot run preview. It no longer starts watch setup.

Migration:

```bash
plur watch find <changed-file>
```

### `watch find` no-op previews use exit code 2

When `watch find` successfully previews a changed file but nothing would run, it
exits 2. Treat this as "valid preview, no runnable target", not as a crash.

### Old runtime cache can be regenerated

Older cache formats may be ignored and regenerated. No manual migration is
required.

## Documentation Highlights

- `docs/output-contracts.md` is the canonical reference for machine-readable
  output, stdout/stderr behavior, and exit codes.
- `docs/features/watch-mode.md` now points to `plur watch find` for watch
  previews.
- `docs/configuration.md` now documents strict keys, CLI-only preview controls,
  target appending, and watch target placement.
- `docs/usage.md` now separates human dry-run, JSON dry-run, and watch preview
  workflows.

## Known Caveats

- Structured JSON exists for successful dry-run and watch-preview plans. Parser,
  config, and runtime errors remain prose on stderr with empty stdout.
- RSpec split mode is experimental. Projects with dynamic examples, shared
  examples, or `before(:context)` setup should validate behavior carefully.
- The small assessment benchmark did not prove broad performance improvement.
  It showed effectively unchanged dry-run overhead and a small slowdown on a
  tiny one-spec fixture run.
