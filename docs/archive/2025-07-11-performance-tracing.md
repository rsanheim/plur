# Performance Tracing in Plur

## Overview

Plur includes built-in performance tracing to help analyze execution bottlenecks and understand where time is spent during test runs.

## Usage

Enable tracing with the `--trace` flag:

```bash
plur --trace                  # Run with default workers
plur --trace -n 4            # Run with 4 workers
plur --trace spec/unit/*_spec.rb  # Trace specific files
```

Trace files are written to the repository's tmp directory:
```
Tracing enabled, writing to: /path/to/plur/tmp/plur-traces/plur-trace-20250528-044523.json
```

## Analyzing Traces

Use the included analyzer script:

```bash
# Analyze the most recent trace
ruby plur/analyze_trace.rb

# Analyze a specific trace with verbose output
ruby plur/analyze_trace.rb -v /tmp/plur-traces/plur-trace-20250528-044523.json
```

## Trace Events

The following operations are traced:

- **file_discovery**: Time to find all spec files
- **worker_pool_init**: Worker goroutine initialization
- **formatter.get_path**: Formatter path resolution (cached after first call)
- **process_spawn**: Time to spawn Ruby subprocess
- **ruby_first_output**: Time from spawn until first output
- **rspec_loaded**: When RSpec reports it has finished loading
- **run_spec_file**: Total time to run each spec file
- **run_specs_parallel**: Overall parallel execution time
- **main.total_execution**: Complete execution time

## Tracing Implementation

The tracing system uses an asynchronous writer to ensure minimal impact on runtime performance:
- Events are sent to a buffered channel (capacity: 1000)
- A background goroutine handles JSON marshaling and file I/O
- Non-blocking sends ensure tracing never slows down execution
- Overhead is typically <1% even with tracing enabled

## Performance Insights

Based on tracing analysis:

### Startup Breakdown (typical values)
- Process spawn: ~1ms per process
- Ruby startup to first output: ~190ms
- RSpec reported load time: ~45ms
- Gap (Ruby interpreter + Bundler + gems): ~145ms

### Overhead Analysis
- Small test suite (2 files): ~8-10% overhead
- Medium test suite (10-50 files): ~1-5% overhead  
- Large test suite (100+ files): <1% overhead

### Key Findings

1. **Ruby startup dominates**: The ~190ms Ruby startup time is the primary bottleneck for small test suites.

2. **Formatter registration is minimal**: The plur formatter is registered once per process at ~6ms, which is negligible compared to Ruby startup.

3. **Plur overhead scales well**: As test suites grow larger, plur's coordination overhead becomes a smaller percentage of total time.

4. **Process spawn is efficient**: Creating new processes takes only ~1ms on modern systems.

## Example Analysis Output

```
Operation Summary:
--------------------------------------------------------------------------------
Operation                         Count  Total(ms)    Avg(ms)    Min(ms)    Max(ms)
--------------------------------------------------------------------------------
run_spec_file                         2    6435.18    3217.59    3188.74    3246.44
main.total_execution                  1    3270.36    3270.36    3270.36    3270.36
run_specs_parallel                    1    3257.56    3257.56    3257.56    3257.56
get_formatter_path                    2      12.38       6.19       6.18       6.19
process_spawn                         2       1.39       0.70       0.68       0.71

Timing Analysis:
--------------------------------------------------------------------------------
Total execution time:        3270.36 ms
Parallel execution time:     3257.56 ms
Longest spec file:           3246.44 ms
Plur overhead:                  11.12 ms (0.3%)
```

## Comparison with turbo_tests

When comparing similar workloads:
- Both tools have similar Ruby startup overhead (~190ms)
- Both report similar RSpec load times (~45ms)
- The performance difference comes from implementation details in work distribution and output handling

## Optimizing Performance

Based on trace analysis:

1. **For small test suites**: The Ruby startup overhead is unavoidable. Consider grouping more tests per file.

2. **For large test suites**: Experiment with worker counts. Often 4-8 workers is optimal, not cores-2.

3. **Cache warming**: The formatter is cached after first use, so subsequent runs in the same session are faster.