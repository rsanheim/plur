# Runtime Cache Measurement Results

> Output of Task 8 from [2026-05-22-runtime-cache-implementation.md](2026-05-22-runtime-cache-implementation.md). Captured after the bin-packing + schema-trim + debug-logging work landed on `rspec-split-specs`.


Use the latest globally installed plur from this repo, run benchmarks against:

* plur itself
* rubocop ~/src/oss/rubocop
* rspec ~/src/oss/rspec
---

# Measurement procedures

* report osbstacles or blockers, do not skip them
* log full runs directly to a ./tmp log file, then include the key summary here along w/ path to evidence

## Run and compare full wall clock time between v1 and v2 cache implementations



## Run v1 and v2 plur, full suite, in debug mode, so you can analyze the imact from runtimecache load and save