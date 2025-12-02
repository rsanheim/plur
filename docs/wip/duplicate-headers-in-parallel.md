# Fix: Duplicate Headers in Multi-Worker Output

## Root Cause (Refined)

The issue is in `PrintResults()` - it always uses the pre-formatted strings (`FormattedFailures`, `FormattedPending`) even in multi-worker mode where they contain duplicate headers.

**What we have:**
- Individual examples ARE being collected into `summary.AllFailures`
- A `parser.FormatFailures([]TestCaseNotification)` method EXISTS that formats with a single header
- But `PrintResults` uses `summary.FormattedFailures` (concatenated strings) instead

**Current code (result.go:143-146):**
```go
} else if summary.HasFailures && summary.FormattedFailures != "" {
    // For RSpec, use the formatted failures
    fmt.Print(summary.FormattedFailures)  // ← BUG: uses concatenated strings
}
```

**Should be using:**
```go
} else if summary.HasFailures && len(summary.AllFailures) > 0 {
    // Use aggregated failures with single header
    fmt.Print(parser.FormatFailures(summary.AllFailures))  // ← Uses individual examples
}
```

## Why Single-Worker Mode Works

In single-worker mode, there's only one `FormattedFailures` string from one worker, so no duplicate headers.

## The Fix

In `PrintResults()`, prioritize using the aggregated individual examples over the pre-formatted strings:

### For Failures (result.go:143-146)

**Before:**
```go
} else if summary.HasFailures && summary.FormattedFailures != "" {
    fmt.Print(summary.FormattedFailures)
}
```

**After:**
```go
} else if summary.HasFailures {
    if len(summary.AllFailures) > 0 {
        // Use aggregated failures - avoids duplicate headers in multi-worker mode
        fmt.Print(parser.FormatFailures(summary.AllFailures))
    } else if summary.FormattedFailures != "" {
        // Fallback to pre-formatted string
        fmt.Print(summary.FormattedFailures)
    }
}
```

### For Pending (result.go:129-132)

**Before:**
```go
if summary.FormattedPending != "" {
    fmt.Print(summary.FormattedPending)
}
```

**After:**
1. First add `AllPending` collection to `TestSummary` (like `AllFailures`)
2. Add `parser.FormatPending()` method to interface and implement it
3. Then:
```go
if len(summary.AllPending) > 0 {
    fmt.Print(parser.FormatPending(summary.AllPending))
} else if summary.FormattedPending != "" {
    fmt.Print(summary.FormattedPending)
}
```

## Phased Approach

### Phase 1: Fix Failures (your request)
1. Change `PrintResults` to use `parser.FormatFailures(summary.AllFailures)` instead of `summary.FormattedFailures`
2. This is a one-line change in the condition

### Phase 2: Fix Pending (later)
1. Add `AllPending` to `TestSummary`
2. Collect pending tests in `BuildTestSummary`
3. Add `FormatPending` to parser interface
4. Implement `FormatPending` in rspec/minitest/passthrough parsers
5. Update `PrintResults` for pending

## Files to Modify (Phase 1 only)

| File | Change |
|------|--------|
| `plur/result.go:143-146` | Use `parser.FormatFailures(summary.AllFailures)` instead of `summary.FormattedFailures` |

## Verification

1. Create a fixture with failures in multiple files
2. Run `plur -n 2` to force multi-worker
3. Verify single "Failures:" header in output
4. Verify all failure details are present
