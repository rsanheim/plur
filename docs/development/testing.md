# Testing Rux

## Test Structure

Rux uses integration tests written in RSpec to ensure correctness.

### Running Tests

```bash
# Run all tests
bin/rake test:ruby

# Run specific test file
bundle exec rspec spec/general_integration_spec.rb

# Run with debugging output
DEBUG=1 bundle exec rspec spec/parallel_execution_spec.rb
```

## Integration Test Approach

Tests use the `default-ruby` project as a test fixture:
- Real RSpec tests that can pass or fail
- Controlled environment for testing parallelism
- Predictable output for assertions

## Key Test Files

- `spec/general_integration_spec.rb` - Core rux functionality
- `spec/parallel_execution_spec.rb` - Parallel execution behavior
- `spec/error_handling_spec.rb` - Error cases and edge conditions
- `spec/doctor_spec.rb` - Doctor command functionality

## Golden Testing

We use the backspin gem for golden testing:
- Captures expected output
- Compares against actual output
- Easy to update when output changes intentionally

## Writing New Tests

1. Use real objects over mocks
2. Keep tests focused and readable
3. Test behavior, not implementation
4. Use integration tests for new features