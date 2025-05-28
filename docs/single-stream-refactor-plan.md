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

### Step 1: Formatter Integration ✅
- [x] Decide on formatter distribution method:
  - **Option A (Implemented)**: Embed formatter as Go string constant, write to XDG cache
  - Formatter cached at `~/.cache/rux/formatters/json_rows_formatter.rb`

### Step 2: Update RunSpecFile Function ✅
- [x] Remove JSON file creation logic (in new implementation)
- [x] Update RSpec command to use custom formatter
- [x] Replace stdout scanner with JSON line parser
- [x] Generate progress output from JSON events

### Step 3: JSON Message Handling ✅
- [x] Create Go structs for JSON message types (JSONMessage in json_message.go)
- [x] Implement streaming JSON parser (ParseJSONMessage)
- [x] Map JSON events to progress indicators

### Step 4: Progress Output Generation ✅
- [x] Remove current stdout progress handling (in new implementation)
- [x] Generate dots/F's from JSON events:
  - `example_passed` → green dot ✅
  - `example_failed` → red F ✅
  - `example_pending` → yellow * ✅
- [x] Maintain colorization logic

### Step 5: Error Handling ✅
- [x] Handle JSON parsing errors gracefully
- [x] Capture partial results if process crashes
- [x] Preserve stderr output for debugging

### Step 6: Full Migration (IN PROGRESS)
- [x] Add feature flag for gradual rollout (--streaming-json)
- [x] Verify all integration tests pass
- [ ] Remove dual formatter implementation entirely
- [ ] Update dry-run output
- [ ] Benchmark performance on larger codebases
- [ ] Document as primary implementation

## Code Changes Overview

### runner.go Changes

```go
// Before
args := []string{"bundle", "exec", "rspec", "--format", "progress", "--format", "json", "--out", jsonFile}

// After (Options A or B - write temp file)
formatterPath := writeFormatterToTemp() // writes embedded string to temp file
args := []string{"bundle", "exec", "rspec", "-r", formatterPath, "--format", "Rux::JsonRowsFormatter"}

// Or Option C (gem installation)
args := []string{"bundle", "exec", "rspec", "-r", "rux/json_rows_formatter", "--format", "Rux::JsonRowsFormatter"}
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
- [ ] Ability to add try a different, more performant formatter option fairly easily

## Next Actions

1. Review and approve this refactoring plan
2. Implement formatter embedding strategy
3. Create proof-of-concept with single spec file
4. Gradually refactor RunSpecFile function
5. Update all integration tests