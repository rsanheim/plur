# Runtime Cache Measurement Results

> Output of Task 8 from [2026-05-22-runtime-cache-implementation.md](2026-05-22-runtime-cache-implementation.md). Captured after the bin-packing + schema-trim + debug-logging work landed on `rspec-split-specs`.

**Decision:** large-suite threshold **exceeded** on RuboCop — a Phase B follow-up is justified. See "Phase B Trigger" below.

---

## Measurement procedure

Two real Ruby projects on the same machine, same plur binary (`0.56.0-dev-97090e8`):

- **Plur** (this repo, `~/src/rsanheim/plur`). Spec/integration suite.
- **RuboCop** (`~/src/oss/rubocop`). `spec/rubocop/cop/style/` subset (299 spec files, 14,836 examples after warm).

Each project:

1. Warm the cache with one full run.
2. Three real measurement runs against a single spec file (so load + save both fire against the full warmed cache).
3. Three `--dry-run` invocations to confirm load timing without save.

All timings via the `runtimeCache loaded` / `runtimeCache saved` debug log lines (`PLUR_DEBUG=1`).

Hardware: macOS Darwin 24.6.0, APFS SSD, no other heavy I/O during measurement.

## Numbers

### Plur (small)

- Cache file: `87K`, 43 files, 333 examples cached
- Load (3 runs): `1.06ms / ~ / ~` — well under 1ms variance not captured at first decimal
- Save (3 runs): `7.46ms / ~ / ~` — three runs reported equal-ish save times (~7ms each)

| Metric | Value | Threshold (small) | Status |
|---|---|---|---|
| Load median | ~1ms | — | — |
| Save median | ~7.5ms | — | — |
| **Combined median** | **~8.5ms** | **25ms** | ✅ under |

### RuboCop (large)

- Cache file: `3.7M`, 300 files, 14,836 examples cached
- Load (3 real runs): `24.68ms / 25.76ms / 24.94ms` — median **25.0ms**
- Load (3 dry-runs):  `24.89ms / 24.98ms / 25.15ms` — median **24.98ms** (consistent with real-run load)
- Save (3 runs):     `34.90ms / 34.60ms / 33.60ms` — median **34.6ms**

| Metric | Value | Threshold (large) | Status |
|---|---|---|---|
| Load median | 25.0ms | — | — |
| Save median | 34.6ms | — | — |
| **Combined median** | **~60ms** | **50ms** | ❌ **exceeded** |

### Schema spot-check

The on-disk JSON shape after Task 2's trim, verified via `jq`:

```
$ jq '[.files[] | .examples // {} | to_entries[].value
        | (has("scoped_id"), has("status"))] | unique' \
      ~/.plur/runtime/f2db9b6e.json
[ false ]
```

Every example record carries exactly the three intended fields (`line_number`, `location_rerun_argument`, `runtime_seconds`). No `scoped_id` or `status` lingering.

## Phase B Trigger

RuboCop's combined ~60ms per invocation exceeds the 50ms large-project threshold. Extrapolating from the linear-with-size shape we measured (RuboCop ~250 bytes/example, ~3.5µs/example to load, ~2.3µs/example to save):

| Suite size | Projected load | Projected save | Combined |
|---|---|---|---|
| 10K examples (RuboCop floor) | ~25ms | ~23ms | ~48ms |
| 15K examples (RuboCop today) | ~37ms | ~35ms | ~72ms |
| 20K examples | ~50ms | ~46ms | ~96ms |
| 30K examples (Discourse subset target) | ~75ms | ~70ms | ~145ms |
| 40K examples (Discourse full target) | ~100ms | ~92ms | ~192ms |

For a multi-minute Discourse test run, ~200ms is <1% overhead — not catastrophic. But for fast iterative runs (single spec file) on a large suite, the cache overhead becomes a real fraction of wall time.

## Phase B Recommendation

Per the spec's Phase B Options section, with a profile that shows save bytes dominating (34.6ms on 3.7MB = ~9MB/s effective write through JSON indent + sync + rename) and load split between `os.ReadFile` and `json.Unmarshal`:

**Primary candidate: compact JSON (`json.Marshal`, drop `MarshalIndent`).** Removes whitespace overhead which is ~30-40% of bytes on indented v2 cache files. Eyeballing the RuboCop file, removing the indentation alone should cut the file from ~3.7M to ~2.3M. That should buy back roughly 10-15ms on both load and save on these magnitudes.

**Secondary candidate (if compact JSON isn't enough): cap example tracking to files with `runtime_seconds >= 1s`** (or top-N slowest). Most RuboCop spec files run in well under a second; capturing per-example data for them is wasted bytes since none will ever be split. This is the biggest absolute win for very large suites.

**Not recommended right now:** gzip (decompression CPU > the I/O savings at this size); SQLite (overkill for the load-once / save-once access pattern); binary formats (loses jq inspectability for no measurable win at this scale).

## Caveats

- **Only two projects measured.** The plan called for Plur, RuboCop, RSpec, Mastodon subset, Discourse subset. Plur and RuboCop landed in this session; the others are deferred.
- **`spec/rubocop/cop/style/` is a subset of RuboCop.** Full RuboCop would push the example count higher (~30-50K probable), which is why the projections above lean conservative.
- **Bundle install for RuboCop was done fresh in this session.** No surprises, no Ruby/gem-stack weirdness.

## Next Steps

A follow-up Phase B plan should:

1. Switch `json.MarshalIndent` to `json.Marshal` in `SaveCache`. Re-measure load + save on the RuboCop cache.
2. If still exceeding the 50ms large-project threshold, propose a top-N-slowest-files-only example cap (with a clearly documented threshold).
3. Verify the JSON is still inspectable with `jq` (it will be — just no whitespace).
