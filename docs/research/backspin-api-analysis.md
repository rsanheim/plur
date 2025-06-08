# Backspin API Analysis: Filters, Matchers, and match_on

## Current State

Backspin currently has three different mechanisms for customizing comparison behavior:

### 1. Filters (Applied During Recording)
```ruby
Backspin.run("example", 
  filter: ->(data) {
    data["stdout"] = normalize_timing(data["stdout"])
    data
  })
```
- **When applied**: During save only
- **Purpose**: Normalize/sanitize data before storage
- **Use case**: Remove non-deterministic values (timestamps, PIDs, etc.)

### 2. Matchers (Applied During Verification)
```ruby
Backspin.run("example",
  matcher: ->(recorded, actual) {
    recorded["status"] == actual["status"] &&
    normalize(recorded["stdout"]) == normalize(actual["stdout"])
  })
```
- **When applied**: During verification only
- **Purpose**: Custom comparison logic for entire command
- **Use case**: Complex cross-tool comparisons

### 3. match_on (Field-Specific Matchers)
```ruby
Backspin.run("example",
  match_on: [:stdout, ->(a, b) { normalize(a) == normalize(b) }])
```
- **When applied**: During verification only
- **Purpose**: Override comparison for specific fields
- **Use case**: Field-specific normalization while keeping other fields exact

## Core Problem: Dynamic Content in CLI Output

The fundamental challenge Backspin addresses is that CLI tools produce output with both:
- **Stable content** - The actual behavior we want to test
- **Dynamic content** - Timestamps, PIDs, paths, random IDs, etc.

Since Backspin is a general-purpose library, it can't know what patterns represent dynamic content for every possible CLI tool. This is why normalization must be user-defined.

## Problems with Current Approach

### 1. Filter/Matcher Separation Confusion
The separation between filters (save-time) and matchers (verify-time) is conceptually confusing:
- Users must understand when each is applied
- Normalization logic often needs to be duplicated
- Mental model doesn't match user intent ("I want to ignore timing differences")

### 2. Asymmetric Normalization
Filters modify the saved data permanently, which means:
- You can't see what was actually captured
- Different tests might need different normalizations of the same output
- Hard to debug when golden files don't match expectations

### 3. Multiple APIs for Similar Goals
Having three different mechanisms (filter, matcher, match_on) for what is fundamentally the same goal (flexible comparison) adds cognitive overhead.

## Proposed Improvements

### 1. Pattern-Based Normalization
Instead of built-in comparators for specific content types, provide a pattern replacement system:

```ruby
Backspin.run("example",
  normalize: {
    stdout: [
      # Replace any decimal number followed by "seconds" with placeholder
      [/\d+\.\d+ seconds/, "[DURATION]"],
      # Replace PID patterns
      [/PID: \d+/, "PID: [PID]"],
      # Replace ISO timestamps
      [/\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}/, "[TIMESTAMP]"],
      # Custom normalization function
      ->(text) { text.gsub(/temp_\w+/, "temp_[ID]") }
    ]
  })
```

This approach:
- Is tool-agnostic
- Makes patterns explicit and debuggable
- Can be shared across tests
- Preserves structure while removing dynamic content

### 2. Dual Storage with Normalization Transparency
Store both raw and normalized versions, making it clear what changed:

```yaml
commands:
  - raw:
      stdout: "Process 12345 completed in 2.456 seconds at 2024-01-15T10:30:45"
    normalized:  
      stdout: "Process [PID] completed in [DURATION] at [TIMESTAMP]"
    normalizations_applied:
      - pattern: '\d{5}'
        replacement: '[PID]'
        matches: ['12345']
      - pattern: '\d+\.\d+ seconds'
        replacement: '[DURATION]'
        matches: ['2.456 seconds']
```

Benefits:
- See exactly what was normalized
- Can verify normalizations are working correctly
- Easy to debug when things don't match

### 3. Structural Comparison Options
For complex outputs, support structural comparison:

```ruby
Backspin.run("json_api",
  normalize: {
    stdout: :parse_json  # Parse as JSON before comparison
  },
  compare: {
    stdout: {
      ignore_fields: ["timestamp", "request_id"],
      ignore_array_order: true,
      fuzzy_numbers: 0.01  # Allow small floating point differences
    }
  })
```

### 4. Learning Mode
Help users identify what needs normalization:

```ruby
# First run: Learning mode
result = Backspin.run("example", mode: :learn) do
  system("my-command")
end

# Backspin runs the command multiple times and reports:
# Detected dynamic content in stdout:
#   - Line 3: "Completed in 1.234 seconds" vs "Completed in 1.567 seconds"
#     Suggested pattern: /\d+\.\d+ seconds/
#   - Line 5: "PID: 12345" vs "PID: 67890"
#     Suggested pattern: /PID: \d+/
```

### 5. Shareable Normalization Profiles
Allow reusable normalization sets:

```ruby
# Define common patterns
module Backspin::Profiles
  RUBY_TEST = {
    normalize: {
      stdout: [
        [/\d+\.\d+ seconds/, "[DURATION]"],
        [/\(files took \d+\.\d+ seconds to load\)/, "(files took [DURATION] to load)"]
      ]
    }
  }
  
  RAILS_SERVER = {
    normalize: {
      stdout: [
        [/Started at .+/, "Started at [TIMESTAMP]"],
        [/Listening on \d+\.\d+\.\d+\.\d+:\d+/, "Listening on [ADDRESS]"]
      ]
    }
  }
end

# Use in tests
Backspin.run("rspec_test", **Backspin::Profiles::RUBY_TEST) do
  system("rspec")
end
```

## Migration Path

### Phase 1: Add Pattern-Based API (Backward Compatible)
```ruby
# Old APIs continue to work
Backspin.run("example", filter: ..., matcher: ...)

# New pattern-based API
Backspin.run("example", 
  normalize: {
    stdout: [[/\d+ seconds/, "[TIME]"]]
  })
```

### Phase 2: Deprecation Warnings
```ruby
# Warning: `filter` is deprecated. Use `normalize` for patterns:
# normalize: { stdout: [[/pattern/, "replacement"]] }
```

### Phase 3: Remove Old APIs (Major Version)
Clean, pattern-based API with clear semantics.

## Use Case Examples

### 1. Cross-Tool Testing (rspec vs rux)
```ruby
# Define normalizations for the tools
timing_patterns = [
  [/\d+\.\d+ seconds/, "[TIME]"],
  [/files took \d+\.\d+ seconds to load/, "files took [TIME] to load"]
]

# First call records rspec output
Backspin.run("rspec_vs_rux",
  normalize: { stdout: timing_patterns }) do
  system("rspec spec/example_spec.rb")
end

# Second call verifies rux matches (with custom matcher for preamble)
Backspin.run!("rspec_vs_rux",
  normalize: { stdout: timing_patterns },
  match_on: [:stdout, ->(rspec, rux) {
    # Skip rux version preamble
    rux_no_preamble = rux.lines[2..].join if rux.include?("rux version")
    rspec.strip == rux_no_preamble.strip
  }]) do
  system("rux spec/example_spec.rb")
end
```

### 2. API Response Testing
```ruby
Backspin.run("api_test",
  normalize: {
    stdout: [
      # Normalize JSON timestamps
      [/"created_at":"[^"]+"/,  '"created_at":"[TIMESTAMP]"'],
      [/"request_id":"[^"]+"/, '"request_id":"[ID]"'],
      # Or use custom function for complex JSON
      ->(json_str) {
        data = JSON.parse(json_str)
        data.delete("timestamp")
        data["items"]&.sort_by! { |i| i["id"] }
        JSON.pretty_generate(data)
      }
    ]
  })
```

### 3. CLI Output Testing
```ruby
Backspin.run("cli_help",
  normalize: {
    stdout: [
      # Version numbers
      [/v\d+\.\d+\.\d+/, "v[VERSION]"],
      # File paths
      [/\/Users\/\w+/, "/Users/[USER]"],
      [/\/home\/\w+/, "/home/[USER]"]
    ]
  })
```

### 4. Database Migration Output
```ruby
Backspin.run("rails_migrate",
  normalize: {
    stdout: [
      # Migration timestamps
      [/== \d{14} \w+:/, "== [TIMESTAMP] [MIGRATION]:"],
      # Execution times
      [/\(\d+\.\d+s\)/, "([TIME]s)"],
      # Schema version
      [/Schema version: \d+/, "Schema version: [VERSION]"]
    ]
  })
```

## Benefits of Pattern-Based Design

1. **Explicit**: Patterns make it clear what's being normalized
2. **Tool-Agnostic**: No assumptions about specific tools or formats
3. **Composable**: Patterns can be combined and reused
4. **Debuggable**: Can see exactly what matched and how it was replaced
5. **Flexible**: Mix patterns with custom functions
6. **Learnable**: Learning mode can suggest patterns

## Advanced Ideas

### 1. Pattern Libraries
```ruby
# Community-contributed patterns
require 'backspin/patterns/ruby'
require 'backspin/patterns/rails'

Backspin.run("test", normalize: {
  stdout: Backspin::Patterns::Ruby::RSPEC_OUTPUT
})
```

### 2. Smart Pattern Detection
```ruby
# Backspin could detect common patterns and suggest normalizations
result = Backspin.analyze("my_recording") do
  3.times { system("my-command") }
end

puts result.suggestions
# Detected variations:
# - /\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}/ (timestamps)
# - /process_\d+/ (process IDs)  
# - /Duration: \d+ms/ (durations)
```

### 3. Interactive Pattern Builder
```ruby
# Interactive mode for building patterns
Backspin.interactive("example") do
  system("my-command")
end
# Opens a UI showing the output with detected dynamic parts highlighted
# User can click to create patterns interactively
```

## Open Questions

1. **Pattern Ordering**: Should patterns be applied in order? What if they overlap?
2. **Performance**: How to efficiently apply many patterns to large outputs?
3. **Security**: Should some patterns be applied during recording (e.g., to remove secrets)?
4. **Reversibility**: Should we support "unnormalizing" for debugging?
5. **Pattern Validation**: How to warn users about patterns that might be too broad?

## Conclusion

The key insight is that Backspin needs to be general-purpose, which means it can't have built-in knowledge of what "timing" or other dynamic content looks like for every possible tool. Instead, a pattern-based approach lets users explicitly define what should be normalized while keeping the library tool-agnostic.

The current filter/matcher/match_on system works but could be simplified into a more cohesive pattern-based API that:
- Makes normalization explicit and debuggable
- Stores both raw and normalized data
- Provides tools to help discover what needs normalization
- Allows sharing patterns between projects and teams

This would make Backspin both more powerful and easier to understand.