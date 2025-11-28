# Setup Scripts

## get-repo

Quickly grab any GitHub repository for clean testing without git history.

### Usage

```bash
# Clone any GitHub repo (auto-converts to SSH)
./script/get-repo https://github.com/example-org/example-project

# Clone with custom directory name
./script/get-repo https://github.com/example-org/example-project my-test-dir

# Works with SSH URLs too
./script/get-repo git@github.com:example-org/example-project.git
```

### What it does

1. **Converts HTTPS to SSH** automatically for faster access
2. **Shallow clone** (--depth 1) for fast download
3. **Removes .git directory** for clean testing environment
4. **Auto-generates directory names** with timestamps

### Requirements

- SSH access to GitHub configured
- `plur` binary installed in PATH (for Ruby testing)

### Quick Testing

```bash
./script/get-repo https://github.com/example-org/example-project
cd example-project-*/
plur                    # Run all specs with default workers
plur --workers 4        # Run with 4 workers (or -n for short)
PARALLEL_TEST_PROCESSORS=2 plur  # Run with env var override
```

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