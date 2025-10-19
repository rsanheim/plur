# Bundler Overhead and Test Optimization Research

## Executive Summary

Bundler adds significant overhead for test execution, especially when running multiple test processes. This research explores optimization strategies used by successful tools and proposes enhancements for Plur.

## Benchmark Results: Bundler Overhead

### Single File Execution (calculator_spec.rb)
```
rspec:              137.1 ms ± 12.3 ms  (baseline)
bin/rspec:          152.7 ms ±  3.9 ms  (11% slower)
bundle exec rspec:  172.0 ms ± 11.9 ms  (25% slower)

Overhead:
- bin/rspec:        +15.6 ms (+11%)
- bundle exec:      +34.9 ms (+25%)
```

### Multiple Files (5 test files)

#### Sequential Execution (worst case)
```
for f in *.rb; do rspec $f; done:        648.1 ms
for f in *.rb; do bundle exec rspec $f; done: 832.5 ms

Bundle exec overhead: 184.4 ms (28% slower)
```

#### Batch Execution (best case)
```
rspec spec/calculator_*_spec.rb:        132.5 ms
bundle exec rspec spec/calculator_*_spec.rb: 169.8 ms

Bundle exec overhead: 37.3 ms (28% slower)
```

### Key Finding
The 25-28% overhead from `bundle exec` is consistent across different execution patterns. For parallel test runners like Plur that spawn multiple processes, this compounds quickly.

## Analysis of Existing Solutions

### 1. test-queue (GitHub/Shopify)

**Architecture:**
- Uses `fork(2)` to create worker processes from a preloaded master
- Master process loads Rails/test environment once
- Workers inherit the loaded environment via fork's copy-on-write

**Key Optimizations:**
- Avoids Ruby interpreter startup for each worker
- Skips Bundler.setup for workers (already done in master)
- Uses Unix domain sockets for work distribution
- Provides `after_fork` hooks for connection resets

**Performance Impact:**
- Eliminates ~130ms Ruby startup per worker
- Removes ~35ms bundle exec overhead per worker
- Near-linear scaling with CPU cores

### 2. Spring (Rails)

**Architecture:**
- Maintains persistent background Rails process
- Uses `Process.fork()` for instant command execution
- Automatically reloads on code changes

**Key Features:**
- Zero configuration - automatic start/stop
- Watches files for changes and restarts when needed
- Integrates with Rails' code reloading

**Performance Impact:**
- Boot time reduction: 50-75% on large apps
- Eliminates entire boot sequence after first run

### 3. parallel_tests

**Architecture:**
- Process-level parallelism (not forking)
- Each process gets dedicated database
- Uses TEST_ENV_NUMBER for environment isolation

**Optimizations:**
- Runtime-based load balancing
- Configurable grouping strategies
- Database preparation parallelization

### 4. Bootsnap (Shopify)

**What it does:**
- Caches compiled Ruby bytecode (YARV instructions)
- Optimizes `require` path resolution
- Caches YAML/JSON parsing

**Performance Impact:**
- 30-75% boot time reduction
- Compile cache: 25% of gains
- Path cache: 75% of gains

## Proposed Plur Optimizations

### Short-term: Smart Command Selection

Implement automatic command optimization based on environment detection:

```go
func (t *Task) OptimizeCommand() string {
    // Priority order for best performance
    
    // 1. Check for binstub (fastest if properly configured)
    if exists("bin/rspec") && !usesSpring("bin/rspec") {
        return "bin/rspec"
    }
    
    // 2. Check if rspec is available directly
    if canExecute("rspec") && hasGemInstalled("rspec") {
        // Verify it's the correct version from Gemfile.lock
        if rspecVersionMatches() {
            return "rspec"
        }
    }
    
    // 3. Fall back to bundle exec (safest but slowest)
    return "bundle exec rspec"
}
```

**Benefits:**
- 25% performance improvement when direct execution is possible
- No configuration needed - automatic detection
- Safe fallback to bundle exec when necessary

### Medium-term: Worker Process Pooling

Implement a worker pool that reuses processes:

```go
type WorkerPool struct {
    workers   []*Worker
    workQueue chan TestFile
    master    *MasterProcess
}

func (wp *WorkerPool) Start() {
    // Start N workers
    for i := 0; i < wp.size; i++ {
        worker := wp.spawnWorker()
        worker.start()
    }
}

func (wp *WorkerPool) ExecuteTest(file TestFile) {
    // Reuse existing worker instead of spawning new process
    wp.workQueue <- file
}
```

**Benefits:**
- Amortize startup cost across multiple test files
- Reduce process spawn overhead
- Better CPU cache utilization

### Long-term: Fork-based Preloading (Ruby Helper)

Create a Ruby preloader helper similar to test-queue:

```ruby
# plur_preloader.rb
class PlurPreloader
  def initialize
    # Load bundler and test framework once
    require 'bundler/setup'
    require 'rspec/core'
    
    @loaded_at = Time.now
  end
  
  def fork_worker(test_files)
    Process.fork do
      # Reset connections after fork
      ActiveRecord::Base.clear_all_connections! if defined?(ActiveRecord)
      
      # Run tests in forked process
      RSpec::Core::Runner.run(test_files)
    end
  end
end
```

**Implementation Strategy:**
1. Create optional Ruby gem `plur-preloader`
2. Detect if gem is available
3. Use preloader for Ruby tests when present
4. Fall back to current approach otherwise

**Expected Performance:**
- 50-70% reduction in startup time for parallel workers
- Near-instant test start after first run
- Scales linearly with CPU cores

## Implementation Recommendations

### Phase 1: Command Optimization (1-2 weeks)
1. Implement smart command selection
2. Add detection for binstubs
3. Verify gem availability without bundle exec
4. Add configuration option to force specific command

### Phase 2: Process Pooling (3-4 weeks)
1. Design worker pool architecture
2. Implement process reuse for same test type
3. Add connection reset handling
4. Benchmark and tune pool size

### Phase 3: Preloader Integration (4-6 weeks)
1. Create plur-preloader Ruby gem
2. Implement fork-based execution
3. Add file watching for code reloads
4. Integrate with main plur binary

## Benchmarking Setup

For accurate performance testing:

```bash
# Create consistent test environment
hyperfine --warmup 5 --min-runs 30 \
  'rspec spec/file.rb' \
  'bundle exec rspec spec/file.rb' \
  'bin/rspec spec/file.rb' \
  'plur spec/file.rb'
```

## Conclusion

Bundler overhead is significant (25-28%) and compounds in parallel test execution. By implementing smart command selection (short-term) and process pooling (medium-term), Plur can achieve 25-50% performance improvements. A fork-based preloader (long-term) could deliver 50-70% improvements, matching or exceeding specialized tools like test-queue.

The key insight: **avoiding repeated process spawning and bundler initialization is more important than the parallelization strategy itself**. Even single-threaded execution with preloading can outperform naive parallel execution with full startup overhead.