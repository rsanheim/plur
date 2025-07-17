# Minitest Implementation Guide

This guide documents the practical implementation details, challenges, and solutions for adding Minitest support to Plur.

## Minitest Execution Model

### Command Structure

Minitest doesn't have a built-in parallel runner like RSpec. We execute tests using Ruby's `-e` flag:

```bash
ruby -Itest -e 'ARGV.each { |f| require f }' test1.rb test2.rb
```

This approach:
- Adds `test` directory to load path with `-I`
- Executes inline Ruby code that requires each test file
- Runs all tests in a single process per worker

### File Detection

Minitest files are detected using these patterns:
- `*_test.rb` - Rails convention
- `test_*.rb` - Traditional Ruby convention
- Files in `test/` directory

Implementation in `file_finder.go`:
```go
case strings.HasSuffix(path, "_test.rb"), 
     strings.HasPrefix(base, "test_"):
    framework = FrameworkMinitest
```

## Critical Implementation Issues

### The Streaming Problem

**Issue**: Initial implementation showed "0 tests, 0 assertions" despite tests running successfully.

**Root Cause**: Using `cmd.CombinedOutput()` which waits for entire command completion:
```go
// WRONG - This breaks real-time output
output, err := cmd.CombinedOutput()
```

**Solution**: Switch to streaming with pipes:
```go
// RIGHT - Enables real-time progress
stdout, _ := cmd.StdoutPipe()
stderr, _ := cmd.StderrPipe()
cmd.Start()

// Stream output line by line
scanner := bufio.NewScanner(stdout)
for scanner.Scan() {
    line := scanner.Text()
    processMinitestOutput(line, outputChan, workerIndex)
}
```

**Lesson**: Streaming is critical for user experience. Always use pipes for real-time feedback.

### Progress Indicator Parsing

Minitest outputs single characters for test progress:
- `.` - Test passed
- `F` - Test failed  
- `E` - Test error
- `S` - Test skipped

Parser implementation:
```go
// Look for lines containing only progress indicators
if matched, _ := regexp.MatchString(`^[\.FES]+$`, line); matched {
    for _, char := range line {
        switch char {
        case '.':
            outputChan <- OutputMessage{WorkerID: workerIndex, Type: "dot"}
        case 'F', 'E':
            outputChan <- OutputMessage{WorkerID: workerIndex, Type: "failure"}
        case 'S':
            outputChan <- OutputMessage{WorkerID: workerIndex, Type: "pending"}
        }
    }
}
```

### Summary Line Differences

**Issue**: Parser looked for "tests" but Minitest outputs "runs":

```
# RSpec:    10 examples, 2 failures, 1 pending
# Minitest: 10 runs, 8 assertions, 2 failures, 0 errors, 1 skips
```

**Solution**: Updated regex pattern:
```go
runsSummaryRegex = regexp.MustCompile(`(\d+) runs?, (\d+) assertions?, (\d+) failures?, (\d+) errors?, (\d+) skips?`)
```

## Output Format Challenges

### Multi-line Failure Format

Minitest failures span multiple lines requiring stateful parsing:

```
  1) Failure:
TestCalculator#test_addition [test/calculator_test.rb:15]:
Expected: 5
  Actual: 4
```

Parser state management:
```go
type MinitestOutputParser struct {
    inFailure     bool
    failureBuffer strings.Builder
    currentTest   string
    currentLocation string
}
```

### Integration Test Updates

Tests needed framework-specific expectations:
```ruby
def expected_output_for_framework(framework)
  case framework
  when :minitest
    /\d+ runs?, \d+ assertions?/
  when :rspec
    /\d+ examples?/
  end
end
```

## Performance Characteristics

### Startup Time
- **RSpec**: ~1-2s framework load time
- **Minitest**: ~0.2-0.5s framework load time
- **Impact**: Minitest tests start faster, beneficial for small test suites

### Memory Usage
- Fewer objects per test than RSpec
- No example group hierarchy overhead
- Simpler assertion tracking
- **Impact**: Lower memory footprint for large test suites

### Parser Efficiency
- Text parsing with regex: O(1) per line
- No JSON parsing overhead
- Simple state machine for multi-line handling
- **Impact**: Slightly faster parsing than RSpec JSON

## Configuration Considerations

### Test Helper Requirements
Minitest projects require proper setup:
```ruby
# Rails projects
require 'test_helper'

# Pure Minitest
require 'minitest/autorun'
```

Plur doesn't modify these requirements - test files must handle their own setup.

### Current Limitations

1. **No JSON formatter** - Must parse unstructured text output
2. **Limited metadata** - Less test information than RSpec provides
3. **No example groups** - Flat test structure only
4. **Basic filtering** - Name-based filtering only
5. **No custom reporters** - Default reporter only

## Lessons Learned

### 1. Framework Assumptions Kill
Don't assume all frameworks work like RSpec. Minitest has fundamentally different:
- Output patterns (text vs JSON)
- Progress reporting (characters vs events)
- Summary format ("runs" vs "examples")

### 2. Integration Tests are Essential
Unit tests didn't catch the streaming issue. Only integration tests that run actual commands revealed the problem.

### 3. Parser State is Complex
Text-based parsers need careful state management. Consider:
- Multi-line patterns
- Interleaved output
- Error recovery
- Buffer management

### 4. Small Details Matter
- "runs" vs "tests" broke parsing
- Progress chars need flushing
- Exit codes differ between frameworks

## Future Enhancements

1. **Custom Formatter**: Create Minitest plugin for structured output
2. **Better Filtering**: Support tags/categories via minitest-reporters
3. **Seed Support**: Capture and report random seed for reproducibility
4. **Rails Integration**: Better support for Rails test types (unit/functional/integration)
5. **Performance Tracking**: Store test runtimes for better distribution

## Debugging Tips

When Minitest support isn't working:

1. **Check streaming**: Ensure pipes are used, not CombinedOutput
2. **Verify regex patterns**: Test against actual Minitest output
3. **Watch for buffering**: Minitest may need explicit flushing
4. **Test with simple cases**: Start with single test file
5. **Compare with direct execution**: Run same command manually

## Testing Minitest Support

Key integration test scenarios:
- Single file with passing tests
- Multiple files with mixed results  
- Syntax errors and load failures
- Empty test files
- Very large test suites

Always test with both Rails and pure Minitest projects to ensure compatibility.