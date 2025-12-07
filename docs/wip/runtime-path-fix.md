# WIP: Plur Performance Investigation

## Status: In Progress

## Original Problem

Plur was 16% slower than turbo_tests on the example-project suite.

## Issue 1: Runtime Path Mismatch (FIXED)

**Cause:** RSpec outputs paths with `./` prefix, glob discovery returns paths without.

**Impact:** 100% cache miss on runtime lookups, essentially random file distribution.

**Fix:** Strip `./` prefix in parser.go (commits 3ffd3cc2, 575194db, etc.)

**Result:** Now at 100% hit rate, but overhead persists.

## Issue 2: 6.4s CPU Overhead (INVESTIGATING)

### Benchmark Data (1 worker, example-project suite)

```
turbo_tests -n 1: 35.95s (User: 21.7s, Sys: 11.8s)
plur -n 1:        42.25s (User: 28.1s, Sys: 11.9s)
                         -------
                  +6.4s extra CPU in plur
```

* System time identical → overhead is NOT I/O or process spawning
* Extra CPU is in the plur Go binary, not in Ruby/RSpec

### rspec invocation comparison

**turbo_tests:**
```
rspec --format TurboTests::JsonRowsFormatter \
      --format ParallelTests::RSpec::RuntimeLogger --out tmp/turbo_rspec_runtime.log \
      spec/...
```

**plur:**
```
bin/rspec -r ~/.plur/formatter/json_rows_formatter.rb \
          --format Plur::JsonRowsFormatter \
          --force-color --tty \
          spec/...
```

### Formatter differences

| Feature | Plur | TurboTests |
|---------|------|------------|
| Events registered | 12 (includes dump_*) | 9 |
| Backtrace handling | RSpec's filter | Raw capture |
| Extra fields | file_path, line_number, run_time | Minimal |

## Issue 2 ROOT CAUSE FOUND: bin/rspec binstub overhead

### Direct comparison (example-project project, single process)

```
rspec (gem command):     35.27s (User: 21.4s)
bin/rspec (binstub):     41.85s (User: 28.1s)
                         ------
Binstub overhead:        +6.6s CPU
```

**The example-project project's binstub adds 6.6 seconds of CPU overhead!**

### Why this affects plur but not turbo_tests

* **turbo_tests** uses `rspec` (the gem's command)
* **plur** uses `bin/rspec` because example-project's `.plur.toml` configures: `cmd = ["bin/rspec"]`

### Solution options

1. **Change example-project's `.plur.toml`** to use `rspec` instead of `bin/rspec` (quick fix for example-project)
2. **Add `--cmd` flag to plur** to override the configured command (general solution)
3. **Document the binstub tradeoff** - binstubs ensure bundler context but add overhead

### Verification: CONFIRMED ✓

After changing example-project's `.plur.toml` from `cmd = ["bin/rspec"]` to `cmd = ["rspec"]`:

```
turbo_tests -n 3: 17.1s
plur -n 3:        17.2s
                  -----
Difference:       0.2% (statistically equivalent!)
```

**Plur is now equal to turbo_tests on example-project.**

## Changes Made

### 1. Fixed path normalization in parser (DONE)

`plur/rspec/parser.go` - Strip `./` prefix when parsing RSpec JSON:
* Line 110: `filePath := strings.TrimPrefix(msg.ExampleGroup.FilePath, "./")`
* Line 183: `FilePath: strings.TrimPrefix(ex.FilePath, "./")`

### 2. Removed unnecessary mutex from RuntimeTracker (DONE)

`plur/runtime_tracker.go` - The tracker is only used single-threaded after all workers complete, so mutex was unnecessary overhead.

### 3. Added hit/miss logging for runtime data (DONE)

`plur/grouper.go` - Added debug logging to show runtime data lookup effectiveness:
```
DEBUG - runtime data lookup hits=50 misses=0 hit_rate=100
```

### 4. Removed unused variable (DONE)

`plur/grouper.go` - Removed unused `totalRuntime` variable.

### 5. Added grouper unit tests (DONE)

`plur/grouper_test.go` - New test file with:
* `TestGroupSpecFilesByRuntime_UsesRuntimeData`
* `TestGroupSpecFilesByRuntime_SlowestFileIsolated`
* `TestGroupSpecFilesByRuntime_BalancedDistribution`

### 6. Updated integration tests (PARTIAL)

`spec/integration/plur_spec/runtime_tracking_spec.rb` - Updated to expect paths without `./` prefix. **PASSING**

`spec/integration/plur_spec/trace_output_spec.rb` - **NEEDS UPDATE** - Still expects `./` prefix in trace output.

## Remaining Work

1. Update `trace_output_spec.rb` to expect paths without `./` prefix
2. Run full test suite to verify no other tests affected
3. Re-run benchmarks to measure actual performance impact
## PRIMARY GOAL

Find out why example-project regressed to ~15% **slower** than turbo tests at _some_ point in the past -- we are not sure the runtime data is a cause, but we should fix it regardless.

## Verification

After clearing old runtime data and running plur:
```bash
$ plur --dry-run -n 8 --debug 2>&1 | grep "runtime data"
DEBUG - runtime data lookup hits=50 misses=0 hit_rate=100
```

The slowest specs are now properly isolated:
```
Worker 0: spec/integration/cli_integration_spec.rb (18.5s) - ALONE
Worker 1: spec/integration/cli_integration_examples_spec.rb (12.9s) - ALONE
Worker 2: spec/integration/pkg/pkg_compare_integration_spec.rb (5.1s) - ALONE
```

## Benchmark Results

Pre-fix: plur 16.1% slower than turbo_tests
Post-fix: Still measuring (grouping is correct but ~3s constant overhead remains)

The 3s overhead is a separate issue - not related to file distribution.
