# StayGold vs VCR: A Comparison

## Overview

StayGold is a CLI testing library inspired by VCR's cassette-based recording and playback approach. While VCR focuses on HTTP interactions, StayGold adapts these concepts for command-line interface testing.

## Core Concepts Borrowed from VCR

### 1. Cassette-Based Recording
**VCR**: Records HTTP interactions to YAML/JSON files called "cassettes"
**StayGold**: Records CLI command outputs to YAML files using the same "cassette" terminology

### 2. Record and Playback Pattern
**VCR**: 
```ruby
VCR.use_cassette("api_call") do
  # HTTP request
end
```

**StayGold**:
```ruby
StayGold.record(record_as: "command_output") do
  Open3.capture3("echo hello")
end

StayGold.verify(cassette: "command_output") do
  Open3.capture3("echo hello")
end
```

### 3. YAML Storage Format
Both use YAML for storing recorded data, making cassettes human-readable and editable.

### 4. Verification/Matching
Both provide mechanisms to verify that actual output matches recorded output.

## Key Differences

### 1. Domain Focus
- **VCR**: HTTP requests and responses
- **StayGold**: CLI commands (stdout, stderr, exit status)

### 2. Recording Target
**VCR** records:
- Request (method, URI, headers, body)
- Response (status, headers, body)

**StayGold** records:
- Command arguments
- stdout output
- stderr output
- Exit status
- Timestamp

### 3. Method Interception
- **VCR**: Hooks into HTTP libraries (WebMock, Faraday, etc.)
- **StayGold**: Overrides `Open3.capture3` method

### 4. API Design

**VCR's block-based API**:
```ruby
VCR.use_cassette("cassette_name", record: :once) do
  # Code that makes HTTP requests
end
```

**StayGold's separate record/verify API**:
```ruby
# Recording
StayGold.record(record_as: "name") do
  # CLI commands
end

# Verification
StayGold.verify(cassette: "name", mode: :strict) do
  # CLI commands
end
```

### 5. Modes and Options

**VCR Recording Modes**:
- `:once` - Record if cassette doesn't exist
- `:new_episodes` - Add new interactions
- `:none` - Never record
- `:all` - Always re-record

**StayGold Verification Modes**:
- `:strict` - Exact match of stdout, stderr, and status
- `:playback` - Return recorded data without executing
- Custom matcher support via blocks

### 6. Request Matching vs Output Verification

**VCR**: Matches requests by multiple criteria (method, URI, headers, body)
**StayGold**: Verifies complete output equality or custom matchers

### 7. Auto-naming from Test Context

Both support automatic cassette naming from RSpec context:
- **VCR**: Infers from test description
- **StayGold**: Builds path from RSpec example group hierarchy

### 8. Sensitive Data Handling

**VCR**: Built-in `filter_sensitive_data` configuration
**StayGold**: No built-in filtering (would need custom implementation)

## What Makes Sense for CLI Testing

### 1. Simpler Recording Structure
CLI commands have simpler output structure (stdout/stderr/status) compared to HTTP's request/response with headers, cookies, etc.

### 2. Direct Method Override
StayGold's approach of overriding `Open3.capture3` is simpler than VCR's need to hook into various HTTP libraries.

### 3. Separate Record/Verify
For CLI testing, separating recording and verification makes sense because:
- You often want to record once and verify multiple times
- CLI outputs might need different verification strategies (exact match vs pattern match)

### 4. Playback Mode
StayGold's playback mode is particularly useful for:
- Testing code that depends on CLI output without running actual commands
- Speeding up tests by avoiding repeated command execution

### 5. Exit Status Tracking
Critical for CLI testing but irrelevant for HTTP - StayGold properly tracks and verifies exit codes.

## Future Considerations

### From VCR that could benefit StayGold:
1. **Recording modes** - Implement `:once`, `:new_episodes`, `:all`
2. **Sensitive data filtering** - Add configuration for filtering passwords, tokens
3. **Multiple command matching** - Support for matching commands by different criteria
4. **Configuration block** - Global configuration for cassette directory, defaults

### CLI-specific needs:
1. **Timing information** - Record command execution time
2. **Environment capture** - Record relevant environment variables
3. **Working directory** - Track where commands were executed
4. **Interactive command support** - Handle commands that require input
5. **Binary output handling** - Support for non-text command outputs

## Conclusion

StayGold successfully adapts VCR's proven cassette-based approach to CLI testing while simplifying the API for the command-line domain. The separation of recording and verification, combined with flexible matching strategies, makes it well-suited for testing command-line tools and scripts.