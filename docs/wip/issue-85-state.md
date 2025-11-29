# Issue #85 Implementation State

## Problem
stdout from tests (like `puts`) is swallowed and never displayed.

## Plan
See `/Users/rsanheim/.claude/plans/refactored-splashing-cloud.md`

## Two-Phase Approach

### Phase 1: Unify Minitest failure handling with RSpec
Make Minitest emit `FormattedFailuresNotification` like RSpec does, so both frameworks use the same code path in `PrintResults`.

### Phase 2: Stream unconsumed stdout
Add stdout streaming via `outputChan` (like stderr already does).

## Completed

1. **Diagram updated** (committed as `89f6c624`):
   - `docs/architecture/test-processing-flow.md` - simplified, notes issue #85

2. **Minitest parser modified** (not committed):
   - `plur/minitest/output_parser.go` lines 120-141
   - Failure lines now return `consumed=true`
   - Emits `FormattedFailuresNotification` with failure buffer content

## Remaining Tasks

1. **result.go** (lines 124-136):
   Remove Minitest special case, use unified `FormattedFailures` path:
   ```go
   // Remove this block:
   if currentJob.IsMinitestStyle() && summary.HasFailures {
       for _, result := range summary.AllResults {
           if result.State == types.StateFailed && result.Output != "" {
               fmt.Print(result.Output)
           }
       }
   } else if summary.HasFailures && summary.FormattedFailures != "" {

   // Replace with:
   if summary.HasFailures && summary.FormattedFailures != "" {
   ```

2. **stream_helper.go** (lines 62-68):
   Stream unconsumed stdout to `outputChan`:
   ```go
   if !consumed {
       collector.AddNotification(types.OutputNotification{
           Event:   types.RawOutput,
           Content: line,
       })
       // ADD THIS:
       if outputChan != nil {
           outputChan <- OutputMessage{
               WorkerID: workerIndex,
               Type:     "stdout",
               Content:  line,
           }
       }
   }
   ```

3. **runner.go** (in `outputAggregator` function, around line 320):
   Add "stdout" case:
   ```go
   case "stdout":
       fmt.Fprintln(os.Stdout, msg.Content)
   ```

4. **Testing**:
   - Run existing tests to verify nothing broke
   - Create integration tests for stdout streaming

## Key Insight
The `consumed` flag now has clear semantics:
- `consumed=true` → parser handled this line, don't stream it
- `consumed=false` → parser didn't consume this line, should be streamed

For Minitest, failure lines are now consumed (will appear via `FormattedFailures` at end).
For RSpec, JSON lines are consumed (structured data), non-JSON lines are not consumed (should stream).
