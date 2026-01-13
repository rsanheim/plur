# Minitest Progress Error Fix

Fix the mismatch between minitest 'E' progress type and output aggregator handling.

## Context

Minitest uses 'E' to indicate an error (exception during test execution, not assertion failure). The current code maps this to "error" which conflicts with the output aggregator's interpretation.

**The Bug:**

1. Minitest outputs 'E' for errors
2. `minitest/output_parser.go:47-50` maps 'E' → `"error"`
3. `stream_helper.go:64-72` sends `OutputMessage{Type: "error", Content: ""}` (progress events have empty Content)
4. `runner.go:325-327` handles "error" as: `fmt.Fprintln(os.Stderr, msg.Content)`
5. **Result: Blank lines printed to stderr instead of 'E' glyph**

**Code Flow:**

```
minitest output 'E'
  → output_parser.NotificationToProgress() returns "error"
  → stream_helper sends OutputMessage{Type: "error", Content: ""}
  → outputAggregator case "error": prints blank line to stderr
```

**Expected Behavior:**

* 'E' should display as a progress glyph (like '.', 'F', '*')
* Currently: dot, failure, pending have dedicated handlers
* "error" type is overloaded for both progress AND error messages

**String-based type system:**

`result.go:37-42` uses `Type string` with magic strings. This makes mismatches easy to introduce and hard to catch at compile time.

## Success Criteria

* [ ] Minitest 'E' outputs display correctly as 'E' glyph
* [ ] Error messages (with Content) still print to stderr
* [ ] Existing progress glyphs (. F *) unchanged
* [ ] Consider typed enum for OutputMessage.Type (optional improvement)
* [ ] Add test verifying 'E' output

## Task List

### Option A: Add distinct "error_progress" type

* [ ] In `minitest/output_parser.go:47-50`, map 'E' to `"error_progress"`
* [ ] In `runner.go:302-336`, add case for "error_progress":
  ```go
  case "error_progress":
      fmt.Print("E")
  ```
* [ ] Keep "error" for actual error messages with content
* [ ] Update `result.go` comment to include new type

### Option B: Map 'E' to "failure" temporarily

* [ ] In `minitest/output_parser.go:47-50`, map 'E' to `"failure"` (displays as 'F')
* [ ] Simpler fix but loses distinction between failures and errors
* [ ] Good temporary fix if error glyph support is deferred

### Option C: Check Content in "error" handler (not recommended)

* [ ] In `runner.go:325-327`, check if Content is empty
* [ ] If empty, treat as progress glyph
* [ ] Brittle: overloads meaning based on content presence

### Recommended: Option A

* [ ] Implement "error_progress" type
* [ ] Add unit test in `minitest/output_parser_test.go` verifying 'E' → "error_progress"
* [ ] Add integration test verifying 'E' displays correctly

### Optional Improvement: Typed enum

* [ ] Define `type OutputMessageType uint8` in `result.go`
* [ ] Define constants: `TypeDot`, `TypeFailure`, `TypePending`, `TypeError`, `TypeErrorProgress`, `TypeStderr`, `TypeStdout`
* [ ] Change `OutputMessage.Type` from `string` to `OutputMessageType`
* [ ] Update all producers and consumers
* [ ] Compile-time safety against type mismatches

## Validation

* [ ] Run `bin/rake test` with a fixture that has minitest errors
* [ ] Verify 'E' displays in progress output
* [ ] Verify other glyphs still work (. F *)
* [ ] Run `bin/rake` for full test suite

## Files to Modify

* `plur/minitest/output_parser.go` - Change 'E' mapping
* `plur/runner.go` - Add error_progress handler
* `plur/result.go` - Update type comment (and optionally add enum)
* `plur/minitest/output_parser_test.go` - Add test for 'E' mapping
