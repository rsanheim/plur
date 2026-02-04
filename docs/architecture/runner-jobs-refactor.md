# Runner command + config refactor notes

## Scope
- Focus areas: runner command construction for RSpec/Minitest, framework selection, and config defaults vs overrides.
- Primary files reviewed:
  - plur/runner.go
  - plur/job/job.go
  - plur/autodetect/defaults.go
  - plur/config/config.go
  - spec/integration/shared/configuration_*_spec.rb
  - spec/integration/plur_spec/minitest_integration_spec.rb
  - spec/integration/plur_spec/framework_selection_spec.rb
  - docs/configuration.md
  - docs/architecture/runner-jobs-framework.md
  - docs/architecture/runner-jobs-rfc.md

## Status
This document captures the pre-refactor analysis and goals. Current behavior is
defined by:
- docs/architecture/runner-jobs-rfc.md
- docs/architecture/runner-jobs-framework.md

### Implementation status (current)
**Done**
- Explicit framework identity and registry (rspec/minitest/go-test/passthrough).
- Job resolver that merges built-ins with user jobs.
- Autodetect uses resolved jobs + explicit pattern inference.
- Run mode command building uses framework defaults and target mode.
- Job env applied consistently in run mode and watch mode.
- Config precedence documented and aligned with behavior.
- Config templates/fixtures updated to remove legacy fields.

**Remaining**
- Optional follow-up refactors from Phase 5 (see implementation plan).

## Pre-refactor behavior (historical)
- Runner builds commands via a job-name switch:
  - RSpec: buildRSpecCommand injects JSON formatter and color flags.
  - Minitest: buildMinitestCommand hardcodes a ruby -Itest -e require list for multi-file groups.
  - Default: job.BuildJobCmd.
- Job name conventions are used for target discovery (job.Job.GetTargetPattern) but parser selection and "minitest style" output depend on exact job name equality (job.Job.CreateParser, job.Job.IsMinitestStyle).
- Autodetect uses embedded defaults when no explicit --use or use= is set, even if the user provided a job override with the same name.
- Global config has RSpec-specific formatter logic in config.ConfigPaths.GetJSONRowsFormatterPath.
- Job env is applied in watch mode, but not in runner builds.

## Findings (ordered, historical)
- 1) Mismatch between name-based target discovery and name-based parser selection: **resolved**
  - job names containing "rspec" or "minitest" get the right target pattern, but only exact names "rspec" or "minitest" get the framework parser and output behavior.
  - A config like [job.rspec-fast] is advertised in docs but would use passthrough parsing and skip RSpec formatter injection.
  - Files: plur/job/job.go (GetConventionBasedTargetPattern vs CreateParser/IsMinitestStyle).
- 2) Autodetect ignores user job overrides unless use= or --use is set: **resolved**
  - If users define [job.rspec] but do not set use=rspec, autodetect will still run built-in defaults.
  - Conflicts with docs examples that imply overriding [job.rspec] without explicit use.
  - Files: plur/autodetect/defaults.go.
- 3) Minitest command builder ignores configured cmd for multi-file groups: **resolved**
  - buildMinitestCommand hardcodes bundle exec ruby -Itest and discards j.Cmd whenever len(files) > 1.
  - Any user customization (e.g., bin/rails test, custom flags) is silently ignored in parallel groups.
  - Files: plur/runner.go.
- 4) RSpec argument injection depends on exact file-arg matching and is O(n*m): **resolved**
  - insertBeforeFiles searches for file names by equality, so it can miss cases where targets are embedded in flags (e.g., "--file={{target}}"), or insert flags at the end.
  - This also depends on BuildJobCmd appending targets at the end, which is not always true for custom cmds using the token in-place.
  - Files: plur/runner.go, plur/job/job.go.
- 5) Job env is inconsistent across commands: **resolved**
  - watch mode applies job.Env, runner does not, so env configuration behaves differently between "spec" and "watch".
  - Files: plur/runner.go, plur/watch/watcher.go.
- 6) Config precedence mismatch between code/tests and docs: **resolved**
  - Integration tests assert config file overrides PARALLEL_TEST_PROCESSORS.
  - docs/configuration.md claims env has higher precedence than config.
- 7) Config schema drift: **resolved**
  - Config templates and fixtures now use job + watch mappings; legacy fields removed.

## Refactor goals
- Single path for command building per job framework, not per job name.
- Explicit, testable framework identity separate from job name.
- Merge user job overrides into autodetect results.
- Move framework-specific config (e.g., formatter provisioning) into framework logic, not global config.
- Reduce arg injection fragility and special-casing.
- Concrete spec lives in: docs/architecture/runner-jobs-framework.md

## Proposed refactor design
### 1) Introduce explicit framework identity
- Add job field: framework (or type) with supported values: rspec, minitest, go-test, passthrough.
- Resolve framework as:
  - If job.framework set: use it.
  - If job name matches a built-in default, use that framework.
  - Otherwise default to passthrough for one-off/custom jobs.
- Replace job.Name == "rspec" / "minitest" checks with framework identity.

### 2) Framework registry + resolver
- Create a framework registry (map[string]FrameworkDef) that provides:
  - Parser factory
  - Command builder
  - Output behavior flags (streamStdout, label, minitest-style failures handling)
  - Optional pre-run hooks (e.g., RSpec formatter install)
- Add a job resolver that merges built-in defaults and user jobs, and selects the final job + framework based on:
  - explicit use= or --use
  - explicit patterns
  - autodetect
- When autodetect resolves "rspec" or "minitest", prefer user-defined job with the same name if present.

### 3) Target expansion strategy (reduces Minitest special-casing)
- Add target_mode on job or framework:
  - args: append/expand targets as arguments (current BuildJobCmd behavior)
  - ruby-require: build a ruby -e require list from targets (current Minitest multi-file behavior)
  - list: join targets into a single argument token for tools that want a list
- Default rspec/go-test to args.
- Default minitest to ruby-require but allow user overrides (so custom cmd is honored).

### 4) Centralize arg injection
- Introduce a command builder helper that tracks the "target position" once (or forces all targets to end) so extra flags can be placed deterministically without scanning.
- RSpec-specific injections (formatter + color flags) should be handled by the RSpec framework builder, not runner.

### 5) Align config semantics
- Decide and document precedence (tests currently assert config > env). Update docs/configuration.md accordingly.
- Update config templates in plur/config_init.go to emit job-based schema.
- Remove unused template fields (spec.command, watch.run.command, type) or map them explicitly if you want to keep them (but repo policy prefers clean breaks).

## Suggested implementation plan (phased)
1) Add framework registry and job resolver types (no behavior change yet):
   - New types in plur/framework or plur/job.
   - Convert CreateParser/IsMinitestStyle to use framework identity.
2) Refactor runner/buildCommands and PrintResults to rely on framework registry:
   - Replace job-name switch with framework builder.
   - Ensure RSpec formatter path is handled in RSpec builder.
3) Implement target_mode and migrate Minitest logic:
   - Preserve current ruby-require behavior for default minitest job.
   - Allow config to override with args mode for custom commands.
4) Merge user job overrides into autodetect:
   - Resolve job by name from combined defaults + user jobs.
   - Add integration tests covering [job.rspec] without use=.
5) Normalize config schema and docs:
   - Update config_init templates to job-based schema.
   - Update docs/configuration.md precedence to match behavior.
   - Remove deprecated/unused fields (per repo policy).

## Test updates to add
- Go unit tests:
  - job framework resolution (explicit framework vs name convention).
  - command builder behavior for rspec/minitest with custom job names.
  - minitest target_mode differences.
- Integration tests:
  - [job.rspec] override without use= should affect autodetect.
  - rspec-fast uses rspec parser/formatter injection in dry-run output.
  - job.Env is honored by "spec" execution (or explicitly removed from schema).
