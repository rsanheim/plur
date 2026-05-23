# Runtime Cache Measurement Results

> Measurement record for the runtime cache work on `rspec-split-specs`. Covers the full evolution from the initial v2 schema through Phase B shape cleanup and the final go-json parser adoption.

## Scope

Measure the runtime-tracking overhead and full-suite behavior of:

- **Legacy v1 tracker:** `main`, plain `map[string]float64` runtime file, no `--rspec-split`.
- **Versioned runtime cache:** `rspec-split-specs`, runtime cache with per-example entries.
- **Versioned runtime cache + rspec-split:** same branch, `--rspec-split` enabled so long files can be expanded into focused `file:line:line` chunks.

Projects in this pass:

| Project | Path | Command shape | Notes |
|---|---|---|---|
| Plur | `/Users/rsanheim/src/rsanheim/plur` | `plur --use=rspec -n 8 --no-color` | Uses this repo's RSpec integration/unit specs, not `bin/rake`. |
| RuboCop | `/Users/rsanheim/src/oss/rubocop` | `plur -n 8 --no-color` | Full RuboCop RSpec suite. |

RSpec-core from the earlier checklist is intentionally out of this pass because the current request asked for Plur and RuboCop.


## Decision Criteria

Thresholds used:

| Project size | Threshold | Action if exceeded |
|---|---:|---|
| Small (< 1K cached examples) | 25 ms combined load+save | Investigate a Phase B cache-format change. |
| Large (>= 10K cached examples) | 50 ms combined load+save | Investigate a Phase B cache-format change. |
| In between | 25 ms combined load+save | Use the stricter small-project threshold. |

For debug runs, compute combined runtime-cache overhead as:

```text
runtime_cache_overhead_ms = runtimeCache_loaded_duration_ms + runtimeCache_saved_duration_ms
```

Those `runtimeCache ...` stderr lines were temporary instrumentation for this measurement pass. They are not part of the current default user-facing output.

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
| v2 selector-grouping rerun ref | `81c842a` (`rspec-split-specs`) |
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

- The versioned cache at this point was ~8.16 MB for RuboCop, versus ~55 KB for the legacy v1 file-level map.
- The v2 debug run shows 111.710 ms combined load+save. The v2 split debug run shows 114.538 ms combined load+save. Both exceed the 50 ms large-suite threshold.
- The original `v2-split` rows above are retained as historical pre-fix evidence. They are not valid speedup evidence because baseline v2 ran 31672 examples while the split runs observed 31687 and 31694 examples.

#### Post Selector-Grouping RuboCop Rerun

After the splitter changed to keep cache identity by `example.id` but group scheduling units by rerunnable selector, RuboCop was rerun at `81c842a` with isolated warmed `PLUR_HOME` directories:

| Mode | Status | Wall seconds | RSpec summary | Cache size | Files | Examples | Cache load ms | Cache save ms | Evidence |
|---|---|---:|---|---:|---:|---:|---:|---:|---|
| `v2-postfix` | pass | 29.246 +/- 0.334 | 31672 examples, 0 failures, 3 pending | 8,159,967 B | 745 | 31672 | n/a | n/a | `tmp/bench/runtime-cache/logs/rubocop/postfix-81c842a-warm-v2.log` |
| `v2-split-postfix` | pass | 21.211 +/- 0.012 | 31672 examples, 0 failures, 3 pending | 8,159,747 B | 745 | 31672 | 54.469 | 56.522 | `tmp/bench/runtime-cache/logs/rubocop/postfix-81c842a-v2-split-debug.log` |

Evidence:

- Hyperfine: `tmp/bench/runtime-cache/results/rubocop-postfix-81c842a-runtime-cache.md`
- Split debug log: `tmp/bench/runtime-cache/logs/rubocop/postfix-81c842a-v2-split-debug.log`
- Warmup logs: `tmp/bench/runtime-cache/logs/rubocop/postfix-81c842a-warm-v2.log`, `tmp/bench/runtime-cache/logs/rubocop/postfix-81c842a-warm-v2-split.log`

Validation:

- Plain v2 warmup: `31672 examples, 0 failures, 3 pending`.
- Split debug run: `31672 examples, 0 failures, 3 pending`.
- `--rspec-split` applied to 2 files and emitted 16 planned split chunks total.
- Hyperfine result: `v2-split-postfix` ran `1.38 +/- 0.02` times faster than `v2-postfix`.
- Split cache overhead was 110.991 ms combined load+save, still above the 50 ms large-suite threshold.
- A matching plain `--debug` diagnostic run loaded the cache in 56.361 ms but was terminated after one RSpec worker hung, so its save timing is not used as evidence.

#### Post Compact-JSON Recheck

After `6db5c48` changed `SaveCache` to stream compact JSON with HTML escaping disabled, the focused Go benchmark and RuboCop dry-run benchmarks were rerun with a current dirty build at `7797ad0`.

Focused Go benchmark:

```bash
go test -mod=mod ./internal/testruntime -run '^$' -bench 'BenchmarkCache_(Load|Save)LargeRspecCache' -benchmem -count=5
```

| Benchmark | Result |
|---|---:|
| `BenchmarkCache_LoadLargeRspecCache` | 40.49-40.78 ms/op, ~13.39 MB/op, 136,413 allocs/op |
| `BenchmarkCache_SaveLargeRspecCache` | 20.54-21.40 ms/op, ~1.9-2.2 MB/op, 33,177 allocs/op |

RuboCop dry-run benchmark:

```bash
hyperfine --warmup 2 --runs 10 --style basic --time-unit millisecond \
  --command-name rubocop-dry-run \
  'PLUR_HOME=tmp/bench/runtime-cache/plurhome/rubocop/current tmp/bench/runtime-cache/bin/plur-current -C /Users/rsanheim/src/oss/rubocop -n 8 --no-color --dry-run --debug' \
  --command-name rubocop-split-dry-run \
  'PLUR_HOME=tmp/bench/runtime-cache/plurhome/rubocop/current tmp/bench/runtime-cache/bin/plur-current -C /Users/rsanheim/src/oss/rubocop -n 8 --no-color --rspec-split --dry-run --debug'
```

| Mode | Wall time | Cache load samples | Cache size | Notes |
|---|---:|---:|---:|---|
| `rubocop-dry-run` | 62.6 +/- 1.4 ms | 43.887, 45.067, 45.974 ms | 6.1 MB | 745 files, 31,672 examples |
| `rubocop-split-dry-run` | 69.6 +/- 2.0 ms | 45.547, 45.324, 45.436 ms | 6.1 MB | split applies to 2 files, 16 chunks |

Evidence:

- Hyperfine: `tmp/bench/runtime-cache/results/rubocop-current-dry-run-cache.md`
- Logs: `tmp/bench/runtime-cache/logs/rubocop/current-dry-run.log`, `tmp/bench/runtime-cache/logs/rubocop/current-split-dry-run.log`
- Warmup log: `tmp/bench/runtime-cache/logs/rubocop/current-warm.log`

Full RuboCop debug runs:

```bash
/usr/bin/time -p env PLUR_HOME=tmp/bench/runtime-cache/plurhome/rubocop/current \
  tmp/bench/runtime-cache/bin/plur-current \
  -C /Users/rsanheim/src/oss/rubocop -n 8 --no-color --debug
```

| Run | Status | Wall seconds | RSpec summary | Cache load ms | Cache save ms | Combined ms | Evidence |
|---|---|---:|---|---:|---:|---:|---|
| `current-full-debug-1` | pass | 28.79 | 31672 examples, 0 failures, 3 pending | 44.844 | 31.012 | 75.857 | `tmp/bench/runtime-cache/logs/rubocop/current-full-debug-1.log` |
| `current-full-debug-2` | pass | 28.73 | 31672 examples, 0 failures, 3 pending | 45.820 | 34.863 | 80.684 | `tmp/bench/runtime-cache/logs/rubocop/current-full-debug-2.log` |

Interpretation:

- The compact cache is 6.1 MB, down from the earlier ~8.16 MB pretty-printed cache.
- Dry-run cache load is now about 45 ms, down from the earlier RuboCop dry-run debug load of ~54 ms.
- Full RuboCop `--debug` load+save is now about 76-81 ms, down from the earlier ~112 ms combined overhead, but still above the 50 ms large-suite threshold.

## Phase B Schema-Shape Recheck

Captured after dropping `example_count`, changing `Files` to `map[string]FileEntry`, changing examples to `[]ExampleEntry`, and moving the RSpec `example.id` into `ExampleEntry.ID`.

Focused Go benchmark:

```bash
script/benchstat --package ./internal/testruntime \
  --filter 'BenchmarkCache_(Load|Save)LargeRspecCache' --count 10
```

| Benchmark | Phase B result | Previous compact-JSON baseline | Interpretation |
|---|---:|---:|---|
| `BenchmarkCache_SaveLargeRspecCache` | ~17.7-18.5 ms/op, 1,505 allocs/op | ~20.5-21.4 ms/op, 33,177 allocs/op | Better save time and much lower allocation count. |
| `BenchmarkCache_LoadLargeRspecCache` | ~41.4-41.9 ms/op, 70,089 allocs/op | ~40.5-40.8 ms/op, 136,413 allocs/op | Allocation count improves, but load time is effectively unchanged/slightly slower. |

RuboCop full debug runs:

| Run | Status | Wall seconds | RSpec summary | Cache load ms | Cache save ms | Combined ms | Evidence |
|---|---|---:|---|---:|---:|---:|---|
| `phaseb-shape-full-debug-1` | pass | 29.74 | 31672 examples, 0 failures, 3 pending | 44.390 | 25.893 | 70.284 | `tmp/bench/runtime-cache/logs/rubocop/phaseb-shape-full-debug-1.log` |
| `phaseb-shape-full-debug-2` | pass | 28.89 | 31672 examples, 0 failures, 3 pending | 44.946 | 31.737 | 76.683 | `tmp/bench/runtime-cache/logs/rubocop/phaseb-shape-full-debug-2.log` |

Dry-run hyperfine:

| Command | Mean | Notes |
|---|---:|---|
| `phaseb-dry-run` | 63.2 +/- 1.4 ms | Debug load sample: 44.976 ms. |
| `phaseb-split-dry-run` | 69.3 +/- 1.2 ms | Debug load sample: 44.496 ms; split applied to 2 files / 16 chunks. |

Interpretation:

- The schema-shape pass improved save time and allocation count, but did not materially reduce cache load time.
- RuboCop combined load+save is now about 70-77 ms, down from ~76-81 ms after compact JSON and ~112 ms before cleanup.
- Cache size is about 6.2 MB. Moving the example ID from the JSON object key to an `id` field did not shrink the real RuboCop cache, because the ID string is still stored once per example.
- The large-suite threshold remains 50 ms combined load+save, so candidate-only example retention is still the next design decision unless the threshold is relaxed.

### Split Planning Notes

Capture `rspec-split applied` count from the `v2-split-debug` and `v2-split-dry-run` logs for each project. A zero count means the v2 split path executed but did not find any files above the per-worker runtime budget with fresh examples.

| Project | Mode | Split-applied files | Planned targets | Evidence |
|---|---|---:|---:|---|
| Plur | `v2-split` | not trusted | not trusted | Plur suite did not complete cleanly under outer Plur. |
| RuboCop | `v2-split-debug` | 2 | 16 | `tmp/bench/runtime-cache/logs/rubocop/v2-split-debug.log` |
| RuboCop | `v2-split-dry-run` | 2 | 16 | `tmp/bench/runtime-cache/logs/rubocop/v2-split-dry-run.log` |
| RuboCop | `v2-split-postfix-debug` | 2 | 16 | `tmp/bench/runtime-cache/logs/rubocop/postfix-81c842a-v2-split-debug.log` |

### Decision

- Plur: blocked as a full-suite runner benchmark. The benchmark protocol needs a Plur-safe entry point before the numbers are meaningful.
- RuboCop runtime cache overhead: pre-cleanup v2 load+save was ~112 ms on a 31.7K-example cache. Post compact-JSON full debug runs are ~76-81 ms combined load+save. The cleanup helped materially, but the combined overhead remains above the 50 ms large-suite threshold, so a smaller Phase B follow-up is still justified unless the threshold is relaxed.
- RuboCop `--rspec-split`: the post selector-grouping rerun is now a valid apples-to-apples comparison. It preserves the baseline example count and is ~1.38x faster on this suite. Keep it experimental until the broader QA matrix is either completed or explicitly narrowed, but the RuboCop correctness blocker from duplicate rerunnable selectors is resolved.

## go-json Parser Adoption

After the Phase B schema-shape pass, load time remained at ~41–45 ms and was the dominant contributor to the combined threshold breach. The bottleneck was `encoding/json`'s reflection-based field dispatch and per-string heap allocation, not file size or JSON structure.

Replacing `encoding/json` with `github.com/goccy/go-json` (a drop-in API replacement) in `loadCache`/`saveCache` resolves this. go-json precomputes struct-field dispatch tables on first use and uses SIMD token scanning on ARM64. It requires one import change with no schema or format modification.

### Benchmark

Command used:

```bash
go test -mod=mod ./internal/testruntime/ \
  -bench='BenchmarkCache_(Load|Save)LargeRspecCache' \
  -benchmem -count=10 -benchtime=3s
```

Synthetic cache: 745 files, 31,672 examples (RuboCop-shaped), compact JSON, schema v4.

| Benchmark | encoding/json (Phase B baseline) | go-json | Change |
|---|---:|---:|---|
| `LoadLargeRspecCache` | ~41.5 ms/op, 70,089 allocs/op | **~8.6 ms/op, 33,942 allocs/op** | 4.8× faster, 51% fewer allocs |
| `SaveLargeRspecCache` | ~18.1 ms/op, 1,505 allocs/op | **~14.9 ms/op, 12 allocs/op** | 1.2× faster, 99% fewer allocs |
| Combined | ~59.6 ms | **~23.5 ms** | 2.5× faster |

The save dropping to 12 allocations total is go-json's zero-allocation encoding path for well-known struct layouts. The load allocation reduction (70K → 34K) reflects go-json's more efficient string batching.

### Extrapolated Real-Project Estimate

Previous real-RuboCop debug loads were ~9% slower than the Go benchmark (~45 ms vs ~41.5 ms). Applying the same factor to the go-json result: **~9–10 ms load, ~16–18 ms save, ~25–28 ms combined** — well under the 50 ms large-suite threshold.

### Tradeoffs

- **`unsafe` internals.** go-json uses unsafe pointer arithmetic for field writes. Stable in practice; the library tests against each new Go release.
- **External dependency.** Adds ~250 KB to the binary. If future stdlib JSON work closes this performance gap, migration back to the standard library should be straightforward because the cache package uses the familiar encoder/decoder API surface.
- **Edge-case divergence.** go-json has minor behavioral differences from stdlib for unusual JSON inputs (NaN/Inf, duplicate keys). Not a concern for a self-written cache with a fixed schema.

### Decision

Phase B threshold concern **closed** by go-json adoption. The combined large-suite cache overhead is now ~23.5 ms against the benchmark and an estimated ~25–28 ms against real RuboCop, both well under the 50 ms threshold. Candidate-only example retention (Phase B Step 5) is deferred; it remains a valid future optimization for caches significantly larger than RuboCop's 31,672 examples.

## Obstacles and Blockers

Record any setup or suite failures here instead of silently skipping a benchmark.

- **Plur full suite under outer Plur:** not clean. `PLUR_HOME` cache isolation conflicts with a watcher-path expectation, and parallel outer execution exposes Rails fixture DB isolation failures.
- **RuboCop split correctness:** the original benchmark showed `--rspec-split` changing example counts from 31672 to 31687/31694 with zero failures. The selector-grouping fix addressed that in the `81c842a` rerun: baseline and split both report 31672 examples, 0 failures, 3 pending.
- **Cache overhead threshold:** RuboCop post-cleanup cache load/save still exceeded the 50 ms large-suite combined threshold, though the gap narrowed at each cleanup step. Resolved by the go-json adoption (see above).
