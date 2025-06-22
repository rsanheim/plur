# Minitest Support - Current Issues and Design Problems

## Status as of 2025-06-22 (UPDATED)

### Recent Commits
- `25b79bca6` Update glob pattern tests to expect 'test files' instead of 'spec files'
- `f9a954d3e` Fix minitest failure integration test expectations
- `d67b9cc02` Fix minitest output streaming and parsing
- `12375da85` Fix minitest command builder tests
- `c3133e631` Add minitest integration tests and fix command building

## Fixed Issues ✅

### 1. ~~Output Capture Problem~~ FIXED
**Previous Issue**: Minitest output showed "0 tests, 0 assertions..." instead of actual results

**Solution Implemented**:
- Changed from `cmd.CombinedOutput()` to streaming with pipes
- Now uses stdout/stderr pipes like RSpec implementation
- Captures output line-by-line in real-time
- Parser updated to look for "runs" instead of "tests"

### 2. ~~Progress Reporting Not Implemented~~ FIXED
**Previous Issue**: No dots (`.`, `F`, `E`) appeared during test execution

**Solution Implemented**:
- Added `isProgressLine()` function to detect lines with only progress indicators
- Streams progress indicators to output channel in real-time
- Properly handles mixed output (progress indicators + test names)
- Now shows colored dots/F/E/S during execution

### 3. ~~Integration Test Failures~~ FIXED
**Previous Issues**:
1. minitest-success tests expected "X runs" but got "0 tests" - FIXED
2. minitest-failures tests didn't capture error output properly - FIXED

**Solutions**:
- Updated parser regex to match "runs" instead of "tests"
- Fixed test expectations to look for output in stdout (not stderr)
- Updated failure detection to look for "Failures:" (plural)

## Remaining Design Challenges

### 1. Type Duplication
**Issue**: Notification types are duplicated in multiple packages
- `rspec/parser.go` defines all notification types
- `minitest/notification_parser.go` defines the same types again
- Main `notifications.go` has the canonical definitions

**Impact**: Code duplication, potential for drift

**Proposed Solution**: 
- Remove duplicated types from parser packages
- Import from main package or create shared types package

### 2. Parser Integration Not Complete
**Issue**: We have parsers but aren't using them yet
- `RunRSpecFiles` still has inline parsing logic
- `RunMinitestFiles` doesn't use the notification parser
- No accumulator to collect notifications

**Proposed Solution**:
- Create NotificationAccumulator
- Refactor runners to use parsers
- Extract common streaming logic

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