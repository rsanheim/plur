# Task Runner Jobs Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `plur rails <task>` and `plur rake <task>` commands that run configured Rails/Rake tasks once per worker, without treating task arguments as files.

**Architecture:** Keep `job.Job` as the named command profile. Add a new `task` runner value to the existing runner/framework registry so task jobs append positional arguments literally, fan out across `WorkerCount`, use Plur worker env, and skip file discovery, target grouping, framework args, test parsing, and runtime tracking. This is intentionally shaped toward a later runner refactor where the public `framework` concept can become `runner`, and where `plur <job> <args>` can dispatch any configured job.

**Tech Stack:** Go, Kong CLI, existing `job.Job` config, existing runtime config merge, existing RSpec integration specs, `bin/rake`.

---

## Short-Term Scope

Ship these commands:

```bash
plur rails db:prepare -n 4
plur rails db:migrate VERSION=20260429000000 -n 4
plur rake db:setup -n 4
```

They run:

```text
job.<name>.cmd + literal CLI args
```

once per worker. They do not discover files. They do not glob arguments. They do not append framework formatter args. They do not parse output as tests.

Default built-in jobs:

```toml
[job.rails]
cmd = ["bin/rails"]
framework = "task"
env = ["RAILS_ENV=test"]

[job.rake]
cmd = ["bundle", "exec", "rake"]
framework = "task"
env = ["RAILS_ENV=test"]
```

Users can override the command through the existing config merge:

```toml
[job.rails]
cmd = ["bundle", "exec", "rails"]
```

Because `rails` inherits the built-in `env`, this still gets `RAILS_ENV=test` unless the user explicitly sets `env`.

## Deliberate Non-Scope

- Do not rename the public TOML key from `framework` to `runner` in this change.
- Do not implement dynamic top-level dispatch for every configured job.
- Do not add recipes like `plur rails setup`.
- Do not keep the one-off `db:create`, `db:migrate`, `db:setup`, or `db:test:prepare` commands after the new task commands replace them.

## File Structure

- Modify `framework/framework.go`
  - Add task runner metadata to the existing registry.
  - Add execution kind metadata so test runners and task runners can be distinguished.
- Modify `job/job.go`
  - Add a task command builder that appends literal args and rejects `{{target}}`.
- Create `job/job_test.go`
  - Cover literal task arg appending and `{{target}}` rejection.
- Create `worker_env.go`
  - Extract reusable Plur worker env construction from `Runner.buildEnv`.
- Modify `runner.go`
  - Keep test runner behavior unchanged, but delegate worker env construction to `buildWorkerEnv`.
- Create `parallel_task_runner.go`
  - Build and execute one command per worker for task jobs.
  - Preserve dry-run output, verbose logging, and deduplicated worker errors.
- Create `parallel_task_runner_test.go`
  - Cover command construction, worker env, serial mode, and rejection of target-based jobs.
- Modify `internal/runtime/defaults.toml`
  - Add built-in `rails` and `rake` jobs.
- Modify `internal/runtime/config_test.go`
  - Cover built-in task jobs and config inheritance for overridden task commands.
- Modify `main.go`
  - Add top-level `rails` and `rake` commands.
  - Remove one-off `db:*` command structs and fields.
- Delete `database.go`
  - Its behavior moves into `parallel_task_runner.go`.
- Delete `database_test.go`
  - `parallel_task_runner_test.go` replaces the old `RunDatabaseTask` coverage.
- Modify `spec/integration/spec/database_tasks_spec.rb`
  - Cover the new `rails` and `rake` commands.
- Modify `README.md`, `docs/usage.md`, `docs/configuration.md`, `docs/examples/plur.toml.example`, and `config_init.go`
  - Document task jobs and update Rails database examples.

---

### Task 1: Add Task Runner Metadata And Job Command Builder

**Files:**
- Modify: `framework/framework.go`
- Modify: `job/job.go`
- Create: `job/job_test.go`

- [ ] **Step 1: Write failing tests for literal task command building**

Create `job/job_test.go`:

```go
package job

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildTaskCmdAppendsArgsLiterally(t *testing.T) {
	j := Job{
		Name: "rails",
		Cmd:  []string{"bin/rails"},
	}

	args, err := BuildTaskCmd(j, []string{"db:migrate", "VERSION=20260429000000"})

	require.NoError(t, err)
	assert.Equal(t, []string{"bin/rails", "db:migrate", "VERSION=20260429000000"}, args)
}

func TestBuildTaskCmdRejectsTargetToken(t *testing.T) {
	j := Job{
		Name: "bad-task",
		Cmd:  []string{"bin/rails", "{{target}}"},
	}

	args, err := BuildTaskCmd(j, []string{"db:prepare"})

	require.Error(t, err)
	assert.Nil(t, args)
	assert.Contains(t, err.Error(), `task runner job "bad-task" cannot use {{target}}`)
}
```

- [ ] **Step 2: Run the failing job tests**

Run:

```bash
bin/rake test:go
```

Expected: FAIL because `BuildTaskCmd` is undefined.

- [ ] **Step 3: Implement `BuildTaskCmd`**

Modify `job/job.go`:

```go
package job

import (
	"fmt"
	"strings"
)
```

Add this function after `BuildJobAllCmd`:

```go
// BuildTaskCmd builds a command for task-style jobs.
// Task jobs append CLI arguments literally and never interpret them as files.
func BuildTaskCmd(job Job, args []string) ([]string, error) {
	if job.UsesTargets() {
		return nil, fmt.Errorf("task runner job %q cannot use {{target}} in cmd", job.Name)
	}

	result := make([]string, 0, len(job.Cmd)+len(args))
	result = append(result, job.Cmd...)
	result = append(result, args...)
	return result, nil
}
```

- [ ] **Step 4: Add task execution metadata to framework registry**

Modify `framework/framework.go` by adding execution kind types near `TargetMode`:

```go
type ExecutionKind int

const (
	ExecutionKindTargets ExecutionKind = iota
	ExecutionKindTask
)
```

Add the field to `Spec`:

```go
ExecutionKind ExecutionKind
```

Set the field on existing registry entries:

```go
ExecutionKind: ExecutionKindTargets,
```

Add the new task runner entry:

```go
"task": {
	Name:          "task",
	Parser:        passthrough.NewOutputParser,
	TargetMode:    TargetModeAppend,
	ExecutionKind: ExecutionKindTask,
},
```

- [ ] **Step 5: Run Go tests for the new builder and registry**

Run:

```bash
bin/rake test:go
```

Expected: PASS for `job` tests and no regressions in framework/runtime config tests.

---

### Task 2: Extract Reusable Worker Env Construction

**Files:**
- Create: `worker_env.go`
- Modify: `runner.go`
- Create: `worker_env_test.go`

- [ ] **Step 1: Write worker env tests**

Create `worker_env_test.go`:

```go
package main

import (
	"testing"

	"github.com/rsanheim/plur/config"
	"github.com/stretchr/testify/assert"
)

func TestBuildWorkerEnvIncludesParallelTaskEnv(t *testing.T) {
	cfg := &config.GlobalConfig{
		WorkerCount: 3,
		FirstIs1:    true,
	}

	env := buildWorkerEnv(cfg, 0, 3, []string{"RAILS_ENV=test"})

	assertEnvContains(t, env, "PARALLEL_TEST_GROUPS=3")
	assertEnvContains(t, env, "TEST_ENV_NUMBER=1")
	assertEnvContains(t, env, "RAILS_ENV=test")
}

func TestBuildWorkerEnvOmitsTestEnvNumberInSerialMode(t *testing.T) {
	cfg := &config.GlobalConfig{
		WorkerCount: 1,
		FirstIs1:    true,
	}

	env := buildWorkerEnv(cfg, 0, 1, []string{"RAILS_ENV=test"})

	assertEnvContains(t, env, "PARALLEL_TEST_GROUPS=1")
	assertEnvContains(t, env, "RAILS_ENV=test")
	assertEnvNotContains(t, env, "TEST_ENV_NUMBER=")
}

func TestBuildWorkerEnvAppendsJobEnvAfterPlurEnv(t *testing.T) {
	cfg := &config.GlobalConfig{
		WorkerCount: 2,
		FirstIs1:    true,
	}

	env := buildWorkerEnv(cfg, 1, 2, []string{"CUSTOM=value"})

	assertEnvContains(t, env, "PARALLEL_TEST_GROUPS=2")
	assertEnvContains(t, env, "TEST_ENV_NUMBER=2")
	assertEnvContains(t, env, "CUSTOM=value")
	assert.Greater(t, len(env), 3)
}
```

- [ ] **Step 2: Run the failing env tests**

Run:

```bash
bin/rake test:go
```

Expected: FAIL because `buildWorkerEnv` is undefined.

- [ ] **Step 3: Implement shared worker env helper**

Create `worker_env.go`:

```go
package main

import (
	"fmt"
	"os"

	"github.com/rsanheim/plur/config"
)

func buildWorkerEnv(cfg *config.GlobalConfig, workerIndex, totalWorkers int, extraEnv []string) []string {
	env := os.Environ()
	env = append(env, fmt.Sprintf("%s=%d", EnvParallelTestGroups, totalWorkers))

	if !cfg.IsSerial() {
		testEnvNumber := GetTestEnvNumber(workerIndex, cfg)
		env = append(env, EnvTestEnvNumber+"="+testEnvNumber)
	}

	env = append(env, extraEnv...)
	return env
}
```

- [ ] **Step 4: Update `Runner.buildEnv` to use the helper**

Modify `runner.go`:

```go
func (r *Runner) buildEnv(workerIndex, totalGroups int) []string {
	return buildWorkerEnv(r.config, workerIndex, totalGroups, r.job.Env)
}
```

Remove now-unused imports from `runner.go` if `fmt` or `os` become unused.

- [ ] **Step 5: Run Go tests**

Run:

```bash
bin/rake test:go
```

Expected: PASS for worker env tests and existing runner env tests.

---

### Task 3: Add The Parallel Task Runner

**Files:**
- Create: `parallel_task_runner.go`
- Create: `parallel_task_runner_test.go`
- Delete later in Task 5: `database.go`

- [ ] **Step 1: Write task runner command construction tests**

Create `parallel_task_runner_test.go`:

```go
package main

import (
	"context"
	"testing"

	"github.com/rsanheim/plur/config"
	"github.com/rsanheim/plur/job"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParallelTaskRunnerBuildsOneCommandPerWorker(t *testing.T) {
	cfg := &config.GlobalConfig{
		WorkerCount: 3,
		FirstIs1:    true,
	}
	j := job.Job{
		Name:      "rails",
		Cmd:       []string{"bin/rails"},
		Framework: "task",
		Env:       []string{"RAILS_ENV=test"},
	}
	runner := &ParallelTaskRunner{
		config: cfg,
		job:    j,
		args:   []string{"db:prepare"},
	}

	commands, err := runner.buildCommands(context.Background())

	require.NoError(t, err)
	require.Len(t, commands, 3)
	assert.Equal(t, []string{"bin/rails", "db:prepare"}, commands[0].Args)
	assert.Equal(t, []string{"bin/rails", "db:prepare"}, commands[1].Args)
	assert.Equal(t, []string{"bin/rails", "db:prepare"}, commands[2].Args)
	assertEnvContains(t, commands[0].Env, "PARALLEL_TEST_GROUPS=3")
	assertEnvContains(t, commands[0].Env, "TEST_ENV_NUMBER=1")
	assertEnvContains(t, commands[0].Env, "RAILS_ENV=test")
	assertEnvContains(t, commands[1].Env, "TEST_ENV_NUMBER=2")
	assertEnvContains(t, commands[2].Env, "TEST_ENV_NUMBER=3")
}

func TestParallelTaskRunnerSerialModeBuildsOneCommandWithoutTestEnvNumber(t *testing.T) {
	cfg := &config.GlobalConfig{
		WorkerCount: 1,
		FirstIs1:    true,
	}
	j := job.Job{
		Name:      "rails",
		Cmd:       []string{"bin/rails"},
		Framework: "task",
		Env:       []string{"RAILS_ENV=test"},
	}
	runner := &ParallelTaskRunner{
		config: cfg,
		job:    j,
		args:   []string{"db:prepare"},
	}

	commands, err := runner.buildCommands(context.Background())

	require.NoError(t, err)
	require.Len(t, commands, 1)
	assertEnvContains(t, commands[0].Env, "PARALLEL_TEST_GROUPS=1")
	assertEnvContains(t, commands[0].Env, "RAILS_ENV=test")
	assertEnvNotContains(t, commands[0].Env, "TEST_ENV_NUMBER=")
}

func TestParallelTaskRunnerRejectsTargetTokenJobs(t *testing.T) {
	cfg := &config.GlobalConfig{
		WorkerCount: 2,
		FirstIs1:    true,
	}
	j := job.Job{
		Name:      "bad",
		Cmd:       []string{"bin/rails", "{{target}}"},
		Framework: "task",
	}
	runner := &ParallelTaskRunner{
		config: cfg,
		job:    j,
		args:   []string{"db:prepare"},
	}

	commands, err := runner.buildCommands(context.Background())

	require.Error(t, err)
	assert.Nil(t, commands)
	assert.Contains(t, err.Error(), `task runner job "bad" cannot use {{target}}`)
}
```

- [ ] **Step 2: Run the failing task runner tests**

Run:

```bash
bin/rake test:go
```

Expected: FAIL because `ParallelTaskRunner` is undefined.

- [ ] **Step 3: Implement task runner command building and dry-run**

Create `parallel_task_runner.go` with:

```go
package main

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/rsanheim/plur/config"
	"github.com/rsanheim/plur/job"
	"github.com/rsanheim/plur/logger"
)

type ParallelTaskRunner struct {
	config *config.GlobalConfig
	job    job.Job
	args   []string
}

func (r *ParallelTaskRunner) Run() error {
	commands, err := r.buildCommands(context.Background())
	if err != nil {
		return err
	}

	taskLabel := strings.Join(r.args, " ")
	toStdErr(r.config.DryRun, "Running %s task '%s' in parallel using %d workers\n", r.job.Name, taskLabel, len(commands))

	if r.config.DryRun {
		for i, cmd := range commands {
			printDryRunWorker(r.config.DryRun, i, cmd)
		}
		return nil
	}

	return r.executeCommands(commands)
}

func (r *ParallelTaskRunner) buildCommands(ctx context.Context) ([]*exec.Cmd, error) {
	commands := make([]*exec.Cmd, r.config.WorkerCount)

	for i := 0; i < r.config.WorkerCount; i++ {
		args, err := job.BuildTaskCmd(r.job, r.args)
		if err != nil {
			return nil, err
		}

		cmd := exec.CommandContext(ctx, args[0], args[1:]...)
		cmd.Env = buildWorkerEnv(r.config, i, r.config.WorkerCount, r.job.Env)
		commands[i] = cmd
	}

	return commands, nil
}

func (r *ParallelTaskRunner) executeCommands(commands []*exec.Cmd) error {
	results := make(chan error, len(commands))
	var wg sync.WaitGroup

	for i, cmd := range commands {
		workerIndex := i
		workerCmd := cmd
		wg.Go(func() {
			logger.Logger.Info("running", "cmd", dryRunString(workerCmd), "worker", workerIndex)

			output, err := workerCmd.CombinedOutput()
			if err != nil {
				results <- fmt.Errorf("worker %d failed: %v\nOutput:\n%s", workerIndex, err, string(output))
				return
			}
			results <- nil
		})
	}

	wg.Wait()
	close(results)

	return formatTaskErrors(results, len(commands))
}

func formatTaskErrors(results <-chan error, workerCount int) error {
	var errors []error
	errorOutputs := make(map[string][]int)

	for err := range results {
		if err == nil {
			continue
		}

		errors = append(errors, err)
		errStr := err.Error()
		if idx := strings.Index(errStr, "\nOutput:\n"); idx != -1 {
			output := errStr[idx+9:]
			workerMatch := regexp.MustCompile(`^worker (\d+) failed:`).FindStringSubmatch(errStr)
			if len(workerMatch) > 1 {
				if workerIdx, parseErr := strconv.Atoi(workerMatch[1]); parseErr == nil {
					errorOutputs[output] = append(errorOutputs[output], workerIdx)
				}
			}
		}
	}

	if len(errors) == 0 {
		return nil
	}

	if len(errorOutputs) == 1 && len(errors) == workerCount {
		for output, workers := range errorOutputs {
			return fmt.Errorf("task failed:\nAll %d workers failed with the same error:\n%s", len(workers), output)
		}
	}

	var uniqueErrors []string
	for output, workers := range errorOutputs {
		if len(workers) == 1 {
			uniqueErrors = append(uniqueErrors, fmt.Sprintf("worker %d failed:\n%s", workers[0], output))
			continue
		}

		workerList := make([]string, len(workers))
		for i, w := range workers {
			workerList[i] = fmt.Sprintf("%d", w)
		}
		uniqueErrors = append(uniqueErrors, fmt.Sprintf("workers [%s] failed with:\n%s", strings.Join(workerList, ", "), output))
	}

	return fmt.Errorf("task failed:\n%s", strings.Join(uniqueErrors, "\n---\n"))
}
```

- [ ] **Step 4: Run task runner tests**

Run:

```bash
bin/rake test:go
```

Expected: PASS for task runner tests.

---

### Task 4: Add Built-In Rails And Rake Task Jobs

**Files:**
- Modify: `internal/runtime/defaults.toml`
- Modify: `internal/runtime/config_test.go`

- [ ] **Step 1: Write runtime config tests for built-in task jobs**

Append to `internal/runtime/config_test.go`:

```go
func TestBuildRuntimeConfigIncludesTaskJobs(t *testing.T) {
	rc, err := BuildRuntimeConfig(&CLIInput{})

	require.NoError(t, err)
	require.Contains(t, rc.Jobs, "rails")
	require.Contains(t, rc.Jobs, "rake")

	assert.Equal(t, []string{"bin/rails"}, rc.Jobs["rails"].Cmd)
	assert.Equal(t, "task", rc.Jobs["rails"].Framework)
	assert.Equal(t, []string{"RAILS_ENV=test"}, rc.Jobs["rails"].Env)

	assert.Equal(t, []string{"bundle", "exec", "rake"}, rc.Jobs["rake"].Cmd)
	assert.Equal(t, "task", rc.Jobs["rake"].Framework)
	assert.Equal(t, []string{"RAILS_ENV=test"}, rc.Jobs["rake"].Env)
}

func TestBuildRuntimeConfigTaskJobCommandOverrideInheritsEnv(t *testing.T) {
	rc, err := BuildRuntimeConfig(&CLIInput{
		Jobs: map[string]job.Job{
			"rails": {Cmd: []string{"bundle", "exec", "rails"}},
		},
	})

	require.NoError(t, err)
	require.Contains(t, rc.Jobs, "rails")
	assert.Equal(t, []string{"bundle", "exec", "rails"}, rc.Jobs["rails"].Cmd)
	assert.Equal(t, "task", rc.Jobs["rails"].Framework)
	assert.Equal(t, []string{"RAILS_ENV=test"}, rc.Jobs["rails"].Env)
	assert.True(t, rc.Inherited["rails"].Framework)
	assert.True(t, rc.Inherited["rails"].Env)
}
```

- [ ] **Step 2: Run failing runtime config tests**

Run:

```bash
bin/rake test:go
```

Expected: FAIL because `rails` and `rake` built-in jobs are not defined.

- [ ] **Step 3: Add built-in task jobs**

Modify `internal/runtime/defaults.toml` after the existing test jobs:

```toml
[defaults.job.rails]
framework = "task"
cmd = ["bin/rails"]
env = ["RAILS_ENV=test"]

[defaults.job.rake]
framework = "task"
cmd = ["bundle", "exec", "rake"]
env = ["RAILS_ENV=test"]
```

- [ ] **Step 4: Run runtime tests**

Run:

```bash
bin/rake test:go
```

Expected: PASS for runtime config tests.

---

### Task 5: Add `plur rails` And `plur rake` Commands

**Files:**
- Modify: `main.go`
- Create: `cmd_task.go`
- Delete: `database.go`
- Delete: `database_test.go`

- [ ] **Step 1: Add command dispatch code**

Create `cmd_task.go`:

```go
package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/rsanheim/plur/config"
	"github.com/rsanheim/plur/framework"
	"github.com/rsanheim/plur/internal/runtime"
	"github.com/rsanheim/plur/job"
)

type RailsCmd struct {
	Args []string `arg:"" name:"task" help:"Rails task and arguments to run once per worker"`
}

type RakeCmd struct {
	Args []string `arg:"" name:"task" help:"Rake task and arguments to run once per worker"`
}

func (c *RailsCmd) Run(parent *PlurCLI) error {
	return RunConfiguredTaskJob("rails", c.Args, parent.globalConfig, parent.runtimeConfig)
}

func (c *RakeCmd) Run(parent *PlurCLI) error {
	return RunConfiguredTaskJob("rake", c.Args, parent.globalConfig, parent.runtimeConfig)
}

func RunConfiguredTaskJob(name string, args []string, cfg *config.GlobalConfig, rc *runtime.RuntimeConfig) error {
	if len(args) == 0 {
		return fmt.Errorf("%s requires a task argument", name)
	}

	currentJob, err := lookupTaskJob(name, rc)
	if err != nil {
		return err
	}

	runner := &ParallelTaskRunner{
		config: cfg,
		job:    currentJob,
		args:   args,
	}
	return runner.Run()
}

func lookupTaskJob(name string, rc *runtime.RuntimeConfig) (job.Job, error) {
	currentJob, ok := rc.Jobs[name]
	if !ok {
		available := make([]string, 0, len(rc.Jobs))
		for jobName := range rc.Jobs {
			available = append(available, jobName)
		}
		sort.Strings(available)
		return job.Job{}, fmt.Errorf("job %q not found. Available jobs: %s", name, strings.Join(available, ", "))
	}

	spec, err := framework.Get(currentJob.Framework)
	if err != nil {
		return job.Job{}, err
	}
	if spec.ExecutionKind != framework.ExecutionKindTask {
		return job.Job{}, fmt.Errorf("job %q uses %q runner; %s command requires a task runner", name, currentJob.Framework, name)
	}

	return currentJob, nil
}
```

- [ ] **Step 2: Wire commands into Kong CLI and remove one-off DB commands**

Modify `main.go`:

```go
type PlurCLI struct {
	// Commands
	Spec      SpecCmd      `cmd:"" help:"Run tests" default:"withargs"`
	Rails     RailsCmd     `cmd:"" name:"rails" help:"Run a Rails task once per worker"`
	Rake      RakeCmd      `cmd:"" name:"rake" help:"Run a Rake task once per worker"`
	Watch     WatchCmd     `cmd:"" help:"Watch for file changes and run tests automatically"`
	Doctor    DoctorCmd    `cmd:"" help:"Diagnose Plur installation and environment"`
	Config    ConfigCmd    `cmd:"" help:"Configuration commands"`
	RailsInit RailsInitCmd `cmd:"" name:"rails:init" help:"Configure a Rails project for parallel testing"`
```

Remove these types and fields:

```go
type DBSetupCmd struct{}
type DBCreateCmd struct{}
type DBMigrateCmd struct{}
type DBPrepareCmd struct{}
DBSetup   DBSetupCmd   `cmd:"" name:"db:setup" help:"Setup test databases"`
DBCreate  DBCreateCmd  `cmd:"" name:"db:create" help:"Create test databases"`
DBMigrate DBMigrateCmd `cmd:"" name:"db:migrate" help:"Migrate test databases"`
DBPrepare DBPrepareCmd `cmd:"" name:"db:test:prepare" help:"Prepare test databases"`
```

- [ ] **Step 3: Delete obsolete database task implementation and tests**

Delete `database.go`.

Delete `database_test.go`. `parallel_task_runner_test.go` covers command construction, worker env, serial mode, and invalid target-token jobs. The integration specs cover error deduplication and real Rails/Rake task execution.

- [ ] **Step 4: Run Go tests**

Run:

```bash
bin/rake test:go
```

Expected: PASS for Go tests.

---

### Task 6: Replace Database Task Integration Specs With Task Job Specs

**Files:**
- Modify: `spec/integration/spec/database_tasks_spec.rb`
- Modify: `fixtures/projects/database-tasks/.plur.toml`

- [ ] **Step 1: Add fixture config for the Rake task fixture**

Create `fixtures/projects/database-tasks/.plur.toml`:

```toml
[job.rake]
cmd = ["bundle", "exec", "rake"]
framework = "task"
```

The built-in `RAILS_ENV=test` env is not required for this fixture, but inheriting it is harmless and keeps the fixture close to Rails task behavior.

- [ ] **Step 2: Rewrite dry-run specs for `plur rails`**

In `spec/integration/spec/database_tasks_spec.rb`, replace `db:setup` dry-run examples with:

```ruby
context "rails task command (dry-run)" do
  it "runs a Rails task in dry-run mode for each worker" do
    Dir.chdir(default_rails_dir) do
      output = run_plur("--dry-run", "rails", "db:prepare", "-n", "3").err

      worker0 = dry_run_worker_line(output, 0)
      expect(worker0).to include("PARALLEL_TEST_GROUPS=3")
      expect(worker0).to include("TEST_ENV_NUMBER=1")
      expect(worker0).to include("RAILS_ENV=test")
      expect(worker0).to include("bin/rails db:prepare")

      worker1 = dry_run_worker_line(output, 1)
      expect(worker1).to include("PARALLEL_TEST_GROUPS=3")
      expect(worker1).to include("TEST_ENV_NUMBER=2")
      expect(worker1).to include("RAILS_ENV=test")
      expect(worker1).to include("bin/rails db:prepare")

      worker2 = dry_run_worker_line(output, 2)
      expect(worker2).to include("PARALLEL_TEST_GROUPS=3")
      expect(worker2).to include("TEST_ENV_NUMBER=3")
      expect(worker2).to include("RAILS_ENV=test")
      expect(worker2).to include("bin/rails db:prepare")
    end
  end

  it "uses legacy first worker env when --no-first-is-1 is set" do
    Dir.chdir(default_rails_dir) do
      output = run_plur("--dry-run", "rails", "db:prepare", "-n", "3", "--no-first-is-1").err

      worker0 = dry_run_worker_line(output, 0)
      expect(worker0).to include("PARALLEL_TEST_GROUPS=3")
      expect(worker0).to include("TEST_ENV_NUMBER=")
      expect(worker0).to include("RAILS_ENV=test")
      expect(worker0).to include("bin/rails db:prepare")
    end
  end

  it "does not set TEST_ENV_NUMBER in serial mode" do
    ENV.delete("TEST_ENV_NUMBER")
    Dir.chdir(default_rails_dir) do
      output = run_plur("--dry-run", "rails", "db:prepare", "-n", "1").err

      worker0 = dry_run_worker_line(output, 0)
      expect(worker0).to include("PARALLEL_TEST_GROUPS=1")
      expect(worker0).to include("RAILS_ENV=test")
      expect(worker0).to include("bin/rails db:prepare")
      expect(worker0).not_to include("TEST_ENV_NUMBER")
    end
  end
end
```

- [ ] **Step 3: Add dry-run spec for literal task args**

Add:

```ruby
describe "task arguments" do
  it "passes task args literally instead of treating them as file patterns" do
    Dir.chdir(default_rails_dir) do
      output = run_plur("--dry-run", "rails", "db:migrate", "VERSION=20260429000000", "-n", "2").err

      expect(output).to include("bin/rails db:migrate VERSION=20260429000000")
      expect(output).not_to include("no test files found")
    end
  end
end
```

- [ ] **Step 4: Update real Rails database execution spec**

Replace:

```ruby
result = run_plur("db:create", "-n", "3", allow_error: true)
```

with:

```ruby
result = run_plur("rails", "db:create", "-n", "3", allow_error: true)
```

Replace:

```ruby
result = run_plur("db:migrate", "-n", "3")
```

with:

```ruby
result = run_plur("rails", "db:migrate", "-n", "3")
```

- [ ] **Step 5: Update verbose and error handling specs**

For verbose Rails task output, assert:

```ruby
result = run_plur("--verbose", "rails", "db:migrate", "-n", "2", allow_error: true)

expect(result.err).to include("INFO")
expect(result.err).to include("running")
expect(result.err).to include("RAILS_ENV=test")
expect(result.err).to include("bin/rails db:migrate")
```

For error handling fixture, replace `db:setup` invocations with `rake db:setup`:

```ruby
result = run_plur("rake", "db:setup", "-n", "2", allow_error: true, env: env)
```

Expect the new generic error prefix:

```ruby
expect(result.err).to include("task failed:")
```

Keep the existing assertions for duplicate and different worker error output.

- [ ] **Step 6: Run the task integration specs**

Run:

```bash
bin/rspec spec/integration/spec/database_tasks_spec.rb
```

Expected: PASS after implementation.

---

### Task 7: Update Documentation And Config Templates

**Files:**
- Modify: `README.md`
- Modify: `docs/usage.md`
- Modify: `docs/configuration.md`
- Modify: `docs/examples/plur.toml.example`
- Modify: `config_init.go`
- Modify: `CHANGELOG.md`

- [ ] **Step 1: Update README database examples**

Replace:

```bash
plur db:create -n 3
plur db:migrate -n 3
plur db:setup -n 3
```

with:

```bash
plur rails db:create -n 3
plur rails db:migrate -n 3
plur rails db:setup -n 3
plur rake db:setup -n 3
```

Add a short paragraph:

```markdown
`plur rails <task>` and `plur rake <task>` run the configured command once per worker. Task arguments are passed literally; they are not treated as test file patterns.
```

- [ ] **Step 2: Update usage docs**

In `docs/usage.md`, add a section:

````markdown
### Running Rails And Rake Tasks Per Worker

```bash
plur rails db:prepare -n 4
plur rails db:migrate VERSION=20260429000000 -n 4
plur rake db:setup -n 4
```

These commands run once per worker with `RAILS_ENV=test`, `PARALLEL_TEST_GROUPS`, and `TEST_ENV_NUMBER`. Positional arguments after `rails` or `rake` are passed literally to the configured command.
````

- [ ] **Step 3: Update configuration docs**

In `docs/configuration.md`, add built-in task job examples:

```toml
[job.rails]
cmd = ["bin/rails"]
framework = "task"
env = ["RAILS_ENV=test"]

[job.rake]
cmd = ["bundle", "exec", "rake"]
framework = "task"
env = ["RAILS_ENV=test"]
```

Add:

```markdown
The `task` runner appends CLI arguments literally and runs the command once per worker. It does not discover files or parse test output.
```

Also add the forward-looking note:

```markdown
The configuration field is currently named `framework` because it began as test-framework selection. Conceptually this is now a runner selection field; future versions may rename the public field to `runner`.
```

- [ ] **Step 4: Update Rails config template**

In `config_init.go`, add task jobs to `railsConfigTemplate`:

```toml
[job.rails]
cmd = ["bin/rails"]
framework = "task"
env = ["RAILS_ENV=test"]

[job.rake]
cmd = ["bundle", "exec", "rake"]
framework = "task"
env = ["RAILS_ENV=test"]
```

- [ ] **Step 5: Update changelog**

Add an unreleased entry to `CHANGELOG.md`:

```markdown
* Add task runner jobs for `plur rails <task>` and `plur rake <task>`, replacing one-off parallel database commands.
```

- [ ] **Step 6: Run docs-sensitive tests**

Run:

```bash
bin/rspec spec/plur/changelog_spec.rb spec/integration/init/config_init_spec.rb
```

Expected: PASS.

---

### Task 8: Full Verification

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

- [ ] **Step 4: Run full test suite**

Run:

```bash
bin/rake
```

Expected: PASS.

---

## Follow-Up Runner Refactor Context

This plan intentionally creates a `task` runner value without doing the larger public rename. The shape should make these later changes straightforward:

- Rename the public config field from `framework` to `runner`.
- Rename the `framework` package or expose a new `runner` package.
- Add dynamic first-argument dispatch so configured jobs can be invoked as `plur <job> <args>`.
- Keep `plur spec` as the test-runner command where positional args are target files.
- Potentially add `plur minitest`, `plur test`, or other runner-specific top-level commands.

The short-term design should avoid Rails-specific internals. `rails` and `rake` are built-in task jobs, not special execution paths.

## Self-Review

- Spec coverage: The plan covers Rails and Rake task commands, literal argument passing, worker-count fanout, worker env, config overrides, error handling, docs, and verification.
- Placeholder scan: No implementation step relies on unspecified behavior.
- Type consistency: The plan uses `framework.ExecutionKind`, `framework.ExecutionKindTask`, `job.BuildTaskCmd`, `buildWorkerEnv`, `ParallelTaskRunner`, `RailsCmd`, `RakeCmd`, and `RunConfiguredTaskJob` consistently across tasks.
