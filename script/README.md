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
- `rux` binary installed in PATH (for Ruby testing)

### Quick Testing

```bash
./script/get-repo https://github.com/example-org/example-project
cd example-project-*/
rux                    # Run all specs with default workers
rux --workers 4        # Run with 4 workers
PARALLEL_TEST_PROCESSORS=2 rux  # Run with env var override
```

## bench

Benchmarks rux against turbo_tests using hyperfine for performance comparison.

### Usage

```bash
# Benchmark any Ruby project
./script/bench ./rux-ruby
./script/bench ./example-project-1234567890
./script/bench /path/to/ruby/project
```

### What it does

1. **Verifies project** has spec directory and dependencies
2. **Installs turbo_tests** temporarily if not present
3. **Runs hyperfine benchmarks** comparing:
   - `turbo_tests`
   - `rux` (default workers)
   - `rux --workers 4`
   - `rux --workers 8`
4. **Exports results** to markdown and JSON
5. **Cleans up** temporary changes

### Requirements

- `hyperfine` (install with `brew install hyperfine`)
- `rux` binary in PATH
- Ruby project with spec directory

### Output

Creates `bench-results.md` and `bench-results.json` in the project directory with detailed performance comparisons.