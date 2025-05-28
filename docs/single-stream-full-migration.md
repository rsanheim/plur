# Full Migration to Single Stream JSON Formatter

## Decision Summary

After implementing and testing both approaches side-by-side, we're fully committing to the single streaming JSON formatter. The dual formatter approach (progress + JSON file) has fundamental flaws:

1. **2x formatting overhead** - Every example is formatted twice by RSpec
2. **Complex synchronization** - Managing stdout and file I/O separately  
3. **No clear performance benefit** from keeping the old approach
4. **Added complexity** from maintaining two implementations

## Migration Plan

### Phase 1: Remove Dual Implementation ✅
- [x] Implement streaming JSON formatter
- [x] Add JSON parsing for streaming messages
- [x] Verify feature parity with integration tests
- [x] Confirm error reporting works correctly

### Phase 2: Full Cutover ✅
- [x] Remove old `RunSpecFile` function
- [x] Rename `RunSpecFileWithStreamingJSON` → `RunSpecFile`
- [x] Remove `--streaming-json` flag
- [x] Remove `RunSpecsInParallelWithFormatter` wrapper
- [x] Update dry-run to show correct formatter commands
- [x] Clean up unused JSON file handling code

### Phase 2.9: simplify benchmark scripts ✅
- [x] Simplified `script/bench` to output only JSON files with naming pattern `YYYYMMDD-HHMMSS-SHA-project.json`
- [x] Updated `script/bench-checkpoint` to run bench, combine JSON results, and generate markdown summary
- [x] Removed all symlinks, help flags, and extra features from both scripts
- [x] Added GitHub commit link in markdown summary
- [x] Outputs follow requested naming pattern for summary files

### Phase 3: Optimization (In Progress)

**Note**: Run benchmarks directly in terminal for clean results (not through Claude due to performance overhead from terminal layers).

#### Completed Optimizations:
- [x] **Eliminated output lock contention** - Replaced mutex-based output with channel-based aggregator
  - Before: Every test result character required mutex lock/unlock
  - After: Single goroutine handles all output via buffered channel
  - Impact: Scales linearly with 25-30 workers without contention
- [x] **Cached formatter path** - GetFormatterPath() called once at startup instead of per spec file
  - Before: File I/O check for every spec file
  - After: sync.Once ensures single computation
  - Impact: Reduces syscalls in hot path

#### Analyze 'mise' impact
- [ ] analyze impact of ruby version manager for `rux` vs `turbo_tests`
   - [ ] my standard login shell uses 'mise activate`, which adds required tools to PATH
   - [ ] another approach mise offers is to setup shims via `mise activate --shims`
   - [ ] details here: https://mise.jdx.dev/dev-tools/shims.html#overview
   - [ ] is it possible that `mise` causes significant overhead in starting each ruby process for rux, given we are going thru a go binary, to shelling out to ruby, as opposed to turbo_tests which is just ruby that spawns more ruby?
- [ ] My hypothesis here is that we _should_ be able to get rux close to the run time of turbo_tests, even for small suites, and perhaps mise is a factor here. In any case, would be good to rule it out.

#### Remaining Optimizations:
- [ ] Optimize JSON parsing (pre-allocate buffers, faster detection)
- [ ] Pool goroutines instead of creating 2 per spec file
- [ ] Pre-allocate string builders with estimated capacity
- [ ] Consider embedding formatter differently (go:embed vs string)
- [ ] Implement --json flag to save results to files
- [ ] Add full failure summary output

### Phase 4: Documentation
- [ ] Update README with new architecture
- [ ] Document formatter specification
- [ ] Add performance tuning guide
- [ ] Document breaking changes and migration path
- [ ] Add troubleshooting guide for common issues

## Code Changes

### 1. Remove Feature Flag (main.go)
```diff
- &cli.BoolFlag{
-     Name:  "streaming-json",
-     Usage: "Use streaming JSON formatter (experimental)",
-     Hidden: true,
- },
```

### 2. Simplify Runner (runner.go)
```diff
- func RunSpecFile(ctx context.Context, ...) TestResult {
-     // All the dual formatter code
- }

- func RunSpecFileWithStreamingJSON(ctx context.Context, ...) TestResult {
+ func RunSpecFile(ctx context.Context, ...) TestResult {
     // Streaming JSON implementation only
}
```

### 3. Update Dry Run (main.go)
```diff
- args := []string{"bundle", "exec", "rspec", "--format", "progress", "--format", "json", "--out", "/tmp/results.json", "--no-color", file}
+ formatterPath, _ := GetFormatterPath()
+ args := []string{"bundle", "exec", "rspec", "-r", formatterPath, "--format", "Rux::JsonRowsFormatter", "--no-color", file}
```

### 4. Clean Up Parallel Execution
```diff
- func RunSpecsInParallel(...) {
-     return RunSpecsInParallelWithFormatter(..., false)
- }
- 
- func RunSpecsInParallelWithFormatter(..., useStreamingJSON bool) {
+ func RunSpecsInParallel(...) {
     // Direct implementation
}
```

## Benefits of Full Migration

1. **Simpler codebase** - One clear path through the code
2. **Easier optimization** - Can focus on optimizing one approach
3. **Better maintainability** - No feature flags or conditional logic
4. **Clearer performance profile** - Direct comparison with turbo_tests

## Performance Expectations

Based on initial benchmarks with rux-ruby (11 specs):
- Current streaming implementation shows similar performance to dual formatter
- This is expected because formatter overhead is small for tiny test suites
- Real benefits should appear with larger test suites where 2x formatting matters more

Next benchmark targets:
- Test with 100+ spec files
- Test with 1000+ examples
- Profile CPU usage per process
- Measure memory usage differences

## Risks and Mitigation

1. **Breaking Change**: Users expecting specific output format
   - Mitigation: Major version bump, clear migration guide

2. **Formatter Distribution**: XDG cache might have permission issues  
   - Mitigation: Fallback to project-local tmp if cache fails

3. **Unknown Edge Cases**: Some RSpec plugins might not work
   - Mitigation: Test with common gems (simplecov, etc.)

4. **Error Scenarios**: Need to handle various failure modes
   - Test with syntax errors in spec files
   - Test with missing gems/dependencies
   - Test with RSpec configuration errors

## Success Criteria

- [ ] CPU usage matches turbo_tests (within 10%)
- [ ] No visible pausing during parallel execution
- [x] All existing integration tests pass (except 2 expecting old error format)
- [x] Real-time progress output maintained
- [x] Error messages appear immediately
- [x] Ability to add different formatter options easily

## Implementation Details

### Formatter Distribution
- Implemented as Go string constant embedded in binary
- Cached at `~/.cache/rux/formatters/json_rows_formatter.rb`
- Written once per user, reused across projects

### JSON Streaming Format
Each line is prefixed with `RUX_JSON:` followed by a JSON object:
- `{"type":"start","count":58,"load_time":0.123}`
- `{"type":"example_passed","description":"...","location":"..."}`
- `{"type":"example_failed","description":"...","exception":{...}}`
- `{"type":"close"}`

## Decision: Proceed with Full Migration ✅

The streaming JSON formatter is working correctly and the dual formatter approach has no advantages. We've successfully removed the complexity and committed to the better architecture. The migration is complete as of commit f244ea087.