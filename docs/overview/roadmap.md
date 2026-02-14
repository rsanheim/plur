# Roadmap

Future plans and potential improvements for Plur.

## Performance

### File Grouping Strategy

* Group small test files together to reduce per-process overhead
* Smart bucketing algorithm that combines short-running files into single worker invocations
* Target: 15-20% improvement on projects with many small spec files

## Features

### Failure Isolation

* Re-run only failed tests on the next invocation
* Track failure state across runs to enable fast feedback loops

### Watch Mode Improvements

* Cancel-and-rerun: interrupt an in-progress test run when new changes arrive
* Process group cleanup: ensure child processes are terminated on Ctrl+C
* Concurrent run guard: prevent overlapping test runs from the debouncer

### CI/CD Optimizations

* Buildkite and GitHub Actions integration for parallel step splitting
* Test timing export for CI-aware load balancing
