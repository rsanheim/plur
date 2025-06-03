# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Backspin is a Ruby gem for characterization testing of command-line interfaces. It records and replays CLI interactions by capturing stdout, stderr, and exit status from shell commands - similar to how VCR works for HTTP interactions. Backspin uses "dubplates" (YAML files) to store recorded command outputs.

## Development Commands

### Setup
```bash
bundle install
bin/setup
```

### Testing
```bash
rake spec                    # Run all tests
rspec spec/[file]           # Run specific test file
rspec spec/[file]:[line]    # Run specific test
```

### Building and Releasing
```bash
bundle exec rake install     # Install gem locally for testing
bundle exec rake release     # Release to RubyGems (updates version, tags, pushes)
```

### Code Quality
```bash
rake standard               # Run Standard Ruby linter
```

## Architecture

### Core Components

**Backspin Module** (`lib/backspin.rb`)
- Main API: `record`, `verify`, `verify!`, `use_dubplate`
- Credential scrubbing logic
- Configuration management

**Command Class** (`lib/backspin.rb`)
- Represents a single CLI execution
- Stores: args, stdout, stderr, status, recorded_at

**Dubplate Class** (`lib/backspin/dubplate.rb`)
- Manages YAML cassette files
- Handles recording/playback sequencing

**RSpecMetadata** (`lib/backspin/rspec_metadata.rb`)
- Auto-generates dubplate names from RSpec context

### Key Design Patterns

- Uses RSpec mocking to intercept `Open3.capture3` calls
- Dubplates are stored as YAML arrays to support multiple commands
- Automatic credential scrubbing for security (AWS keys, API tokens, passwords)
- VCR-style recording modes: `:once`, `:all`, `:none`, `:new_episodes`

### Testing Approach

- Integration-focused tests that exercise the full stack
- Uses temporary directories (`tmp/backspin`) cleaned after each test
- Tests use real shell commands (`echo`, `date`, etc.)
- Configuration is reset between tests to avoid side effects

## Common Development Tasks

### Adding New Features
1. Write integration tests in `spec/backspin/`
2. Implement in appropriate module (usually `lib/backspin.rb`)
3. Update README.md if adding public API
4. Run tests with `rake spec`

### Debugging Tests
- Dubplates are saved to `spec/backspin/` during tests
- Check YAML files to see recorded command outputs
- Use `VERBOSE=1` for additional output during tests

### Updating Credential Patterns
- Add patterns to `DEFAULT_CREDENTIAL_PATTERNS` in `lib/backspin.rb`
- Test with appropriate fixtures in specs