# Usage

## Basic Commands

### Running Tests

```bash
# Run all specs with the default worker count
plur

# Explicit "spec" command (same as default)
plur spec

# Specify number of workers
plur -n 4
plur --workers 8

# Dry run - see what would be executed
plur --dry-run

# Run from another directory (like git -C)
plur -C path/to/project
```

Explicit path inputs are normalized before discovery and execution. For example,
`./spec/models/user_spec.rb` is run as `spec/models/user_spec.rb`.

### Selecting Test Framework

Plur auto-detects your test framework (RSpec or Minitest) based on directory structure, but you can override this:

```bash
# Run RSpec tests explicitly
plur --use=rspec

# Run Minitest tests
plur --use=minitest

# Set default in config file
echo 'use = "rspec"' > .plur.toml
plur  # Now runs RSpec by default
```

**Projects with both spec/ and test/ directories**:

When both exist, plur defaults to RSpec. Use the `--use` flag to select:

```bash
plur                    # Runs RSpec tests (default)
plur --use=rspec        # Explicitly run RSpec tests
plur --use=minitest     # Run Minitest tests
```

Or set a permanent default in `.plur.toml`:

```toml
use = "minitest"  # Override default to use Minitest
```

### Minitest Integration Notes

Plur integrates with Minitest as a standard minitest plugin: a
`minitest/plur_plugin.rb` file (written to `$PLUR_HOME/config/ruby/`) is added
to ruby's load path, where minitest's own plugin discovery finds it. The
plugin replaces minitest's progress and summary reporters with one that
reports structured results to plur; test-written stdout streams through
untouched. On minitest 6, where plugin loading is opt-in, plur's worker
script calls `Minitest.load_plugins` itself.

Setting `MT_NO_PLUGINS=1` (minitest's own plugin opt-out) disables plur's
plugin too; plur then reports zero results while the suite's native output
streams through, with exit codes still reflecting the suite result.

### Excluding Tests From Discovery

Use `--exclude-pattern` (repeatable) to drop matching files from the test plan
before workers run. Patterns use doublestar semantics. Patterns that match no
selected files are ignored.

```bash
# Skip a single file
plur --exclude-pattern 'spec/legacy/old_spec.rb'

# Skip all system specs
plur --exclude-pattern 'spec/system/**/*_spec.rb'

# Multiple patterns OR together
plur --exclude-pattern 'spec/system/**/*_spec.rb' \
     --exclude-pattern 'spec/legacy/**/*_spec.rb'
```

Excludes can also be configured per-job in `.plur.toml`. CLI excludes are
*additive on top of* configured excludes — they do not replace them. See
[Configuration](configuration.md) for details.

### Watch Mode

```bash
# Watch for changes and re-run tests
plur watch

# Install the watcher binary if needed
plur watch install

# Customize debounce/timeout and ignore patterns
plur watch run --debounce 250 --timeout 60 --ignore "vendor/**" --ignore "tmp/**"
```

### Doctor Command

```bash
# Run diagnostics and troubleshooting
plur doctor
```

The `Runtime Data:` block reports the cache file path along with its
size, file count, and example count when the cache exists (e.g.
`21K / 13 files / 68 examples`), or `(file does not exist)` on a
fresh project.

### Rails And Rake Commands

Run Rails' standard [`db:test:prepare`](https://github.com/rails/rails/blob/c8d223147de1907b4bd3779b5733a55ab9ba9858/activerecord/lib/active_record/railties/databases.rake#L535-L557) task with Plur to recreate test databases for all workers. The task targets test databases, so `RAILS_ENV=test` is not required.

```bash
plur rails db:test:prepare      # Run db:test:prepare n times
plur rails db:test:prepare -n 8 # Run db:test:prepare 8 times
```

`plur rails` and `plur rake` run the command once per worker and set `PARALLEL_TEST_GROUPS` and `TEST_ENV_NUMBER`. Plur does not change `RAILS_ENV`.

```bash
plur rails db:setup                           # Run db:setup n times with TEST_ENV_NUMBER set
plur rails db:drop db:create RAILS_ENV=test   # Drop and recreate your test databases
plur rails app:my_task                        # Run an app Rake task n times
plur rake app:my_task                         # Run the same task with the Rake alias
plur rake app:my_task -n 1 -- --my-flag       # Pass Rake-specific flags after --
```

All `plur rake` and `plur rails` commands run once per worker, with `PARALLEL_TEST_GROUPS` and `TEST_ENV_NUMBER` set.
Use `--dry-run` to print commands without running them:

```text
> plur --dry-run rails db:test:prepare -n4
[dry-run] Running rails command 'db:test:prepare' in parallel using 4 workers
[dry-run] Worker 0: PARALLEL_TEST_GROUPS=4 TEST_ENV_NUMBER=1 bin/rails db:test:prepare
[dry-run] Worker 1: PARALLEL_TEST_GROUPS=4 TEST_ENV_NUMBER=2 bin/rails db:test:prepare
[dry-run] Worker 2: PARALLEL_TEST_GROUPS=4 TEST_ENV_NUMBER=3 bin/rails db:test:prepare
[dry-run] Worker 3: PARALLEL_TEST_GROUPS=4 TEST_ENV_NUMBER=4 bin/rails db:test:prepare
```

Arguments are passed as-is and are not treated as test file patterns. Put Plur flags like `-n` before `--`. Arguments after `--` are passed through to Rails/Rake.

## Command Line Options

### Global Options

* `-n, --workers NUMBER` - Number of parallel workers (default: 4)
* `--dry-run` - Show what would run without executing
* `-h, --help` - Show help
* `-v, --verbose` - Enable verbose logging
* `--version` - Show version

### Environment Variables

* `PLUR_WORKERS` - Override number of workers (`PARALLEL_TEST_PROCESSORS` is a legacy fallback)
* `PLUR_DEBUG` - Enable debug logging
* `PLUR_CONFIG_FILE` - Load a specific config file
* `PLUR_HOME` - Override Plur's home directory (`~/.plur`)

## Parallelism

### Default Behavior

Plur uses 4 workers by default:
- Override with `-n` or `--workers`
- Respects `PLUR_WORKERS` (or the legacy `PARALLEL_TEST_PROCESSORS`) if set
- Project config can set a different default

### Manual Control

```bash
# Use all cores
plur -n $(nproc)

# Conservative - half the cores
plur -n $(( $(nproc) / 2 ))

# CI environments often benefit from more workers
plur -n $(( $(nproc) + 2 ))
```

## Output Formats

### Progress Output (Default)

Shows dots for test progress:
```
....F...*...
```
- `.` - Passing test
- `F` - Failing test
- `*` - Pending test

## Performance Monitoring

### Basic Timing

RSpec output includes timing details, for example:
```
Finished in 0.35 seconds (files took 0.12 seconds to load)
```

### Debugging Test Failures

```bash
# Run with debug output
plur --debug

# Check which files would run
plur --dry-run | grep "file_spec.rb"

# Run doctor for diagnostics
plur doctor
```

### Performance Tuning

1. **Start with the default**: Try `4` workers first
2. **Measure and adjust**: Experiment with different worker counts
3. **Consider test characteristics**:
- Many small tests: More workers
- Few large tests: Fewer workers
- I/O heavy tests: More workers than CPU cores

## Runtime Tracking

Plur records per-file runtime data to `$PLUR_HOME/runtime/<project-hash>.json`
so it can balance subsequent worker assignments using historical timing.

The on-disk format is a versioned runtime cache:

```json
{
  "meta": {
    "schema_version": 4,
    "plur_version": "0.56.0-dev-abc1234"
  },
  "run": {
    "cwd": "/Users/example/src/my-project",
    "last_run_at": "2026-05-22T15:04:05Z"
  },
  "files": {
    "spec/slow_spec.rb": {
      "mtime_unix_nano": 1778610000000000000,
      "size_bytes": 12345,
      "runtime_seconds": 12.34,
      "examples": [
        {
          "id": "./spec/slow_spec.rb[1:1]",
          "line_number": 12,
          "location_rerun_argument": "./spec/slow_spec.rb:12",
          "runtime_seconds": 0.40
        }
      ]
    }
  }
}
```

Behavior:

- File aggregates are rewritten only by default/full-file RSpec runs. Focused
  (`spec/foo_spec.rb:42`), tag-filtered (`--tag=…`), `--fail-fast`, aborted,
  and `--`-passthrough runs are classified as *partial* and merge
  per-example observations without overwriting the file aggregate.
- `--dry-run` never writes the cache.
- Invalid, corrupt, or unsupported schema files are ignored and replaced on the next
  successful default run.
- Old v1 caches (`map[string]float64`) are ignored and regenerated.
- Shared examples are attributed to their rerunnable owning spec file
  (the file the focused target points back to), not the support file
  whose source contains the shared block. The runtime cache stores only
  the fields needed for future balancing: RSpec `id`, rerunnable target,
  owner line, and runtime.

### `--rspec-split` (EXPERIMENTAL)

`--rspec-split` is an opt-in, RSpec-only flag that expands long-running
spec files into focused `file:line:line:line` targets, then lets the
existing runtime grouper balance them across workers.

```bash
plur --rspec-split -n 8
PLUR_RSPEC_SPLIT=1 plur -n 8
```

How it works:

- Splitting requires `--rspec-split == true`, an RSpec job, and worker
  count greater than 1.
- For each file, plur consults the runtime cache. If the recorded
  `mtime_unix_nano`/`size_bytes` still match the source file, plur considers
  the example data fresh enough for split planning.
- A file is split only if its historical `runtime_seconds` exceeds the
  per-worker budget (`total_runtime / worker_count`). This is the simple
  experimental rule — no multipliers, no floors.
- Split chunks are built by bin-packing the file's cached per-example
  runtimes using longest-processing-time greedy: each example lands in
  the bin with the smallest current sum, so a single heavy example ends
  up isolated in its own chunk. Examples with no recorded runtime fall
  back to the file's mean per-example runtime.
- Each chunk's summed runtime feeds back into the grouper as the
  target's runtime weight, so worker balancing reflects the actual
  bin-pack distribution. The generated `file:line:line:...` targets are
  not persisted in the cache.

Known pitfalls:

- `before(:all)` / `before(:context)` state may run once per chunk
  process instead of once per file, which can break suites that rely on
  shared context fixtures.
- Suites that define examples dynamically (from environment, time,
  random data, database state, or metaprogramming) may produce different
  example sets between cache generation and split execution.
- Shared examples and custom DSLs can produce surprising source
  locations. Plur stores `id` and `location_rerun_argument` so
  divergence can be debugged.
- Splitting is cache-driven: a cold run (no runtime cache entries yet) falls back
  to file-level grouping. The next default run populates the cache.

Splitting is intentionally experimental. The semantics may change as
real-world data is collected; do not rely on stable split behavior yet.

## Next Steps

- See [Configuration](configuration.md) for customization
- See [Architecture](architecture/index.md) for technical details
- See [Features](features/index.md) for detailed feature documentation
