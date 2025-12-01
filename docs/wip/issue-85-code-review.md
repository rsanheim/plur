# Code Review: Issue #85 - Stdout Streaming Implementation

*Reviewed: 2025-11-29*
*Branch: `stdout-behavior`*
*Reviewer: Collette (Claude Code)*

## Summary

This PR adds real-time stdout streaming for RSpec tests while explicitly avoiding it for Minitest. The approach adds a `streamStdout` boolean parameter to `streamTestOutput()` that conditionally sends unconsumed stdout lines to the output aggregator. The implementation is surgical and focused, with good documentation.

**Overall Assessment**: This is a clean, well-reasoned implementation that correctly solves the problem for RSpec. However, there are some concerns about output ordering and a potential edge case with empty lines.

---

## Strengths

1. **Clear Problem Understanding**: The design decision to only stream for RSpec (where JSON parsing gives clear `consumed` semantics) while avoiding Minitest (where `consumed=false` for nearly everything) shows good analysis of the existing behavior.

2. **Good Documentation**: The comments in `stream_helper.go` (lines 69-71) and `runner.go` (lines 226-227) explain the *why* clearly, which is exactly what comments should do.

3. **Minimal Changes**: The implementation touches only the essential files and doesn't over-engineer the solution.

4. **Follows Existing Patterns**: The new "stdout" message type follows the same pattern as "stderr" in `outputAggregator`.

5. **Updated Architecture Docs**: The `test-processing-flow.md` changes accurately reflect the new behavior.

---

## Critical Issues

### 1. Potential Output Ordering Issue with Multiple Workers

*Location*: `stream_helper.go` lines 72-78, `runner.go` line 329

When multiple workers are running concurrently, stdout from different workers will interleave unpredictably. Consider this scenario:

```
Worker 0: puts "Starting test A"
Worker 1: puts "Starting test B"
Worker 0: puts "Finishing test A"
Worker 1: puts "Finishing test B"
```

Output could appear as:
```
Starting test A
Starting test B
Finishing test B
Finishing test A
```

**Question**: Is this acceptable? The existing stderr streaming has the same behavior, but users may have stronger expectations about stdout ordering. At minimum, this should be documented.

**Suggestion**: Consider prefixing worker output with worker ID for debugging clarity, or documenting that output from concurrent workers will interleave.

### 2. Empty Line Handling

*Location*: `rspec/parser.go` lines 146-153

```go
// Not a JSON line - return as raw output
if line != "" {
    notifications = append(notifications, types.OutputNotification{
        Event:   types.RawOutput,
        Content: line,
    })
}

return notifications, false // Line was not consumed
```

Empty lines return `consumed=false` but no notification is added. The current `stream_helper.go` code will stream these empty lines:

```go
if !consumed {
    collector.AddNotification(types.OutputNotification{...})
    if outputChan != nil && streamStdout {
        outputChan <- OutputMessage{...}
    }
}
```

This means empty lines get streamed to stdout and also added to the collector. However, the RSpec parser already handles this by not adding a notification for empty lines. The asymmetry is confusing.

**Suggestion**: Either:
* Make `stream_helper.go` skip empty lines for streaming: `if !consumed && line != "" && streamStdout`
* Or document that empty lines will appear in real-time output

---

## Suggestions for Improvement

### 1. API Design: Consider Using a Configuration Object

*Location*: `stream_helper.go` line 38-45

The function signature is getting long:

```go
func streamTestOutput(
    stdout, stderr io.Reader,
    parser types.TestOutputParser,
    collector *TestCollector,
    outputChan chan<- OutputMessage,
    workerIndex int,
    streamStdout bool, // <-- new parameter
) (stderrOutput string)
```

While adding a single boolean is fine for now, if you anticipate adding more streaming options (like controlling stderr streaming, or adding worker prefixes), consider grouping these into a config struct:

```go
type StreamConfig struct {
    StreamStdout bool
    // Future: StreamStderr bool, WorkerPrefix string, etc.
}
```

**Verdict**: Keep as-is for now, but be aware of the trajectory.

### 2. Comment in OutputMessage Should Stay Organized

*Location*: `result.go` line 36

```go
Type     string // "dot", "failure", "pending", "error", "stderr", "stdout"
```

Minor: consider grouping by category:
* Progress: `"dot"`, `"failure"`, `"pending"`
* Raw output: `"error"`, `"stderr"`, `"stdout"`

Not important, just a nit.

---

## Edge Cases Analysis

| Edge Case | Risk Level | Notes |
|-----------|------------|-------|
| Mixed puts/progress timing | Low | Channel-based design ensures per-worker ordering |
| Very long stdout lines | Low | `ScannerBufferSize` (256KB) handles this; truncation is reasonable failure |
| Binary output | Low-Medium | Could garble terminal with control sequences (same risk as stderr today) |
| Concurrent stdout | Medium | Output interleaves but won't corrupt - document this |

---

## Testing Recommendations

Add these integration tests:

1. **Basic stdout streaming for RSpec**:
   * Create a spec with `puts "Hello from test"` inside an `it` block
   * Verify "Hello from test" appears in output

2. **Stdout doesn't duplicate for Minitest**:
   * Create a Minitest test with `puts "Hello"`
   * Verify "Hello" appears exactly once (not twice)

3. **Stdout with multiple workers**:
   * Create multiple slow specs each with distinct puts output
   * Run with `-n 2` or more
   * Verify all puts output appears (order may vary)

4. **Stdout interleaved with progress indicators**:
   * Create spec: `puts "before"; expect(true).to be true; puts "after"`
   * Verify both "before" and "after" appear, with a dot between them

5. **Empty stdout** (edge case):
   * Create spec with `puts ""` or `puts` (no args)
   * Verify no crash, empty line handling is sensible

---

## Consistency with Stderr Streaming

The implementation is consistent with existing stderr handling:

| Aspect | stderr | stdout (new) |
|--------|--------|--------------|
| Real-time streaming | Yes | Yes (RSpec only) |
| Message type | "stderr" | "stdout" |
| Output destination | `os.Stderr` | `os.Stdout` |
| Also stored | In `stderrBuilder` | In `collector` via `AddNotification` |

The only asymmetry is the conditional nature of stdout streaming, which is intentional and well-documented.

---

## Future Compatibility: Minitest Stdout Streaming

**Assessment**: This implementation makes it *easier* to eventually fix Minitest stdout streaming.

The infrastructure is now in place:
1. "stdout" message type exists
2. `outputAggregator` handles it
3. The `streamStdout` parameter is the toggle point

To enable Minitest streaming in the future, you would need to:
1. Make the Minitest parser return `consumed=true` for progress lines (`.`, `F`, `E`, `S`)
2. Set `streamStdout=true` for Minitest

The current comment accurately describes the blocker:
> Minitest returns consumed=false for everything

Once Minitest's parser is improved to properly consume progress/summary lines, you can flip the switch.

---

## Final Recommendations

1. **Document the interleaving behavior** - either in code comments or user-facing docs. Users should know that stdout from parallel workers may interleave.

2. **Consider the empty line case** - decide if empty lines should be streamed or skipped, and make the behavior consistent.

3. **Add integration tests** - the testing recommendations above will serve as guardrails.

4. **Ship it** - the implementation is solid. The issues raised are minor and can be addressed in follow-up work.

---

## Files Referenced

| File | Lines | Description |
|------|-------|-------------|
| `plur/stream_helper.go` | 66-78 | New stdout streaming logic |
| `plur/runner.go` | 226-228, 327-329 | streamStdout flag, aggregator case |
| `plur/result.go` | 36 | OutputMessage comment |
| `plur/rspec/parser.go` | 145-154 | RSpec consumed semantics |
| `plur/minitest/output_parser.go` | 99-144 | Minitest consumed semantics |
