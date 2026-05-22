# Runtime Cache Measurement Results

> Output of Task 8 from [2026-05-22-runtime-cache-implementation.md](2026-05-22-runtime-cache-implementation.md). Captured after the bin-packing + schema-trim + debug-logging work landed on `rspec-split-specs`.

## Requirements
* escalate sandbox for requests around getting the suites runnable
* report osbstacles or blockers, do not skip them blindly
* log full runs directly to a ./tmp log file, then include the key summary here along w/ path to evidence
* 

Use the latest globally installed plur from this repo, run benchmarks against:

* plur itself
* rubocop ~/src/oss/rubocop
* rspec ~/src/oss/rspec

You should compare:

* v1 cache implementation
* v1 cache ipmlmeentation in verbose
* v1 cache implementation in dry-run mode
* v2 cache implementation
* v2 cache ipmlmeentation in verbose
* v2 cache ipmlmeentation in debug mode
* v2 cache ipmlmeentation in dry-run mode


For the `debug` mode, you should look at the logs and measure the impact of the runtimecache load and save.

## Results