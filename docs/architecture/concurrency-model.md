# Concurrency Model

## Worker Pool Architecture

Plur uses a goroutine-based worker pool for parallel test execution.

### Key Components

- **Worker Goroutines**: Each worker runs tests independently
- **Job Channel**: Distributes test files to workers
- **Result Channel**: Collects test results from workers
- **Aggregator**: Combines results and handles output

### Distribution Strategy

Plur uses intelligent test distribution with two strategies:

1. **Runtime-based distribution** (preferred): When historical runtime data is available, tests are distributed based on their previous execution times to balance workload across workers
2. **Size-based distribution** (fallback): When no runtime data exists, tests are distributed based on file sizes to approximate workload balance

Runtime data is automatically collected and stored in `~/.cache/plur/runtimes/{project-hash}.json`.

## Channel Design

Channels provide thread-safe communication:
- Buffered job channel prevents blocking
- Unbuffered result channel for backpressure
- No lock contention in output aggregation

## Resource Management

- Worker count auto-detection: `CPU cores - 2`
- Graceful shutdown on interrupts
- Process cleanup on worker failures