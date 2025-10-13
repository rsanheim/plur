# Analysis: plur spec/test Command Behavior

## Current Behavior Analysis

### Problem Statement

The user expects the following behavior when a project has both `spec/` and `test/` directories:

1. `plur test` → should run all minitest files inside test/
2. `plur spec` → should run all rspec files inside spec/
3. `plur spec test/` → should run all minitest files inside test/ (explicit path override)

However, the current implementation doesn't support this workflow properly.

## Current Implementation Findings

### 1. Command Structure

From `plur/main.go:170-179`:
- There is **only** a `SpecCmd` struct - no `TestCmd` exists
- `SpecCmd` is marked as `default:"withargs"` making it the default command
- This means both `plur` and `plur spec` run the same code path

### 2. Task Detection Logic

From `plur/internal/task/task.go:314-328` in `DetectFramework()`:
```go
// Check for test/ directory first (minitest)
if exists("test") {
    return NewMinitestTask()
}
// Check for spec/ directory (rspec)
if exists("spec") {
    return NewRSpecTask()
}
// Default to RSpec for backward compatibility
return NewRSpecTask()
```

**Critical Issue**: When both `spec/` and `test/` exist, it **always picks minitest** because it checks `test/` first.

### 3. Command Execution Flow

From `plur/main.go:30-43` in `SpecCmd.Run()`:
```go
// Priority: CLI --use, config use, auto-detect
taskName := r.Use
if taskName == "" && parent.Use != "" {
    taskName = parent.Use
}
if taskName == "" {
    detectedTask := task.DetectFramework()
    taskName = detectedTask.Name
}
```

The task selection priority is:
1. CLI flag `--use`
2. Config file `use` setting
3. Auto-detection (which has the test-first bias)

### 4. Pattern Handling

From `plur/glob.go:39-42`, when a directory is passed as a pattern:
```go
if fileInfo.IsDir() {
    // Directory: use task's test glob pattern within this directory
    dirPattern := filepath.Join(pattern, "**", "*"+suffix)
    matches, err = doublestar.FilepathGlob(dirPattern)
}
```

This means `plur spec test/` would look for files matching the current task's suffix (e.g., `_spec.rb` for RSpec or `_test.rb` for minitest) within the `test/` directory.

## Root Causes

### Issue 1: No Separate 'test' Command
- There's no `TestCmd` struct to handle `plur test`
- The `spec` command name is misleading when running minitest

### Issue 2: Auto-detection Bias
- `DetectFramework()` always prefers `test/` over `spec/`
- No smart detection based on which command was invoked
- No ability to have both frameworks active simultaneously

### Issue 3: Command Semantics Mismatch
- `plur spec test/` tries to be smart but the logic is convoluted
- The task's test suffix determines what files are found, not the directory

## Proposed Solution

### Option 1: Smart Command-Based Detection (Recommended)

1. **Add a `TestCmd` struct** that's an alias/wrapper for SpecCmd but hints at minitest
2. **Make detection command-aware**:
   - `plur test` → prefer minitest task
   - `plur spec` → prefer rspec task
   - `plur` (no command) → use existing detection
3. **Support explicit directory patterns properly**:
   - When given a directory path, detect framework based on the path
   - `plur spec test/` → use minitest for test/ directory
   - `plur test spec/` → use rspec for spec/ directory

### Option 2: Require Explicit Configuration

1. **When both directories exist**, require a config file:
```toml
# .plur.toml
use = "rspec"  # or "minitest"

# Or define both and switch with --use
[task.rspec]
# ...

[task.minitest]
# ...
```

2. **Error clearly** when both exist without config:
```
Error: Both spec/ and test/ directories found.
Please specify which framework to use:
  - Run: plur --use=rspec
  - Run: plur --use=minitest
  - Or set 'use = "rspec"' in .plur.toml
```

### Option 3: Run Both Frameworks (Complex)

1. **Detect and run both** when both directories exist
2. **Aggregate results** from both frameworks
3. This adds significant complexity to the runner and output formatting

## Implementation Plan for Option 1

### Step 1: Add TestCmd
```go
type TestCmd struct {
    Patterns []string `arg:"" optional:"" help:"Test files or patterns to run (default: test/**/*_test.rb)"`
    Use      string   `short:"u" help:"Task configuration to use" default:""`
}

func (t *TestCmd) Run(parent *PlurCLI) error {
    // Set hint for framework detection
    parent.commandHint = "test"
    // Delegate to existing spec logic
    spec := SpecCmd{Patterns: t.Patterns, Use: t.Use}
    return spec.Run(parent)
}
```

### Step 2: Update PlurCLI struct
```go
type PlurCLI struct {
    Spec    SpecCmd  `cmd:"" help:"Run RSpec tests" default:"withargs"`
    Test    TestCmd  `cmd:"" help:"Run Minitest tests"`
    // ... other commands

    commandHint string // internal field for detection
}
```

### Step 3: Enhance DetectFramework
```go
func DetectFramework(hint string, patterns []string) *Task {
    // If explicit directory pattern given, use that
    if len(patterns) > 0 {
        for _, pattern := range patterns {
            if pattern == "test" || strings.HasPrefix(pattern, "test/") {
                if exists("test") {
                    return NewMinitestTask()
                }
            }
            if pattern == "spec" || strings.HasPrefix(pattern, "spec/") {
                if exists("spec") {
                    return NewRSpecTask()
                }
            }
        }
    }

    // Use command hint if available
    if hint == "test" && exists("test") {
        return NewMinitestTask()
    }
    if hint == "spec" && exists("spec") {
        return NewRSpecTask()
    }

    // Fall back to existing detection
    if exists("test") {
        return NewMinitestTask()
    }
    if exists("spec") {
        return NewRSpecTask()
    }

    return NewRSpecTask() // default
}
```

### Step 4: Update SpecCmd.Run()
```go
func (r *SpecCmd) Run(parent *PlurCLI) error {
    // ... existing priority logic ...
    if taskName == "" {
        // Pass hint and patterns to detection
        detectedTask := task.DetectFramework(parent.commandHint, r.Patterns)
        taskName = detectedTask.Name
    }
    // ...
}
```

## Current Test Coverage Review

### Existing Test Coverage

Based on review of the current test suite, here's what's already tested:

1. **Minitest Integration** (`spec/integration/plur_spec/minitest_integration_spec.rb`)
   - Tests minitest projects with `--use minitest` flag
   - Tests auto-detection for minitest projects (with only test/ directory)
   - Does NOT test behavior when both directories exist

2. **Configuration Tests** (`spec/integration/shared/configuration_integration_spec.rb`)
   - Tests minitest configuration via config file
   - Tests task switching with `--use` flag
   - Tests custom task definitions

3. **Go Unit Tests** (`plur/internal/task/task_test.go`)
   - Tests `DetectFramework()` with test/, spec/, and both directories
   - **Important**: Line 390-409 confirms that when both exist, minitest is preferred
   - Tests task building for both RSpec and Minitest

4. **Manual Testing Results** (from /tmp/plur-test-both)
   - `plur` → runs minitest (test/)
   - `plur spec` → **still runs minitest** (ignores command name!)
   - `plur test` → runs minitest (appears to work but is actually default behavior)
   - `plur spec spec/` → **FAILS** - looks for _test.rb files in spec/
   - `plur spec test/` → runs minitest files in test/
   - `plur --use=rspec` → correctly runs rspec

### Missing Test Coverage

The following scenarios are NOT currently tested in the integration suite:

1. **Projects with both spec/ and test/ directories**
   - No fixtures with both directories
   - No integration tests for this scenario

2. **Command-aware behavior**
   - No tests for `plur spec` expecting RSpec when both exist
   - No tests for `plur test` (if it existed) expecting Minitest

3. **Explicit directory patterns**
   - No tests for `plur spec test/` behavior
   - No tests for `plur test spec/` behavior (cross-directory execution)

4. **Error cases**
   - No test for the error when `plur spec spec/` can't find _test.rb files

## Testing Requirements

### Test Cases to Add

1. **Project with only spec/**:
   - `plur` → runs rspec
   - `plur spec` → runs rspec
   - `plur test` → error or runs nothing

2. **Project with only test/**:
   - `plur` → runs minitest
   - `plur spec` → error or runs nothing
   - `plur test` → runs minitest

3. **Project with both spec/ and test/**:
   - `plur` → runs minitest (current behavior, could change)
   - `plur spec` → runs rspec
   - `plur test` → runs minitest
   - `plur spec test/` → runs minitest files in test/
   - `plur test spec/` → runs rspec files in spec/

4. **With --use flag**:
   - `plur --use=rspec test/` → tries to find _spec.rb files in test/
   - `plur --use=minitest spec/` → tries to find _test.rb files in spec/

5. **With config file**:
   - Config with `use = "rspec"` makes rspec the default
   - Multiple task definitions can be switched with --use

## Migration Considerations

### Breaking Changes
- Adding `plur test` command is **additive**, not breaking
- Changing auto-detection when both directories exist **could be breaking**
  - Current: always picks minitest when both exist
  - Proposed: command-aware selection

### Backward Compatibility Path
1. **Phase 1**: Add `test` command without changing detection
2. **Phase 2**: Add warning when both directories exist without explicit config
3. **Phase 3**: Change detection to be command-aware (minor version bump)

## Updated Analysis Based on Test Coverage

### Key Findings

1. **`plur test` already works** - But it's accidental! Kong's `default:"withargs"` on SpecCmd means any unknown command gets treated as an argument to spec. So `plur test` is actually running `plur spec test`, which works by coincidence.

2. **Command name is completely ignored** - When both directories exist, `plur spec` still runs minitest because DetectFramework() doesn't receive any context about which command was used.

3. **Directory patterns partially work** - `plur spec test/` does run minitest files, but `plur spec spec/` fails because it looks for the wrong file suffix.

4. **No existing tests for mixed projects** - There are zero integration tests for projects with both spec/ and test/ directories.

## Refined Implementation Plan

Given the findings, here's the updated approach:

### Phase 1: Add Proper TestCmd (Quick Win)
1. Create explicit `TestCmd` struct (currently works by accident)
2. Pass command context to detection logic
3. Add integration tests for both directories scenario

### Phase 2: Fix Directory Pattern Behavior
1. When explicit directory is given, detect framework from directory name
2. Fix the issue where `plur spec spec/` fails

### Phase 3: Smart Command-Aware Detection
1. Modify DetectFramework to accept command hint
2. `plur spec` → prefers RSpec
3. `plur test` → prefers Minitest
4. Keep current behavior for bare `plur` (backward compatible)

## Conclusion

The current implementation has a fundamental limitation where it can only run one framework at a time and has a hard-coded preference for minitest when both directories exist. The solution requires:

1. **Add explicit `test` command** - Currently works accidentally, needs proper implementation
2. **Make framework detection command-aware** - Pass command context to detection
3. **Fix directory pattern detection** - Infer framework from explicit directory paths
4. **Add comprehensive tests** - No coverage exists for mixed directory projects

This provides the intuitive behavior the user expects while maintaining backward compatibility and the simplicity of the current single-task execution model.

### Priority Test Fixture Needed

Create `fixtures/projects/mixed-rspec-minitest/` with:
- `spec/` directory with RSpec tests
- `test/` directory with Minitest tests
- Integration tests for all command combinations
- Tests for explicit directory patterns