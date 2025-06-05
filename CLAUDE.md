# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project: Rux - Fast parallel test runner for Ruby/RSpec

Production-ready Go implementation, ~13% faster than turbo_tests/parallel_tests.

## 🚨 CRITICAL: Always use `bin/rake`, never bare `rake`

```bash
# Daily workflow commands (in order of frequency):
bin/rake install         # Build & install to $GOPATH/bin - USE CONSTANTLY
bin/rake                 # Run ALL tests & lints before committing
bin/rake test:ruby       # Run integration specs only
bin/rake standard:fix    # Fix Ruby lint issues

# Never do this:
# rake anything         ❌ WRONG - breaks bundler context  
# go build             ❌ WRONG - missing version info
# cd rux && go build   ❌ WRONG - use bin/rake install
```

## Quick Reference

### Rux Commands
```bash
rux                      # Run tests (auto-detect workers)
rux -n 4                 # Specify workers (often fastest)
rux --dry-run            # Preview what will run
rux --trace              # Performance profiling
rux doctor               # Debug installation issues
rux watch                # Auto-run tests on file changes (experimental)
```

### Common Fixes
- **"cannot load such file -- backspin"** → `bundle install` at root
- **"go: inconsistent vendoring"** → `cd rux && go mod vendor`
- **"watcher binary not found"** → Binary is embedded and extracted to ~/.cache/rux/bin/
- **Tests fail in rake but pass alone** → Use `bin/rake` not `rake`

### Project Structure
- `rux/` - Go source (main binary)
- `spec/` - Integration tests (USE THESE as guardrails)
- `rux-ruby/` - Example Ruby project for testing
- `vendor/backspin/` - Vendored golden testing gem

### Architecture Notes
- Worker pool with goroutines
- Runtime-based test distribution (tracks execution times)
- Channel-based output aggregation (no lock contention)
- Compatible with PARALLEL_TEST_PROCESSORS env var

### Development Cycle
1. Make changes
2. `bin/rake install` - Test globally
3. `bin/rake` - Run everything
4. Fix issues with `bin/rake standard:fix`
5. `git add -A && git commit`

## Testing from Outside-In

ALWAYS use integration specs as guardrails:
- `spec/general_integration_spec.rb` - Core functionality
- `spec/parallel_execution_spec.rb` - Parallelism
- `spec/error_handling_spec.rb` - Error cases
- `spec/doctor_spec.rb` - Doctor command with backspin

Run via: `bin/rake test:ruby` or `bundle exec rspec spec/[file]`