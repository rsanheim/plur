# StayGold: Characterization Testing for CLIs

## Overview

StayGold is a Ruby library for characterization testing of command-line interfaces, inspired by [VCR](https://github.com/vcr/vcr). While VCR records and replays HTTP interactions, StayGold records and replays CLI interactions - capturing stdout, stderr, and exit status from shell commands.

The name "StayGold" comes from the idea of preserving "golden master" outputs - a common pattern in characterization testing where you capture the current behavior of a system as the expected behavior.

## Purpose

- **Test any CLI**: Written in Ruby but can test CLIs in any language via `system`, `Open3`, etc.
- **Regression detection**: Capture current behavior and detect when it changes
- **Speed up tests**: Replay recorded interactions instead of running real commands
- **Document behavior**: YAML cassettes serve as documentation of expected outputs

## Core Concepts

### Cassettes
Like VCR, StayGold stores recordings in "cassettes" - YAML files containing:
- Command arguments
- stdout output
- stderr output
- Exit status
- Timestamp
- (Future) Environment variables
- (Future) Working directory

### Recording Mode
Intercepts CLI calls and saves outputs to cassette files:
```ruby
StayGold.record(record_as: "echo_hello") do
  Open3.capture3("echo hello")
end
```

### Playback Mode (Future)
Replays recorded interactions without executing actual commands:
```ruby
StayGold.use_cassette("echo_hello") do
  Open3.capture3("echo hello")  # Returns recorded output, doesn't run command
end
```

## API Design

### Core Methods

#### `StayGold.record(options = {}, &block)`
Records CLI interactions within the block.

Options:
- `record_as`: Name for the cassette file (without .yaml extension)
- `path`: Directory to store cassettes (default: tmp/stay_gold)
- `record_mode`: :new_episodes, :once, :none, :all (like VCR)

Returns: Result object with:
- `commands`: Array of command objects
- `cassette_path`: Path to the generated cassette file

#### `StayGold.verify(options = {}, &block)`
Verifies CLI output against recorded cassettes.

Options:
- `cassette`: Name of the cassette to verify against (auto-generated if not provided)
- `mode`: Verification mode
  - `:strict` (default) - Exact match of stdout, stderr, and exit status
  - `:playback` - Returns recorded output without running commands
- `matcher`: Custom verification lambda/proc for flexible matching

Returns: VerifyResult object with:
- `verified?`: Whether the output matched expectations
- `output`: The actual stdout
- `diff`: Differences between expected and actual stdout
- `stderr_diff`: Differences between expected and actual stderr
- `command_executed?`: Whether the command was actually run (false in playback mode)
- `error_message`: Human-readable error description

Examples:
```ruby
# Strict verification (default)
result = StayGold.verify(cassette: "echo_test") do
  Open3.capture3("echo hello")
end
expect(result.verified?).to be true

# Playback mode - doesn't run command
result = StayGold.verify(cassette: "slow_command", mode: :playback) do
  Open3.capture3("slow_command")  # Not executed!
end

# Custom matcher for flexible verification
result = StayGold.verify(cassette: "version", 
                        matcher: ->(recorded, actual) { 
                          recorded["stdout"].start_with?("ruby")
                        }) do
  Open3.capture3("ruby --version")
end
```

#### `StayGold.configure(&block)` (Future)
Global configuration:
```ruby
StayGold.configure do |config|
  config.cassette_library_dir = "spec/cassettes"
  config.default_cassette_options = {
    record: :new_episodes
  }
end
```

### Command Interception

Need to intercept various Ruby methods for running commands:
- `Open3.capture3`
- `Open3.capture2`
- `Open3.capture2e`
- `system`
- `Kernel#``
- `%x{}`
- `IO.popen`

Each intercepted command should create a command object with:
- `class`: The method used (e.g., Open3::Capture3)
- `args`: Array of command arguments
- `output`: stdout/stderr/status

## Implementation Constraints

1. **Minimal dependencies**: Keep it simple, maybe just YAML
2. **Non-invasive**: Should not break existing code
3. **Thread-safe**: Multiple tests might run in parallel
4. **Deterministic**: Handle timestamps, PIDs, temp paths
5. **Cross-platform**: Handle Windows vs Unix line endings, paths

## Open Questions

1. **Command parsing**: How to properly parse complex shell commands?
   - `"echo hello"` vs `["echo", "hello"]`
   - Shell expansion, pipes, redirects

2. **Matching strategy**: How to match recorded vs actual commands?
   - Exact match on args?
   - Regular expressions?
   - Custom matchers like VCR?

3. **Sensitive data**: How to filter passwords, API keys?
   - Filter patterns?
   - Before_record hooks?

4. **Binary output**: How to handle non-text output?
   - Base64 encode?
   - Store separately?

5. **Performance**: How to minimize overhead when recording?
   - Lazy loading cassettes?
   - In-memory cache?

6. **Integration with RSpec**: Custom matchers?
   ```ruby
   expect { system("rux --version") }.to stay_gold
   ```

## TODOs

### Phase 1: Minimal Recording (Completed ✅)
- [x] Basic module structure
- [x] Intercept Open3.capture3
- [x] Create Result object
- [x] Write YAML cassettes
- [x] Make existing spec pass

### Phase 2: Enhanced Recording (Partially Complete)
- [ ] Support all command methods (system, backticks, etc.)
- [x] Auto-generated cassette names from RSpec context
- [ ] Configurable cassette directory
- [ ] Handle binary output

### Phase 3: Playback (Completed ✅)
- [x] Load cassettes
- [x] Match commands to recordings
- [x] Return recorded output (playback mode)
- [x] Handle missing recordings (raises CassetteNotFoundError)

### Phase 4: Advanced Features (Partially Complete)
- [ ] Filter sensitive data
- [x] Custom matchers
- [x] Verification modes (strict, playback)
- [ ] Record modes (like VCR)
- [ ] RSpec integration helpers
- [ ] Minitest integration

## Usage Examples

### Basic Recording
```ruby
# spec/cli_spec.rb
it "prints version" do
  result = StayGold.record(record_as: "rux_version") do
    stdout, stderr, status = Open3.capture3("rux --version")
    expect(stdout).to match(/rux v\d+\.\d+\.\d+/)
  end
end
```

### With Playback (Future)
```ruby
it "runs quickly on replay" do
  StayGold.use_cassette("slow_command") do
    # First run: takes 5 seconds
    # Subsequent runs: instant
    system("sleep 5 && echo done")
  end
end
```

### Auto-naming (Future)
```ruby
describe "git commands" do
  around { |ex| StayGold.record(&ex) }
  
  it "shows status" do
    # Cassette: git_commands/shows_status.yaml
    system("git status")
  end
end
```

## Inspiration from VCR

Key concepts to borrow:
1. **Cassette metaphor**: Familiar to Ruby developers
2. **Record modes**: Control when to record vs replay
3. **Configuration block**: Central place for settings
4. **Hook system**: before_record, before_playback
5. **Request matching**: Flexible matching strategies
6. **Serializers**: Pluggable storage formats

## Benefits for Rux Development

1. **Test rux output**: Ensure consistent formatting, colors, timing info
2. **Regression tests**: Detect when output changes unexpectedly  
3. **Performance tests**: Compare execution times across versions
4. **Cross-platform**: Test Unix vs Windows output differences
5. **Documentation**: Cassettes show expected output for different scenarios

## Current Implementation Status

### What's Working
- ✅ Basic recording and playback via `use_cassette`
- ✅ Separate record/verify APIs for explicit control
- ✅ Auto-generated cassette names from RSpec context
- ✅ Multiple verification modes (strict, playback, custom matchers)
- ✅ VCR-compatible record modes (:once, :all, :none, :new_episodes)
- ✅ Comprehensive test coverage (31 specs, 100% passing)

### Known Limitations
- 🚧 Only intercepts `Open3.capture3` (not system, backticks, etc.)
- 🚧 No thread safety (uses singleton method definitions)
- 🚧 No sensitive data filtering
- 🚧 No binary output handling
- 🚧 Simplified :new_episodes mode (just appends)
- 🚧 No command argument matching

### Recommended Usage for Rux

```ruby
# Basic output testing
it "shows version" do
  output = StayGold.use_cassette("rux_version") do
    stdout, _, _ = Open3.capture3("rux --version")
    stdout
  end
  expect(output).to match(/rux v\d+\.\d+\.\d+/)
end

# Performance regression testing
it "completes within time limit" do
  start = Time.now
  StayGold.use_cassette("rux_performance") do
    Open3.capture3("rux spec/fixtures/sample_specs")
  end
  duration = Time.now - start
  
  # First run records actual time
  # Subsequent runs are instant (playback)
  expect(duration).to be < 0.1 # Ensures cassette was used
end

# Error output testing
it "shows helpful error for missing files" do
  _, stderr, status = StayGold.use_cassette("rux_missing_file") do
    Open3.capture3("rux spec/does_not_exist.rb")
  end
  
  expect(stderr).to include("No spec files found")
  expect(status).not_to eq(0)
end
```