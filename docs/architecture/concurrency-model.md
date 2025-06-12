# Concurrency Model

## Worker Pool Architecture

Rux uses a goroutine-based worker pool for parallel test execution.

### Key Components

- **Worker Goroutines**: Each worker runs tests independently
- **Job Channel**: Distributes test files to workers
- **Result Channel**: Collects test results from workers
- **Aggregator**: Combines results and handles output

### Distribution Strategy

Currently uses round-robin distribution. Future versions will support:
- Runtime-based distribution
- File-size based grouping
- Custom distribution strategies

## Channel Design

Channels provide thread-safe communication:
- Buffered job channel prevents blocking
- Unbuffered result channel for backpressure
- No lock contention in output aggregation

## Resource Management

- Worker count auto-detection: `CPU cores - 2`
- Graceful shutdown on interrupts
- Process cleanup on worker failures