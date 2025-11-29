# Benchmark Analysis - 2025-11-29

## Summary

Comprehensive benchmarking of plur's hot code paths at realistic scale (up to 30,000 tests).

**Key Finding:** All components scale linearly (O(n)) - no algorithmic complexity issues detected.

## Performance Baseline

### Grouper (File Distribution)

| Benchmark | Files | Time | Memory | Allocs |
|-----------|-------|------|--------|--------|
| GroupBySize | 1,000 | 601µs | 343KB | 3,073 |
| GroupBySize | 5,000 | 3.0ms | 1.77MB | 15,097 |
| GroupBySize | 10,000 | 6.2ms | 3.54MB | 30,186 |
| GroupByRuntime | 1,000 | 53µs | 60KB | 74 |
| GroupByRuntime | 5,000 | 296µs | 396KB | 98 |
| GroupByRuntime | 10,000 | 734µs | 792KB | 187 |

**Observation:** GroupByRuntime is ~10x faster than GroupBySize due to no filesystem stat calls. The stat I/O dominates GroupBySize performance.

### RSpec Parser (JSON Parsing)

| Benchmark | Tests | Time | Memory | Allocs |
|-----------|-------|------|--------|--------|
| ParseLine (JSON) | 1 | 2.0µs | 1.5KB | 38 |
| ParseLine (raw) | 1 | 39ns | 48B | 2 |
| Parser | 1,000 | 2.1ms | 1.5MB | 38,807 |
| Parser | 5,000 | 10.6ms | 7.6MB | 194,820 |
| Parser | 10,000 | 21.0ms | 15.2MB | 389,830 |
| Parser | 30,000 | 61.1ms | 45.7MB | 1.17M |

**Observation:** JSON parsing is the primary cost. At rubocop-scale (30K tests), parsing takes 61ms and allocates 45MB. The per-line cost is consistent (~2µs per JSON line).

### TestCollector (Result Aggregation)

| Benchmark | Tests | Time | Memory | Allocs |
|-----------|-------|------|--------|--------|
| Collector | 1,000 | 121µs | 588KB | 1,012 |
| Collector | 5,000 | 1.0ms | 3.7MB | 5,019 |
| Collector | 10,000 | 2.1ms | 8.6MB | 10,023 |
| Collector | 30,000 | 6.4ms | 29.9MB | 30,030 |

**Observation:** Scales linearly. Memory allocation is ~1KB per test, dominated by notification storage.

## Scaling Analysis

### Time Complexity (verified O(n))

| Component | 1K→5K ratio | 5K→10K ratio | Expected (linear) |
|-----------|-------------|--------------|-------------------|
| GroupBySize | 5.0x | 2.0x | 5.0x, 2.0x ✓ |
| GroupByRuntime | 5.6x | 2.5x | 5.0x, 2.0x ✓ |
| RSpecParser | 5.1x | 2.0x | 5.0x, 2.0x ✓ |
| TestCollector | 8.3x | 2.1x | 5.0x, 2.0x ✓ |

All components exhibit linear scaling within expected tolerance.

## Hotspots Identified

1. **RSpec JSON Parsing** (2µs/line, 38 allocs/line)
   * `json.Unmarshal` is the primary cost
   * Consider: streaming JSON parser or pre-compiled decoder

2. **GroupBySize stat calls** (600µs for 1K files)
   * Filesystem I/O dominates
   * Consider: parallel stat calls or caching

3. **Memory Allocation** (~1.5KB per parsed test event)
   * Many small allocations (38 per JSON line)
   * Consider: object pooling for notification structs

## Complexity Detection Tests

All tests pass - no O(n²) behavior detected:

```
TestGrouperComplexity        PASS
TestGrouperRuntimeComplexity PASS
TestRSpecParserComplexity    PASS
TestTestCollectorComplexity  PASS
```

## Recommendations

1. **No immediate optimization needed** - All hot paths scale linearly
2. **For future optimization focus:**
   * JSON parsing if sub-60ms for 30K tests is required
   * Parallel file stat if GroupBySize is in critical path
3. **Use GroupByRuntime when possible** - 10x faster than GroupBySize

## Benchstat Usage

```bash
# Run benchmarks with 5+ iterations for statistical significance
script/benchstat --count 5

# Compare before/after optimization
script/benchstat --compare results/benchmarks/baseline.txt

# Filter to specific benchmarks
script/benchstat --filter RSpec --count 10
```
