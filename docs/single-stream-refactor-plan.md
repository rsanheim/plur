# Single Stream Refactoring Plan for Rux

## Current State Analysis

### Current Architecture (Dual Stream)
1. **RunSpecFile** function:
   - Creates temp JSON file
   - Runs RSpec with `--format progress --format json --out jsonFile`
   - Reads progress dots from stdout in real-time
   - Parses JSON file after completion
   - Mutex locks for coordinating output

2. **Problems**:
   - Double formatting overhead (2x CPU usage)
   - Complex synchronization between stdout and file I/O
   - Mutex contention causing visible pausing
   - Delayed error reporting (only after JSON parsing)

### Proposed Architecture (Single Stream)
1. **RunSpecFile** refactored:
   - No JSON file creation
   - Runs RSpec with custom formatter only
   - Reads streaming JSON from stdout
   - Parses JSON messages in real-time
   - Generates progress dots from JSON events

2. **Benefits**:
   - Eliminates double formatting overhead
   - Simplifies synchronization (one stream per process)
   - Real-time progress and error reporting
   - Reduced mutex contention

## Refactoring Steps

### Step 1: Formatter Integration
- [ ] Decide on formatter distribution method:
  - **Option A (Recommended)**: Embed formatter as Go string constant
  - **Option B**: Write formatter to temp file at runtime
  - **Option C**: Require users to install formatter gem

### Step 2: Update RunSpecFile Function
- [ ] Remove JSON file creation logic
- [ ] Update RSpec command to use custom formatter
- [ ] Replace stdout scanner with JSON line parser
- [ ] Generate progress output from JSON events

### Step 3: JSON Message Handling
- [ ] Create Go structs for JSON message types:
  ```go
  type JSONMessage struct {
      Type    string          `json:"type"`
      Example *ExampleResult  `json:"example,omitempty"`
      Group   *GroupInfo      `json:"group,omitempty"`
      // ... other fields
  }
  ```
- [ ] Implement streaming JSON parser
- [ ] Map JSON events to progress indicators

### Step 4: Progress Output Generation
- [ ] Remove current stdout progress handling
- [ ] Generate dots/F's from JSON events:
  - `example_passed` → green dot
  - `example_failed` → red F
  - `example_pending` → yellow *
- [ ] Maintain colorization logic

### Step 5: Error Handling
- [ ] Handle JSON parsing errors gracefully
- [ ] Capture partial results if process crashes
- [ ] Preserve stderr output for debugging

### Step 6: Testing & Migration
- [ ] Update integration tests
- [ ] Add feature flag for gradual rollout?
- [ ] Benchmark performance improvements
- [ ] Document breaking changes (if any)

## Code Changes Overview

### runner.go Changes

```go
// Before
args := []string{"bundle", "exec", "rspec", "--format", "progress", "--format", "json", "--out", jsonFile}

// After
formatterPath := writeFormatterToTemp() // or embed directly
args := []string{"bundle", "exec", "rspec", "-r", formatterPath, "--format", "Rux::JsonRowsFormatter"}
```

### New JSON Parsing Logic

```go
scanner := bufio.NewScanner(stdout)
for scanner.Scan() {
    line := scanner.Text()
    if strings.HasPrefix(line, "RUX_JSON:") {
        jsonStr := strings.TrimPrefix(line, "RUX_JSON:")
        var msg JSONMessage
        if err := json.Unmarshal([]byte(jsonStr), &msg); err != nil {
            // Handle error
            continue
        }
        handleJSONMessage(msg, colorOutput)
    } else {
        // Non-JSON output (errors, warnings)
        fmt.Fprintln(os.Stderr, line)
    }
}
```

## Risk Mitigation

1. **Backward Compatibility**:
   - Consider keeping file-based JSON as fallback
   - Add `--legacy-formatter` flag if needed

2. **Performance Testing**:
   - Benchmark on various project sizes
   - Test with high concurrency (many workers)
   - Verify memory usage doesn't increase

3. **Error Scenarios**:
   - Test with syntax errors in spec files
   - Test with missing gems/dependencies
   - Test with RSpec configuration errors

## Success Criteria

- [ ] CPU usage matches turbo_tests (within 10%)
- [ ] No visible pausing during parallel execution
- [ ] All existing integration tests pass
- [ ] Real-time progress output maintained
- [ ] Error messages appear immediately

## Next Actions

1. Review and approve this refactoring plan
2. Implement formatter embedding strategy
3. Create proof-of-concept with single spec file
4. Gradually refactor RunSpecFile function
5. Update all integration tests