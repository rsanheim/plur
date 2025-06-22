# Minitest Support - Current Issues and Design Problems

## Status as of 2025-06-22

### Recent Commits
- `c3133e631` Add minitest integration tests and fix command building
- `592de6254` Implement basic minitest execution
- `11ae346de` Refactor command building with CommandBuilder interface
- `ab8bcb964` Add minitest module with parser and command builder
- `fff099f0b` Add framework type support for minitest

## Known Issues

### 1. Output Capture Problem
**Issue**: Minitest output shows "0 tests, 0 assertions..." instead of actual results

**Root Cause**: In `RunMinitestFiles` (runner.go:501-594):
- Uses `cmd.CombinedOutput()` which waits for entire command to finish
- No real-time output streaming like RSpec implementation
- Initial empty summary is captured before actual test output

**Manual Test Results**:
```bash
# What rux shows:
0 tests, 0 assertions, 0 failures, 0 errors, 0 skips

# What actually runs:
8 runs, 23 assertions, 0 failures, 0 errors, 0 skips
```

### 2. Progress Reporting Not Implemented
**Issue**: No dots (`.`, `F`, `E`) appear during test execution

**Root Cause**:
- RSpec uses custom JSON formatter for streaming
- Minitest outputs plain text progress indicators
- Current implementation doesn't scan output line-by-line

**Design Challenge**:
- RSpec has structured JSON output with clear message types
- Minitest has unstructured text output that needs parsing
- Need to detect progress indicators mixed with other output

### 3. Integration Test Failures
1. **minitest-success tests**: Expect "X runs" but get "0 tests"
2. **minitest-failures tests**: Don't capture error output properly

## Design Differences: RSpec vs Minitest

### RSpec Implementation
- Uses custom `rux_formatter.rb` that outputs JSON
- Structured messages with types: `example_passed`, `example_failed`, etc.
- Clean separation between progress events and output
- Streaming parser can easily identify message boundaries

### Minitest Challenge
- No custom formatter (using standard output)
- Progress indicators mixed with test output
- Need to parse unstructured text in real-time
- Must preserve all output while extracting progress

## Current Implementation Details

### File Discovery (glob.go)
- `FindTestFiles()` - Framework-aware file discovery
- `FindMinitestFiles()` - Finds `*_test.rb` files
- Works correctly, no issues here

### Command Building (minitest/command.go)
- Properly builds commands following parallel_tests pattern
- Strips `test/` prefix and `.rb` extension for require
- Uses array syntax: `["file1", "file2"].each { |f| require f }`
- Works correctly when tested manually

### Output Parsing (minitest/parser.go)
- `ParseOutput()` - Extracts summary from output
- Works correctly on complete output
- Issue is getting the complete output in the first place

### Runner Integration (runner.go)
- `RunSpecFile()` - Dispatches correctly to framework-specific runner
- `RunMinitestFiles()` - Uses `cmd.CombinedOutput()` (THE MAIN ISSUE)
- `parseMinitestOutput()` - Works but gets incomplete output

## Why This Is Challenging

1. **Streaming vs Batch Processing**
   - RSpec: Designed for streaming with JSON messages
   - Minitest: Designed for human-readable output
   - Need to parse human-readable output in real-time

2. **Output Format Differences**
   - RSpec: `{"type": "example_passed", ...}`
   - Minitest: Just `.` mixed with other output
   - No clear message boundaries in minitest

3. **Progress Indicator Detection**
   - Need to detect single characters (`.`, `F`, `E`) in output stream
   - These might appear anywhere in a line
   - Must distinguish from other output

4. **Preserving Output Fidelity**
   - Must capture all output for final display
   - While also parsing for progress indicators
   - Cannot lose any output in the process

## Potential Solutions

### Option 1: Line-by-Line Streaming (Recommended)
- Use stdout/stderr pipes like RSpec
- Scan each line for progress indicators
- Build complete output while streaming
- Parse summary when detected

### Option 2: Custom Minitest Reporter
- Create a minitest reporter that outputs JSON
- Would require users to add gem/require
- Goes against "simple" philosophy

### Option 3: Post-Process with Verbose Mode
- Use minitest `-v` flag for more structured output
- Parse verbose output after execution
- Loses real-time progress reporting

## Testing Manual Commands

```bash
# Current command that works manually:
bundle exec ruby -Itest -e '["calculator_test", "string_helper_test"].each { |f| require f }'

# Output includes:
Run options: --seed 24071

# Running:

in test_titleize
......in test_addition
..

Finished in 0.000601s, 13311.1473 runs/s, 38269.5484 assertions/s.

8 runs, 23 assertions, 0 failures, 0 errors, 0 skips
```

The key insight is that minitest does output progress indicators, but they're mixed with other output like "in test_titleize". The challenge is capturing and parsing this in real-time.