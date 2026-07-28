# Setup Scripts

## bench-suite

Runs the pinned benchmark manifest with hyperfine and writes structured trend
results under a host-scoped output directory.

### Usage

```bash
# Run every project in benchmarks/projects.yml
./script/bench-suite
# Run one manifest project
./script/bench-suite --only backspin
# Write into the project tmp tree
./script/bench-suite --out ./tmp/bench
# Skip the static HTML report hook
./script/bench-suite --no-report
# See all options
./script/bench-suite --help
```

### What it does

1. Reads `benchmarks/projects.yml`
2. Clones or updates each pinned target under the output root
3. Installs the target bundle in `vendor/bundle`
4. Runs one hyperfine invocation per target
5. Writes `hyperfine.json`, enriched `result.json`, `output.log`, `run.json`, and append-only `index.jsonl`
6. Refreshes the static report with `script/bench-report`

### Requirements

* `hyperfine` (install with `brew install hyperfine`)
* `plur` binary in PATH (`bin/rake install`)
* Bundler and git for provisioning target projects

### Output

By default, results are saved under `tmp/bench/<host>/`:
* `runs/<run-id>/run.json` - run metadata
* `runs/<run-id>/<project>/hyperfine.json` - raw hyperfine export
* `runs/<run-id>/<project>/result.json` - hyperfine data with plur/target/host metadata
* `index.jsonl` - one trend row per hyperfine result element
* `site/index.html` - static dashboard from `script/bench-report`

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
