# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project: Plur - Fast parallel test runner for Ruby/RSpec

Production-ready Go implementation, ~13% faster than turbo_tests/parallel_tests.

## 🚨 IMPORTANT: Always use `bin/rake`, never bare `rake`

```bash
# Daily workflow commands (in order of frequency):
bin/rake install              # Build & install to $GOPATH/bin - USE CONSTANTLY
bin/rake                      # Run ALL tests & lints before committing
bin/rake test:default_ruby    # Test plur on default-ruby fixture project (quick check)
bin/rake test                 # Run full Ruby test suite
bin/rake standard:fix         # Fix Ruby lint issues

# Never do this:
# rake anything         ❌ WRONG - breaks bundler context
# go build             ❌ WRONG - missing version info
# cd plur && go build   ❌ WRONG - use bin/rake install
```

### bin/rake build vs bin/rake install

* **bin/rake build** - Fast local build using `go build` (creates `plur/plur`)
  * Version detection may be incorrect (uses runtime git describe in CWD)
  * Used by CI for speed
  * Fine for testing, not for distribution

* **bin/rake install** - Production dev install using `goreleaser build` (installs to `$GOPATH/bin/plur`)
  * Version is correctly embedded via ldflags at build time
  * Always shows consistent version regardless of CWD
  * Use this for your daily workflow

## Quick Reference

### Plur Commands
```bash
plur                      # Run tests (auto-detect workers)
plur -n 4                 # Specify workers (often fastest)
plur -C path/to/project   # Change to directory before running (like git -C)
plur --dry-run            # Preview what will run
plur doctor               # Debug installation issues
plur watch                # Auto-run tests on file changes (experimental)
plur spec                      # Run tests with detected job
```

### Configuration Files

Plur supports TOML configuration files for persistent settings:

```toml
# .plur.toml or ~/.plur.toml
workers = 4              # Number of parallel workers
color = true             # Enable colored output
use = "rspec"            # Default job to use (can be overridden with --use)

[job.rspec]
cmd = ["bin/rspec"]        # Override default command

[job.custom-lint]
cmd = ["bundle", "exec", "rubocop"]
target_pattern = "**/*.rb"
```

Configuration precedence: CLI flags > `.plur.toml` (local) > `~/.plur.toml` (global) > defaults

See [Configuration Documentation](docs/configuration.md#job-configuration) for full details on creating custom jobs.

### Common Fixes
- **"cannot load such file -- backspin"** → `bundle install` at root
- **"go: inconsistent vendoring"** → `cd plur && go mod vendor`
- **"watcher binary not found"** → Binary is embedded and extracted to ~/.cache/plur/bin/
- **Tests fail in rake but pass alone** → Use `bin/rake` not `rake`
- **Testing fixtures is cumbersome** → Use `plur -C fixtures/minitest-success` instead of `cd`

### Framework Detection (Updated Behavior)
When both `spec/` and `test/` directories exist:
- **Current behavior (as of commit 7796831)**: Plur defaults to RSpec
- **Previous behavior**: Plur defaulted to Minitest
- **Rationale**: RSpec is typically the primary framework in projects with both directories
- **Override**: Use `plur --use=minitest` or set `use = "minitest"` in `.plur.toml`

### Project Structure
- `plur/` - Go source (main binary)
- `spec/` - Full Ruby test suite for plur itself
- `fixtures/projects/default-ruby/` - Ruby fixture project for testing plur
- `fixtures/projects/default-rails/` - Rails fixture project for testing plur
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

### Multi-Platform Builds & Docker

```bash
# Build Linux binaries (amd64 & arm64)
bin/rake build:linux

# Install plur in Docker container
script/install-plur-docker CONTAINER_NAME

# Install in docker-compose container
script/install-plur-docker SERVICE_NAME -C COMPOSE_PREFIX

# Use plur in container
docker exec CONTAINER_NAME plur
```

Cross-compilation uses Go's GOOS/GOARCH with CGO_ENABLED=0 for static binaries.

## Testing from Outside-In

ALWAYS use integration specs as guardrails:
- `spec/integration/plur_spec/general_integration_spec.rb` - Core functionality
- `spec/integration/plur_spec/parallel_execution_spec.rb` - Parallelism
- `spec/integration/plur_spec/error_handling_spec.rb` - Error cases
- `spec/integration/plur_doctor/doctor_spec.rb` - Doctor command with backspin

Run all specs via: `bin/rake test` or target specific: `bundle exec rspec spec/[file]`

### Go Testing Guidelines
- Use testify assertions (`assert` and `require`) for all new Go tests
- `require` for critical assertions that should stop the test
- `assert` for non-critical assertions that can continue
- Only add descriptive messages when the assertion itself isn't self-explanatory (e.g., complex conditions or domain-specific checks)

## Kong CLI Patterns

**IMPORTANT**: When implementing Kong subcommands, be aware that Kong executes commands in reverse order (from deepest subcommand up to parent). Parent commands must check the context to avoid running when a subcommand is invoked. See `docs/development/kong-cli-patterns.md` for critical implementation details.

## MCP Server Integration

This project includes MCP (Model Context Protocol) servers configured in `.mcp.json`:

- **GitHub MCP**: Create/manage PRs and issues, access repo metadata
- **CircleCI MCP**: Check CI status, run pipelines, debug failures

### Quick CI Status Check
```bash
# Check current branch CI status via CircleCI MCP
mcp__circleci__get_latest_pipeline_status
```

For complex GitHub searches, prefer `gh` CLI for better control:
```bash
# Search with specific fields
gh search repos --language=go --stars=">50" glob --json name,owner,stargazersCount

# Get commit info
gh api repos/owner/repo/commits/SHA --jq '{sha: .sha, message: .commit.message}'
```

## Documentation Guidelines

Keep documentation focused on the **current state** of the project:
- Document what exists and works today, not future plans
- Remove inline references to "coming soon", "will support", etc.
- Future plans belong only in `docs/overview/roadmap.md`
- When features are implemented, move them from roadmap to main docs

## Output Formatting

- No ANSI color codes in output (keep it plain text)
- Use simple ASCII for emphasis: `>>>`, `✓`, `✗`

## ⚠️ No Backward Compatibility Without Explicit Instruction

**NEVER** keep old code around for backward compatibility unless explicitly instructed to do so. This includes:
- No deprecated aliases or wrapper functions
- No "backward compatibility" comments or code paths
- No maintaining old method names or interfaces
- Remove old code immediately when refactoring

When renaming or refactoring:
1. Change the code directly
2. Update all references
3. Delete the old implementation completely
4. Do NOT leave deprecated versions "for compatibility"

This is a hard rule. Break things if needed - we prefer clean breaks over technical debt.

## Planning and Estimates

**NEVER** include time estimates or effort calculations in plans or documentation. Focus on:
- Clear description of what needs to be done
- Dependencies and ordering
- Testing strategy
- Success criteria

Skip any discussion of "hours", "days", or "effort estimates" - we don't track or care about those.
