# Google Go Style Guide - Key Principles for Plur

This document distills the most relevant principles from the [Google Go Style Guide](https://google.github.io/styleguide/go/guide) and [Best Practices](https://google.github.io/styleguide/go/best-practices) for the plur project.

## Core Principles (In Order of Importance)

1. **Clarity** - Code should make its purpose and rationale obvious to readers
2. **Simplicity** - Accomplish goals in the most straightforward way possible
3. **Concision** - High signal-to-noise ratio without sacrificing clarity
4. **Maintainability** - Code should be easy to edit and evolve over time
5. **Consistency** - Align with broader Go ecosystem and codebase patterns

## Naming Conventions

### Getters
* **No "Get" prefix** for simple accessors
  ```go
  // Bad
  func GetVersion() string
  func (c *Config) GetTimeout() time.Duration

  // Good
  func Version() string
  func (c *Config) Timeout() time.Duration
  ```

### Avoid Repetition
* **Don't repeat package context** in names
  ```go
  // Bad (in package logger)
  func LogError(msg string)
  func LogVerbose(msg string)

  // Good
  func Error(msg string)
  func Verbose(msg string)
  // Usage: logger.Error("failed")
  ```

* **Don't repeat receiver type** in method names
  ```go
  // Bad
  func (c *Config) ConfigPath() string

  // Good
  func (c *Config) Path() string
  ```

### Package Names
* Use **short, lowercase, singular** names
* Avoid generic names like `utils`, `helpers`, `common`
* Package name should describe its purpose: `task`, `watch`, `format`

### Variable Names
* **Short names** for short scopes (`i` for index, `err` for error)
* **Descriptive names** for package-level or long-lived variables
* **Acronyms** should be all caps or all lowercase: `URL`, `url` (not `Url`)

## Error Handling

### Error Wrapping
* Use `%w` verb for wrapping errors (Go 1.13+) to preserve error chain
* Place `%w` at the **end** of format string for clarity
* Add context that helps debugging

```go
// Bad
return fmt.Errorf("failed to open: %v", err)  // loses error chain
return fmt.Errorf("%w: failed to open file", err)  // %w not at end

// Good
return fmt.Errorf("failed to open file %s: %w", filename, err)
return fmt.Errorf("config validation failed: %w", err)
```

### Error Flow
* **Return errors, don't log and return** - let caller decide what to do
* **Check errors immediately** after function calls
* **Handle errors only once** - either return them OR handle them, not both

```go
// Bad - logs AND returns
func loadConfig() error {
    data, err := os.ReadFile("config.toml")
    if err != nil {
        log.Printf("failed to read config: %v", err)  // Don't log here
        return err  // Caller will handle
    }
    return nil
}

// Good - just return
func loadConfig() error {
    data, err := os.ReadFile("config.toml")
    if err != nil {
        return fmt.Errorf("failed to read config: %w", err)
    }
    return nil
}
```

### Panics
* **Never panic in libraries** - return errors instead
* Only panic in `init()` for **unrecoverable** initialization failures
* Panics are acceptable for truly impossible conditions (programmer errors)

## Package Structure

### Organization
* **Group by purpose**, not by type
* Keep packages **focused and cohesive**
* Use `internal/` for packages that shouldn't be imported by others
* Avoid circular dependencies

```
plur/
  main.go           # CLI wiring, minimal logic
  runner.go         # Test execution
  watch/            # File watching functionality
    processor.go    # Event processing
    job.go         # Job definitions
    tokens.go      # Template tokens
  internal/         # Private packages
    task/          # Task detection
    format/        # Output formatting
```

### File Size
* Keep files **under 500 lines** when possible
* Split by **logical boundaries**, not arbitrary line counts
* One concept per file (e.g., `tokens.go` for token handling)

## Documentation

### Package Comments
Every package should have a package comment:
```go
// Package task provides test framework detection and task configuration.
// It supports RSpec, Minitest, and custom test frameworks through a unified interface.
package task
```

### Function Comments
* **Export functions** need comments starting with the function name
* Explain **WHY** and **WHAT** (when non-obvious), not just restate code
* Document **preconditions** and **side effects**

```go
// Bad - obvious comment
// Version returns the version string
func Version() string

// Good - explains context
// Version returns the build version, including git commit if available.
// Returns "unknown" if version information was not embedded at build time.
func Version() string
```

### Comments Should Be Maintenance-Free
* Don't reference temporal context ("recently added", "now supports")
* Don't document history ("used to do X, now does Y")
* Keep comments evergreen - they should remain true as code evolves

## Testing

### Test Organization
* Use **table-driven tests** for multiple similar cases
* Name test cases clearly with the `name` field
* Use `t.Run()` for subtests to get better test output

```go
func TestTokenBuild(t *testing.T) {
    tests := []struct {
        name     string
        path     string
        pattern  string
        expected Tokens
    }{
        {
            name:    "simple file",
            path:    "lib/foo.rb",
            pattern: "lib/*.rb",
            expected: Tokens{Path: "lib/foo.rb", Dir: "lib"},
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := BuildTokens(tt.path, tt.pattern)
            assert.Equal(t, tt.expected, got)
        })
    }
}
```

### Assertions
* Use **testify** for better assertions and error messages
* Use `require` for critical assertions (stops test on failure)
* Use `assert` for non-critical checks (continues on failure)

```go
// Critical - can't continue if this fails
require.NoError(t, err)
require.NotNil(t, config)

// Non-critical - can check multiple values
assert.Equal(t, expected, actual)
assert.Contains(t, output, "success")
```

### Test Helpers
* Test helpers should **set up state**, not make assertions
* Pass `t *testing.T` as first parameter and call `t.Helper()`
* Keep assertion logic in the test function itself

```go
// Good - helper sets up, test asserts
func createTestConfig(t *testing.T) *Config {
    t.Helper()
    cfg := &Config{Workers: 4}
    return cfg
}

func TestConfig(t *testing.T) {
    cfg := createTestConfig(t)
    assert.Equal(t, 4, cfg.Workers)  // Assertion in test, not helper
}
```

## Concurrency

### Goroutines
* **Always ensure goroutines terminate** - no leaks
* Use `sync.WaitGroup` or channels to coordinate
* Pass data through channels, not shared memory

```go
// Good - WaitGroup ensures cleanup
var wg sync.WaitGroup
for _, item := range items {
    wg.Add(1)
    go func(item Item) {
        defer wg.Done()
        process(item)
    }(item)  // Pass as parameter, don't capture
}
wg.Wait()
```

### Channels
* **Specify direction** in function signatures when possible
* **Close channels** from the sender, never the receiver
* Use buffered channels when you know the capacity

```go
// Good - direction specified
func producer(out chan<- string) {
    defer close(out)  // Sender closes
    out <- "data"
}

func consumer(in <-chan string) {
    for msg := range in {  // Range handles close
        process(msg)
    }
}
```

### Context
* Pass `context.Context` as **first parameter**
* Use for cancellation, deadlines, and request-scoped values
* Never store contexts in structs

```go
// Good
func FetchData(ctx context.Context, id string) (*Data, error) {
    // Check for cancellation
    select {
    case <-ctx.Done():
        return nil, ctx.Err()
    default:
    }
    // ... fetch data
}
```

## Performance

### Optimization
* **Measure first** - don't guess about performance
* **Optimize for clarity** unless performance is proven critical
* Profile before optimizing

### Preallocation
* Preallocate slices when size is known
* Don't over-preallocate "just in case"

```go
// Good - size known
results := make([]Result, 0, len(inputs))

// Bad - arbitrary preallocation
results := make([]Result, 0, 1000)  // Why 1000?
```

## Code Organization Best Practices

### Initialization
* Use `var` for zero values
* Use `:=` for non-zero values
* Group related declarations

```go
var (
    defaultTimeout = 30 * time.Second
    maxRetries     = 3
)

func process() {
    ctx := context.Background()  // Non-zero value
    var results []string         // Zero value
}
```

### Early Returns
* Return early for error conditions
* Keep the "happy path" at the minimum indentation

```go
// Good - early return
func process(data []byte) error {
    if len(data) == 0 {
        return errors.New("empty data")
    }

    if !isValid(data) {
        return errors.New("invalid data")
    }

    // Happy path with minimal indentation
    result := parse(data)
    return save(result)
}
```

### Constants
* Define constants for **magic numbers**
* Group related constants with `iota` when appropriate
* Use typed constants when the type matters

```go
// Good
const (
    maxWorkers        = 10
    defaultTimeout    = 30 * time.Second
    bufferSize        = 1024
)

// Good - typed constants
type State int

const (
    StateIdle State = iota
    StateRunning
    StateStopped
)
```

## Summary

Following these guidelines will help plur maintain:
* **Clarity** - Code intentions are obvious
* **Consistency** - Follows Go community standards
* **Maintainability** - Easy to modify and extend
* **Reliability** - Proper error handling and concurrency
* **Performance** - Efficient without sacrificing clarity

The key is balancing these principles appropriately for each situation, always favoring clarity and correctness over premature optimization.