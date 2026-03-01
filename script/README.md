# Setup Scripts

## bench

Benchmarks plur against turbo_tests using hyperfine for performance comparison.

### Usage

```bash
# Benchmark default projects (default-ruby and example-project)
./script/bench

# Benchmark a specific project (--project, or -p for short)
./script/bench --project ./path/to/project

# Benchmark multiple projects
./script/bench --project ./project1 --project ./project2

# Specify worker count (--workers, or -n for short)
./script/bench --project ./project --workers 4

# Create combined summary files
./script/bench --checkpoint

# See all options
./script/bench --help
```

### What it does

1. **Verifies project** has spec directory
2. **Runs hyperfine benchmarks** comparing `turbo_tests` vs `plur` at specified worker count
3. **Exports results** to timestamped JSON files in `results/` directory
4. Optionally creates **checkpoint summaries** combining multiple project results

### Requirements

* `hyperfine` (install with `brew install hyperfine`)
* `plur` binary in PATH (`bin/rake install`)
* `turbo_tests` gem installed in the target project
* Ruby project with spec directory

### Output

Results are saved to `results/` directory with timestamps:
* `{timestamp}-{commit}-{project}.json` - Hyperfine benchmark data
* `{timestamp}-{commit}-summary.md` - Combined summary (with `--checkpoint`)

## benchmark-memory

Runs Go memory benchmarks for plur's critical output handling paths.

### Usage

```bash
# Run all memory benchmarks
./script/benchmark-memory

# Filter to specific benchmarks
./script/benchmark-memory Collector

# Verbose output
./script/benchmark-memory -v
```

### What it does

1. Runs Go benchmarks with `-benchmem` flag
2. Tests memory allocations in TestCollector, StreamHelper, and related code
3. Provides guidance for before/after comparisons using `benchstat`

### Requirements

* Go toolchain
* Optional: `benchstat` for comparing results (`go install golang.org/x/perf/cmd/benchstat@latest`)