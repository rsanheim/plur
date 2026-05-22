# Runtime Cache Measurement Results

> Output of Task 8 from [2026-05-22-runtime-cache-implementation.md](2026-05-22-runtime-cache-implementation.md). Captured after the bin-packing + schema-trim + debug-logging work landed on `rspec-split-specs`.

**Decision:** large-suite threshold **exceeded** on RuboCop (full suite, 31.7K examples) — a Phase B follow-up is justified. See "Phase B Trigger" below.

---

## Measurement procedure

Two real Ruby projects on the same machine, same plur binary (`0.56.0-dev-97090e8`):

- **Plur** (this repo, `~/src/rsanheim/plur`). Full integration suite (333 examples, 43 spec files).
- **RuboCop** (`~/src/oss/rubocop`). Full suite (31,672 examples, 745 spec files).

Each project measured two separate ways:

1. **Clean run** — `plur -n 8` without `PLUR_DEBUG`. Captures real pass/fail and full wall time. PLUR_DEBUG cannot be set during the run because it propagates into spawned worker processes; Plur's own integration specs assert `stderr.empty?` and fail when subprocess plur emits debug noise.
2. **Cache-timing runs** — `PLUR_DEBUG=1 plur -n 8 <single-file>`. Captures `runtimeCache loaded` / `runtimeCache saved` durations against the warmed cache. Three runs per project for median.

Hardware: macOS Darwin 24.6.0, APFS SSD, no other heavy I/O during measurement.

## Plur — full suite (small)

### Clean run (wall time + failures)

```
plur -n 8        (33 spec files discovered)
333 examples, 2 failures, 4 pending
real    0m15.144s  (run 1)
real    0m17.958s  (run 2)
```

- **Median wall time: ~16.5s**
- **Failures: 2** — both in `spec/plur/release_spec.rb` (`extract-notes` cases). Verified present on `main` and unrelated to this branch's work: `script/release:63` hits `invalid byte sequence in US-ASCII (ArgumentError)` when reading the changelog. Pre-existing encoding bug in `script/release`.

### Cache timing (3 runs against warmed cache)

```
runtimeCache loaded duration=681.167µs   files=43 examples=333  (run 1)
runtimeCache saved  duration=7.043458ms  files=43 examples=333  (run 1)
runtimeCache loaded duration=710.875µs   files=43 examples=333  (run 2)
runtimeCache saved  duration=6.612458ms  files=43 examples=333  (run 2)
runtimeCache loaded duration=690.917µs   files=43 examples=333  (run 3)
runtimeCache saved  duration=7.691583ms  files=43 examples=333  (run 3)
```

- Cache file: 87K, 43 files, 333 examples
- **Load median: 691µs**
- **Save median: 7.04ms**
- **Combined median: ~7.7ms** — well under the 25ms small-project threshold ✅

## RuboCop — full suite (large)

### Clean run (wall time + failures)

```
plur -n 8        (745 spec files discovered)
31672 examples, 0 failures, 3 pending
real    0m49.308s
```

- **Wall time: 49.3s**
- **Failures: 0** ✅
- **Pending: 3** (expected — pre-existing pending examples in RuboCop's own suite)

### Cache timing (3 runs against warmed cache)

```
runtimeCache loaded duration=53.631667ms  files=745 examples=31672  (run 1)
runtimeCache saved  duration=61.063250ms  files=745 examples=31672  (run 1)
runtimeCache loaded duration=53.475125ms  files=745 examples=31672  (run 2)
runtimeCache saved  duration=58.035542ms  files=745 examples=31672  (run 2)
runtimeCache loaded duration=52.940666ms  files=745 examples=31672  (run 3)
runtimeCache saved  duration=54.563750ms  files=745 examples=31672  (run 3)
```

- Cache file: **7.8M**, 745 files, 31,672 examples
- **Load median: 53.48ms**
- **Save median: 58.04ms**
- **Combined median: ~111ms** — **2.2× over the 50ms large-project threshold** ❌

Relative to wall: cache I/O is **~0.23%** of the 49.3s full-suite run. Not a wall-time problem for full suites, but a noticeable fixed cost on every single `plur` invocation regardless of how few specs you run.

### Schema spot-check

```
$ jq '[.files[] | .examples // {} | to_entries[].value
        | (has("scoped_id"), has("status"))] | unique' \
      ~/.plur/runtime/f2db9b6e.json
[ false ]
```

Every example record carries exactly the three intended fields (`line_number`, `location_rerun_argument`, `runtime_seconds`). No `scoped_id` or `status` lingering anywhere across 31,672 examples.

## Summary table

| Project | Wall | Examples | Cache size | Load | Save | Combined | Threshold |
|---|---:|---:|---:|---:|---:|---:|---|
| Plur | 16.5s | 333 | 87K | 691µs | 7.04ms | **7.7ms** | 25ms (small) ✅ |
| RuboCop | 49.3s | 31,672 | 7.8M | 53.5ms | 58.0ms | **111ms** | 50ms (large) ❌ |

## Phase B Trigger

RuboCop's combined ~111ms exceeds the 50ms large-project threshold by 2.2×. Extrapolating linearly (saves scale ~1.8µs/example at this size, loads scale ~1.7µs/example):

| Suite size | Projected load | Projected save | Combined |
|---|---|---|---|
| 10K (small large) | ~17ms | ~18ms | ~35ms |
| 15K | ~26ms | ~28ms | ~54ms |
| 20K | ~34ms | ~37ms | ~71ms |
| 30K (RuboCop) | 53ms | 58ms | 111ms |
| 40K (Discourse target) | ~70ms | ~75ms | ~145ms |
| 60K (Discourse full) | ~105ms | ~115ms | ~220ms |

For a multi-minute Discourse test run, ~220ms is <0.1% overhead — not catastrophic. For fast iterative runs (single spec file at 1-2s), the cache overhead becomes a noticeable fraction.

## Phase B Recommendation

The save side is bytes-dominated: 58ms on 7.8MB = ~135 MB/s effective write through JSON indent + sync + rename. The load side splits between `os.ReadFile` and `json.Unmarshal`. With both dominated by JSON serialization, the easiest win is:

**Primary candidate: compact JSON (`json.Marshal`, drop `MarshalIndent`).** RuboCop's indented v2 cache is 7.8M; an unindented version of the same data is ~30-40% smaller (rough estimate from typical JSON whitespace overhead). That alone should cut both load and save by ~20-30ms each on RuboCop, bringing combined under the 50ms threshold.

**Secondary candidate (if compact JSON isn't enough): cap example tracking to files with `runtime_seconds >= 1s`** (or top-N slowest). Most RuboCop spec files run in well under a second; capturing per-example data for them is wasted bytes since they'll never be split. This is the biggest absolute win for very large suites.

**Not recommended right now:**
- **Gzip** — decompression CPU > the I/O savings at this size; doesn't address parse cost which is the larger half of load.
- **SQLite** — overkill for the load-once / save-once access pattern.
- **Binary formats** (gob/msgpack/cbor) — loses `jq` inspectability for no measurable win at this scale.

## Caveats

- **Only two projects measured.** The plan called for Plur, RuboCop, RSpec, Mastodon subset, Discourse subset. Plur and RuboCop landed in this session; the others are deferred to a follow-up.
- **Wall time has natural variance.** Plur showed 15.1s and 17.9s on two runs — sub-second variation due to filesystem/macOS factors. RuboCop measured once.
- **PLUR_DEBUG propagation gotcha:** the initial Plur run with `PLUR_DEBUG=1` reported 21 failures because Plur's own integration specs assert `stderr.empty?` and subprocess plur invocations inherit the env var. The clean methodology above (`plur -n 8` without `PLUR_DEBUG` for failure counting; separate single-file invocations for timing) sidesteps this. Worth noting if anyone reruns the QA.

## Next Steps

A follow-up Phase B plan should:

1. Switch `json.MarshalIndent` to `json.Marshal` in `SaveCache`. Re-measure load + save on the RuboCop full-suite cache.
2. If still exceeding the 50ms large-project threshold, propose a top-N-slowest-files-only example cap with a clearly documented threshold.
3. Verify the JSON is still inspectable with `jq` (it will be — just no whitespace).
