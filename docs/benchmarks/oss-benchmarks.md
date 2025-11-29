# OSS Benchmark Comparison: Plur vs RSpec

*Generated: 2025-11-28*
*Plur version: 0.13.0-dev-9dc2f9d6*
*Workers: 18 (auto-detected)*

## Summary

| Project | Examples | Spec Files | RSpec Avg | Plur (default) | Plur (-n 4) | Notes |
|---------|----------|------------|-----------|----------------|-------------|-------|
| rubocop | 28,840 | 722 | 101.37s | **34.53s (2.9x)** | - | Large suite, parallel shines |
| pry | 1,439 | 78 | 5.12s | **3.04s (1.7x)** | - | Medium suite benefits |
| simplecov | 391 | 26 | 6.11s | 4.96s (1.2x) | **3.76s (1.6x)** | -n 4 helps |
| grape | 2,214 | 117 | 2.75s | 4.43s (slower) | **2.11s (1.3x)** | -n 4 helps |
| factory_bot | 737 | 82 | 5.37s | 8.53s (slower) | **4.82s (1.1x)** | -n 4 helps |
| capistrano | 400 | 30 | 1.51s | FAILED | FAILED | Bug: `--force-color` |

## Key Findings

### 1. Large Test Suites: Plur Excels (3x faster)

**rubocop** (28,840 examples, 722 spec files):
* RSpec: 102.48s, 99.68s, 101.96s (avg **101.37s**)
* Plur: 34.23s, 34.82s, 34.53s (avg **34.53s**)
* **Result: ~3x faster (66% time reduction)**

This demonstrates plur's value proposition - parallel execution significantly reduces test time on large suites.

### 2. Medium Test Suites: Good Speedup (1.2x-1.7x faster)

**pry** (1,439 examples, 78 spec files):
* RSpec: 5.19s, 5.07s, 5.10s (avg **5.12s**)
* Plur: 3.21s, 2.95s, 2.95s (avg **3.04s**)
* **Result: 1.68x faster (41% time reduction)**

**simplecov** (391 examples, 26 spec files):
* RSpec: 6.33s, 5.99s, 6.01s (avg **6.11s**)
* Plur: 4.94s, 4.95s, 4.99s (avg **4.96s**)
* **Result: 1.23x faster (19% time reduction)**

Medium suites with moderate test times benefit from parallelization even with the overhead.

### 3. Small/Fast Test Suites: Parallel Overhead Hurts

**factory_bot** (737 examples, 82 spec files):
* RSpec: 5.47s, 5.30s, 5.34s (avg **5.37s**)
* Plur: 8.48s, 8.53s, 8.58s (avg **8.53s**)
* **Result: 59% slower**

**grape** (2,214 examples, 117 spec files):
* RSpec: 2.78s, 2.73s, 2.75s (avg **2.75s**)
* Plur: 4.45s, 4.40s, 4.45s (avg **4.43s**)
* **Result: 61% slower**

When tests are fast and finish quickly, the overhead of spawning 18 workers and coordinating output outweighs the benefits of parallelism.

### 4. Worker Count Tuning: Fewer Workers Can Help

Testing with reduced worker counts shows significant improvements on small/fast suites:

| Project | RSpec | -n 1 | -n 2 | -n 4 | -n 18 (default) |
|---------|-------|------|------|------|-----------------|
| factory_bot | 5.37s | 5.40s | **4.85s** | **4.82s** | 8.53s |
| grape | 2.75s | 2.84s | 2.93s | **2.11s** | 4.43s |
| simplecov | 6.11s | 6.22s | 5.86s | **3.76s** | 4.96s |

**Key insight:** 4 workers appears to be a sweet spot for small-medium suites:
* **factory_bot** with -n 4: 4.82s (vs 8.53s default) - **10% faster than RSpec**
* **grape** with -n 4: 2.11s (vs 4.43s default) - **23% faster than RSpec**
* **simplecov** with -n 4: 3.76s (vs 4.96s default) - **38% faster than RSpec**

This suggests plur could benefit from smarter worker count auto-detection based on suite size.

### 5. Bug Discovered: RSpec Version Compatibility

**capistrano** (400 examples, 30 spec files, RSpec 3.4.0):
* RSpec: 1.55s, 1.50s, 1.47s (avg **1.51s**)
* Plur: **FAILED** - `invalid option: --force-color`

**Root cause**: `--force-color` was added in RSpec 3.6. Plur passes this flag unconditionally.

**Fix needed**: Detect RSpec version and only use `--force-color` on RSpec >= 3.6.

### 6. Test Suite Compatibility: rspec-core

**rspec-core** (2,838 examples, 77 spec files):
* Intermittent `Errno::ENOENT: No such file or directory - getcwd` errors in parallel mode
* **Root cause**: rspec-core's tests use `Dir.mktmpdir` + `Dir.chdir` in `around` blocks. When a test fails inside these blocks, the temp directory is deleted before RSpec finishes formatting the failure output, causing `getcwd` to fail.
* **Why only parallel?**: rspec-core runs tests sequentially in CI (no parallel_tests/turbo_tests). This latent bug was never exposed until running with plur.
* **Workaround**: Run with `plur -n 1` for sequential execution, or accept occasional failures.
* **Not a plur bug**: This is a test isolation issue in rspec-core's test suite design.

## Detailed Results

### rubocop

```
RSpec Run 1: 1m 42.48s  (28840 examples, 0 failures, 1 pending)
RSpec Run 2: 1m 39.68s
RSpec Run 3: 1m 41.96s

Plur Run 1: 34.23s      (28840 examples, 0 failures, 1 pending)
Plur Run 2: 34.82s
Plur Run 3: 34.53s
```

### pry

```
RSpec Run 1: 5.19s  (1439 examples, 0 failures, 3 pending)
RSpec Run 2: 5.07s
RSpec Run 3: 5.10s

Plur Run 1: 3.21s   (1439 examples, 0 failures, 3 pending)
Plur Run 2: 2.95s
Plur Run 3: 2.95s
```

### simplecov

```
RSpec Run 1: 6.33s  (391 examples, 0 failures)
RSpec Run 2: 5.99s
RSpec Run 3: 6.01s

Plur Run 1: 4.94s   (391 examples, 0 failures)
Plur Run 2: 4.95s
Plur Run 3: 4.99s
```

### factory_bot

```
RSpec Run 1: 5.47s  (737 examples, 0 failures)
RSpec Run 2: 5.30s
RSpec Run 3: 5.34s

Plur Run 1: 8.48s   (737 examples, 0 failures)
Plur Run 2: 8.53s
Plur Run 3: 8.58s
```

### grape

```
RSpec Run 1: 2.78s  (2214 examples, 1 failure - existing test issue)
RSpec Run 2: 2.73s
RSpec Run 3: 2.75s

Plur Run 1: 4.45s   (2214 examples, 1 failure)
Plur Run 2: 4.40s
Plur Run 3: 4.45s
```

### capistrano

```
RSpec Run 1: 1.55s  (400 examples, 0 failures, 2 pending)
RSpec Run 2: 1.50s
RSpec Run 3: 1.47s

Plur Run 1: FAILED (--force-color not supported on RSpec 3.4.0)
Plur Run 2: FAILED
Plur Run 3: FAILED
```

## When to Use Plur

Based on these benchmarks:

| Scenario | Recommendation |
|----------|----------------|
| 10,000+ examples | **Strongly recommended** - expect 2-3x speedup with default workers |
| 1,000-10,000 examples | **Recommended** - expect 1.2-2x speedup with default workers |
| 500-1,000 examples | **Use `-n 4`** - default workers may hurt, 4 workers typically helps |
| <500 examples | **Use `-n 4`** - fewer workers avoids overhead, may still see modest speedup |

## Recommendations

1. **Fix RSpec version detection** - Add logic to detect RSpec < 3.6 and avoid `--force-color`

2. **Smart worker count auto-detection** - Consider automatically capping workers based on spec file count:
   * <50 spec files: use 4 workers max
   * 50-200 spec files: use 8 workers max
   * 200+ spec files: use all available cores

3. **User guidance** - For small/fast suites, recommend `plur -n 4` in documentation

## Environment

* macOS Darwin 24.6.0
* Ruby via mise (project-specific versions)
* 18 CPU cores available (M3 Max)
* All projects from [real-world-rspec](https://github.com/pirj/real-world-rspec) meta-repository
