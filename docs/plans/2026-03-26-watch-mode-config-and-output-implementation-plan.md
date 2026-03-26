# Watch Mode Config And Output Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `plur watch` merge built-in and user watch mappings, then fix prompt/output behavior so watch mode renders clean shell-style commands without duplicate failure warnings.

**Architecture:** Keep watch-resolution logic additive by appending user `[[watch]]` entries to the built-in mappings already resolved for the selected job, and validate config against builtin+user mappings together. For output, funnel watch prompt rendering, echoed commands, and watch-level logs through one synchronized stderr writer so prompt redraws and debug lines cannot trample each other, while job failures are reported exactly once.

**Tech Stack:** Go, Kong CLI config loading, `slog` with the custom logger handler, testify, RSpec integration specs, `bin/rspec`, `bin/rake`

---

## Current Branch State

- `cmd_watch_test.go` already contains `TestLoadWatchConfigurationMergesUserAndDefaultWatches`. Keep that regression and make it pass; do not replace it with a weaker assertion.
- `spec/integration/plur_watch/watch_integration_spec.rb` already contains `keeps default watches when user config adds a custom watch mapping`. Keep that example and make it pass from the current branch state.
- The old `docs/watch-output-bugs.md` transcript examples (S1, S2, S3) are acceptance criteria, not background notes. Convert them into executable test cases during implementation.

## File Structure

- Modify: `cmd_watch.go`
  Responsibility: watch-mode config loading, prompt lifecycle, and wiring a shared interactive output writer into the runtime.
- Modify: `cmd_watch_test.go`
  Responsibility: runtime regression proving `loadWatchConfiguration` keeps builtin and user watch mappings.
- Modify: `autodetect/defaults.go`
  Responsibility: merged builtin+user watch handling for validation.
- Modify: `autodetect/defaults_test.go`
  Responsibility: helper-level regression coverage for validation using merged watch lists.
- Modify: `logger/logger.go`
  Responsibility: construct a logger that writes through the shared interactive stderr writer during watch mode.
- Create: `watch/terminal_output.go`
  Responsibility: synchronized interactive writer that owns prompt rendering, log-safe newline insertion, and echoed command formatting.
- Create: `watch/terminal_output_test.go`
  Responsibility: unit tests for prompt reuse and newline-before-log behavior.
- Modify: `watch/watcher.go`
  Responsibility: route command echoing through `TerminalOutput` and stop double-reporting failing job exits.
- Modify: `watch/file_event_handler.go`
  Responsibility: emit one watch-level warning when job execution fails.
- Modify: `watch/file_event_handler_test.go`
  Responsibility: regression test proving a failing job yields one watch-level warning.
- Modify: `spec/support/plur_watch_helper.rb`
  Responsibility: helpers for temp watch specs and prompt/output assertions.
- Modify: `spec/integration/plur_watch/watch_integration_spec.rb`
  Responsibility: end-to-end regressions for additive config merging and the S1/S2/S3 output behavior.
- Modify: `docs/configuration.md`
  Responsibility: document additive `[[watch]]` semantics.
- Modify: `docs/features/watch-mode.md`
  Responsibility: remove outdated broken-output guidance and describe the current prompt behavior.

Add this small writer-injection helper to `logger/logger.go` and use it only from watch mode:

```go
func NewWithWriter(w io.Writer) *slog.Logger {
	return slog.New(NewCustomTextHandler(w, &slog.HandlerOptions{
		Level: &logLevel,
	}))
}
```

Do not redesign the logger package beyond that constructor.

### Task 1: Fix additive watch merging first

**Files:**
- Modify: `cmd_watch.go`
- Modify: `cmd_watch_test.go`
- Modify: `autodetect/defaults.go`
- Modify: `autodetect/defaults_test.go`
- Modify: `spec/integration/plur_watch/watch_integration_spec.rb`
- Test: `cmd_watch_test.go`
- Test: `autodetect/defaults_test.go`
- Test: `spec/integration/plur_watch/watch_integration_spec.rb`

- [ ] **Step 1: Add the missing validation regression and keep the existing runtime/integration regressions**

Add this helper-level regression to `autodetect/defaults_test.go` without removing the existing branch-local tests:

```go
func TestMergeAllWatchesIncludesBuiltinDefaults(t *testing.T) {
	userWatches := []watch.WatchMapping{
		{
			Name:    "custom-config-watch",
			Source:  "config/**/*.yml",
			Targets: []string{"spec/config_spec.rb"},
			Jobs:    []string{"rspec"},
		},
	}

	merged := mergeAllWatches(userWatches)

	var names []string
	for _, mapping := range merged {
		names = append(names, mapping.Name)
	}

	assert.Contains(t, names, "custom-config-watch")
	assert.Contains(t, names, "lib-to-spec")
	assert.Contains(t, names, "spec-files")
}
```

Ensure the existing branch-local runtime regression stays present in `cmd_watch_test.go`:

```go
func TestLoadWatchConfigurationMergesUserAndDefaultWatches(t *testing.T) {
	cli := &PlurCLI{
		WatchMappings: []watch.WatchMapping{
			{
				Name:    "custom-config-watch",
				Source:  "config/**/*.yml",
				Targets: []string{"spec/config_spec.rb"},
				Jobs:    []string{"rspec"},
			},
		},
	}

	resolved, watches, err := loadWatchConfiguration(cli, "rspec")
	require.NoError(t, err)
	require.Equal(t, "rspec", resolved.Name)

	var names []string
	for _, mapping := range watches {
		names = append(names, mapping.Name)
	}

	assert.Contains(t, names, "custom-config-watch")
	assert.Contains(t, names, "lib-to-spec")
	assert.Contains(t, names, "spec-files")
}
```

Ensure the existing branch-local Ruby regression stays present in `spec/integration/plur_watch/watch_integration_spec.rb`:

```ruby
it "keeps default watches when user config adds a custom watch mapping" do
  Dir.mktmpdir do |tmpdir|
    config_path = File.join(tmpdir, ".plur.toml")
    File.write(config_path, <<~TOML)
      [[watch]]
      name = "custom-config-watch"
      source = "config/**/*.yml"
      targets = ["spec/config_spec.rb"]
      jobs = ["rspec"]
    TOML

    stdout, stderr, status = Open3.capture3(
      {"PLUR_CONFIG_FILE" => config_path},
      plur_binary, "watch", "find", "lib/calculator.rb",
      chdir: default_ruby_dir.to_s
    )

    expect(stderr).to eq("")
    expect(stdout).to include('msg="checking watch" file=lib/calculator.rb')
    expect(stdout).to include('msg="found rules" name=lib-to-spec source=lib/**/*.rb')
    expect(stdout).to include('msg="found files" files=spec/calculator_spec.rb')
    expect(status.exitstatus).to eq(0)
  end
end
```

- [ ] **Step 2: Run the focused regressions and confirm they fail**

Run:

```bash
mise exec -- go test . -run TestLoadWatchConfigurationMergesUserAndDefaultWatches -count=1
mise exec -- go test ./autodetect -run TestMergeAllWatchesIncludesBuiltinDefaults -count=1
mise exec -- bin/rspec spec/integration/plur_watch/watch_integration_spec.rb \
  --example "keeps default watches when user config adds a custom watch mapping"
```

Expected:

- the new `./autodetect` test fails to compile because `mergeAllWatches` does not exist yet
- `TestLoadWatchConfigurationMergesUserAndDefaultWatches` fails because only the custom mapping is returned
- the Ruby integration regression exits non-zero and reports `msg="found rules" count=0`

- [ ] **Step 3: Implement additive watch merging for runtime and validation**

In `autodetect/defaults.go`, add a helper that always prepends built-in watch mappings before any user mappings:

```go
func mergeAllWatches(userWatches []watch.WatchMapping) []watch.WatchMapping {
	if len(userWatches) == 0 {
		return builtinDefaults.Defaults.Watches
	}

	merged := make([]watch.WatchMapping, 0, len(builtinDefaults.Defaults.Watches)+len(userWatches))
	merged = append(merged, builtinDefaults.Defaults.Watches...)
	merged = append(merged, userWatches...)
	return merged
}
```

Then use it in validation:

```go
func ValidateConfig(userJobs map[string]job.Job, userWatches []watch.WatchMapping) error {
	resolvedJobs, _, err := buildResolvedJobs(userJobs)
	if err != nil {
		return err
	}

	if err := validateResolvedJobs(resolvedJobs); err != nil {
		return err
	}

	return watch.ValidateConfig(resolvedJobs, mergeAllWatches(userWatches))
}
```

In `cmd_watch.go`, stop treating user mappings as a replacement set:

```go
func loadWatchConfiguration(cli *PlurCLI, explicitJobName string) (*autodetect.ResolveJobResult, []watch.WatchMapping, error) {
	result, err := autodetect.ResolveJob(explicitJobName, cli.Job, nil)
	if err != nil {
		return nil, nil, err
	}

	watches := make([]watch.WatchMapping, 0, len(result.Watches)+len(cli.WatchMappings))
	watches = append(watches, result.Watches...)
	watches = append(watches, cli.WatchMappings...)

	logInheritedFields(result.Name, result.Inherited)
	return result, watches, nil
}
```

Do not deduplicate by name in this change. Appending is the desired semantics for this branch.

- [ ] **Step 4: Re-run the regressions and confirm they pass**

Run:

```bash
mise exec -- go test . -run TestLoadWatchConfigurationMergesUserAndDefaultWatches -count=1
mise exec -- go test ./autodetect -run TestMergeAllWatchesIncludesBuiltinDefaults -count=1
mise exec -- bin/rspec spec/integration/plur_watch/watch_integration_spec.rb \
  --example "keeps default watches when user config adds a custom watch mapping"
```

Expected: all three commands PASS.

- [ ] **Step 5: Commit the config merge fix**

```bash
git add cmd_watch.go cmd_watch_test.go autodetect/defaults.go autodetect/defaults_test.go spec/integration/plur_watch/watch_integration_spec.rb
git commit -m "Fix watch configuration merging"
```

### Task 2: Lock down prompt rendering and single-warning behavior in Go tests

**Files:**
- Create: `watch/terminal_output_test.go`
- Modify: `watch/file_event_handler_test.go`
- Test: `watch/terminal_output_test.go`
- Test: `watch/file_event_handler_test.go`

- [ ] **Step 1: Add unit tests for the S1/S2/S3 mechanics**

Create `watch/terminal_output_test.go` with prompt and log-order regressions:

```go
package watch

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTerminalOutput_EchoCommandUsesActivePromptLine(t *testing.T) {
	var buf bytes.Buffer
	output := NewTerminalOutput(&buf)

	require.NoError(t, output.ShowPrompt())
	require.NoError(t, output.EchoCommand([]string{"bundle", "exec", "rspec", "spec/calculator_spec.rb"}))

	assert.Equal(t, "[plur] > bundle exec rspec spec/calculator_spec.rb\n", buf.String())
}

func TestTerminalOutput_WriteStartsNewLineWhenPromptIsVisible(t *testing.T) {
	var buf bytes.Buffer
	output := NewTerminalOutput(&buf)

	require.NoError(t, output.ShowPrompt())
	_, err := output.Write([]byte("14:06:41 - DEBUG - watch path=\"spec/calculator_spec.rb\"\n"))

	require.NoError(t, err)
	assert.Equal(t, "[plur] > \n14:06:41 - DEBUG - watch path=\"spec/calculator_spec.rb\"\n", buf.String())
}
```

Extend `watch/file_event_handler_test.go` with the watch-level failure logging regression:

```go
func TestFileEventHandler_HandleBatch_LogsJobFailureOnce(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "spec"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "spec", "user_spec.rb"), []byte("# spec"), 0644))

	originalLogger := logger.Logger
	defer func() { logger.Logger = originalLogger }()

	var logBuf bytes.Buffer
	logger.Logger = slog.New(logger.NewCustomTextHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelWarn}))

	handler := &FileEventHandler{
		Jobs: map[string]job.Job{
			"rspec": {Name: "rspec", Cmd: []string{"bundle", "exec", "rspec", "{{target}}"}},
		},
		Watches: []WatchMapping{
			{Name: "spec-files", Source: "spec/**/*_spec.rb", Jobs: []string{"rspec"}},
		},
		CWD: tmpDir,
		Executor: func(j job.Job, targets []string, cwd string) error {
			return errors.New("exit status 1")
		},
	}

	handler.HandleBatch([]string{"spec/user_spec.rb"})

	output := logBuf.String()
	assert.Contains(t, output, "Job execution failed")
	assert.NotContains(t, output, "Job execution error")
}
```

Update imports in `watch/file_event_handler_test.go` to include `bytes`, `errors`, `log/slog`, and `github.com/rsanheim/plur/logger`.

- [ ] **Step 2: Run the Go tests and confirm they fail**

Run:

```bash
mise exec -- go test ./watch -run 'TestTerminalOutput|TestFileEventHandler_HandleBatch_LogsJobFailureOnce' -count=1
```

Expected:

- compile failure because `NewTerminalOutput` and `ShowPrompt`/`EchoCommand` do not exist yet
- or a test failure because `FileEventHandler` still logs `Job execution error`

- [ ] **Step 3: Add the testable output primitive and normalize the warning message**

Create `watch/terminal_output.go`:

```go
package watch

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
)

type TerminalOutput struct {
	writer        io.Writer
	mu            sync.Mutex
	promptVisible bool
}

var defaultTerminalOutput = NewTerminalOutput(os.Stderr)

func NewTerminalOutput(w io.Writer) *TerminalOutput {
	return &TerminalOutput{writer: w}
}

func SetTerminalOutput(output *TerminalOutput) func() {
	previous := defaultTerminalOutput
	defaultTerminalOutput = output
	return func() {
		defaultTerminalOutput = previous
	}
}

func (o *TerminalOutput) Write(p []byte) (int, error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.promptVisible {
		if _, err := fmt.Fprint(o.writer, "\n"); err != nil {
			return 0, err
		}
		o.promptVisible = false
	}

	return o.writer.Write(p)
}

func (o *TerminalOutput) ShowPrompt() error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.promptVisible {
		return nil
	}

	_, err := fmt.Fprint(o.writer, "[plur] > ")
	if err == nil {
		o.promptVisible = true
	}
	return err
}

func (o *TerminalOutput) EchoCommand(args []string) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	line := strings.Join(args, " ")
	if o.promptVisible {
		_, err := fmt.Fprintf(o.writer, "%s\n", line)
		o.promptVisible = false
		return err
	}

	_, err := fmt.Fprintf(o.writer, "[plur] > %s\n", line)
	return err
}
```

In `watch/file_event_handler.go`, log the canonical message once at the orchestration layer:

```go
if err := h.executor()(j, targets, h.CWD); err != nil {
	logger.Logger.Warn("Job execution failed", "job", jobName, "error", err)
}
```

- [ ] **Step 4: Re-run the Go tests and confirm they pass**

Run:

```bash
mise exec -- go test ./watch -run 'TestTerminalOutput|TestFileEventHandler_HandleBatch_LogsJobFailureOnce' -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit the output primitives**

```bash
git add watch/terminal_output.go watch/terminal_output_test.go watch/file_event_handler.go watch/file_event_handler_test.go
git commit -m "Add watch terminal output primitives"
```

### Task 3: Wire the interactive writer through watch mode and make S1/S2/S3 pass end-to-end

**Files:**
- Modify: `cmd_watch.go`
- Modify: `logger/logger.go`
- Modify: `watch/watcher.go`
- Modify: `spec/support/plur_watch_helper.rb`
- Modify: `spec/integration/plur_watch/watch_integration_spec.rb`
- Test: `spec/integration/plur_watch/watch_integration_spec.rb`

- [ ] **Step 1: Add end-to-end watch regressions for prompt reuse, debug log placement, and single warning output**

Extend `spec/support/plur_watch_helper.rb` with two small helpers:

```ruby
def with_temp_watch_spec(filename, contents)
  path = default_ruby_dir.join("spec", filename)
  File.write(path, contents)
  yield path
ensure
  File.delete(path) if path.exist?
end

def expect_prompt_command(output, command)
  expect(output).to include("[plur] > #{command}")
  expect(output).not_to include("[plur] >\n[plur] > #{command}")
end
```

Then add these integration examples to `spec/integration/plur_watch/watch_integration_spec.rb`:

```ruby
it "echoes the command on the active prompt line" do
  with_temp_watch_spec("watch_prompt_spec.rb", <<~RUBY) do |spec_path|
    RSpec.describe "watch prompt" do
      it "passes" do
        expect(1).to eq(1)
      end
    end
  RUBY

    original = spec_path.read
    result = run_plur_watch(
      timeout: 10,
      until_output: "bundle exec rspec spec/watch_prompt_spec.rb"
    ) do
      spec_path.write(original + "\n# trigger prompt\n")
    end

    expect_prompt_command(result.err, "bundle exec rspec spec/watch_prompt_spec.rb")
  end
end

it "moves debug log lines off the prompt before printing them" do
  with_temp_watch_spec("watch_debug_spec.rb", <<~RUBY) do |spec_path|
    RSpec.describe "watch debug" do
      it "passes" do
        expect(1).to eq(1)
      end
    end
  RUBY

    original = spec_path.read
    result = run_plur_watch(
      timeout: 10,
      until_output: "bundle exec rspec spec/watch_debug_spec.rb"
    ) do
      spec_path.write(original + "\n# trigger debug\n")
    end

    expect(result.err).to include("[plur] > \n")
    expect(result.err).not_to include("[plur] > 14:")
  end
end

it "logs a failing watched job once" do
  with_temp_watch_spec("watch_failure_spec.rb", <<~RUBY) do |spec_path|
    RSpec.describe "watch failure" do
      it "fails" do
        expect(1).to eq(2)
      end
    end
  RUBY

    original = spec_path.read
    result = run_plur_watch(
      timeout: 10,
      until_output: "Job execution failed"
    ) do
      spec_path.write(original + "\n# trigger failure\n")
    end

    expect(result.err.scan("Job execution failed").size).to eq(1)
    expect(result.err).not_to include("Job execution error")
  end
end
```

- [ ] **Step 2: Run the integration regressions and confirm they fail**

Run:

```bash
bin/rspec spec/integration/plur_watch/watch_integration_spec.rb \
  --example "echoes the command on the active prompt line"
bin/rspec spec/integration/plur_watch/watch_integration_spec.rb \
  --example "moves debug log lines off the prompt before printing them"
bin/rspec spec/integration/plur_watch/watch_integration_spec.rb \
  --example "logs a failing watched job once"
```

Expected:

- the first example fails because prompt and echoed command are split across separate writes/streams
- the second example fails because the debug line is rendered directly after `[plur] > `
- the third example fails because both `Job execution failed` and `Job execution error` appear in the captured output

- [ ] **Step 3: Wire `TerminalOutput` into watch mode and collapse failure logging to one layer**

Add the watch-mode logger constructor in `logger/logger.go`:

```go
func NewWithWriter(w io.Writer) *slog.Logger {
	return slog.New(NewCustomTextHandler(w, &slog.HandlerOptions{
		Level: &logLevel,
	}))
}
```

In `watch/watcher.go`, replace direct `fmt.Printf` command echoing and return non-zero exits to the caller without logging them here:

```go
func RunCommand(args []string) error {
	if len(args) == 0 {
		return nil
	}

	if err := defaultTerminalOutput.EchoCommand(args); err != nil {
		return err
	}

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
```

```go
func ExecuteJob(j job.Job, targetFiles []string, cwd string) error {
	logger.Logger.Info("Executing job", "job", j.Name, "targets", fmt.Sprintf("%+v", targetFiles))

	// ... build cmd as today ...

	if err := defaultTerminalOutput.EchoCommand(cmd); err != nil {
		return err
	}

	execCmd := exec.Command(cmd[0], cmd[1:]...)
	execCmd.Dir = cwd
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr
	execCmd.Env = append(os.Environ(), j.Env...)
	return execCmd.Run()
}
```

In `cmd_watch.go`, create one shared interactive writer for the whole watch session, point the watch logger at it, and use it for prompt redraws:

```go
terminalOutput := watch.NewTerminalOutput(os.Stderr)
restoreOutput := watch.SetTerminalOutput(terminalOutput)
defer restoreOutput()

originalLogger := logger.Logger
logger.Logger = logger.NewWithWriter(terminalOutput)
defer func() {
	logger.Logger = originalLogger
}()
```

Keep the prompt queue, but render the prompt through `TerminalOutput`:

```go
case <-promptChan:
	_ = terminalOutput.ShowPrompt()
```

For the interactive Enter-to-run-all-tests path, handle the new `RunCommand` error in `cmd_watch.go` so failures are still reported once:

```go
case "":
	cmd := job.BuildJobAllCmd(resolvedJob)
	if err := watch.RunCommand(cmd); err != nil {
		logger.Logger.Warn("Job execution failed", "job", resolvedJob.Name, "error", err)
	}
	showPrompt()
```

Do not reintroduce `Job execution error` anywhere in this change.

- [ ] **Step 4: Run focused verification, then broad watch verification**

Run:

```bash
mise exec -- go test . ./autodetect ./watch -run 'TestLoadWatchConfigurationMergesUserAndDefaultWatches|TestMergeAllWatchesIncludesBuiltinDefaults|TestTerminalOutput|TestFileEventHandler_HandleBatch_LogsJobFailureOnce' -count=1
bin/rspec spec/integration/plur_watch/watch_integration_spec.rb
bin/rake test:default_ruby
```

Expected:

- focused Go regressions PASS
- `watch_integration_spec.rb` PASS, including the new S1/S2/S3 examples
- `bin/rake test:default_ruby` PASS as a quick outside-in check against the default Ruby fixture project

- [ ] **Step 5: Commit the watch runtime/output fix**

```bash
git add cmd_watch.go logger/logger.go watch/watcher.go spec/support/plur_watch_helper.rb spec/integration/plur_watch/watch_integration_spec.rb
git commit -m "Fix watch prompt and failure output"
```

### Task 4: Refresh docs to match the implemented behavior

**Files:**
- Modify: `docs/configuration.md`
- Modify: `docs/features/watch-mode.md`
- Test: `docs/configuration.md`
- Test: `docs/features/watch-mode.md`

- [ ] **Step 1: Confirm the docs still describe stale watch behavior**

Run:

```bash
rg -n "Multiple \"plur> \" prompts appearing|janky terminal|Watch mode uses \\[\\[watch\\]\\] entries" docs/features/watch-mode.md docs/configuration.md
```

Expected:

- `docs/features/watch-mode.md` still describes broken prompt behavior as a known issue
- `docs/configuration.md` does not yet say that user `[[watch]]` entries are additive to the built-in mappings for the resolved job

- [ ] **Step 2: Update the watch docs with the current behavior**

In `docs/configuration.md`, add an explicit additive-semantics paragraph under “Watch Configuration”:

```md
Plur always starts with the built-in watch mappings for the resolved job and then appends any user-defined `[[watch]]` entries from configuration. User mappings are additive; they do not replace built-in mappings like `lib-to-spec`, `app-to-spec`, or `spec-files`.
```

In `docs/features/watch-mode.md`, replace the stale “Concurrent Output” limitation with current behavior:

```md
### Interactive Output

Watch mode keeps a shell-style prompt on stderr:

- when idle it shows `[plur] > `
- when a file change triggers a run, the echoed command reuses that prompt line
- debug/info log lines force a newline before rendering so they never print on top of the prompt
- failing runs emit one watch-level warning after the test process exits
```

Do not leave the old “multiple `plur>` prompts appearing” wording behind anywhere in the file.

- [ ] **Step 3: Verify the stale text is gone and the new text is present**

Run:

```bash
rg -n "additive; they do not replace|shell-style prompt on stderr|multiple \"plur> \" prompts appearing|janky terminal" docs/configuration.md docs/features/watch-mode.md
```

Expected:

- the additive-semantics and shell-style prompt text are found
- the stale “multiple `plur>` prompts appearing” and “janky terminal” language is not found

- [ ] **Step 4: Run a final repo-level verification sweep for this branch**

Run:

```bash
git diff --check
mise exec -- go test . ./autodetect ./watch -count=1
bin/rspec spec/integration/plur_watch/watch_integration_spec.rb
bin/rake test:default_ruby
```

Expected: all commands PASS with no diff-check errors.

- [ ] **Step 5: Commit the docs refresh**

```bash
git add docs/configuration.md docs/features/watch-mode.md
git commit -m "Document watch config and output behavior"
```

## Self-Review

### Spec Coverage

- Additive config semantics: covered by Task 1.
- Validation/config-loading alignment: covered by Task 1.
- Prompt reuse, debug-log newline handling, and single warning behavior: covered by Tasks 2 and 3.
- Public docs reflecting the implemented behavior: covered by Task 4.

### Placeholder Scan

- No `TODO`, `TBD`, or “handle appropriately” placeholders remain.
- Every code-changing step includes concrete snippets.
- Every verification step names exact commands and expected outcomes.

### Type Consistency

- `mergeAllWatches` is referenced only after being defined in Task 1.
- `TerminalOutput`, `ShowPrompt`, `EchoCommand`, and `SetTerminalOutput` are introduced in Task 2 before Task 3 wires them into runtime code.
- The single canonical warning string is `Job execution failed`; `Job execution error` is intentionally removed.
