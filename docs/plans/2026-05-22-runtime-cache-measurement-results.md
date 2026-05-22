# Runtime Cache Measurement Results

> Output of Task 8 from [2026-05-22-runtime-cache-implementation.md](2026-05-22-runtime-cache-implementation.md). Captured after the bin-packing + schema-trim + debug-logging work landed on `rspec-split-specs`.

## Scope

Measure the runtime-tracking overhead and full-suite behavior of:

- **Legacy v1 tracker:** `main`, plain `map[string]float64` runtime file, no `--rspec-split`.
- **v2 runtime cache:** `rspec-split-specs`, versioned runtime cache with per-example entries.
- **v2 runtime cache + rspec-split:** same branch, `--rspec-split` enabled so long files can be expanded into focused `file:line:line` chunks.

Projects in this pass:

| Project | Path | Command shape | Notes |
|---|---|---|---|
| Plur | `/Users/rsanheim/src/rsanheim/plur` | `plur --use=rspec -n 8 --no-color` | Uses this repo's RSpec integration/unit specs, not `bin/rake`. |
| RuboCop | `/Users/rsanheim/src/oss/rubocop` | `plur -n 8 --no-color` | Full RuboCop RSpec suite. |

RSpec-core from the earlier checklist is intentionally out of this pass because the current request asked for Plur and RuboCop.

## Benchmark Protocol

Use the existing benchmark flow where it fits:

- `script/bench-git` is the baseline harness for comparing refs. It creates worktrees, builds `plur`, and runs `hyperfine`.
- Direct `hyperfine` commands are used for the extra runtime-cache modes that `script/bench-git` does not expose yet: `--verbose`, `--debug`, `--dry-run`, and `--rspec-split`.
- The direct commands follow the same shape as `script/bench-git`: explicit binaries, `-C <project>`, fixed worker count, warmups, exported Markdown/JSON evidence.

All cache, log, and hyperfine output must stay inside this repository's `./tmp` tree.

### Build Inputs

Build one binary per implementation. `script/bench-git` can do this when only comparing plain `plur -n <workers>` across refs. For the full mode matrix, build stable binaries into `tmp/bench/runtime-cache/bin/` so all direct `hyperfine` commands use the same artifacts:

```bash
mkdir -p tmp/bench/runtime-cache/bin tmp/bench/runtime-cache/worktrees
git worktree add tmp/bench/runtime-cache/worktrees/main main
go build -mod=mod \
  -ldflags "-X github.com/rsanheim/plur/internal/buildinfo.Version=rspec-split-specs-29c8031" \
  -o tmp/bench/runtime-cache/bin/plur-v2 .
go build -C tmp/bench/runtime-cache/worktrees/main -mod=mod \
  -ldflags "-X github.com/rsanheim/plur/internal/buildinfo.Version=main-5f272dd" \
  -o /Users/rsanheim/src/rsanheim/plur/tmp/bench/runtime-cache/bin/plur-v1 .
```

Record the refs:

```bash
git rev-parse --short HEAD
git rev-parse --short main
```

Optional plain ref baseline with the existing harness:

```bash
script/bench-git --refs main rspec-split-specs \
  --project /Users/rsanheim/src/oss/rubocop \
  --workers 8 \
  --warmup 1 \
  --runs 2 \
  --ignore-failure
```

For Plur itself, the direct commands below include `--use=rspec` to force the Ruby spec suite; `script/bench-git` does not currently expose extra `plur` args.

### Cache Isolation

Use a separate `PLUR_HOME` for each project + implementation + mode family so one cache format cannot influence another:

```text
tmp/bench/runtime-cache/plurhome/<project>/v1
tmp/bench/runtime-cache/plurhome/<project>/v2
tmp/bench/runtime-cache/plurhome/<project>/v2-split
```

Before measuring a family, warm it with one successful full run:

```bash
PLUR_HOME=<home> <binary> <project args>
```

For `v2-split`, warm with the non-split v2 command first. `--rspec-split` needs a complete v2 aggregate cache before it can make split decisions. `hyperfine --warmup 1` is enough for non-dry-run modes because the warmup run populates that mode's cache. Dry-run modes need an explicit non-dry warmup because `--dry-run` never writes runtime data.

### Measured Modes

Run each mode through `hyperfine`, export JSON/Markdown, and capture separate debug logs for the modes that expose cache timings:

| Mode | Binary | Extra flags | Full-suite run? | Purpose |
|---|---|---|---|---|
| `v1` | `plur-v1` | none | yes | Legacy tracker steady-state baseline. |
| `v1-verbose` | `plur-v1` | `--verbose` | yes | Checks verbose logging overhead for legacy tracker. |
| `v1-dry-run` | `plur-v1` | `--dry-run` | no | Measures plan-only cost with legacy cache loaded. |
| `v2` | `plur-v2` | none | yes | v2 cache without splitting. |
| `v2-verbose` | `plur-v2` | `--verbose` | yes | Checks verbose logging overhead for v2 cache. |
| `v2-debug` | `plur-v2` | `--debug` | yes | Emits `runtimeCache loaded` / `runtimeCache saved` timings. |
| `v2-dry-run` | `plur-v2` | `--dry-run` | no | Measures plan-only cost with v2 cache loaded. |
| `v2-split` | `plur-v2` | `--rspec-split` | yes | Measures the experimental split execution path. |
| `v2-split-debug` | `plur-v2` | `--rspec-split --debug` | yes | Captures split decision logs and cache load/save timings. |
| `v2-split-dry-run` | `plur-v2` | `--rspec-split --dry-run --debug` | no | Confirms planned split fan-out without rewriting the cache. |

For Plur, add `--use=rspec` to every measured command so the benchmark is the Ruby spec suite. For RuboCop, use autodetection.

### Direct Hyperfine Shape

Project variables:

```bash
BENCH_ROOT=/Users/rsanheim/src/rsanheim/plur/tmp/bench/runtime-cache
PLUR_ROOT=/Users/rsanheim/src/rsanheim/plur
RUBOCOP_ROOT=/Users/rsanheim/src/oss/rubocop
V1=$BENCH_ROOT/bin/plur-v1
V2=$BENCH_ROOT/bin/plur-v2
```

Plur command template:

```bash
hyperfine --warmup 1 --runs 2 --style basic --time-unit second \
  --export-json "$BENCH_ROOT/results/plur-runtime-cache.json" \
  --export-markdown "$BENCH_ROOT/results/plur-runtime-cache.md" \
  --command-name "v1" "PLUR_HOME=$BENCH_ROOT/plurhome/plur/v1 $V1 -C $PLUR_ROOT --use=rspec -n 8 --no-color" \
  --command-name "v2" "PLUR_HOME=$BENCH_ROOT/plurhome/plur/v2 $V2 -C $PLUR_ROOT --use=rspec -n 8 --no-color" \
  --command-name "v2-split" "PLUR_HOME=$BENCH_ROOT/plurhome/plur/v2-split $V2 -C $PLUR_ROOT --use=rspec -n 8 --no-color --rspec-split"
```

RuboCop uses the same shape without `--use=rspec`:

```bash
hyperfine --warmup 1 --runs 2 --style basic --time-unit second \
  --export-json "$BENCH_ROOT/results/rubocop-runtime-cache.json" \
  --export-markdown "$BENCH_ROOT/results/rubocop-runtime-cache.md" \
  --command-name "v1" "PLUR_HOME=$BENCH_ROOT/plurhome/rubocop/v1 $V1 -C $RUBOCOP_ROOT -n 8 --no-color" \
  --command-name "v2" "PLUR_HOME=$BENCH_ROOT/plurhome/rubocop/v2 $V2 -C $RUBOCOP_ROOT -n 8 --no-color" \
  --command-name "v2-split" "PLUR_HOME=$BENCH_ROOT/plurhome/rubocop/v2-split $V2 -C $RUBOCOP_ROOT -n 8 --no-color --rspec-split"
```

Run a second hyperfine pass for the logging/dry-run modes. These are separated from the full-suite pass so dry runs do not distort full-suite timing:

```bash
hyperfine --warmup 1 --runs 2 --style basic --time-unit second \
  --export-json "$BENCH_ROOT/results/<project>-runtime-cache-modes.json" \
  --export-markdown "$BENCH_ROOT/results/<project>-runtime-cache-modes.md" \
  --command-name "v1-verbose" "PLUR_HOME=$BENCH_ROOT/plurhome/<project>/v1-verbose $V1 -C <project-root> <project-extra-args> -n 8 --no-color --verbose" \
  --command-name "v1-dry-run" "PLUR_HOME=$BENCH_ROOT/plurhome/<project>/v1-dry-run $V1 -C <project-root> <project-extra-args> -n 8 --no-color --dry-run" \
  --command-name "v2-verbose" "PLUR_HOME=$BENCH_ROOT/plurhome/<project>/v2-verbose $V2 -C <project-root> <project-extra-args> -n 8 --no-color --verbose" \
  --command-name "v2-debug" "PLUR_HOME=$BENCH_ROOT/plurhome/<project>/v2-debug $V2 -C <project-root> <project-extra-args> -n 8 --no-color --debug" \
  --command-name "v2-dry-run" "PLUR_HOME=$BENCH_ROOT/plurhome/<project>/v2-dry-run $V2 -C <project-root> <project-extra-args> -n 8 --no-color --dry-run" \
  --command-name "v2-split-debug" "PLUR_HOME=$BENCH_ROOT/plurhome/<project>/v2-split-debug $V2 -C <project-root> <project-extra-args> -n 8 --no-color --rspec-split --debug" \
  --command-name "v2-split-dry-run" "PLUR_HOME=$BENCH_ROOT/plurhome/<project>/v2-split-dry-run $V2 -C <project-root> <project-extra-args> -n 8 --no-color --rspec-split --dry-run --debug"
```

Before the dry-run mode pass, warm the dry-run `PLUR_HOME` directories with matching non-dry commands:

```bash
PLUR_HOME=$BENCH_ROOT/plurhome/<project>/v1-dry-run $V1 -C <project-root> <project-extra-args> -n 8 --no-color
PLUR_HOME=$BENCH_ROOT/plurhome/<project>/v2-dry-run $V2 -C <project-root> <project-extra-args> -n 8 --no-color
PLUR_HOME=$BENCH_ROOT/plurhome/<project>/v2-split-dry-run $V2 -C <project-root> <project-extra-args> -n 8 --no-color
```

For cache timing evidence, run the debug modes once with output redirected to logs:

```bash
PLUR_HOME=$BENCH_ROOT/plurhome/<project>/v2-debug \
  $V2 -C <project-root> <project-extra-args> -n 8 --no-color --debug \
  > "$BENCH_ROOT/logs/<project>/v2-debug.log" 2>&1

PLUR_HOME=$BENCH_ROOT/plurhome/<project>/v2-split-debug \
  $V2 -C <project-root> <project-extra-args> -n 8 --no-color --rspec-split --debug \
  > "$BENCH_ROOT/logs/<project>/v2-split-debug.log" 2>&1
```

### Evidence Layout

```text
tmp/bench/runtime-cache/
  bin/
    plur-v1
    plur-v2
  logs/
    plur/
      warmup-*.log
      run-*.log
    rubocop/
      warmup-*.log
      run-*.log
  results/
    <project>-runtime-cache.json
    <project>-runtime-cache.md
    <project>-runtime-cache-modes.json
    <project>-runtime-cache-modes.md
```

The hyperfine JSON/Markdown files are the wall-time source of truth. Debug logs are the cache-timing source of truth.

## Decision Criteria

Use the thresholds from [the measurement plan](2026-05-22-runtime-cache-measurement-and-bin-packing-plan.md#decision-criteria):

| Project size | Threshold | Action if exceeded |
|---|---:|---|
| Small (< 1K cached examples) | 25 ms combined load+save | Investigate a Phase B cache-format change. |
| Large (>= 10K cached examples) | 50 ms combined load+save | Investigate a Phase B cache-format change. |
| In between | 25 ms combined load+save | Use the stricter small-project threshold. |

For debug runs, compute combined runtime-cache overhead as:

```text
runtime_cache_overhead_ms = runtimeCache_loaded_duration_ms + runtimeCache_saved_duration_ms
```

Dry runs only load the cache. They must not save or modify the runtime file.

## Results

### Environment

| Item | Value |
|---|---|
| Date | 2026-05-22 |
| Host OS/arch | macOS / arm64 |
| Worker count | 8 |
| v1 ref | `5f272dd` (`main`) |
| v2 ref | `29c8031` (`rspec-split-specs`) |
| Plur Ruby | Ruby 4.0.3, RSpec 3.13 |
| RuboCop Ruby | Ruby 4.0.5, RSpec 3.13 |

### Plur

Plur's own full RSpec suite is not a clean target for outer `plur -n 8` benchmarking yet. The benchmark command completed, but each mode exited non-zero. The failures are environmental / suite-isolation blockers rather than runtime-cache regressions:

- Setting `PLUR_HOME` for cache isolation breaks `spec/integration/watch/watch_spec.rb:17`, which expects the watcher path under `ENV["HOME"]/.plur/bin/...`.
- Running the suite itself through outer Plur parallelism also exposes Rails fixture database isolation failures in `spec/integration/spec/rails_rake_spec.rb`.

Evidence:

- Hyperfine: `tmp/bench/runtime-cache/results/plur-runtime-cache.md`
- Logs: `tmp/bench/runtime-cache/logs/plur/v1.log`, `tmp/bench/runtime-cache/logs/plur/v2.log`, `tmp/bench/runtime-cache/logs/plur/v2-split.log`

| Mode | Status | Wall seconds | RSpec summary | Project cache size | Files | Examples | Evidence |
|---|---|---:|---|---:|---:|---:|---|
| `v1` | non-zero | 15.937 +/- 0.284 | 333 examples, 1 failure, 4 pending | 2,753 B | 43 | n/a | `tmp/bench/runtime-cache/logs/plur/v1.log` |
| `v2` | non-zero | 16.847 +/- 0.184 | 333 examples, 1 failure, 4 pending | 89,097 B | 43 | 333 | `tmp/bench/runtime-cache/logs/plur/v2.log` |
| `v2-split` | non-zero | 16.891 +/- 0.335 | 333 examples, 3 failures, 4 pending | 89,127 B | 43 | 333 | `tmp/bench/runtime-cache/logs/plur/v2-split.log` |

Conclusion for this project: do not use these timings to compare runtime tracker implementations. The suite needs a benchmark-safe entry point, probably `bin/rspec` with `PLUR_BINARY=<candidate>` for integration coverage, or a Plur-safe subset that excludes specs which intentionally invoke Plur/rails/watch behavior with shared fixtures.

### RuboCop

RuboCop is the clean full-suite comparison. All measured full-run modes exited zero.

Evidence:

- Primary hyperfine: `tmp/bench/runtime-cache/results/rubocop-runtime-cache.md`
- Mode hyperfine: `tmp/bench/runtime-cache/results/rubocop-runtime-cache-modes.md`
- Logs: `tmp/bench/runtime-cache/logs/rubocop/*.log`

| Mode | Status | Wall seconds | RSpec summary | Cache size | Files | Examples | Cache load ms | Cache save ms | Evidence |
|---|---|---:|---|---:|---:|---:|---:|---:|---|
| `v1` | pass | 43.124 +/- 7.202 | 31672 examples, 0 failures, 3 pending | 54,675 B | 750 | n/a | n/a | n/a | `tmp/bench/runtime-cache/logs/rubocop/v1.log` |
| `v1-verbose` | pass | 28.637 +/- 0.747 | 31672 examples, 0 failures, 3 pending | 54,923 B | 750 | n/a | n/a | n/a | `tmp/bench/runtime-cache/logs/rubocop/v1-verbose.log` |
| `v1-dry-run` | pass | 0.015 +/- 0.000 | dry-run only | 54,848 B | 750 | n/a | n/a | n/a | `tmp/bench/runtime-cache/logs/rubocop/v1-dry-run.log` |
| `v2` | pass | 29.000 +/- 0.212 | 31672 examples, 0 failures, 3 pending | 8,159,753 B | 745 | 31672 | n/a | n/a | `tmp/bench/runtime-cache/logs/rubocop/v2.log` |
| `v2-verbose` | pass | 28.790 +/- 0.182 | 31672 examples, 0 failures, 3 pending | 8,160,009 B | 745 | 31672 | n/a | n/a | `tmp/bench/runtime-cache/logs/rubocop/v2-verbose.log` |
| `v2-debug` | pass | 28.453 +/- 0.340 | 31672 examples, 0 failures, 3 pending | 8,160,001 B | 745 | 31672 | 53.248 | 58.462 | `tmp/bench/runtime-cache/logs/rubocop/v2-debug.log` |
| `v2-dry-run` | pass | 0.069 +/- 0.001 | dry-run only | 8,160,034 B | 745 | 31672 | 54.459 | n/a | `tmp/bench/runtime-cache/logs/rubocop/v2-dry-run-debug.log` |
| `v2-split` | pass, count differs | 22.755 +/- 0.194 | 31694 examples, 0 failures, 3 pending | 8,159,871 B | 745 | 31672 | n/a | n/a | `tmp/bench/runtime-cache/logs/rubocop/v2-split.log` |
| `v2-split-debug` | pass, count differs | 21.416 +/- 0.329 | 31687 examples, 0 failures, 3 pending | 8,159,651 B | 745 | 31672 | 52.315 | 62.223 | `tmp/bench/runtime-cache/logs/rubocop/v2-split-debug.log` |
| `v2-split-dry-run` | pass | 0.076 +/- 0.001 | dry-run only | 8,160,123 B | 745 | 31672 | 54.211 | n/a | `tmp/bench/runtime-cache/logs/rubocop/v2-split-dry-run.log` |

Notes:

- The v2 cache is ~8.16 MB for RuboCop, versus ~55 KB for the legacy v1 file-level map.
- The v2 debug run shows 111.710 ms combined load+save. The v2 split debug run shows 114.538 ms combined load+save. Both exceed the 50 ms large-suite threshold.
- The `v2-split` timings are not safe to treat as a win yet because the example count changes. Baseline v2 ran 31672 examples; split runs observed 31687 and 31694 examples. That is a correctness blocker for `--rspec-split`, likely caused by repeated line targets when multiple examples share a definition line.

### Split Planning Notes

Capture `rspec-split applied` count from the `v2-split-debug` and `v2-split-dry-run` logs for each project. A zero count means the v2 split path executed but did not find any files above the per-worker runtime budget with fresh examples.

| Project | Mode | Split-applied files | Planned targets | Evidence |
|---|---|---:|---:|---|
| Plur | `v2-split` | not trusted | not trusted | Plur suite did not complete cleanly under outer Plur. |
| RuboCop | `v2-split-debug` | 2 | 16 | `tmp/bench/runtime-cache/logs/rubocop/v2-split-debug.log` |
| RuboCop | `v2-split-dry-run` | 2 | 16 | `tmp/bench/runtime-cache/logs/rubocop/v2-split-dry-run.log` |

### Decision

- Plur: blocked as a full-suite runner benchmark. The benchmark protocol needs a Plur-safe entry point before the numbers are meaningful.
- RuboCop runtime cache overhead: Phase B is warranted. v2 load+save is ~112 ms on a 31.7K-example cache, above the 50 ms large-suite threshold.
- RuboCop `--rspec-split`: keep experimental and do not advertise the speedup yet. It is faster in wall time here, but it changes the executed example count.

## Obstacles and Blockers

Record any setup or suite failures here instead of silently skipping a benchmark.

- **Plur full suite under outer Plur:** not clean. `PLUR_HOME` cache isolation conflicts with a watcher-path expectation, and parallel outer execution exposes Rails fixture DB isolation failures.
- **RuboCop split correctness:** `--rspec-split` changed example counts from 31672 to 31687/31694 with zero failures. This must be fixed before timing improvements count.
- **Cache overhead threshold:** RuboCop v2 cache load/save exceeds the measurement plan's threshold.
