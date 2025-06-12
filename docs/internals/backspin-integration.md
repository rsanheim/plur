# Backspin Integration

## Overview

Rux uses [backspin](https://github.com/jasonkarns/backspin) for golden/snapshot testing of CLI output. This integration allows us to record expected behavior once and verify it remains consistent across changes.

## How Rux Uses Backspin

### Integration Tests

Rux's integration tests use backspin to verify that the test runner produces consistent output across different scenarios:

```ruby
# spec/general_integration_spec.rb
Backspin.run("basic_single_file") do
  # Record the output of running a single spec file
  system("cd default-ruby && bundle exec rux spec/models/user_spec.rb")
end
```

### Doctor Command Testing

The `rux doctor` command uses backspin to verify its diagnostic output:

```ruby
# spec/doctor_spec.rb
Backspin.run("rux_doctor", 
  match_on: [:stdout, ->(recorded, actual) {
    # Custom matcher to handle dynamic content like versions
    normalize_output(recorded) == normalize_output(actual)
  }]
) do
  system("rux doctor")
end
```

## Key Concepts

### 1. Recording vs Verification

- **Recording Mode**: When a golden file doesn't exist, backspin records the output
- **Verification Mode**: When a golden file exists, backspin compares new output against it
- **Update Mode**: Set `GOLDEN=true` environment variable to update existing recordings

### 2. Normalization Strategies

Rux uses several strategies to handle dynamic content in test output:

#### Pattern-Based Normalization

```ruby
# Remove timing information that varies between runs
output.gsub(/\d+\.\d+ seconds/, "[DURATION]")

# Normalize file paths
output.gsub(/\/Users\/\w+/, "/Users/[USER]")

# Remove ANSI color codes
output.gsub(/\e\[([;\d]+)?m/, "")
```

#### Custom Matchers

For complex comparisons, backspin supports custom matchers:

```ruby
Backspin.run("cross_tool_comparison",
  match_on: [:stdout, ->(rspec_output, rux_output) {
    # Skip rux version preamble
    rux_normalized = rux_output.lines[2..].join if rux_output.include?("rux version")
    
    # Compare normalized outputs
    normalize(rspec_output) == normalize(rux_normalized)
  }]
)
```

### 3. Golden File Storage

Golden files are stored in `fixtures/backspin/` with descriptive names:

```
fixtures/backspin/
├── basic_single_file.yml
├── rux_doctor_golden.yml
├── parallel_execution.yml
└── error_handling.yml
```

## Best Practices

### 1. Keep Recordings Focused

Each backspin recording should test one specific behavior:

```ruby
# Good: Specific test for parallel execution
Backspin.run("parallel_4_workers") do
  system("rux -n 4")
end

# Bad: Testing too many things at once
Backspin.run("everything") do
  system("rux && rux doctor && rux --help")
end
```

### 2. Handle Dynamic Content Appropriately

Choose the right normalization strategy based on the content type:

- **Timestamps/Durations**: Use pattern replacement
- **Process IDs**: Use pattern replacement
- **File Paths**: Normalize to canonical form
- **Random IDs**: Replace with placeholders
- **Versions**: Consider if changes are significant

### 3. Document Normalization Logic

Always explain why normalization is needed:

```ruby
# Normalize timing because execution speed varies by machine
output.gsub(/Finished in \d+\.\d+ seconds/, "Finished in [DURATION]")

# Remove absolute paths since they differ between environments
output.gsub(Dir.pwd, "[PROJECT_ROOT]")
```

### 4. Review Golden File Changes

When golden files change:

1. Verify the change is intentional
2. Ensure normalization is working correctly
3. Check that the change doesn't break existing functionality
4. Update with `GOLDEN=true bundle exec rspec`

## Common Patterns

### Testing CLI Output

```ruby
RSpec.describe "CLI output" do
  it "produces consistent help text" do
    Backspin.run("help_output") do
      system("rux --help")
    end
  end
end
```

### Cross-Tool Comparison

```ruby
RSpec.describe "RSpec compatibility" do
  it "produces equivalent output to rspec" do
    # First call records rspec output
    Backspin.run("rspec_comparison") do
      system("rspec spec/example_spec.rb")
    end
    
    # Second call verifies rux matches
    Backspin.run!("rspec_comparison",
      match_on: [:stdout, method(:normalize_and_compare)]
    ) do
      system("rux spec/example_spec.rb")
    end
  end
end
```

### Handling Errors

```ruby
RSpec.describe "Error handling" do
  it "captures both stdout and stderr" do
    Backspin.run("error_output",
      # Normalize stack traces which include line numbers
      normalize: {
        stderr: [[/:\d+:in/, ":[LINE]:in"]]
      }
    ) do
      system("rux spec/failing_spec.rb")
    end
  end
end
```

## Debugging Tips

### 1. Viewing Differences

When a test fails, backspin shows a diff of expected vs actual:

```
Expected stdout to match recorded output
Diff:
- Finished in 1.234 seconds
+ Finished in 2.567 seconds
```

### 2. Inspecting Golden Files

Golden files are YAML and can be inspected directly:

```bash
cat fixtures/backspin/basic_single_file.yml
```

### 3. Updating Recordings

To update a specific recording:

```bash
GOLDEN=true bundle exec rspec spec/general_integration_spec.rb -e "basic_single_file"
```

### 4. Temporary Bypass

For debugging, temporarily bypass backspin:

```ruby
# Skip backspin for debugging
if ENV['DEBUG']
  output = `rux doctor 2>&1`
  puts output
else
  Backspin.run("rux_doctor") do
    system("rux doctor")
  end
end
```


## See Also

- [Snapshot Testing](snapshot-testing.md) - General snapshot testing patterns
- [Backspin GitHub Repository](https://github.com/jasonkarns/backspin)
- Research documents:
  - [Filter vs Match Research](../research/backspin-filter-vs-match-research.md)
  - [IO Capture Design](../research/backspin-io-capture-design.md)
  - [API Analysis](../research/backspin-api-analysis.md)