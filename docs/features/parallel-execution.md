# Parallel Execution

Plur's core feature is running RSpec tests in parallel for faster feedback.

## How It Works

1. **Test Discovery**: Finds test files based on patterns (supports globs like `**/*_spec.rb`)
2. **Worker Pool**: Creates N worker processes
3. **Intelligent Distribution**: 
   - Uses saved runtime data when available for optimal load balancing
   - Falls back to file-size-based distribution for new projects
   - Runtime data is automatically collected and cached per project
4. **Aggregation**: Combines results from all workers
5. **Reporting**: Shows unified test results with timing information

## Worker Count

### Auto-Detection (Default)
```bash
plur  # Uses CPU cores - 2
```

### Manual Control
```bash
plur -n 4         # Use 4 workers
plur --workers 8  # Use 8 workers
PARALLEL_TEST_PROCESSORS=10 plur # Use 10 workers
```