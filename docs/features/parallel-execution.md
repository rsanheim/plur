# Parallel Execution

Plur's core feature is running test targets in parallel for faster feedback.

## How It Works

1. Plur discovers test targets from CLI patterns, job configuration, or
   framework defaults.
2. It creates worker processes based on `--workers`, `-n`, config, or
   `PARALLEL_TEST_PROCESSORS`.
3. It distributes targets by historical runtime when data is available, falling
   back to file size for new projects.
4. It runs each worker command and aggregates the result.

## Worker Count

```bash
plur             # Default 4 workers
plur -n 4        # Use 4 workers
plur --workers 8 # Use 8 workers
PARALLEL_TEST_PROCESSORS=10 plur
```

## Runtime Tracking

Plur records per-file runtime data to `$PLUR_HOME/runtime/<project-hash>.json`
so later runs can balance worker assignments using historical timing.

File aggregates are rewritten only by default/full-file RSpec runs. Focused
targets such as `spec/foo_spec.rb:42`, tag-filtered runs, `--fail-fast`, aborted
runs, and `--` passthrough runs are treated as partial and merge per-example
observations without overwriting the file aggregate.

Behavior:

- `--dry-run` never writes the cache.
- Invalid, corrupt, or unsupported schema files are ignored and replaced on the
  next successful default run.
- Old v1 caches (`map[string]float64`) are ignored and regenerated.
- Shared examples are attributed to their rerunnable owning spec file, not the
  support file whose source contains the shared block.

The runtime cache stores only fields needed for future balancing: RSpec example
IDs, rerunnable targets, owner lines, and runtime.

## Experimental RSpec Splitting

`--rspec-split` is an opt-in, RSpec-only flag that expands long-running spec
files into focused `file:line:line:line` targets, then lets the existing runtime
grouper balance them across workers.

```bash
plur --rspec-split -n 8
PLUR_RSPEC_SPLIT=1 plur -n 8
```

Splitting requires:

- `--rspec-split == true`
- an RSpec job
- worker count greater than 1
- fresh enough runtime cache data for the source file

How it works:

- A file is split only if its historical runtime exceeds the per-worker budget
  (`total_runtime / worker_count`).
- Split chunks are built by bin-packing cached per-example runtimes using a
  longest-processing-time greedy algorithm.
- Examples with no recorded runtime fall back to the file's mean per-example
  runtime.
- Each chunk's summed runtime feeds back into the grouper as that target's
  runtime weight.
- Generated `file:line:line:...` targets are not persisted in the cache.

Known pitfalls:

- `before(:all)` / `before(:context)` state may run once per chunk process
  instead of once per file.
- Dynamically generated examples may differ between cache generation and split
  execution.
- Shared examples and custom DSLs can produce surprising source locations.
- A cold run with no runtime cache falls back to file-level grouping. The next
  default run can populate cache data for future split planning.

Splitting is intentionally experimental. Its semantics may change as real-world
data is collected.
