# Rails/Rake Job Commands Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `plur rails <args>` and `plur rake <args>` commands that run configured Rails/Rake jobs once per requested worker.

**Architecture:** Keep Rails and Rake as ordinary `job.Job` entries resolved through the existing runtime config. Reuse the existing `Runner` for parallel command execution by adding a small method that builds one command per worker from `job.Cmd + args`, using the existing worker env and worker execution paths. Add one Kong command struct for `rails` and expose `rake` as its Kong alias; do not add a new runner/framework abstraction for this stepping stone.

**Tech Stack:** Go, Kong CLI, existing `internal/runtime` job config, existing `Runner`, RSpec integration specs, `bin/rake`.

---

## Review Corrections

The earlier plan was over-designed for the first useful version.

- Do not add `env = ["RAILS_ENV=test"]` to built-in `rails` or `rake` jobs. Environment should come from the user's shell or explicit config because silently forcing Rails test mode is surprising.
- Do not add a public `task` framework/runner value. The existing `passthrough` framework already represents commands whose output is not parsed as RSpec or Minitest.
- Do not add a separate `ParallelTaskRunner`. The existing `Runner` already builds worker env, prints dry-run commands, logs worker commands, streams output, and runs workers in parallel.
- Do not add a standalone `lookupTaskJob`. `RuntimeConfig.Jobs` is already the resolved job map, including built-ins and user overrides.
- Keep this as a narrow replacement for the one-off database commands, not a general `plur <job> <args>` dispatcher.

## Short-Term Scope

Ship these commands:

```bash
plur rails db:prepare -n 4
plur rails db:migrate VERSION=20260429000000 -n 4
plur rake db:setup -n 4
```

Each command runs:

```text
job.<name>.cmd + literal CLI args
```

once per requested worker. It does not discover files, glob arguments, add framework formatter arguments, parse test output, or save runtime data.

Default built-in jobs:

```toml
[defaults.job.rails]
framework = "passthrough"
cmd = ["bin/rails"]

[defaults.job.rake]
framework = "passthrough"
cmd = ["bundle", "exec", "rake"]
```

Users can still override either job through the existing config merge:

```toml
[job.rails]
cmd = ["bundle", "exec", "rails"]
```

## Deliberate Non-Scope

- Do not rename the public TOML key from `framework` to `runner`.
- Do not add dynamic top-level dispatch for every configured job.
- Do not add a `task` framework, `TaskRunner`, `ParallelTaskRunner`, or execution-kind registry.
- Do not support arbitrary command names beyond the explicit `rails` and `rake` commands.
- Do not keep the old `db:create`, `db:migrate`, `db:setup`, or `db:test:prepare` commands after the new commands replace them.
- Do not redesign `--` passthrough handling in this change. If task-specific flags need special handling later, add that in a focused follow-up.

## File Structure

- Modify `internal/runtime/defaults.toml`
  - Add built-in `rails` and `rake` jobs as `passthrough` jobs with no default env.
- Modify `internal/runtime/config_test.go`
  - Cover built-in Rails/Rake jobs and user command/env overrides.
- Modify `runner.go`
  - Add a small method on `Runner` that runs `job.Cmd + args` once per requested worker.
  - Reuse `buildEnv`, `printDryRunWorker`, `executeWorkers`, and passthrough parsing.
- Modify `runner_test.go`
  - Cover per-worker command construction, env behavior, dry-run behavior, and failure exit behavior for the new Runner method.
- Create `cmd_rails_rake.go`
  - Add `RailsCmd` with `aliases:"rake"` and a small helper that chooses `rails` or `rake` from Kong's parse context, reads `parent.runtimeConfig.Jobs[name]`, and calls the Runner method.
- Modify `main.go`
  - Wire `rails` and `rake` commands into Kong.
  - Remove old flat database command structs and fields.
- Delete `database.go`
  - Its database-specific behavior is replaced by normal job execution.
- Delete `database_test.go`
  - Runner and integration tests replace this coverage.
- Modify `spec/integration/spec/database_tasks_spec.rb`
  - Replace old `db:*` specs with `rails` and `rake` command specs.
- Modify `README.md`, `docs/usage.md`, `docs/configuration.md`, `docs/examples/plur.toml.example`, and `config_init.go`
  - Document Rails/Rake job commands and explicit env configuration.

---

### Task 1: Add Built-In Rails And Rake Jobs

**Files:**
- Modify: `internal/runtime/defaults.toml`
- Modify: `internal/runtime/config_test.go`

- [ ] **Step 1: Add failing runtime config tests**

Add tests that assert:

- `BuildRuntimeConfig(&CLIInput{})` includes `rails` and `rake`.
- `rails` resolves to `cmd = ["bin/rails"]` and `framework = "passthrough"`.
- `rake` resolves to `cmd = ["bundle", "exec", "rake"]` and `framework = "passthrough"`.
- Neither built-in job sets `Env`.
- A user override can set only `cmd` while inheriting `framework = "passthrough"`.
- A user override can set explicit job env without Plur choosing a Rails environment.

Run:

```bash
bin/rake test:go
```

Expected: FAIL because the built-in jobs are not defined yet.

- [ ] **Step 2: Add the defaults**

Add to `internal/runtime/defaults.toml`:

```toml
[defaults.job.rails]
framework = "passthrough"
cmd = ["bin/rails"]

[defaults.job.rake]
framework = "passthrough"
cmd = ["bundle", "exec", "rake"]
```

- [ ] **Step 3: Verify runtime tests**

Run:

```bash
bin/rake test:go
```

Expected: PASS for runtime config tests.

---

### Task 2: Teach The Existing Runner To Run Job Args Per Worker

**Files:**
- Modify: `runner.go`
- Modify: `runner_test.go`

- [ ] **Step 1: Add failing Runner tests**

Add focused tests for a new Runner method named `RunArgsPerWorker` or similar:

- Given `WorkerCount: 3`, job `cmd = ["bin/rails"]`, and args `["db:prepare"]`, it builds three commands with `bin/rails db:prepare`.
- Worker env uses existing behavior: `PARALLEL_TEST_GROUPS=3`, `TEST_ENV_NUMBER=1`, `2`, `3`.
- Serial mode (`WorkerCount: 1`) omits `TEST_ENV_NUMBER`.
- Job env is appended only when the job explicitly sets `Env`.
- Dry-run prints each worker and returns nil without executing.
- A non-zero worker exit returns an error from the Runner method.

Run:

```bash
bin/rake test:go
```

Expected: FAIL because the new Runner method does not exist.

- [ ] **Step 2: Implement command building on `Runner`**

Add a method to `runner.go` that:

- Requires at least one arg and returns a clear error otherwise.
- Creates exactly `r.config.WorkerCount` commands.
- Builds each command by copying `r.job.Cmd` and appending the literal args.
- Sets env with `r.buildEnv(workerIndex, r.config.WorkerCount)`.
- Prints a short summary through `toStdErr`.
- Uses `printDryRunWorker` for dry-run.
- Uses `r.executeWorkers(commands)` for real execution.
- Returns an error if any worker result is not successful.

Keep this method in `runner.go` because it is another execution mode of the existing runner, not a new runner.

- [ ] **Step 3: Verify Runner tests**

Run:

```bash
bin/rake test:go
```

Expected: PASS for the new Runner tests and existing Runner tests.

---

### Task 3: Add `plur rails` And `plur rake`

**Files:**
- Create: `cmd_rails_rake.go`
- Modify: `main.go`
- Delete: `database.go`
- Delete: `database_test.go`

- [ ] **Step 1: Add command structs**

Create `cmd_rails_rake.go` with:

- `RailsCmd` containing positional `Args []string`.
- `RailsCmd.Run` chooses job name `"rails"` for `plur rails` and `"rake"` for the Kong alias `plur rake`.

The shared helper should:

- Read `j, ok := parent.runtimeConfig.Jobs[name]`.
- Return a clear error if the job is missing.
- Create a Runner with the selected job and no test files.
- Call the new Runner method with the command args.

Do not add a separate job lookup abstraction. Directly reading from the resolved job map is enough for two explicit commands.

- [ ] **Step 2: Wire commands into Kong**

Modify `PlurCLI` in `main.go`:

```go
Rails RailsCmd `cmd:"" name:"rails" aliases:"rake" help:"Run a Rails or Rake command once per worker"`
```

Remove:

- `DBSetupCmd`
- `DBCreateCmd`
- `DBMigrateCmd`
- `DBPrepareCmd`
- `DBSetup`, `DBCreate`, `DBMigrate`, and `DBPrepare` fields on `PlurCLI`

- [ ] **Step 3: Delete old database implementation**

Delete:

- `database.go`
- `database_test.go`

This project does not keep old wrappers for compatibility unless explicitly requested.

- [ ] **Step 4: Verify Go tests**

Run:

```bash
bin/rake test:go
```

Expected: PASS.

---

### Task 4: Replace Database Integration Specs

**Files:**
- Modify: `spec/integration/spec/database_tasks_spec.rb`
- Modify or create: `fixtures/projects/database-tasks/.plur.toml`

- [ ] **Step 1: Update dry-run specs**

Replace old `db:setup` dry-run examples with `rails db:prepare` examples:

```ruby
output = run_plur("--dry-run", "rails", "db:prepare", "-n", "3").err
```

Assert:

- three workers are printed
- each worker includes `PARALLEL_TEST_GROUPS=3`
- each worker has the expected `TEST_ENV_NUMBER`
- each worker includes `bin/rails db:prepare`
- no worker includes `RAILS_ENV=test` unless the spec explicitly sets that env; clear `ENV["RAILS_ENV"]` around these examples if the outer test process has it set

Keep the serial-mode and `--no-first-is-1` assertions, updated to `rails db:prepare`.

- [ ] **Step 2: Add literal arg coverage**

Add a dry-run spec:

```ruby
output = run_plur("--dry-run", "rails", "db:migrate", "VERSION=20260429000000", "-n", "2").err
```

Assert the output includes:

```text
bin/rails db:migrate VERSION=20260429000000
```

and does not include `no test files found`.

- [ ] **Step 3: Update real Rails database execution**

Replace:

```ruby
run_plur("db:create", "-n", "3", allow_error: true)
run_plur("db:migrate", "-n", "3")
```

with:

```ruby
run_plur("rails", "db:create", "-n", "3", allow_error: true, env: {"RAILS_ENV" => "test"})
run_plur("rails", "db:migrate", "-n", "3", env: {"RAILS_ENV" => "test"})
```

The explicit env keeps the test behavior without baking Rails test mode into the built-in job.

- [ ] **Step 4: Update Rake error fixture specs**

Configure the fixture if needed:

```toml
[job.rake]
cmd = ["bundle", "exec", "rake"]
```

Replace `db:setup` invocations with:

```ruby
run_plur("rake", "db:setup", "-n", "2", allow_error: true, env: env)
```

Assert non-zero status and useful worker output. Do not preserve database-specific error wording unless the new Runner method intentionally emits it.

- [ ] **Step 5: Verify focused integration specs**

Run:

```bash
bin/rspec spec/integration/spec/database_tasks_spec.rb
```

Expected: PASS.

---

### Task 5: Update Documentation And Templates

**Files:**
- Modify: `README.md`
- Modify: `docs/usage.md`
- Modify: `docs/configuration.md`
- Modify: `docs/examples/plur.toml.example`
- Modify: `config_init.go`
- Modify: `CHANGELOG.md`

- [ ] **Step 1: Replace old database commands**

Replace examples like:

```bash
plur db:create -n 3
plur db:migrate -n 3
plur db:setup -n 3
```

with:

```bash
plur rails db:create -n 3
plur rails db:migrate -n 3
plur rails db:prepare -n 3
plur rake db:setup -n 3
```

- [ ] **Step 2: Document built-in jobs**

Show:

```toml
[job.rails]
cmd = ["bin/rails"]
framework = "passthrough"

[job.rake]
cmd = ["bundle", "exec", "rake"]
framework = "passthrough"
```

Then state plainly that users choose any Rails environment through their shell or explicit job env.

- [ ] **Step 3: Explain behavior plainly**

Document:

- `plur rails <args>` and `plur rake <args>` run once per worker.
- Args are appended literally to the configured job command.
- The commands set Plur's worker env (`PARALLEL_TEST_GROUPS`, `TEST_ENV_NUMBER`) through existing Runner behavior.
- They inherit the shell environment and any explicit job env.
- They do not discover files or parse test output.

- [ ] **Step 4: Verify docs-sensitive tests**

Run:

```bash
bin/rspec spec/plur/changelog_spec.rb spec/integration/init/config_init_spec.rb
```

Expected: PASS.

---

### Task 6: Full Verification

**Files:**
- No new edits expected.

- [ ] **Step 1: Run focused Go tests**

Run:

```bash
bin/rake test:go
```

Expected: PASS.

- [ ] **Step 2: Run focused integration specs**

Run:

```bash
bin/rspec spec/integration/spec/database_tasks_spec.rb
```

Expected: PASS.

- [ ] **Step 3: Run Rails fixture check**

Run:

```bash
bin/rake test:default_rails
```

Expected: PASS.

- [ ] **Step 4: Run full build**

Run:

```bash
bin/rake
```

Expected: PASS.

---

## Follow-Up Context

This leaves the larger runner naming problem alone. A future change can rename `framework` to `runner`, split the package, or add dynamic configured-job dispatch if there is a concrete product need. This change should only prove that named Rails/Rake jobs can run across Plur workers without pretending task execution is a new subsystem.

## Self-Review

- Spec coverage: Covers Rails/Rake commands, literal argument passing, worker fanout, explicit env behavior, config overrides, removal of old database commands, docs, and verification.
- Placeholder scan: No task depends on `TBD`, future planner work, or unspecified abstractions.
- Simplicity check: Uses existing `RuntimeConfig.Jobs`, existing `Runner`, existing `passthrough` parser, existing worker env, and existing dry-run/logging paths. Removed the proposed `task` framework, `ParallelTaskRunner`, execution-kind metadata, `BuildTaskCmd`, and `lookupTaskJob`.
