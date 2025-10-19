# Consolidation Analysis: plur spec and plur watch

## Executive Summary

After analyzing the codebase, I've identified significant opportunities to consolidate `plur spec` and `plur watch` commands. Currently, they operate on fundamentally different architectures - `spec` uses a sophisticated parallel execution system while `watch` uses a simple serial execution approach. The consolidation would bring major benefits including parallel test execution in watch mode, consistent formatting and output handling, and easier maintenance.

## Current Architecture

### plur spec
* **Parallel Execution**: Uses a worker pool pattern with goroutines and channels
* **Smart Grouping**: Groups test files by runtime history or file size for optimal distribution
* **Structured Output**: Uses custom JSON formatter for parsing test results
* **Runtime Tracking**: Tracks test execution times for future optimization
* **Framework Support**: Full support for RSpec and Minitest with extensible CommandBuilder interface

### plur watch  
* **Serial Execution**: Runs tests one at a time using simple exec.Command
* **File Watching**: Uses embedded e-dant/watcher binary for file system monitoring
* **File Mapping**: Maps source files to test files (lib→spec, app→spec)
* **Debouncing**: Prevents rapid re-runs with configurable delay
* **Interactive Mode**: Supports manual test runs via Enter key

## What IS Shared

1. **Configuration Infrastructure**
   * GlobalConfig struct (config.go:20-30)
   * ConfigPaths management 
   * Framework detection (RSpec/Minitest)
   * Color output handling

2. **Test Discovery**
   * FindTestFiles functions (glob.go)
   * ExpandGlobPatterns for pattern matching
   * Framework-specific file suffixes

3. **Command Building**
   * CommandBuilder interface (command_builder.go:10)
   * RSpecCommandBuilder and MinitestCommandBuilder
   * Framework-specific argument construction

4. **Core Types**
   * TestFramework enum
   * Test state management
   * Logger infrastructure

## What is NOT Shared

1. **Test Execution**
   * spec: Sophisticated parallel runner with worker pools (runner.go:299-395)
   * watch: Simple serial execution with exec.Command (watch.go:226-249)

2. **Output Handling**
   * spec: Channel-based output aggregator with structured parsing
   * watch: Direct stdout/stderr passthrough

3. **Result Processing**
   * spec: WorkerResult struct with detailed metrics
   * watch: No result tracking or aggregation

4. **Runtime Features**
   * spec: RuntimeTracker for execution time optimization
   * watch: No runtime tracking

5. **Test Grouping**
   * spec: Intelligent file grouping algorithms
   * watch: No grouping, runs files individually

## Consolidation Opportunities

### 1. Unified Test Execution Engine

Create a shared `TestRunner` interface that both commands can use:

```go
type TestRunner interface {
    Run(files []string, config *RunConfig) (*TestResults, error)
    SupportsParallel() bool
}

type RunConfig struct {
    Workers     int
    Command     string
    Framework   TestFramework
    ColorOutput bool
    JSONOutput  bool
    Interactive bool  // For watch mode
}
```

### 2. Parallel Execution in Watch Mode

The biggest win would be enabling parallel test execution in watch mode:

* When multiple files change, run them in parallel
* When running all tests (Enter key), use full parallel execution
* Make parallelism configurable for watch mode (maybe fewer workers)

### 3. Consistent Output Formatting

Both modes should use the same output handling:
* Structured test results parsing
* Consistent progress indicators
* Unified error reporting
* Same colorization logic

### 4. Shared File Mapping Logic

The FileMapper from watch mode could be enhanced and used by spec:
* Enable spec to run tests for changed implementation files
* Support reverse mapping (spec→implementation) for focused testing
* Integrate with runtime tracking for smarter test selection

### 5. Configuration Consolidation

Create command-specific configuration sections:
```toml
[spec]
workers = 8
command = "bundle exec rspec"

[watch]
workers = 4  # Fewer workers for interactive mode
debounce = 100
command = "bundle exec rspec"
```

## Areas Where Consolidation Does NOT Make Sense

### 1. Interactive Features
Watch mode's interactive prompt and command handling are unique and should remain separate.

### 2. File System Monitoring
The watcher binary integration is specific to watch mode and doesn't benefit spec.

### 3. Debouncing Logic
File change debouncing is watch-specific and would complicate the spec command.

### 4. Exit Behavior
* spec: Exits after test run completes
* watch: Continues running until explicitly stopped

## Implementation Plan

### Phase 1: Extract Common Test Runner
1. Create TestRunner interface and implementations
2. Move parallel execution logic to shared package
3. Migrate spec command to use new runner

### Phase 2: Enhance Watch Mode
1. Integrate TestRunner into watch command
2. Add parallel execution support
3. Implement consistent output handling

### Phase 3: Advanced Features
1. Share runtime tracking between modes
2. Implement smart test selection based on file changes
3. Add TUI features for both modes

## Alternative Approaches

### Option A: Minimal Integration
Only share the parallel execution engine, keeping everything else separate. This would be the safest approach but provides fewer benefits.

### Option B: Full Unification
Create a single command with modes: `plur test --watch`. This would be the most unified but might confuse existing users.

### Option C: Gradual Convergence (Recommended)
Start by sharing the execution engine and output handling, then gradually move more features to shared components based on user feedback.

## Additional Considerations

### 1. Performance Impact
Watch mode might need different performance characteristics:
* Faster startup time for individual test runs
* Lower memory usage for long-running process
* Different worker pool management

### 2. Testing Strategy
The consolidation should be extensively tested:
* Integration tests for both modes
* Performance benchmarks
* User experience testing

### 3. Migration Path
* Maintain backward compatibility
* Provide migration guide for configuration changes
* Consider feature flags for new behaviors

## Follow-up Questions

1. **User Experience**: Should watch mode default to parallel execution, or should it be opt-in?

2. **Configuration**: Should we support different commands for spec vs watch, or enforce consistency?

3. **Output Modes**: Should watch mode support JSON output for CI integration scenarios?

4. **Resource Management**: How should we handle worker pool lifecycle in long-running watch mode?

5. **Error Recovery**: How should watch mode handle persistent test failures (e.g., syntax errors)?

## Recommended Next Steps

1. **Spike: Shared Test Runner** - Create a proof of concept for the shared execution engine
2. **User Research** - Survey users about their watch mode usage patterns
3. **Performance Testing** - Benchmark parallel execution in watch mode scenarios
4. **API Design** - Design the public API for the shared components
5. **Incremental Rollout** - Start with optional parallel watch mode behind a flag