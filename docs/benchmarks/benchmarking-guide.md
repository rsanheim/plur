# Benchmarking Plur Against OSS Projects

This guide explains how to benchmark plur against real-world RSpec projects.

## Prerequisites

* Clone the [real-world-rspec](https://github.com/pirj/real-world-rspec) meta-repository
* Ensure `mise` is installed for Ruby version management
* Build and install plur: `bin/rake install`

## Setup

```bash
# Clone real-world-rspec if not already present
cd ~/src/oss
git clone https://github.com/pirj/real-world-rspec.git

# Create benchmark output directory
mkdir -p /path/to/plur/tmp/oss-benchmarks
```

## Running Benchmarks

For each project, run 3 iterations of both RSpec and plur to get stable averages.

### Single Project Benchmark

```bash
PROJECT=rubocop
BENCH_DIR=/path/to/plur/tmp/oss-benchmarks/$PROJECT

mkdir -p $BENCH_DIR
cd ~/src/oss/real-world-rspec/$PROJECT

# Install dependencies
bundle check || bundle install

# Run RSpec 3 times
for i in 1 2 3; do
  bundle exec rspec --format progress 2>&1 | tee $BENCH_DIR/rspec-run-$i.log
done

# Run plur 3 times (default workers)
for i in 1 2 3; do
  plur 2>&1 | tee $BENCH_DIR/plur-run-$i.log
done

# Run plur with specific worker counts
for workers in 1 2 4 8; do
  plur -n $workers 2>&1 | tee $BENCH_DIR/plur-n$workers.log
done
```

### Extract Timing

```bash
# Extract timing from logs
grep -E "Finished in" $BENCH_DIR/*.log
```

## Recommended Test Projects

### Large suites (expect 2-3x speedup)
* **rubocop** - 28,840 examples, 722 spec files

### Medium suites (expect 1.2-2x speedup)
* **pry** - 1,439 examples, 78 spec files
* **simplecov** - 391 examples, 26 spec files

### Small/fast suites (may be slower due to overhead)
* **factory_bot** - 737 examples, 82 spec files
* **grape** - 2,214 examples, 117 spec files

### Known issues
* **capistrano** - Uses RSpec 3.4.0 which doesn't support `--force-color`
* **graphql-ruby** - Bundle issues (missing rspec-core in Gemfile)
* **rspec** - Meta-gem with no specs

## Analyzing Results

Key metrics to capture:
* Total runtime from "Finished in X seconds"
* Example count and failure count
* File load time (impacts parallel overhead)

Calculate speedup:
```
speedup = rspec_avg / plur_avg
```

## Worker Count Tuning

For projects where default workers cause overhead, test with fewer workers:

```bash
# Test different worker counts
plur -n 1   # Serial (baseline)
plur -n 2   # Minimal parallelism
plur -n 4   # Light parallelism
plur -n 8   # Moderate parallelism
plur        # Default (auto-detect, typically CPU count)
```

## Troubleshooting

### Bundle version conflicts
Some projects require older bundler. Use:
```bash
gem install bundler:2.7.2
bundle _2.7.2_ install
bundle _2.7.2_ exec rspec
```

### RSpec version compatibility
Projects using RSpec < 3.6 will fail with `--force-color` error. This is a known plur bug.

### Missing specs
Some meta-gems (like `rspec` itself) have no spec directory - skip these.
