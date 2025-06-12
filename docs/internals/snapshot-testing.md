# Snapshot Testing

## Overview

Snapshot testing (also known as golden testing or approval testing) is a testing technique where:

1. You capture the output of your code once (the "snapshot" or "golden" file)
2. Future test runs compare new output against this snapshot
3. Changes are flagged as test failures unless explicitly approved

This approach is particularly valuable for testing CLI tools, where the exact output format matters.

## Why Snapshot Testing?

### Benefits

- **Comprehensive Coverage**: Captures entire output, not just specific assertions
- **Easy Updates**: When output intentionally changes, update the snapshot
- **Visual Diffs**: See exactly what changed when tests fail
- **Low Maintenance**: No need to update multiple assertions when output format changes

### Challenges

- **Dynamic Content**: Timestamps, IDs, and paths vary between runs
- **Cross-Platform**: File paths and line endings differ
- **Noise**: Minor formatting changes can cause failures
- **Review Burden**: Snapshot updates need careful review

## Snapshot Testing in Rux

Rux uses snapshot testing extensively to ensure compatibility with RSpec's output format.

### Example: Basic Snapshot Test

```ruby
RSpec.describe "Rux output" do
  it "matches expected format for a simple test run" do
    Backspin.run("simple_test") do
      system("rux spec/example_spec.rb")
    end
  end
end
```

### Handling Dynamic Content

Dynamic content is the biggest challenge in snapshot testing. Here are strategies Rux uses:

#### 1. Pattern Replacement

Replace dynamic patterns with stable placeholders:

```ruby
def normalize_output(output)
  output
    .gsub(/\d+\.\d+ seconds/, "[DURATION]")
    .gsub(/pid: \d+/, "pid: [PID]")
    .gsub(/\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}/, "[TIMESTAMP]")
end
```

#### 2. Structural Comparison

For structured output (JSON, YAML), compare structure rather than exact text:

```ruby
def compare_json_output(expected, actual)
  expected_data = JSON.parse(expected)
  actual_data = JSON.parse(actual)
  
  # Ignore specific fields
  expected_data.delete("timestamp")
  actual_data.delete("timestamp")
  
  expect(actual_data).to eq(expected_data)
end
```

#### 3. Fuzzy Matching

Allow small variations in numeric values:

```ruby
def numbers_match?(expected, actual, tolerance = 0.01)
  expected_num = expected.to_f
  actual_num = actual.to_f
  (expected_num - actual_num).abs <= tolerance
end
```

## Patterns and Anti-Patterns

### Good Patterns

#### 1. Focused Snapshots

Each snapshot should test one specific behavior:

```ruby
# Good: Specific snapshot for error formatting
Backspin.run("error_format") do
  system("rux spec/failing_spec.rb")
end

# Good: Separate snapshot for parallel execution
Backspin.run("parallel_output") do
  system("rux -n 4")
end
```

#### 2. Semantic Normalization

Normalize based on meaning, not just format:

```ruby
# Good: Semantic placeholder
output.gsub(/Took \d+\.\d+ seconds/, "Took [DURATION]")

# Less clear: Generic pattern
output.gsub(/\d+\.\d+/, "[NUMBER]")
```

#### 3. Documented Normalization

Explain why each normalization exists:

```ruby
output
  # Remove ANSI codes since they're not relevant to the test
  .gsub(/\e\[([;\d]+)?m/, "")
  # Normalize paths since they differ between machines
  .gsub(Dir.pwd, "[PROJECT_ROOT]")
  # Remove timing since it's non-deterministic
  .gsub(/\d+\.\d+s/, "[TIME]")
```

### Anti-Patterns

#### 1. Over-Normalization

Don't normalize away important differences:

```ruby
# Bad: Removes too much information
output.gsub(/\d+/, "[NUMBER]")  # Loses important counts

# Good: Specific normalization
output.gsub(/pid: \d+/, "pid: [PID]")  # Only PIDs
```

#### 2. Fragile Patterns

Avoid patterns that break with minor changes:

```ruby
# Bad: Too specific to current format
output.gsub(/^Done in \d+\.\d+ seconds\.$/, "[TIMING]")

# Good: More flexible
output.gsub(/\d+\.\d+ seconds/, "[DURATION]")
```

#### 3. Mixing Concerns

Keep test logic separate from normalization:

```ruby
# Bad: Business logic in normalization
def normalize(output)
  if output.include?("error")
    "TEST FAILED"  # Don't hide actual errors!
  else
    output
  end
end

# Good: Preserve actual output
def normalize(output)
  output.gsub(/at line \d+/, "at line [LINE]")
end
```

## Advanced Techniques

### 1. Multi-Stage Normalization

Apply normalizations in a specific order:

```ruby
class OutputNormalizer
  STAGES = [
    :remove_ansi_codes,
    :normalize_paths,
    :normalize_timestamps,
    :normalize_durations
  ]
  
  def normalize(output)
    STAGES.reduce(output) do |text, stage|
      send(stage, text)
    end
  end
  
  private
  
  def remove_ansi_codes(text)
    text.gsub(/\e\[([;\d]+)?m/, "")
  end
  
  def normalize_paths(text)
    text
      .gsub(Dir.pwd, "[PWD]")
      .gsub(Dir.home, "[HOME]")
      .gsub(/\/tmp\/\w+/, "/tmp/[TMPDIR]")
  end
  
  # ... other stages
end
```

### 2. Recording Modes

Support different recording strategies:

```ruby
module SnapshotMode
  # Always update snapshots
  RECORD = :record
  
  # Update only if missing
  RECORD_MISSING = :record_missing
  
  # Never update (CI mode)
  VERIFY_ONLY = :verify_only
  
  # Interactive approval
  APPROVE = :approve
end

Backspin.configure do |config|
  config.mode = ENV["CI"] ? SnapshotMode::VERIFY_ONLY : SnapshotMode::RECORD_MISSING
end
```

### 3. Snapshot Metadata

Store additional context with snapshots:

```ruby
Backspin.run("complex_test",
  metadata: {
    ruby_version: RUBY_VERSION,
    platform: RUBY_PLATFORM,
    recorded_at: Time.now.iso8601,
    git_sha: `git rev-parse HEAD`.strip
  }
) do
  system("rux spec/")
end
```

### 4. Differential Snapshots

Record only differences from a baseline:

```ruby
# Record baseline
baseline = Backspin.run("rspec_baseline") do
  system("rspec")
end

# Record only differences
Backspin.run("rux_differences",
  baseline: baseline,
  record_only: :differences
) do
  system("rux")
end
```

## Tools and Alternatives

### Ruby Tools

1. **Backspin** - Rux's choice, simple and focused
2. **Approvals** - Port of ApprovalTests
3. **VCR** - Primarily for HTTP, but demonstrates patterns

### Other Languages

1. **Jest Snapshots** (JavaScript) - Integrated snapshot testing
2. **Insta** (Rust) - Powerful with inline snapshots
3. **Go Golden Files** - Convention-based approach

### Key Differences

- **Storage Format**: YAML vs JSON vs custom
- **Inline vs External**: Snapshots in test files vs separate files
- **Update Mechanism**: Environment variables vs CLI flags vs interactive

## Best Practices for Rux

### 1. Snapshot Organization

```
fixtures/backspin/
├── output/           # CLI output snapshots
│   ├── help.yml
│   ├── version.yml
│   └── doctor.yml
├── integration/      # Full integration test snapshots
│   ├── single_file.yml
│   ├── parallel.yml
│   └── rails_app.yml
└── compatibility/    # RSpec compatibility snapshots
    ├── formatters.yml
    ├── options.yml
    └── reporter.yml
```

### 2. Review Process

When updating snapshots:

1. **Understand the Change**: Why did the output change?
2. **Verify Correctness**: Is the new output correct?
3. **Check Side Effects**: Does this affect other tests?
4. **Update Documentation**: Document significant changes

### 3. CI Integration

```ruby
# CI configuration
RSpec.configure do |config|
  if ENV["CI"]
    # Fail if snapshots would change
    Backspin.configure do |c|
      c.mode = :verify_only
    end
  end
end
```

### 4. Debugging Failed Snapshots

```ruby
# Temporarily disable normalization for debugging
Backspin.run("debug_test",
  normalize: ENV["DEBUG"] ? {} : { stdout: [...] }
) do
  system("rux spec/failing_spec.rb")
end
```


## Conclusion

Snapshot testing is a powerful technique for ensuring CLI tools like Rux maintain consistent behavior. The key to success is:

1. **Strategic Normalization**: Handle dynamic content without losing important information
2. **Clear Organization**: Keep snapshots focused and well-organized
3. **Good Tooling**: Use tools like backspin that make snapshot testing easy
4. **Team Practices**: Establish clear processes for reviewing snapshot changes

When done well, snapshot testing provides confidence that changes don't break existing behavior while making it easy to evolve the tool's output format.

## See Also

- [Backspin Integration](backspin-integration.md) - How Rux specifically uses backspin
- [Testing Best Practices](/docs/testing.md) - General testing guidelines
- Research documents:
  - [Backspin API Analysis](../research/backspin-api-analysis.md)
  - [Filter vs Match Research](../research/backspin-filter-vs-match-research.md)