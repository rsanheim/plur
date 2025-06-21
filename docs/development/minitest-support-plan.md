# Minitest Support & Framework Abstraction Plan

## Overview

This document outlines the plan to add Minitest support to rux while establishing proper abstractions for supporting multiple test frameworks. The approach prioritizes incremental changes, maintains backward compatibility, and uses Minitest as a forcing function to drive better design.

## Current Architecture Analysis

### Tight Coupling Points

1. **TestResult struct** - Contains `rspec.FailureDetail` and `rspec.JSONOutput`
2. **Runner** - Hardcodes RSpec formatter path and command structure  
3. **Result formatting** - Calls `rspec.FormatFailure()` directly
4. **File patterns** - Assumes `*_spec.rb` pattern
5. **Output parsing** - Expects RSpec's JSON format

### Strengths to Preserve

- Streaming output model for real-time feedback
- Parallel execution architecture
- Runtime-based test distribution
- Minimal dependencies

## Design Principles

1. **Concrete First** - Build Minitest support, then extract abstractions
2. **No Special Cases** - Avoid if/else proliferation for each framework
3. **Maintain Performance** - Keep streaming and parallel execution unchanged
4. **Backward Compatible** - Existing RSpec projects must work without changes
5. **Progressive Disclosure** - Auto-detection with explicit override options

## Implementation Phases

### Phase 0: Create more fixture projects for testing and verification

* create `fixtures/projects/minitest-success` project
    * A basic test project using latest minitest in a 'stock' way
    * Add some some basic tests that all pass
* create `fixtures/projects/minitest-failures` project
    * A basic test project using latest minitest in a 'stock' way
    * Add a mix of tests, some failing and some failing
    * For verifying failure detection and output
* create `fixtures/projects/testunit-success` project
    * A basic test project using latest ruby test-unit in a 'stock' way
    * Add some some basic tests that all pass
* create `fixtures/projects/testunit-failures` project
    * A basic test project using latest test-unit in a 'stock' way
    * Add a mix of tests, some failing and some failing
    * For verifying failure detection and output
* create a spec helper method to run the tests in these projects using the default Ruby way, so we can compare against and see
how they run by default

### Phase 1: Decouple Core Types from RSpec

**Goal**: Remove `rspec` package imports from core types

#### 1.1 Create Framework-Agnostic Types

```go
// result.go - NEW generic types
type TestFailure struct {
    Description string
    FilePath    string
    LineNumber  int
    Message     string
    Backtrace   []string
}

type TestExample struct {
    Description string
    Status      string // "passed", "failed", "pending"
    Duration    float64
}
```

#### 1.2 Update TestResult

```go
// runner.go - UPDATED
type TestResult struct {
    SpecFile     string
    Success      bool
    Output       string
    Error        error
    Duration     time.Duration
    FileLoadTime time.Duration
    
    // Framework-agnostic fields (was rspec-specific)
    Failures     []TestFailure  // Changed from []rspec.FailureDetail
    Examples     []TestExample  // New - for runtime tracking
    ExampleCount int
    FailureCount int
    PendingCount int
    
    // Raw formatted output (framework provides this)
    FormattedFailures string
    FormattedSummary  string
}
```

#### 1.3 Update TestSummary

```go
// result.go - UPDATED
type TestSummary struct {
    TotalExamples     int
    TotalFailures     int
    AllFailures       []TestFailure // Changed from []rspec.FailureDetail
    // ... rest unchanged
}
```

### Phase 2: Define Test Framework Interface

**Goal**: Create abstraction for test framework operations

#### 2.1 Core Framework Interface

```go
// framework/framework.go
package framework

type Framework interface {
    // Identity
    Name() string
    DefaultCommand() string
    
    // Detection
    DetectProject(dir string) bool
    TestFilePattern() string  // e.g., "*_spec.rb" or "*_test.rb"
    
    // Execution
    BuildCommand(files []string, options CommandOptions) []string
    RequiresFormatter() bool
    GetFormatterPath(formatterDir string) (string, error)
    
    // Output parsing
    ParseStreamingOutput(line string) (*TestEvent, error)
}

type CommandOptions struct {
    FormatterPath string
    ColorOutput   bool
    BaseCommand   string // Override from config
}

type TestEvent struct {
    Type      EventType
    Example   *TestExample
    Failure   *TestFailure
    LoadTime  float64
    
    // For formatted output events
    FormattedOutput string
    OutputType      string // "failures", "summary"
}

type EventType int
const (
    EventLoadComplete EventType = iota
    EventExamplePassed
    EventExampleFailed
    EventExamplePending
    EventFormattedOutput
    EventRunComplete
)
```

#### 2.2 Framework Registry

```go
// framework/registry.go
var frameworks = map[string]Framework{
    "rspec":    &RSpecFramework{},
    "minitest": &MinitestFramework{},
}

func Get(name string) (Framework, error) {
    if f, ok := frameworks[name]; ok {
        return f, nil
    }
    return nil, fmt.Errorf("unknown framework: %s", name)
}

func Detect(dir string) Framework {
    // Check each framework's DetectProject method
    for _, f := range frameworks {
        if f.DetectProject(dir) {
            return f
        }
    }
    return frameworks["rspec"] // Default
}
```

### Phase 3: Implement RSpec Framework Adapter

**Goal**: Move existing RSpec logic into framework implementation

#### 3.1 RSpec Framework Implementation

```go
// framework/rspec.go
type RSpecFramework struct{}

func (r *RSpecFramework) Name() string { return "rspec" }
func (r *RSpecFramework) DefaultCommand() string { return "bundle exec rspec" }

func (r *RSpecFramework) DetectProject(dir string) bool {
    // Check for spec/ directory or .rspec file
    specDir := filepath.Join(dir, "spec")
    if _, err := os.Stat(specDir); err == nil {
        return true
    }
    // ... check for .rspec, Gemfile with rspec-core, etc.
    return false
}

func (r *RSpecFramework) BuildCommand(files []string, opts CommandOptions) []string {
    cmd := strings.Fields(opts.BaseCommand)
    
    if opts.FormatterPath != "" {
        cmd = append(cmd, "-r", opts.FormatterPath, 
                    "--format", "Rux::JsonRowsFormatter")
    }
    
    if !opts.ColorOutput {
        cmd = append(cmd, "--no-color")
    } else {
        cmd = append(cmd, "--force-color", "--tty")
    }
    
    return append(cmd, files...)
}

func (r *RSpecFramework) ParseStreamingOutput(line string) (*TestEvent, error) {
    // Adapt existing rspec.ParseStreamingMessage
    msg, err := rspec.ParseStreamingMessage(line)
    if err != nil || msg == nil {
        return nil, err
    }
    
    // Convert to generic TestEvent
    return convertRSpecMessage(msg), nil
}
```

### Phase 4: Add Minitest Support

**Goal**: Implement Minitest framework with JSON output

#### 4.1 Minitest JSON Formatter (Ruby)

```ruby
# formatter/minitest_json_rows_formatter.rb
require 'json'
require 'minitest'

module Rux
  class MinitestJsonRowsFormatter < Minitest::StatisticsReporter
    def start
      super
      io.puts "RUX_JSON:#{JSON.generate({type: 'start', count: options[:total]})}"
    end
    
    def record(result)
      event = {
        type: status_type(result),
        example: {
          description: result.name,
          full_description: "#{result.klass}##{result.name}",
          location: result.source_location.join(':'),
          file_path: result.source_location[0],
          line_number: result.source_location[1],
          status: result.result_code,
          run_time: result.time
        }
      }
      
      if result.failure
        event[:example][:exception] = format_exception(result.failure)
      end
      
      io.puts "RUX_JSON:#{JSON.generate(event)}"
    end
    
    def report
      super
      io.puts "RUX_JSON:#{JSON.generate({
        type: 'summary',
        example_count: count,
        failure_count: failures,
        skip_count: skips,
        duration: total_time
      })}"
    end
    
    private
    
    def status_type(result)
      case result.result_code
      when '.' then 'example_passed'
      when 'F' then 'example_failed'
      when 'S' then 'example_pending'
      when 'E' then 'example_failed'
      end
    end
    
    def format_exception(failure)
      {
        class: failure.error.class.name,
        message: failure.message,
        backtrace: failure.backtrace
      }
    end
  end
end
```

#### 4.2 Minitest Framework Implementation

```go
// framework/minitest.go
type MinitestFramework struct{}

func (m *MinitestFramework) Name() string { return "minitest" }
func (m *MinitestFramework) DefaultCommand() string { return "ruby -Itest" }

func (m *MinitestFramework) DetectProject(dir string) bool {
    // Check for test/ directory
    testDir := filepath.Join(dir, "test")
    if _, err := os.Stat(testDir); err == nil {
        // Look for test files
        matches, _ := filepath.Glob(filepath.Join(testDir, "**/*_test.rb"))
        return len(matches) > 0
    }
    return false
}

func (m *MinitestFramework) TestFilePattern() string {
    return "*_test.rb"
}

func (m *MinitestFramework) BuildCommand(files []string, opts CommandOptions) []string {
    cmd := strings.Fields(opts.BaseCommand)
    
    if opts.FormatterPath != "" {
        // Require our formatter
        cmd = append(cmd, "-r", opts.FormatterPath)
        // Add minitest/autorun if not already in base command
        if !strings.Contains(opts.BaseCommand, "minitest/autorun") {
            cmd = append(cmd, "-r", "minitest/autorun")
        }
    }
    
    return append(cmd, files...)
}

func (m *MinitestFramework) ParseStreamingOutput(line string) (*TestEvent, error) {
    // Similar structure to RSpec parser
    if !strings.HasPrefix(line, "RUX_JSON:") {
        return nil, nil
    }
    
    var msg minitestMessage
    if err := json.Unmarshal([]byte(line[8:]), &msg); err != nil {
        return nil, err
    }
    
    return convertMinitestMessage(&msg), nil
}
```

### Phase 5: Update Core Components

**Goal**: Wire framework abstraction through the system

#### 5.1 Update Config

```go
// config.go
type Config struct {
    // ... existing fields ...
    Framework    string // "rspec", "minitest", or auto-detect
    SpecCommand  string
    TestCommand  string // For minitest
}
```

#### 5.2 Update SpecCmd

```go
// main.go
func (r *SpecCmd) Run(parent *RuxCLI) error {
    // ... existing setup ...
    
    // Detect or get configured framework
    var fw framework.Framework
    if config.Framework != "" {
        fw, err = framework.Get(config.Framework)
        if err != nil {
            return err
        }
    } else {
        fw = framework.Detect(".")
    }
    
    logger.Logger.Debug("detected framework", "name", fw.Name())
    
    // Update file discovery based on framework
    if len(r.Patterns) == 0 {
        r.Patterns = []string{fmt.Sprintf("**/%s", fw.TestFilePattern())}
    }
    
    // Pass framework to executor
    executor := NewTestExecutor(config, specFiles, fw)
    // ...
}
```

#### 5.3 Update TestExecutor

```go
// execution.go
type TestExecutor struct {
    config    *Config
    specFiles []string
    framework framework.Framework
}

func (e *TestExecutor) buildCommand(files []string) []string {
    baseCmd := e.config.SpecCommand
    if baseCmd == "" {
        baseCmd = e.framework.DefaultCommand()
    }
    
    opts := framework.CommandOptions{
        BaseCommand:   baseCmd,
        ColorOutput:   e.config.ColorOutput,
        FormatterPath: e.getFormatterPath(),
    }
    
    return e.framework.BuildCommand(files, opts)
}
```

#### 5.4 Update Runner

```go
// runner.go - Update RunSpecFile
func RunSpecFile(ctx context.Context, framework framework.Framework, ...) TestResult {
    // Build command using framework
    args := /* use framework.BuildCommand */
    
    // In output parsing goroutine:
    event, err := framework.ParseStreamingOutput(line)
    if event != nil {
        switch event.Type {
        case framework.EventExamplePassed:
            outputChan <- OutputMessage{Type: "dot"}
            // ... handle event
        }
    }
}
```

### Phase 6: Configuration Support

**Goal**: Allow framework configuration via TOML

```toml
# .rux.toml

# Explicit framework selection (optional, auto-detected if not set)
framework = "minitest"

# Framework-specific commands
[spec]
command = "bundle exec rspec"

[test]  # New section for minitest
command = "ruby -Itest"

# Future: framework-specific options
[frameworks.minitest]
require_autorun = true
test_dir = "test"

[frameworks.rspec]
require_spec_helper = true
spec_dir = "spec"
```

## Testing Strategy

### 1. Unit Tests
- Test each framework implementation in isolation
- Test framework detection logic
- Test output parsing for each framework

### 2. Integration Tests
- Create minimal test projects for each framework
- Test full execution flow
- Verify output formatting matches expectations

### 3. Compatibility Tests
- Ensure existing RSpec projects work unchanged
- Test projects with both RSpec and Minitest files
- Verify configuration precedence

### 4. Performance Tests
- Ensure no regression in execution speed
- Verify streaming behavior unchanged
- Test parallel execution with both frameworks

## Migration Strategy

1. **Feature Branch**: All work on `minitest-support` branch
2. **Incremental PRs**:
   - PR 1: Decouple types (Phase 1)
   - PR 2: Add framework interface (Phase 2)
   - PR 3: RSpec adapter (Phase 3)
   - PR 4: Minitest support (Phase 4)
   - PR 5: Wire everything together (Phase 5)
3. **Beta Testing**: Hidden `--framework` flag for early testing
4. **Documentation**: Update before general release

## Future Extensibility

This architecture sets up support for:

### Additional Ruby Frameworks
- Test::Unit
- Cucumber
- Minitest::Spec

### Other Languages
- Go tests (`go test`)
- JavaScript/Node (`jest`, `mocha`)
- Python (`pytest`)

### Framework-Specific Features
- Custom test discovery
- Framework-specific performance optimizations
- Native parallel execution integration

## Open Questions

1. **Formatter Management**: Should each framework manage its own formatter installation?
2. **Mixed Projects**: How to handle projects with both RSpec and Minitest?
3. **File Mapping**: Should watch mode file mapping be framework-aware?
4. **Output Unification**: How much should we normalize output across frameworks?

## Success Criteria

1. Minitest projects run with same parallelization as RSpec
2. No performance regression for RSpec projects
3. Configuration remains simple and optional
4. Adding new frameworks requires minimal code changes
5. Existing RSpec projects work without any changes

## Timeline Estimate

- Phase 1-2: 1 week (refactoring)
- Phase 3-4: 1 week (implementation)
- Phase 5-6: 1 week (integration)
- Testing & Documentation: 1 week

Total: ~1 month for full implementation

---

*This is a living document. As implementation proceeds, we'll update based on discoveries and feedback.*