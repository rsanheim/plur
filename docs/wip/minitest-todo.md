# Minitest TODOs

This is the single source of truth for remaining Minitest support tasks.

## Current Issues to Fix

- [ ] Fix "Running" prelude output for minitest tests - we still say 'spec files' when its 'test files'. Also, if it's really 1 spec or test file, we should never say "in parallel" - that doesn't make sense. If it's one file we can't parallelize.

Example: 
  ```
    > plur test/calculator_test.rb 
    plur version v0.7.6-0.20250628065354-2392e7f82dde
    Running 1 spec files in parallel using 1 workers (20 cores available)...
    Using size-based grouped execution: 1 file across 18 workers
  ```

- [ ] Create an enum for "dot", "fail", "error", etc - and use that everywhere for progress tracking. No more repeating those strings throughout for progress tracking.

- [ ] Handle Error exceptions differently from assertion Failures (currently both counted as failures)

## Future Enhancements

- [ ] Add runtime tracking for better test distribution
- [ ] Support custom Minitest reporters
- [ ] Add Test::Unit support
- [ ] Optimize channel buffer sizes
- [ ] Add framework-specific configuration options

## Testing & Documentation

- [ ] Unit tests for MinitestOutputParser
- [ ] Update user documentation for minitest usage
- [ ] Add minitest examples to README
- [ ] Document framework detection logic
- [ ] Add troubleshooting guide for minitest

## Known Issues

- [ ] No error recovery for malformed output
- [ ] Limited minitest reporter support (only default reporter)

---

Last updated: 2025-01-11