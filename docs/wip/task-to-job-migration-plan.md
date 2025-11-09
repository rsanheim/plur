# Task to Job Migration Plan

**Date:** 2025-11-09
**Author:** Claude Code (Collette)
**Status:** DRAFT - For Review (Simplified)

## Executive Summary

Consolidate the Task and Job concepts in plur into a single Job concept that handles both parallel test execution (`plur spec`) and watch mode (`plur watch`). This eliminates code duplication, simplifies configuration, and provides a unified way to specify commands for both modes.

## Current State

### Two Overlapping Concepts

**Task** (`internal/task/task.go`):
- Used for parallel test execution
- Contains: Name, Description, Run command, SourceDirs, TestGlob
- Framework-specific logic (RSpec, Minitest)
- Used by: `plur spec`, `plur test`

**Job** (`watch/job.go`):
- Used for watch mode command execution
- Contains: Name, Cmd array, Env variables
- Template-based with {{target}} substitution
- Used by: `plur watch`

### The Problem

1. **Duplication**: Both Task and Job define "how to run commands"
2. **Inconsistency**: Different configuration for watch vs spec modes
3. **Complexity**: Task merging logic, separate autodetection systems
4. **Maintenance**: Two parallel systems to maintain and test

## Proposed Solution: Unified Job Model

### New Job Structure

```go
type Job struct {
    Name          string   `toml:"-" json:"name"`
    Cmd           []string `toml:"cmd" json:"cmd"`
    Env           []string `toml:"env,omitempty" json:"env,omitempty"`
    TargetPattern string   `toml:"target_pattern,omitempty" json:"target_pattern,omitempty"` // NEW
}
```

### Key Design Decisions

1. **TargetPattern field**: Optional glob pattern for file discovery
   - Used by `plur spec` to find all files to process in parallel
   - Example: `"spec/**/*_spec.rb"` for RSpec tests
   - Example: `"**/*.go"` for Go files
   - Not needed for jobs that don't need file discovery (single commands)

2. **Parser selection**: Hardcoded based on job name
   - Job named "rspec" → use RSpec parser
   - Job named "minitest" → use Minitest parser
   - No exposed configuration needed

3. **No backward compatibility**: Clean break from Task
   - Remove all Task code completely
   - Users must migrate to Job configuration
   - Simpler codebase without legacy support

## Migration Phases

### Phase 1: Move Job Package and Extend with TargetPattern

**Goal**: Move Job out of watch package and extend it for general use.

**File consolidation**:
- Create `plur/job.go` with Job struct and BuildJobCmd
- Move WatchMapping/MultiString to `watch/processor.go`
- **Delete** `watch/job.go` entirely

**Changes**:
1. **Move Job to top-level package** (`plur/job.go`):
   - Move `Job` struct from `watch/job.go` to `plur/job.go`
   - Move `BuildJobCmd()` function to `plur/job.go`
   - Move `BuildJobAllCmd()` function to `plur/job.go`
   - Move `WatchMapping` and `MultiString` to `watch/processor.go` (already uses WatchMapping)
   - **Delete** `watch/job.go` entirely (no longer needed)
   - Update imports across codebase: `watch.Job` → `Job` (main package)

   **Rationale**:
   - Job is used by both watch and non-watch modes, so it shouldn't be in the watch package
   - WatchMapping is only used by EventProcessor, so keeping them together in `processor.go` makes sense
   - Eliminates an entire file (`watch/job.go`)

2. **Move WatchMapping and MultiString to processor.go** (`watch/processor.go`):
   - Move `WatchMapping` struct from `watch/job.go` to top of `watch/processor.go` (before EventProcessor definition)
   - Move `MultiString` type and `UnmarshalTOML` method to `watch/processor.go`
   - EventProcessor already uses WatchMapping, so they belong together

3. **Delete watch/job.go**:
   - File no longer needed after moving Job, WatchMapping, and MultiString

4. Add `TargetPattern` field to Job struct (`plur/job.go`)
   ```go
   type Job struct {
       Name          string   `toml:"-" json:"name"`
       Cmd           []string `toml:"cmd" json:"cmd"`
       Env           []string `toml:"env,omitempty" json:"env,omitempty"`
       TargetPattern string   `toml:"target_pattern,omitempty" json:"target_pattern,omitempty"`
   }
   ```

5. **Update BuildJobCmd signature** to accept multiple targets (`plur/job.go`):
   ```go
   func BuildJobCmd(job *Job, targets []string) []string {
       // Implementation as described in D2 above
   }
   ```

6. **Update watch.go call sites** to use new import and array:
   ```go
   // Change import from:
   "github.com/rsanheim/plur/watch"

   // To (if top-level):
   // Just use Job directly in main package (no import needed)

   // Change call from:
   cmd := watch.BuildJobCmd(job, relTarget)

   // To:
   cmd := BuildJobCmd(job, []string{relTarget})
   ```

7. Create passthrough parser (`plur/passthrough/parser.go`):
   ```go
   // Package passthrough provides a simple parser that forwards output unchanged
   type PassthroughParser struct{}

   func NewOutputParser() *PassthroughParser {
       return &PassthroughParser{}
   }

   // Implements TestOutputParser interface but just forwards output
   func (p *PassthroughParser) ProcessLine(line string) { ... }
   ```

8. Add helper methods to Job (`plur/job.go`):

   **Required imports for plur/job.go:**
   ```go
   import (
       "strings"

       "github.com/rsanheim/plur/minitest"
       "github.com/rsanheim/plur/passthrough"
       "github.com/rsanheim/plur/rspec"
       "github.com/rsanheim/plur/types"
   )
   ```

   **Helper methods:**
   ```go
   func (j *Job) GetTargetPattern() string {
       return j.TargetPattern
   }

   func (j *Job) GetTargetSuffix() string {
       // Extract suffix from pattern like "spec/**/*_spec.rb" → "_spec.rb"
       // Used by ExpandGlobPatterns when user passes directory: "spec/models" → "spec/models/**/*_spec.rb"
       pattern := j.TargetPattern
       if pattern == "" {
           return ""
       }
       lastStar := strings.LastIndex(pattern, "*")
       if lastStar == -1 {
           return ""
       }
       return pattern[lastStar+1:]
   }

   func (j *Job) CreateParser() (types.TestOutputParser, error) {
       switch j.Name {
       case "rspec":
           return rspec.NewOutputParser(), nil
       case "minitest":
           return minitest.NewOutputParser(), nil
       default:
           return passthrough.NewOutputParser(), nil
       }
   }

   func (j *Job) IsMinitestStyle() bool {
       return j.Name == "minitest"
   }
   ```

9. Update embedded defaults (`watch/defaults.toml`):
    ```toml
    [defaults.ruby.job.rspec]
    cmd = ["bundle", "exec", "rspec", "{{target}}"]
    target_pattern = "spec/**/*_spec.rb"

    [defaults.ruby.job.minitest]
    cmd = ["bundle", "exec", "ruby", "-Itest", "{{target}}"]
    target_pattern = "test/**/*_test.rb"

    [defaults.ruby.job.rake]
    cmd = ["bundle", "exec", "rake", "{{target}}"]
    # No target_pattern - not a parallel runner
    ```

**Tests**:
- Unit tests for Job in new location (ensure imports work)
- Unit tests for WatchMapping/MultiString in processor.go
- Unit tests for new Job methods
- Unit tests for passthrough parser
- Unit tests for BuildJobCmd with single and multiple targets
- Integration test for watch mode with new imports
- Verify watch/job.go is deleted and nothing imports it

### Phase 2: Add Job-Based File Discovery

**Goal**: Create file discovery functions that use Job instead of Task.

**Changes**:
1. Add Job-based discovery to `glob.go`:
   ```go
   func FindFilesFromJob(job *Job) ([]string, error) {
       pattern := job.GetTargetPattern()
       matches, err := doublestar.FilepathGlob(pattern)
       if err != nil {
           return nil, fmt.Errorf("error finding test files: %w", err)
       }
       return matches, nil
   }

   func ExpandPatternsFromJob(patterns []string, job *Job) ([]string, error) {
       // Same logic as ExpandGlobPatterns but uses job.GetTargetSuffix()
       // Handles directory expansion: "spec/models" + suffix "_spec.rb" → "spec/models/**/*_spec.rb"
       seenFiles := make(map[string]struct{})
       suffix := job.GetTargetSuffix()

       for _, pattern := range patterns {
           // Check if directory → expand with suffix
           // Check if file → validate suffix
           // Check if glob → expand directly
           // (Same logic as current ExpandGlobPatterns)
       }

       return allFiles, nil
   }
   ```

2. Keep existing Task-based functions temporarily

**Tests**:
- Unit tests for FindFilesFromJob
- Unit tests for ExpandPatternsFromJob with directories, files, and globs

### Phase 3: Switch SpecCmd to Use Job

**Goal**: Migrate `plur spec` command to use Job instead of Task.

**Changes**:
1. **Load Job instead of Task** (`main.go`):
   ```go
   // Remove task detection/loading
   job := getJobFromConfigOrDefaults(parent.Jobs, parent.Use)
   ```

2. **Update file discovery**:
   ```go
   testFiles := FindFilesFromJob(job)
   ```

3. **Pass Job to execution**:
   ```go
   executor := &TestExecutor{currentJob: job}
   ```

4. **Update TestExecutor** (`execution.go`):
   ```go
   type TestExecutor struct {
       currentJob *Job  // Use Job instead of Task (Job is now in main package)
       // Remove currentTask
   }
   ```

5. **Update RunTestFiles** (`runner.go`):
   ```go
   func RunTestFiles(..., job *Job) WorkerResult {
       parser := job.CreateParser()

       // Build command using framework-specific wrapper or BuildJobCmd directly
       var args []string
       switch job.Name {
       case "rspec":
           args = buildRSpecCommand(job, testFiles, globalConfig)
       case "minitest":
           args = buildMinitestCommand(job, testFiles, globalConfig)
       default:
           args = BuildJobCmd(job, testFiles)
       }

       // Execute command with args
       cmd := exec.CommandContext(ctx, args[0], args[1:]...)
       // ... rest of execution
   }
   ```

**Tests**: Integration tests for `plur spec` with Job

### Phase 4: Remove Task Dependencies

**Goal**: Remove all Task usage from codebase.

**Changes**:
1. **Delete Task-based functions**:
   - Remove `FindTestFiles(task)` from `glob.go`
   - Remove `ExpandGlobPatterns(..., task)` from `glob.go`

2. **Update result formatting** (`result.go`):
   - Use `job.IsMinitestStyle()` instead of `task.IsMinitestStyle()`

3. **Remove Task configuration** (`main.go`):
   - Remove `TaskConfig` field from `PlurCLI`
   - Remove `mergeTaskConfig()` function
   - Remove `getTaskWithOverrides()` function
   - Remove `validateTaskExists()` function
   - Only parse `[job.*]` sections

4. **Replace framework-specific command building with wrapper logic**:
   - Delete `Task.BuildCommand()` (replaced by `BuildJobCmd` + wrappers)
   - Create `buildRSpecCommand(job, files, globalConfig)` in runner.go:
     ```go
     func buildRSpecCommand(job *Job, files []string, globalConfig *config.GlobalConfig) []string {
         // Start with base command from BuildJobCmd
         args := BuildJobCmd(job, files)

         // Add formatter if available
         if globalConfig.ConfigPaths != nil {
             formatterPath := globalConfig.ConfigPaths.GetJSONRowsFormatterPath()
             if formatterPath != "" {
                 // Insert before files
                 args = insertBeforeFiles(args, files, "-r", formatterPath, "--format", "Plur::JsonRowsFormatter")
             }
         }

         // Add color flags
         if !globalConfig.ColorOutput {
             args = insertBeforeFiles(args, files, "--no-color")
         } else {
             args = insertBeforeFiles(args, files, "--force-color", "--tty")
         }

         return args
     }
     ```

   - Create `buildMinitestCommand(job, files, globalConfig)` in runner.go:
     ```go
     func buildMinitestCommand(job *Job, files []string, globalConfig *config.GlobalConfig) []string {
         // For multiple files, use special -e require pattern
         if len(files) > 1 {
             cmd := []string{"bundle", "exec", "ruby", "-Itest"}
             requires := []string{}
             for _, file := range files {
                 testFile := strings.TrimPrefix(file, "test/")
                 testFile = strings.TrimSuffix(testFile, ".rb")
                 requires = append(requires, testFile)
             }
             requireList := `"` + strings.Join(requires, `", "`) + `"`
             cmd = append(cmd, "-e", `[`+requireList+`].each { |f| require f }`)
             return cmd
         }
         // For single file, use BuildJobCmd directly
         return BuildJobCmd(job, files)
     }
     ```

   - Update `RunTestFiles` to use appropriate builder based on job name

   **Rationale**: Keeps framework-specific logic isolated in runner.go, similar to current Task.BuildCommand approach. Custom user jobs can define flags directly in Job.Cmd.

**Tests**: Ensure framework-specific commands work correctly

### Phase 5: Delete Task Package

**Goal**: Remove task package entirely.

**Changes**:
1. **Delete** `plur/internal/task/` directory and all files
2. **Remove** all Task imports from codebase
3. **Verify** no references remain via grep/search

**Tests**: Full test suite passes without task package

### Phase 6: Unify Autodetection

**Goal**: Single autodetection system for both spec and watch modes.

**Changes**:
1. **Use watch/defaults.go** for all autodetection
2. **Remove** duplicate framework detection logic
3. **Simplify SpecCmd** - just call `GetAutodetectedDefaults()`

**Tests**: Autodetection works for both modes

### Phase 7: Add Database Support (Optional Enhancement)

**Goal**: Use rake job for database commands.

**Changes**:
1. Update database commands to use rake Job
2. Pass rake tasks as targets: `"db:create"`, `"db:migrate"`

**Tests**: Database commands work with Job

## Implementation Strategy

### Step-by-Step Approach

1. **Phase 1** - Add TargetPattern to Job (low risk, additive only)
2. **Phase 2** - Create Job-based file discovery functions
3. **Phase 3** - Switch SpecCmd to use Job (the big migration)
4. **Phase 4** - Remove all Task dependencies from codebase
5. **Phase 5** - Delete task package entirely
6. **Phase 6** - Unify autodetection
7. **Phase 7** - (Optional) Add database support

### Testing Strategy

- **Unit tests**: For each new Job method and file discovery function
- **Integration tests**: Full `plur spec` execution with Job
- **Regression tests**: Ensure all existing functionality works
- **Fixture tests**: Test with default-ruby and default-rails projects
- **Manual testing**: Verify autodetection and custom configurations

## Configuration Examples

### Current (Task-based)

```toml
# .plur.toml - Current format with Tasks
[task.rspec]
run = "bin/rspec"
source_dirs = ["spec", "lib", "app"]
test_glob = "spec/**/*_spec.rb"

[[watch]]
source = "lib/**/*.rb"
targets = ["spec/{{dir_relative}}/{{name}}_spec.rb"]
jobs = "rspec"

[job.rspec]  # Separate job definition for watch
cmd = ["bin/rspec", "{{target}}"]
```

**Problems**: Two configurations, task vs job, duplication

### After Migration (Job-only)

```toml
# .plur.toml - Unified Job format
[job.rspec]
cmd = ["bin/rspec", "{{target}}"]
target_pattern = "spec/**/*_spec.rb"

[[watch]]
source = "lib/**/*.rb"
targets = ["spec/{{dir_relative}}/{{name}}_spec.rb"]
jobs = "rspec"
```

**Benefits**:
1. **Single job definition** works for both `plur spec` and `plur watch`
2. **No task section** needed
3. **No merge behavior** - simpler config model
4. **Consistent** across all commands

### Example: Custom Test Runner

```toml
[job.custom-test]
cmd = ["bin/custom-runner", "{{target}}"]
target_pattern = "tests/**/*_test.js"

[[watch]]
source = "src/**/*.js"
targets = ["tests/{{name}}_test.js"]
jobs = "custom-test"
```

### Example: Non-Test Job (No TargetPattern)

```toml
[job.lint]
cmd = ["bundle", "exec", "rubocop"]
# No target_pattern - runs as single command, not parallel

[[watch]]
source = "**/*.rb"
jobs = "lint"
```

## Success Metrics

### Code Quality
- [ ] ~226 lines removed (`internal/task/` directory deleted)
- [ ] No Task references in codebase
- [ ] Single autodetection system (watch/defaults.go)
- [ ] Simplified configuration parsing (no task merging)
- [ ] Reduced test complexity (no dual paths)

### Functionality
- [ ] All existing tests pass
- [ ] `plur spec` works with Jobs
- [ ] `plur watch` continues to work
- [ ] Custom jobs can be defined
- [ ] Autodetection provides correct defaults
- [ ] RSpec and Minitest parsers work correctly

### User Experience
- [ ] Simpler configuration model (single [job.*] section)
- [ ] Unified behavior between spec and watch modes
- [ ] No backward compatibility burden

## No Backward Compatibility

**Important**: This migration completely removes Task support with no fallback or migration path. The only user is the project maintainer, so:
- No deprecation warnings
- No support for `[task.*]` configs
- No migration helpers or documentation
- Clean break - Task is gone, Job is the only way

If old Task configs exist, they will simply not work. This is intentional and acceptable.

---

## Risks and Mitigations

### Risk 1: Parser Selection for Custom Jobs

**Impact**: Custom job names (not "rspec" or "minitest") won't get parsed output

**Mitigation**:
- Use passthrough parser for custom jobs (pipes output directly)
- Output will be raw/interleaved but this is acceptable for custom frameworks
- RSpec and Minitest users get formatted output as expected

### Risk 2: Framework-Specific Command Building

**Impact**: RSpec needs formatter/color flags, Minitest needs special require pattern

**Mitigation**:
- Wrap BuildJobCmd with framework-specific functions in runner.go
- `buildRSpecCommand()` adds formatter and color flags before files
- `buildMinitestCommand()` handles `-e` require pattern for multiple files
- Custom jobs define flags directly in Job.Cmd template

### Risk 3: Test Coverage During Migration

**Impact**: Temporary dual-path code could hide bugs

**Mitigation**:
- Phase 2 adds Job-based path with comprehensive unit tests
- Phase 3 switches only after validating Job path works
- Keep fixture project integration tests running throughout all phases
- Each phase includes explicit test verification before proceeding

## Design Decisions

### D1: Parser Selection - Passthrough for Custom Jobs

**Decision**: Use passthrough parser for non-rspec/minitest jobs

```go
func (j *Job) CreateParser() (types.TestOutputParser, error) {
    switch j.Name {
    case "rspec":
        return rspec.NewOutputParser(), nil
    case "minitest":
        return minitest.NewOutputParser(), nil
    default:
        return passthrough.NewOutputParser(), nil  // New: passthrough parser
    }
}
```

**Passthrough parser**: Pipes process output directly to STDOUT without parsing
- Output will be interleaved and potentially messy
- Acceptable tradeoff for custom test frameworks
- Users with rspec/minitest get formatted output
- Users with custom frameworks get raw output

**Implementation**: Create new `passthrough` parser package that implements `TestOutputParser` interface but just forwards output.

### D2: Command Building - Use BuildJobCmd for Both Parallel and Watch

**Decision**: Use `BuildJobCmd` for both parallel execution and watch mode

**Current state**:
- `BuildJobCmd(job, target string)` - watch mode, single file
- `Task.BuildCommand(files []string)` - parallel mode, multiple files

**Proposed**:
- Change signature: `BuildJobCmd(job *Job, targets []string)`
- Works for both modes:
  - Parallel execution: `BuildJobCmd(job, []string{"spec/foo.rb", "spec/bar.rb", "spec/baz.rb"})`
  - Watch mode: `BuildJobCmd(job, []string{"spec/foo.rb"})`

**Implementation**:
```go
func BuildJobCmd(job *Job, targets []string) []string {
    // Find and replace {{target}} token with all target files
    result := []string{}
    foundToken := false

    for _, part := range job.Cmd {
        if part == "{{target}}" {
            // Replace entire {{target}} element with all targets
            result = append(result, targets...)
            foundToken = true
        } else if strings.Contains(part, "{{target}}") {
            // Token is part of a string (e.g., "--file={{target}}")
            // Expand to multiple args: ["--file=spec/foo.rb", "--file=spec/bar.rb"]
            for _, target := range targets {
                result = append(result, strings.ReplaceAll(part, "{{target}}", target))
            }
            foundToken = true
        } else {
            result = append(result, part)
        }
    }

    // If no {{target}} token, append all targets at end
    if !foundToken {
        result = append(result, targets...)
    }

    return result
}
```

**Examples**:
```go
// RSpec with multiple files
job.Cmd = ["bundle", "exec", "rspec", "{{target}}"]
BuildJobCmd(job, []string{"spec/foo.rb", "spec/bar.rb"})
// → ["bundle", "exec", "rspec", "spec/foo.rb", "spec/bar.rb"]

// Watch mode with single file
BuildJobCmd(job, []string{"spec/foo.rb"})
// → ["bundle", "exec", "rspec", "spec/foo.rb"]

// Custom flag pattern
job.Cmd = ["my-runner", "--file={{target}}"]
BuildJobCmd(job, []string{"test1.rb", "test2.rb"})
// → ["my-runner", "--file=test1.rb", "--file=test2.rb"]
```

**Rationale**:
- Single unified command building function
- Template-based approach handles common cases
- Works seamlessly for both parallel and watch modes

**Framework-specific args**: RSpec formatter/color flags and Minitest require patterns are handled by wrapping `BuildJobCmd` with framework-specific functions in runner.go. This keeps the special logic isolated while allowing custom user jobs to define their own flags directly in `Job.Cmd`. See Phase 4 for implementation details.

### D3: TargetPattern - Single Pattern String

**Decision**: TargetPattern is a single string, not array

```go
TargetPattern string  `toml:"target_pattern,omitempty"`
```

**Rationale**:
- Sufficient for common cases
- Projects needing multiple patterns can define separate jobs
- Simpler configuration and implementation
- Can revisit if user demand warrants it

**Example with multiple jobs**:
```toml
[job.rspec]
cmd = ["bundle", "exec", "rspec", "{{target}}"]
target_pattern = "spec/**/*_spec.rb"

[job.minitest]
cmd = ["bundle", "exec", "ruby", "-Itest", "{{target}}"]
target_pattern = "test/**/*_test.rb"
```

### D4: Autodetection - Check spec/ First, Then test/

**Decision**: Same logic as watch/defaults.go

**Precedence**:
1. Check for `spec/` directory → use rspec job
2. Check for `test/` directory → use minitest job
3. Default to rspec

**No warnings**: Don't warn when both exist, just follow documented precedence

**Implementation**: Reuse existing autodetection from `watch/defaults.go` - already has this logic

## Appendix: Key Code Locations

### Files to Modify

**Phase 1** (Move Job and extend):
- `plur/job.go` - NEW: Move Job struct and BuildJobCmd here
- `plur/watch/processor.go` - Move WatchMapping and MultiString here (from job.go)
- `plur/watch/job.go` - DELETE: Entire file removed
- `plur/watch.go` - Update imports to use Job from main package
- `plur/watch_find.go` - Update imports
- `plur/watch/defaults.toml` - Add target_pattern to job configs
- `plur/job_test.go` - NEW: Tests for Job
- `plur/passthrough/parser.go` - NEW: Create passthrough parser
- `plur/passthrough/parser_test.go` - NEW: Test passthrough parser

**Phase 2** (File discovery):
- `plur/glob.go` - Add FindFilesFromJob(), ExpandPatternsFromJob()

**Phase 3** (Switch SpecCmd):
- `plur/main.go` - Load Job, remove task logic
- `plur/execution.go` - Use Job instead of Task
- `plur/runner.go` - Accept Job parameter, add framework-specific wrappers

**Phase 4** (Remove Task deps):
- `plur/glob.go` - Remove Task functions
- `plur/result.go` - Use Job
- `plur/main.go` - Remove task config parsing
- `plur/runner.go` - Remove Task.BuildCommand calls

**Phase 5** (Delete):
- `plur/internal/task/` - Delete entire directory

**Phase 6** (Unify autodetection):
- `plur/main.go` - Use watch/defaults.go for autodetection
- Remove duplicate detection logic

### Current Task Usage (To Replace)

**Task is used in**:
- `main.go` - SpecCmd loads and validates Task
- `glob.go` - FindTestFiles(task), ExpandGlobPatterns(..., task)
- `execution.go` - TestExecutor.currentTask
- `runner.go` - RunTestFiles(..., task)
- `result.go` - task.IsMinitestStyle()
- `watch.go` - task.GetWatchDirs() (can use watch mappings instead)

**All will be replaced with Job equivalents**

### File Discovery Flow

**Before (Task)**:
```
task.TestGlob = "spec/**/*_spec.rb"
  ↓
FindTestFiles(task)
  ↓
doublestar.FilepathGlob(task.TestGlob)
  ↓
[list of files]
  ↓
Group by runtime/size → Distribute to workers
```

**After (Job)**:
```
job.TargetPattern = "spec/**/*_spec.rb"
  ↓
FindFilesFromJob(job)
  ↓
doublestar.FilepathGlob(job.TargetPattern)
  ↓
[list of files]
  ↓
Group by runtime/size → Distribute to workers
```

Same mechanism, different source.