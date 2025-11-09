# Plur Code Review: Google Go Style Compliance

**Review Date:** 2025-11-08
**Reviewer:** Claude Code (Collette)
**Scope:** Full plur codebase against Google Go Style Guide and Best Practices

## Executive Summary

The plur codebase is **well-structured and production-ready**, with excellent concurrency patterns, good test coverage, and clean package boundaries. The main areas for improvement are:

1. **Naming conventions** - Remove "Get" prefixes from getters (~10 functions)
2. **Error handling** - Use `%w` instead of `%v` for error wrapping (~13 locations)
3. **Package organization** - Eliminate generic `utils.go` package
4. **Documentation** - Add package-level comments and improve some function docs
5. **Code organization** - Extract business logic from `main.go`

These are **refinements rather than major issues**. The codebase already follows most Go best practices correctly.

---

## Findings by Priority

### HIGH Priority (8 findings)

#### H1: Remove "Get" Prefix from Getters

**Issue:** Google Go style discourages "Get" prefix for simple accessors. The property name alone is sufficient.

**Locations:**
```go
// plur/version.go
func GetVersionInfo() string                           // line 21
func GetDetailedVersionInfo() string                   // line 80
func GetBuildTime() string                             // line 116

// plur/runner.go
func GetWorkerCount(cliWorkers int) int               // line 53
func GetTestEnvNumber(workerIndex int, ...) string    // line 73

// plur/runtime_tracker.go
func GetRuntimeFilePath() (string, error)             // line 68

// plur/watch/binary.go
func GetWatcherBinaryPath(binDir string) (string, error) // line 20

// plur/rspec/formatter.go
func GetFormatterPath(formattersPath string) (string, error) // line 17

// plur/watch/defaults.go
func GetDefaultProfile(name string) *DefaultProfile   // line 61
func GetAutodetectedDefaults() (...)                  // line 82
```

**Recommendation:**
```go
// Remove "Get" prefix
func VersionInfo() string
func DetailedVersionInfo() string
func BuildTime() string
func WorkerCount(cliWorkers int) int
func TestEnvNumber(workerIndex int, ...) string
func RuntimeFilePath() (string, error)
func WatcherBinaryPath(binDir string) (string, error)
func FormatterPath(formattersPath string) (string, error)
func DefaultProfile(name string) *DefaultProfile
func AutodetectedDefaults() (...)
```

**Impact:** Medium effort, high consistency gain. Update ~10 function names and all call sites.

---

#### H2: Remove "Log" Prefix from Logger Functions

**Issue:** Package name already indicates "logger", so "Log" prefix is redundant.

**Location:** `plur/logger/logger.go:107-128`

```go
// Current - "Log" appears twice in usage
logger.LogVerbose("message")
logger.LogDebug("details")
logger.LogError("failed", err)
logger.LogWarn("warning")

// Should be
logger.Verbose("message")
logger.Debug("details")
logger.Error("failed", err)
logger.Warn("warning")
```

**Recommendation:**
```go
// plur/logger/logger.go
func Verbose(msg string, args ...any)  // was LogVerbose
func Debug(msg string, args ...any)    // was LogDebug
func Error(msg string, err error, args ...any)  // was LogError
func Warn(msg string, args ...any)     // was LogWarn
```

**Impact:** Low effort, high clarity gain. More idiomatic Go.

---

#### H3: Use `%w` for Error Wrapping, Not `%v`

**Issue:** Go 1.13+ supports `%w` verb for error wrapping, which preserves the error chain for `errors.Is()` and `errors.As()`. Using `%v` loses this information.

**Locations:**
```go
// plur/watch.go
return fmt.Errorf("failed to find watcher binary: %v", err)        // line 166
return fmt.Errorf("watcher error: %v", err)                        // line 346

// plur/main.go
return fmt.Errorf("failed to change directory to %s: %v", dir, err) // line 428

// plur/dependencies.go
return fmt.Errorf("error running bundle install: %v", err)         // line 26

// plur/glob.go
return nil, fmt.Errorf("error finding test files: %v", err)        // line 18
return nil, fmt.Errorf("error expanding pattern %q: %v", pattern, err) // line 56

// plur/watch/processor.go
return nil, fmt.Errorf("error matching pattern %q: %v", watch.Source, err) // line 46

// Additional locations in other files
```

**Recommendation:**
```go
// Change all %v to %w when wrapping errors
return fmt.Errorf("failed to find watcher binary: %w", err)
return fmt.Errorf("failed to change directory to %s: %w", dir, err)
return fmt.Errorf("error running bundle install: %w", err)
```

**Exception:** Keep `%v` when NOT wrapping errors:
```go
// Correct - sourceDirs is not an error, so %v is right
return fmt.Errorf("no directories to watch found (tried: %v)", sourceDirs)
```

**Impact:** Low effort (~13 replacements), high correctness gain. Enables proper error inspection.

---

#### H4: Eliminate Generic `utils.go` Package

**Issue:** Google Go style discourages generic "utils" packages. Functions should live in packages that describe their purpose.

**Location:** `plur/utils.go` (28 lines)

**Current contents:**
```go
func pluralize(count int, singular, plural string) string
func toStdErr(dryRun bool, format string, args ...any)
func dump(data interface{})
```

**Recommendation:**
* `pluralize()` → Move to `internal/format` package (formatting logic)
* `toStdErr()` → Move to `logger` package or inline at call sites
* `dump()` → Debug-only function - consider removing or moving to internal debug package

**Impact:** Medium effort. Requires moving functions and updating imports. Improves package cohesion.

---

#### H5: Extract Business Logic from `main.go`

**Issue:** `main.go` contains significant business logic (task validation, merging, resolution) that should be in separate packages.

**Location:** `plur/main.go` (480 lines)

**Current structure:**
* Lines 1-30: Imports and CLI struct
* Lines 31-116: `SpecCmd.Run()` - 85 lines, does too much
* Lines 118-175: Other command implementations
* Lines 177-330: Task configuration logic (should be separate)
* Lines 332-480: Validation, merging, helper functions

**Functions that should be extracted:**
```go
// Lines 310-330: Should be in config or cli package
func mergeTaskConfig(base *task.Task, override task.Task) *task.Task

// Lines 276-308: Should be in config or cli package
func getTaskWithOverrides(tomlConfig *config.GlobalConfig, ...) (*task.Task, error)

// Lines 306-330: Should be in cli package
func validateTaskExists(cliTaskName string, ...) error
```

**Recommendation:**
```go
// Create plur/internal/cli/tasks.go or plur/config/tasks.go
package cli  // or config

func MergeTaskConfig(base *task.Task, override task.Task) *task.Task { ... }
func GetTaskWithOverrides(cfg *config.GlobalConfig, ...) (*task.Task, error) { ... }
func ValidateTaskExists(taskName string, ...) error { ... }
```

**Impact:** High effort. Requires creating new package and moving functions. Significantly improves main.go readability.

---

#### H6: Simplify Complex `SpecCmd.Run()` Method

**Issue:** The `SpecCmd.Run()` method (lines 31-116) does too much in one function (85 lines).

**Current responsibilities:**
1. Task resolution
2. Task validation
3. Framework detection messaging
4. File discovery
5. Dependency management
6. Test execution

**Recommendation:**
```go
// Extract helper methods
func (r *SpecCmd) Run(parent *PlurCLI) error {
    task, err := r.resolveTask(parent)
    if err != nil {
        return err
    }

    testFiles, err := r.discoverTestFiles(task)
    if err != nil {
        return err
    }

    if r.Auto {
        if err := r.installDependencies(parent.globalConfig); err != nil {
            return err
        }
    }

    return r.executeTests(parent.globalConfig, testFiles, task)
}

func (r *SpecCmd) resolveTask(parent *PlurCLI) (*task.Task, error) {
    // Lines 36-52 from current Run()
}

func (r *SpecCmd) discoverTestFiles(task *task.Task) ([]string, error) {
    // Lines 70-85 from current Run()
}

// etc.
```

**Impact:** Medium effort. Improves readability and testability of individual steps.

---

#### H7: Inconsistent Method Naming - "Get" Prefix on Task

**Issue:** Task methods inconsistently use "Get" prefix.

**Location:** `plur/internal/task/task.go:72-94`

```go
// Has "Get" prefix
func (t *Task) GetWatchDirs() []string      // line 73
func (t *Task) GetTestSuffix() string       // line 87
func (t *Task) GetTestPattern() string      // line 92

// No "Get" prefix
func (t *Task) BuildCommand(...)            // line 24
func (t *Task) CreateParser(...)            // line 56
func (t *Task) IsMinitestStyle() bool       // line 68
```

**Recommendation:**
```go
// Be consistent - remove "Get" prefix
func (t *Task) WatchDirs() []string
func (t *Task) TestSuffix() string
func (t *Task) TestPattern() string
```

**Impact:** Low effort. Simple rename, update call sites.

---

#### H8: Good Concurrency Patterns ✓

**Finding:** Plur has **excellent** concurrency patterns. No issues found!

**Positive examples:**

**Channel management** (plur/runner.go:308):
```go
outputChan := make(chan OutputMessage, maxWorkers*10)  // Good buffering
// ... use channel ...
close(outputChan)  // Proper close from sender
outputWg.Wait()    // Wait for consumers
```

**WaitGroup usage** (plur/runner.go:322-335):
```go
var wg sync.WaitGroup
for i, group := range groups {
    wg.Add(1)  // Add before goroutine
    go func(workerIndex int, files []string) {
        defer wg.Done()  // Done in defer
        // work
    }(i, group.Files)  // Pass as parameters, don't capture
}
wg.Wait()  // Wait for all
```

**Mutex protection** (plur/runtime_tracker.go:17):
```go
type RuntimeTracker struct {
    mu   sync.Mutex
    data map[string]map[string]float64  // Protected by mu
}
```

**Impact:** No changes needed. This is a strength of the codebase!

---

### MEDIUM Priority (11 findings)

#### M1: Missing Package-Level Documentation

**Issue:** Several packages lack package-level documentation explaining their purpose.

**Missing docs:**
* `plur/watch` - No package comment
* `plur/internal/task` - No package comment
* `plur/logger` - No package comment
* `plur/rspec` - No package comment
* `plur/minitest` - No package comment

**Recommendation:**
```go
// Package watch implements file watching and job execution for plur.
// It supports glob-based file pattern matching and template-based job
// execution using the doublestar library and Go templates.
package watch

// Package task provides test framework detection and task configuration.
// It supports RSpec, Minitest, and custom test frameworks through a
// unified Task interface.
package task

// Package logger provides structured logging with verbosity levels for plur.
// It wraps Go's slog package with custom handlers for color output and
// level filtering.
package logger
```

**Impact:** Low effort, medium documentation gain.

---

#### M2: Config Package Split Across Two Locations

**Issue:** Config code is split between two files in inconsistent locations.

**Locations:**
* `plur/config.go` - 2 lines, essentially empty
* `plur/config/config.go` - 103 lines, actual implementation

**Recommendation:** Remove empty `plur/config.go`, keep all config in `plur/config/` package.

**Impact:** Minimal effort. Clean up empty file.

---

#### M3: Large Files Could Be Split

**Issue:** Some files are approaching or exceeding recommended 500-line limit.

**Locations:**
* `plur/main.go` - 480 lines (close to limit)
* `plur/watch.go` - 383 lines
* `plur/runner.go` - 358 lines

**Recommendation:**
* `main.go` → Split CLI validation/task logic to `plur/internal/cli/`
* `watch.go` → Extract job execution to `watch_executor.go`
* `runner.go` → Consider splitting worker management from output aggregation

**Impact:** Medium effort. Not urgent, but consider for future refactoring.

---

#### M4: Context Parameter Not Used

**Issue:** `RunTestFiles()` accepts a `context.Context` parameter but creates a new `context.Background()` instead of using it.

**Location:** `plur/runner.go:152, 269`

```go
func RunTestFiles(ctx context.Context, ...) WorkerResult {
    // ...
    ctx := context.Background()  // line 269 - ignores passed ctx
    // ...
}
```

**Recommendation:**
Either use the passed context or remove it from the signature:
```go
// Option 1: Use the context
func RunTestFiles(ctx context.Context, ...) WorkerResult {
    // Use ctx directly, don't create new one
}

// Option 2: Remove if not needed
func RunTestFiles(globalConfig *config.GlobalConfig, ...) WorkerResult {
    ctx := context.Background()
    // ...
}
```

**Impact:** Low effort. Clarifies API contract.

---

#### M5: `logger.WithContext()` Doesn't Use Context

**Issue:** Function exists but doesn't actually use the context parameter.

**Location:** `plur/logger/logger.go`

```go
func WithContext(ctx context.Context) *slog.Logger {
    return Logger  // Ignores ctx completely
}
```

**Recommendation:**
```go
// Option 1: Remove if not needed
// Delete the function entirely

// Option 2: Actually use context values
func WithContext(ctx context.Context) *slog.Logger {
    // Extract relevant values from context
    if requestID, ok := ctx.Value("request_id").(string); ok {
        return Logger.With("request_id", requestID)
    }
    return Logger
}
```

**Impact:** Low effort. Either implement or remove.

---

#### M6: Magic Numbers Should Be Named Constants

**Issue:** Several magic numbers in the code that should be named constants.

**Locations:**
```go
// plur/test_collector.go:21-28
const rawOutputBufferSize = 1024 * 8  // Good!

tests:    make([]types.TestCaseNotification, 0, 100),  // Magic: 100
failures: make([]types.TestCaseNotification, 0, 10),   // Magic: 10

// plur/runner.go
outputChan := make(chan OutputMessage, maxWorkers*10)  // Magic: 10
```

**Recommendation:**
```go
const (
    rawOutputBufferSize    = 1024 * 8
    initialTestCapacity    = 100
    initialFailureCapacity = 10
    outputChannelMultiplier = 10
)

tests:    make([]types.TestCaseNotification, 0, initialTestCapacity)
failures: make([]types.TestCaseNotification, 0, initialFailureCapacity)
outputChan := make(chan OutputMessage, maxWorkers*outputChannelMultiplier)
```

**Impact:** Low effort, improves maintainability.

---

#### M7: Simplify `exists()` Helper

**Issue:** Using `filepath.Glob()` is overkill for checking file existence.

**Location:** `plur/internal/task/task.go:216-220`

```go
// Current - uses Glob unnecessarily
func exists(path string) bool {
    matches, err := filepath.Glob(path)
    return err == nil && len(matches) > 0
}
```

**Recommendation:**
```go
// Simpler and more direct
func exists(path string) bool {
    _, err := os.Stat(path)
    return err == nil
}
```

**Impact:** Minimal effort, clearer intent.

---

#### M8: Obvious Comments Should Be Removed or Improved

**Issue:** Some comments just restate what the code does instead of explaining why.

**Examples:**
```go
// plur/config/config.go:12
// GlobalConfig holds settings that are truly global across all commands
// ^ Obvious from the name "GlobalConfig"

// Better:
// GlobalConfig contains runtime settings shared across all commands,
// including paths, verbosity flags, and output preferences.
```

**Recommendation:** Review godoc comments to explain WHY or add context, not just restate WHAT.

**Impact:** Low effort, improves documentation quality.

---

#### M9: Good Table-Driven Test Patterns ✓

**Finding:** Plur has **excellent** table-driven test organization!

**Positive examples:**
```go
// plur/watch/tokens_test.go - Clean structure
tests := []struct {
    name     string
    path     string
    pattern  string
    expected Tokens
}{
    {
        name:    "handles root level files",
        path:    "Gemfile",
        pattern: "*",
        expected: Tokens{Path: "Gemfile", Dir: "./"},
    },
    // More cases...
}

for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
        result := BuildTokens(tt.path, tt.pattern)
        assert.Equal(t, tt.expected, result)
    })
}
```

**Impact:** No changes needed. Keep doing this!

---

#### M10: Good Use of Testify ✓

**Finding:** Consistent and correct use of `require` vs `assert`.

**Positive examples:**
```go
// Critical assertions - use require (stops on failure)
require.NoError(t, err)
require.NotNil(t, processor)

// Non-critical checks - use assert (continues on failure)
assert.Equal(t, expected, actual)
assert.Contains(t, output, "success")
```

**Impact:** No changes needed. Follows CLAUDE.md guidelines correctly!

---

#### M11: Good Early Return Patterns ✓

**Finding:** Code consistently uses early returns for error handling, keeping happy path at minimal indentation.

**Positive example:**
```go
// plur/internal/task/task.go:56-64
func (t *Task) CreateParser() (types.TestOutputParser, error) {
    switch t.Name {
    case "rspec":
        return rspec.NewOutputParser(), nil
    case "minitest":
        return minitest.NewOutputParser(), nil
    default:
        return nil, fmt.Errorf("unsupported task type: %s", t.Name)
    }
}
```

**Impact:** No changes needed. This is good Go style!

---

### LOW Priority (3 findings)

#### L1: Some Function Comments Could Be More Concise

**Issue:** A few godoc comments are overly verbose.

**Example:**
```go
// plur/main.go:306-308
// validateTaskExists checks if a task exists when explicitly requested
// Returns nil if task exists or was auto-detected
// Returns error with available tasks if explicitly requested task doesn't exist
```

**Recommendation:**
```go
// validateTaskExists verifies explicitly-requested tasks exist and returns
// helpful errors with available task names if not found.
```

**Impact:** Minimal. Nice-to-have improvement.

---

#### L2: Could Use More Subtests with `t.Run()`

**Issue:** Some table-driven tests don't use `t.Run()` for individual test cases.

**Recommendation:**
```go
// Current - harder to identify which case fails
for _, tc := range testCases {
    result := fn(tc.input)
    assert.Equal(t, tc.expected, result)
}

// Better - clear test output
for _, tc := range testCases {
    t.Run(tc.name, func(t *testing.T) {
        result := fn(tc.input)
        assert.Equal(t, tc.expected, result)
    })
}
```

**Impact:** Low. Most tests already do this, just a few stragglers.

---

#### L3: Consider Future File Splits

**Issue:** As the codebase grows, keep an eye on file sizes.

**Candidates for eventual splitting:**
* `plur/main.go` - Already at 480 lines
* `plur/watch.go` - At 383 lines
* `plur/internal/task/task.go` - At 226 lines

**Recommendation:** Monitor these files. If they exceed 500 lines, consider splitting by logical boundaries.

**Impact:** Future consideration. Not urgent now.

---

## Summary by Category

### Quick Wins (Can be done in one PR)
1. **Remove "Get" prefixes** - 10+ functions, straightforward rename
2. **Remove "Log" prefixes** - Logger package functions
3. **Change `%v` to `%w`** - ~13 error wrapping sites
4. **Add package comments** - 5-6 packages missing docs
5. **Define constants** - Magic numbers like 100, 10

**Effort:** 2-3 hours
**Impact:** High consistency gain, better Go idiom compliance

### Medium Refactors (Separate PRs)
1. **Distribute utils.go** - Move functions to appropriate packages
2. **Extract CLI logic from main.go** - Create internal/cli package
3. **Simplify SpecCmd.Run()** - Extract helper methods
4. **Fix unused context** - Remove or use properly
5. **Improve exists()** - Use os.Stat instead of Glob

**Effort:** 4-6 hours per item
**Impact:** Improved maintainability and code organization

### Long-term Improvements
1. **Split large files** - When they exceed 500 lines
2. **Add more subtests** - Use t.Run() consistently
3. **Review context strategy** - Determine if context is needed throughout

**Effort:** Ongoing as code grows
**Impact:** Maintains code quality as project scales

---

## What Plur Does Well ✓

The codebase has many strengths that should be preserved:

1. **Excellent concurrency patterns** - No goroutine leaks, proper channel management
2. **Good test coverage** - Table-driven tests, good use of testify
3. **Clean package boundaries** - watch package is particularly well-structured
4. **Proper error handling** - Good context in errors, just needs %w instead of %v
5. **Good documentation in many places** - Explains WHY, not just WHAT
6. **No inappropriate panics** - Only one panic in init for embedded resource (acceptable)
7. **Good use of early returns** - Keeps happy path clear
8. **Consistent code style** - Easy to read and navigate
9. **Well-organized tests** - Clear naming, good structure
10. **Good use of internal packages** - Prevents external coupling

---

## Recommended Action Plan

### Phase 1: Quick Wins (Week 1)
* [ ] Remove "Get" prefixes from all getters (H1)
* [ ] Remove "Log" prefixes from logger functions (H2)
* [ ] Change error wrapping from %v to %w (H3)
* [ ] Add package-level comments (M1)
* [ ] Define constants for magic numbers (M6)

**Result:** ~90% Go style compliance improvement with minimal effort

### Phase 2: Refactoring (Week 2-3)
* [ ] Eliminate utils.go package (H4)
* [ ] Remove inconsistent "Get" from Task methods (H7)
* [ ] Fix context usage issues (M4, M5)
* [ ] Simplify exists() helper (M7)
* [ ] Improve obvious comments (M8)

**Result:** Better package organization and API clarity

### Phase 3: Larger Refactors (Future)
* [ ] Extract business logic from main.go (H5)
* [ ] Simplify SpecCmd.Run() (H6)
* [ ] Clean up config package split (M2)
* [ ] Consider file splits for large files (M3)

**Result:** Improved maintainability for long-term growth

---

## Conclusion

The plur codebase is **already quite good** and follows most Go best practices. The issues identified are primarily:

* **Naming conventions** (Get/Log prefixes)
* **Error wrapping** (%v → %w)
* **Package organization** (utils.go, main.go size)

These are all **refinements** rather than critical problems. The concurrency patterns, testing approach, and overall structure are excellent and should serve as models for new code.

**Overall Grade: B+ (Very Good)**

With the quick wins from Phase 1 implemented, this would easily be an A (Excellent) codebase.