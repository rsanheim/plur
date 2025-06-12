# Parallel Execution

Rux's core feature is running RSpec tests in parallel for faster feedback.

## How It Works

1. **Test Discovery**: Finds all `*_spec.rb` files recursively
2. **Worker Pool**: Creates N worker processes
3. **Intelligent Distribution**: 
   - Uses historical runtime data when available for optimal load balancing
   - Falls back to file-size-based distribution for new projects
   - Runtime data is automatically collected and cached per project
4. **Aggregation**: Combines results from all workers
5. **Reporting**: Shows unified test results with timing information

## Worker Count

### Auto-Detection (Default)
```bash
rux  # Uses CPU cores - 2
```

### Manual Control
```bash
rux -n 4         # Use 4 workers
rux --workers 8  # Use 8 workers
```

### Environment Variable
```bash
export PARALLEL_TEST_PROCESSORS=6
rux  # Uses 6 workers
```

## Performance Benefits

- **~13% faster** than turbo_tests/parallel_tests
- **Linear scaling** up to CPU core count
- **Low overhead** from Go implementation
- **Efficient I/O** with channel-based aggregation

## Best Practices

1. **Start with defaults** - Auto-detection works well
2. **Monitor CPU usage** - Ensure workers aren't starved
3. **Consider test characteristics**:
   - Many small tests → More workers
   - Few large tests → Fewer workers
   - I/O heavy → More than CPU count

## Comparison with Other Tools

| Feature | Rux | parallel_tests | turbo_tests |
|---------|-----|----------------|-------------|
| Language | Go | Ruby | Ruby |
| Speed | Fastest | Slower | Slow |
| Memory | Low | High | High |
| Setup | Zero-config | Complex | Moderate |