# Test Framework Type Refactor

## Overview
Convert TestFramework from a string constant to a struct that encapsulates the file patterns and conventions of each test framework.

## Current State
TestFramework is currently just a string type:
```go
type TestFramework string

const (
    FrameworkRSpec    TestFramework = "rspec"
    FrameworkMinitest TestFramework = "minitest"
)
```

This leads to scattered framework-specific logic throughout the codebase.

## Implementation Plan

### 1. Create New TestFramework Type (in `config.go`)

```go
// TestFramework represents a test framework with its file patterns and conventions
type TestFramework struct {
    name       string
    suffix     string    // Primary file suffix like "_spec.rb"
    directory  string    // Default test directory like "spec/"
}

// Framework instances
var (
    FrameworkRSpec = TestFramework{
        name:      "rspec",
        suffix:    "_spec.rb",
        directory: "spec/",
    }
    
    FrameworkMinitest = TestFramework{
        name:      "minitest",
        suffix:    "_test.rb",
        directory: "test/",
    }
)

// String returns the framework name for compatibility
func (f TestFramework) String() string {
    return f.name
}

// IsTestFile checks if a file path matches this framework's pattern
func (f TestFramework) IsTestFile(path string) bool {
    base := filepath.Base(path)
    if f.name == "minitest" {
        // Minitest supports both _test.rb and test_*.rb
        return strings.HasSuffix(base, "_test.rb") || 
               (strings.HasPrefix(base, "test_") && strings.HasSuffix(base, ".rb"))
    }
    return strings.HasSuffix(path, f.suffix)
}
```

### 2. Update ParseFrameworkType to Return Struct

```go
func ParseFrameworkType(frameworkType string) TestFramework {
    if frameworkType == "" {
        return DetectTestFramework()
    }
    switch frameworkType {
    case "rspec":
        return FrameworkRSpec
    case "minitest":
        return FrameworkMinitest
    default:
        return FrameworkRSpec
    }
}
```

### 3. Update Direct String Comparisons

Since TestFramework is now a struct, update comparisons:

```go
// In various files, change:
if framework == FrameworkMinitest

// To:
if framework.name == "minitest"
// Or add an equals method if preferred
```

### 4. Replace getTestFileSuffix Function

```go
// Remove getTestFileSuffix entirely
// Replace calls like:
suffix := getTestFileSuffix(framework)

// With:
suffix := framework.suffix
```

### 5. Update File Pattern Matching

```go
// In glob.go isTestFile function:
func isTestFile(path string, framework TestFramework) bool {
    return framework.IsTestFile(path)
}

// In FindSpecFiles/FindMinitestFiles, use the patterns directly
```

### 6. Fix UI Messages Separately

For the "Running X spec files" issue, we'll handle it directly in execution.go:

```go
// Determine the terminology based on framework
fileType := "spec files"
if e.specCmd.GetFramework().name == "minitest" {
    fileType = "test files"
}

// Use singular when appropriate
if len(e.specFiles) == 1 {
    fileType = strings.TrimSuffix(fileType, "s") + " file"
}

// Only show "in parallel" when actually parallel
if actualWorkers > 1 && len(e.specFiles) > 1 {
    fmt.Printf("Running %d %s in parallel using %d workers...\n", 
        len(e.specFiles), fileType, actualWorkers)
} else {
    fmt.Printf("Running %d %s...\n", len(e.specFiles), fileType)
}
```

## Files to Update

1. `config.go` - Define new TestFramework type
2. `glob.go` - Remove getTestFileSuffix, update isTestFile
3. `execution.go` - Fix "Running X spec files" messages
4. `main.go` - Update framework comparisons
5. `runner.go` - Update framework comparisons
6. `parser_factory.go` - Update framework comparisons
7. `command_builder.go` - Update framework comparisons

## Benefits

1. **Encapsulation** - Test framework patterns in one place
2. **Type safety** - Can't mix up framework types
3. **Extensibility** - Easy to add new frameworks
4. **Simple** - Just the essential framework properties
5. **Clear separation** - UI concerns stay in UI code

## What This Doesn't Include

- No UI/display methods (those belong in the presentation layer)
- No plural/singular logic (that's a UI concern)
- Just the core framework patterns and detection logic

## Testing Considerations

- Verify framework detection still works
- Test that file pattern matching works for both frameworks
- Ensure all existing tests pass
- No JSON serialization changes needed (TestFramework is never persisted)