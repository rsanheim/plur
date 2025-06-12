# Testing Rux

## Test Structure

Rux uses "outside-in" integration tests written in RSpec to ensure correctness.

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

Many tests use projects from the `fixtures/projects` directory for common scenarios.
The 'default-ruby' project and 'default-rails' project are good starting points.

## Key Test Files

- `spec/general_integration_spec.rb` - Core rux functionality
- `spec/parallel_execution_spec.rb` - Parallel execution behavior
- `spec/error_handling_spec.rb` - Error cases and edge conditions
- `spec/doctor_spec.rb` - Doctor command functionality

## Golden Testing

We use the [backspin](https://github.com/rsanheim/backspin) gem for golden testing in a few spots.
- Records actual output on the first run to records in `fixtures/backspin`
- Compares actual output to the recorded output
- Easy to update

## Writing New Tests

1. Use real objects over mocks
2. Keep tests focused and readable
3. Test behavior, not implementation
4. Use integration tests for new features