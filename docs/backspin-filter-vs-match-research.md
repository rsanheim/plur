# Research: Filter vs Match in Snapshot Testing Tools

## Overview
This document analyzes how comparable snapshot/golden testing tools handle the distinction between record-time filtering and compare-time matching/normalization.

## Tools Analyzed

### 1. VCR (Ruby)
VCR strongly emphasizes record-time filtering for security:

```ruby
VCR.configure do |c|
  # These are applied at record time - data never hits the cassette
  c.filter_sensitive_data('<API_KEY>') { ENV['API_KEY'] }
  c.filter_sensitive_data('<PASSWORD>') { 'secret123' }
  
  # Limited compare-time options - mainly for matching requests
  c.default_cassette_options = {
    match_requests_on: [:method, :uri]  # Not really normalization
  }
end
```

**Philosophy**: "Sensitive data should never be recorded"
- Cassettes are often checked into version control
- Once filtered, original values are permanently lost
- Very security-focused

### 2. Jest Snapshots (JavaScript)
Jest provides both approaches with equal support:

```javascript
// Record-time: Custom serializers
expect.addSnapshotSerializer({
  test: (val) => val && val.hasOwnProperty('timestamp'),
  print: (val) => {
    const copy = {...val};
    copy.timestamp = '[TIMESTAMP]';
    return JSON.stringify(copy);
  }
});

// Compare-time: Property matchers
expect(user).toMatchSnapshot({
  id: expect.any(Number),
  createdAt: expect.any(Date),
  password: '[REDACTED]'  // But this might already be in snapshot!
});
```

**Philosophy**: "Flexibility for different use cases"
- Property matchers are very popular for handling dynamic content
- Serializers less common but available for security needs
- Emphasizes developer experience

### 3. Insta (Rust)
Insta makes the clearest distinction between the two concepts:

```rust
// Settings can specify both
let mut settings = insta::Settings::new();

// Redactions: Applied at record time to structured data
settings.add_redaction(".id", "[ID]");
settings.add_redaction(".api_key", "[REDACTED]");

// Filters: Applied at compare time to strings
settings.add_filter(r"\d{4}-\d{2}-\d{2}", "[DATE]");
settings.add_filter(r"[\w\-\.]+@[\w\-\.]+", "[EMAIL]");

settings.bind(|| {
    insta::assert_yaml_snapshot!(data);
});
```

**Philosophy**: "Different problems need different solutions"
- **Redactions** = security/privacy (record-time)
- **Filters** = normalization (compare-time)
- Clear naming makes intent obvious

### 4. Approval Tests
Minimal built-in transformation support:

```java
// Most implementations rely on external diff tools
Approvals.verify(output, new Options()
    .withScrubber(s -> s.replaceAll("\\d+ms", "[TIME]ms"))
    .withReporter(new DiffReporter())
);
```

**Philosophy**: "Keep it simple, let humans decide"
- Focus on the approval workflow
- Transformations are often ad-hoc in test code
- Relies on good diff tools

### 5. Go Golden Files
Very simple approach:

```go
// Typically just in test code
got := RunCommand()
if *update {
    ioutil.WriteFile("testdata/golden.txt", []byte(got), 0644)
}
want, _ := ioutil.ReadFile("testdata/golden.txt")

// Normalization happens before comparison
got = normalizeOutput(got)
want = normalizeOutput(want)
```

**Philosophy**: "Convention over configuration"
- No framework features, just patterns
- All transformations in user code
- Simple but requires discipline

## Key Insights

### 1. Security vs Flexibility Trade-off
- **VCR** prioritizes security: filter at record time, data never stored
- **Jest** prioritizes flexibility: compare-time matchers handle most cases
- **Insta** explicitly supports both with different names

### 2. Debugging Experience
Tools that preserve original data (compare-time) provide better debugging:
- Can see what actually changed
- Can adjust normalization without re-recording
- Easier to understand test failures

### 3. Performance Considerations
- Record-time: No runtime overhead, but less flexible
- Compare-time: Runtime overhead, but more flexible

### 4. Common Patterns

**Always filter at record time:**
- Passwords, API keys, tokens
- Personal information (PII)
- Anything that shouldn't be in version control

**Usually normalize at compare time:**
- Timestamps, durations
- Process IDs, random IDs
- File paths (cross-platform)
- Whitespace differences

## Recommendations for Backspin

Based on this research, here's what I recommend:

### 1. Keep Both, But Clarify the Distinction

```ruby
Backspin.run("example",
  # Security: Applied at record time, data never stored
  redact: {
    stdout: [
      [/password: \w+/, "password: [REDACTED]"],
      [/api_key=\w+/, "api_key=[REDACTED]"]
    ]
  },
  # Normalization: Applied at compare time
  normalize: {
    stdout: [
      [/\d+\.\d+ seconds/, "[DURATION]"],
      [/pid: \d+/, "pid: [PID]"]
    ]
  }
)
```

### 2. Use Clear Naming
- `redact` (not `filter`) - makes security intent clear
- `normalize` (not `match`) - makes comparison intent clear
- Avoid ambiguous terms

### 3. Show Both in Recordings
Like Insta, store both versions when redacting:

```yaml
commands:
  - raw: "[REDACTED]"  # User sees redaction happened
    actual_raw: "password: super_secret"  # Never stored
    redacted: true
    stdout: "Login successful"
```

### 4. Provide Guidance
```ruby
# Backspin detects potential issues
# WARNING: Detected possible password in output (line 3)
# Consider using `redact` to prevent storage:
#   redact: { stdout: [[/password: \S+/, "password: [REDACTED]"]] }
```

### 5. Safe Defaults
- Compare-time normalization by default (preserves data)
- Require explicit action for redaction
- Warn when patterns might be too broad

## Conclusion

The filter/match separation in Backspin makes sense, but the naming and purpose could be clearer. Most successful tools recognize that:

1. **Security-sensitive data** needs record-time filtering (redaction)
2. **Dynamic content** needs compare-time normalization
3. These are fundamentally different concerns that deserve different treatment

The best approach is to:
- Keep both mechanisms
- Use clear, intent-revealing names (`redact` vs `normalize`)
- Document when to use each
- Provide good defaults and warnings
- Store enough information for debugging without compromising security