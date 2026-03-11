# Exclude Pattern CLI Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a repeatable `--exclude-pattern` flag to `plur spec` that removes matching test files from plur's discovered file set before worker grouping, with visible debug logging in both dry-run and real runs.

**Architecture:** Keep file discovery consolidated in `glob.go` by extending the existing discovery path rather than adding a parallel abstraction. Apply exclude matching after include expansion and before runner construction, then emit discovery/exclusion summaries through the existing stderr logger with a dry-run-prefixed message in dry-run mode.

**Tech Stack:** Go, Kong CLI parsing, `github.com/bmatcuk/doublestar/v4`, testify, RSpec integration specs, `bin/rake`

---

### Task 1: Lock down exclude filtering in Go unit tests

**Files:**
- Modify: `glob_test.go`
- Modify: `glob.go`
- Test: `glob_test.go`

**Step 1: Write the failing test**

Add unit tests in `glob_test.go` that describe the desired filtering behavior for both discovery entry points:

```go
func TestFindFilesFromJobAppliesExcludePatterns(t *testing.T) {
	rspecJob := job.Job{Name: "rspec", TargetPattern: "spec/**/*_spec.rb"}

	files, err := FindFilesFromJob(rspecJob, []string{"spec/system/**/*_spec.rb"})

	require.NoError(t, err)
	assert.ElementsMatch(t, []string{
		"spec/models/user_spec.rb",
		"spec/models/post_spec.rb",
	}, files)
}

func TestExpandPatternsFromJobAppliesExcludePatterns(t *testing.T) {
	rspecJob := job.Job{Name: "fast", Framework: "rspec"}

	files, err := ExpandPatternsFromJob(
		[]string{"spec"},
		rspecJob,
		[]string{"spec/models/system_spec.rb"},
	)

	require.NoError(t, err)
	assert.NotContains(t, files, "spec/models/system_spec.rb")
}
```

Also add one bad-pattern test so the helper fails loudly on an invalid exclude glob:

```go
_, err := FindFilesFromJob(rspecJob, []string{"["})
require.Error(t, err)
assert.Contains(t, err.Error(), "exclude")
```

**Step 2: Run test to verify it fails**

Run:

```bash
go test . -run 'TestFindFilesFromJob|TestExpandPatternsFromJob' -count=1
```

Expected: FAIL because the discovery functions do not accept exclude patterns yet.

**Step 3: Write minimal implementation**

In `glob.go`, extend the existing discovery flow instead of adding a new abstraction:

```go
func FindFilesFromJob(j job.Job, excludePatterns []string) ([]string, error) {
	patterns, err := framework.TargetPatternsForJob(j)
	if err != nil {
		return nil, err
	}
	return expandGlobPatterns(patterns, excludePatterns)
}

func filterExcludedFiles(files []string, excludePatterns []string) ([]string, []string, error) {
	if len(excludePatterns) == 0 {
		return dedupeAndSort(files), nil, nil
	}

	var kept []string
	var excluded []string
	for _, file := range dedupeAndSort(files) {
		matched, err := matchesAnyExclude(file, excludePatterns)
		if err != nil {
			return nil, nil, err
		}
		if matched {
			excluded = append(excluded, file)
			continue
		}
		kept = append(kept, file)
	}

	return kept, excluded, nil
}
```

Keep the helper internal to `glob.go`. Normalize paths before matching, sort the
final slices for deterministic tests/logging, and update both `FindFilesFromJob`
and `ExpandPatternsFromJob` to reuse the same exclude helper.

**Step 4: Run test to verify it passes**

Run:

```bash
go test . -run 'TestFindFilesFromJob|TestExpandPatternsFromJob' -count=1
```

Expected: PASS with the new exclude-aware discovery path.

**Step 5: Commit**

```bash
git add glob.go glob_test.go
git commit -m "Add exclude filtering to glob discovery"
```

### Task 2: Add CLI plumbing and integration coverage

**Files:**
- Create: `spec/integration/shared/exclude_pattern_spec.rb`
- Modify: `main.go`
- Modify: `glob.go`
- Test: `spec/integration/shared/exclude_pattern_spec.rb`

**Step 1: Write the failing test**

Create `spec/integration/shared/exclude_pattern_spec.rb` with integration coverage for the CLI behavior:

```ruby
RSpec.describe "Exclude pattern CLI" do
  it "excludes files during autodiscovery" do
    chdir(default_ruby_dir) do
      result = run_plur("--dry-run", "--exclude-pattern", "spec/models/**/*_spec.rb")

      expect(result).to be_success
      expect(result.err).not_to include("spec/models/user_spec.rb")
      expect(result.err).to include("spec/calculator_spec.rb")
    end
  end

  it "applies repeated exclude patterns" do
    chdir(default_ruby_dir) do
      result = run_plur(
        "--dry-run",
        "--exclude-pattern", "spec/models/**/*_spec.rb",
        "--exclude-pattern", "spec/services/**/*_spec.rb"
      )

      expect(result.err).not_to include("spec/models/user_spec.rb")
      expect(result.err).not_to include("spec/services/email_service_spec.rb")
    end
  end

  it "errors when excludes remove every selected file" do
    chdir(default_ruby_dir) do
      result = run_plur_allowing_errors(
        "--dry-run",
        "spec/models",
        "--exclude-pattern", "spec/models/**/*_spec.rb"
      )

      expect(result.exit_status).not_to eq(0)
      expect(result.err).to include("no test files remain after applying exclude patterns")
    end
  end
end
```

Add one minitest example in the same file to prove the feature stays file-level
across frameworks:

```ruby
result = run_plur("--use", "minitest", "--dry-run", "--exclude-pattern", "test/string_helper_test.rb")
expect(result.err).to include("test/calculator_test.rb")
expect(result.err).not_to include("test/string_helper_test.rb")
```

**Step 2: Run test to verify it fails**

Run:

```bash
bin/rspec spec/integration/shared/exclude_pattern_spec.rb
```

Expected: FAIL because Kong/main do not recognize `--exclude-pattern` yet.

**Step 3: Write minimal implementation**

Update `SpecCmd` in `main.go` to accept a repeatable exclude flag:

```go
type SpecCmd struct {
	Patterns        []string `arg:"" optional:"" help:"Spec files or patterns to run (default: spec/**/*_spec.rb)"`
	ExcludePatterns []string `help:"Exclude test files matching pattern (repeatable)" name:"exclude-pattern"`
	Tags            []string `help:"Filter RSpec by tag (repeatable)" name:"tag"`
}
```

Then pass `r.ExcludePatterns` through the existing discovery branches:

```go
if len(r.Patterns) > 0 {
	testFiles, err = ExpandPatternsFromJob(r.Patterns, currentJob, r.ExcludePatterns)
} else {
	testFiles, err = FindFilesFromJob(currentJob, r.ExcludePatterns)
}
```

Return the new post-exclusion error when the filtered file list is empty.

**Step 4: Run test to verify it passes**

Run:

```bash
bin/rspec spec/integration/shared/exclude_pattern_spec.rb
```

Expected: PASS with the CLI flag wired through discovery for both RSpec and Minitest.

**Step 5: Commit**

```bash
git add main.go glob.go spec/integration/shared/exclude_pattern_spec.rb
git commit -m "Add exclude-pattern CLI flag"
```

### Task 3: Route discovery reporting through the structured logger

**Files:**
- Modify: `main.go`
- Modify: `utils.go`
- Modify: `spec/integration/shared/exclude_pattern_spec.rb`
- Test: `spec/integration/shared/exclude_pattern_spec.rb`

**Step 1: Write the failing test**

Extend `spec/integration/shared/exclude_pattern_spec.rb` with logging assertions for both dry-run and real runs:

```ruby
it "shows discovery and exclusion info through the logger in dry-run mode" do
  chdir(default_ruby_dir) do
    result = run_plur(
      "--debug",
      "--dry-run",
      "--exclude-pattern", "spec/models/**/*_spec.rb"
    )

    expect(result.err).to include("[dry-run]")
    expect(result.err).to include("test file discovery")
    expect(result.err).to include('excluded=2')
    expect(result.err).to include('remaining=')
  end
end

it "shows the same discovery event in real runs" do
  chdir(default_ruby_dir) do
    result = run_plur(
      "--debug",
      "spec/models",
      "--exclude-pattern", "spec/models/system_spec.rb"
    )

    expect(result.err).to include("test file discovery")
    expect(result.err).not_to include("[dry-run]")
  end
end
```

**Step 2: Run test to verify it fails**

Run:

```bash
bin/rspec spec/integration/shared/exclude_pattern_spec.rb
```

Expected: FAIL because discovery/exclusion details are not emitted through the logger yet.

**Step 3: Write minimal implementation**

Add a small helper for discovery logging that uses the existing stderr logger
for both execution paths:

```go
func logDiscoverySummary(cfg *config.GlobalConfig, msg string, attrs ...any) {
	if cfg.DryRun {
		msg = "[dry-run] " + msg
	}
	logger.Logger.Debug(msg, attrs...)
}
```

Use it in `main.go` after discovery and exclusion are complete:

```go
logDiscoverySummary(cfg, "test file discovery",
	"job", currentJob.Name,
	"framework", currentJob.Framework,
	"patterns", r.Patterns,
	"exclude_patterns", r.ExcludePatterns,
	"discovered", discoveredCount,
	"excluded", len(excludedFiles),
	"remaining", len(testFiles),
)
```

If excluded files are present, emit a second debug event with the actual file
list so `--debug` users can see exactly what was removed. Keep existing
`printDryRunWorker` output unchanged.

If discovery-specific `toStdErr` calls are touched while wiring this in, move
only those touched paths onto the logger rather than attempting a broad output
refactor.

**Step 4: Run test to verify it passes**

Run:

```bash
bin/rspec spec/integration/shared/exclude_pattern_spec.rb
```

Expected: PASS with matching debug output in dry-run and real runs.

**Step 5: Commit**

```bash
git add main.go utils.go spec/integration/shared/exclude_pattern_spec.rb
git commit -m "Log exclude filtering through discovery logger"
```

### Task 4: Update help text and user-facing docs

**Files:**
- Modify: `spec/integration/shared/plur_integration_spec.rb`
- Modify: `docs/usage.md`
- Modify: `docs/overview/project-status.md`
- Test: `spec/integration/shared/plur_integration_spec.rb`

**Step 1: Write the failing test**

Add a help assertion to `spec/integration/shared/plur_integration_spec.rb`:

```ruby
expect(result.out).to include("exclude-pattern")
```

**Step 2: Run test to verify it fails**

Run:

```bash
bin/rspec spec/integration/shared/plur_integration_spec.rb
```

Expected: FAIL because the help text does not mention the new flag yet.

**Step 3: Write minimal implementation**

Update the `main.go` flag help text and document the new flag in `docs/usage.md`
and `docs/overview/project-status.md`.

Example docs snippet:

```md
plur --exclude-pattern 'spec/system/**/*_spec.rb'
plur spec spec/models --exclude-pattern 'spec/models/system_spec.rb'
```

Keep docs limited to current behavior: file-level exclusion only, implemented by
plur before splitting work across workers.

**Step 4: Run test to verify it passes**

Run:

```bash
bin/rspec spec/integration/shared/plur_integration_spec.rb
```

Expected: PASS with updated CLI help output.

**Step 5: Commit**

```bash
git add main.go spec/integration/shared/plur_integration_spec.rb docs/usage.md docs/overview/project-status.md
git commit -m "Document exclude-pattern CLI flag"
```

### Task 5: Run focused verification, then broader project checks

**Files:**
- Test: `glob_test.go`
- Test: `spec/integration/shared/exclude_pattern_spec.rb`
- Test: `spec/integration/shared/plur_integration_spec.rb`

**Step 1: Run focused Go coverage**

Run:

```bash
go test . -run 'TestFindFilesFromJob|TestExpandPatternsFromJob' -count=1
```

Expected: PASS

**Step 2: Run focused integration specs**

Run:

```bash
bin/rspec spec/integration/shared/exclude_pattern_spec.rb spec/integration/shared/plur_integration_spec.rb
```

Expected: PASS

**Step 3: Run broader project verification**

Run:

```bash
bin/rake test:default_ruby
bin/rake test
```

Expected: PASS for both commands.

**Step 4: Commit final verification note**

If any doc or code changes were needed during verification, commit them:

```bash
git add <touched files>
git commit -m "Finish exclude-pattern verification"
```

If no new changes were needed, skip this commit.
