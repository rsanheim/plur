# Refactoring PrintResults for Better Design

## Problem Statement

The `PrintResults` function in `rux/result.go` is currently 110+ lines with complex conditional logic. This is primarily due to:

1. **Different parser return semantics**: RSpec provides pre-formatted output while Minitest doesn't
2. **Mixed responsibilities**: Formatting logic is scattered between PrintResults, parsers, and helper functions
3. **Framework-specific assumptions**: Helper functions like `FormatTestFailure()` assume RSpec patterns

## Current State Analysis

### RSpec
- Provides `FormattedFailuresNotification` with complete, formatted failure output
- Provides `FormattedSummaryNotification` with formatted summary
- PrintResults uses these when available, falls back to manual formatting

### Minitest
- Only provides raw `TestCaseNotification` objects
- No formatted output
- Missing backtraces (TODO in parser)
- Doesn't distinguish between errors and failures properly

### Issues in PrintResults
- 7 different conditional branches
- Duplicated color handling logic
- RSpec-specific formatting applied to all frameworks ("Failure/Error:", "rspec" command)
- Mixed concerns: display logic + formatting logic

## Proposed Solution

### 1. Extend TestOutputParser Interface

```go
type TestOutputParser interface {
    // Existing methods...
    ParseLine(line string) ([]types.TestNotification, bool)
    NotificationToProgress(notification types.TestNotification) (string, bool)
    FormatSummary(suite *types.SuiteNotification, totalExamples int, totalFailures int, totalPending int, wallTime float64, loadTime float64) string
    
    // New methods for formatting
    FormatFailures(failures []types.TestCaseNotification) string      // Format individual failure details
    FormatFailuresList(failures []types.TestCaseNotification) string  // Format list of failures with file:line
    ColorizeSummary(summary string, hasFailures bool) string
}
```

### 2. Parser Implementations

#### RSpec Parser
```go
func (p *RSpecParser) FormatFailures(failures []types.TestCaseNotification) string {
    // Return empty - RSpec provides FormattedFailuresNotification
    // This is only called as fallback
    return formatRSpecFailures(failures)
}

func (p *RSpecParser) FormatFailuresList(failures []types.TestCaseNotification) string {
    var sb strings.Builder
    for _, failure := range failures {
        sb.WriteString(fmt.Sprintf("rspec %s:%d # %s\n",
            failure.FilePath,
            failure.LineNumber,
            failure.FullDescription))
    }
    return sb.String()
}
```

#### Minitest Parser
```go
func (p *MinitestParser) FormatFailures(failures []types.TestCaseNotification) string {
    var sb strings.Builder
    for i, failure := range failures {
        sb.WriteString(fmt.Sprintf("  %d) %s:\n", i+1, failure.GetTestID()))
        sb.WriteString(fmt.Sprintf("%s#%s [%s:%d]:\n", 
            failure.ClassName, failure.MethodName, 
            failure.FilePath, failure.LineNumber))
        
        if failure.Exception != nil {
            sb.WriteString(failure.Exception.Message)
            sb.WriteString("\n")
            // Add backtrace when available
            for _, line := range failure.Exception.Backtrace {
                sb.WriteString("    " + line + "\n")
            }
        }
    }
    return sb.String()
}

func (p *MinitestParser) FormatFailuresList(failures []types.TestCaseNotification) string {
    // Minitest doesn't typically show a re-run command list
    return ""
}
```

### 3. Simplified PrintResults (~20 lines)

```go
func PrintResults(summary TestSummary, colorOutput bool) {
    parser, err := NewTestOutputParser(summary.Framework)
    if err != nil {
        // Fallback to basic output
        fmt.Printf("%d examples, %d failures\n", summary.TotalExamples, summary.TotalFailures)
        return
    }

    // Print failures if any
    if summary.HasFailures {
        if summary.FormattedFailures != "" {
            fmt.Print(summary.FormattedFailures)
        } else if len(summary.AllFailures) > 0 {
            fmt.Print(parser.FormatFailures(summary.AllFailures))
        }
    }

    // Print summary
    summaryText := summary.FormattedSummary
    if summaryText == "" {
        summaryText = parser.FormatSummary(nil, summary.TotalExamples, 
            summary.TotalFailures, summary.TotalPending,
            summary.WallTime.Seconds(), summary.TotalFileLoadTime.Seconds())
    }
    
    if colorOutput {
        summaryText = parser.ColorizeSummary(summaryText, summary.HasFailures)
    }
    fmt.Println(summaryText)

    // Print failed examples list (RSpec only)
    if failedList := parser.FormatFailuresList(summary.AllFailures); failedList != "" {
        fmt.Println("\nFailed examples:")
        fmt.Print(failedList)
    }

    // Print errored files
    for _, result := range summary.ErroredFiles {
        if result.State == StateError && result.Output != "" {
            fmt.Print(result.Output)
        }
    }
}
```

### 4. Additional Improvements Needed

1. **Minitest Parser**:
   - Capture and parse backtraces properly
   - Distinguish between errors and failures
   - Track assertion counts correctly

2. **General**:
   - Move `FormatTestFailure` and `FormatFailedExamples` from result.go
   - Remove `ExtractFailingLine` from rspec package, make it parser-specific

## Benefits

1. **Cleaner separation of concerns**: Each parser handles its own formatting
2. **Easier to maintain**: PrintResults focuses only on orchestration
3. **Framework-agnostic**: No RSpec assumptions in shared code
4. **Extensible**: Easy to add new test frameworks
5. **Testable**: Each parser's formatting can be tested independently

## Migration Path

1. First, extend the interface with default implementations
2. Move existing formatting logic to respective parsers
3. Refactor PrintResults to use new methods
4. Remove old formatting functions
5. Update tests to cover new parser methods