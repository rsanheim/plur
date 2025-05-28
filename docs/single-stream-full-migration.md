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

### Phase 2: Full Cutover (Current)
- [ ] Remove old `RunSpecFile` function
- [ ] Rename `RunSpecFileWithStreamingJSON` → `RunSpecFile`
- [ ] Remove `--streaming-json` flag
- [ ] Remove `RunSpecsInParallelWithFormatter` wrapper
- [ ] Update dry-run to show correct formatter commands
- [ ] Clean up unused JSON file handling code

### Phase 3: Optimization
- [ ] Profile formatter caching overhead
- [ ] Consider embedding formatter differently (go:embed vs string)
- [ ] Optimize JSON parsing (pre-allocate buffers?)
- [ ] Remove mutex locking where possible

### Phase 4: Documentation
- [ ] Update README with new architecture
- [ ] Document formatter specification
- [ ] Add performance tuning guide

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

## Decision: Proceed with Full Migration

The streaming JSON formatter is working correctly and the dual formatter approach has no advantages. Let's remove the complexity and commit to the better architecture.