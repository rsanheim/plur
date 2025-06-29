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
rux -C path/to/project   # Change to directory before running (like git -C)
rux --dry-run            # Preview what will run
rux --trace              # Performance profiling
rux doctor               # Debug installation issues
rux watch                # Auto-run tests on file changes (experimental)
rux spec --command=bin/rspec  # Override default test command
```

### Configuration Files

Rux supports TOML configuration files for persistent settings:

```toml
# .rux.toml or ~/.rux.toml
command = "bin/rspec"    # Override default "bundle exec rspec"
workers = 4              # Number of parallel workers
color = true             # Enable colored output
```

Configuration precedence: CLI flags > `.rux.toml` (local) > `~/.rux.toml` (global) > defaults


### Common Fixes
- **"cannot load such file -- backspin"** → `bundle install` at root
- **"go: inconsistent vendoring"** → `cd rux && go mod vendor`
- **"watcher binary not found"** → Binary is embedded and extracted to ~/.cache/rux/bin/
- **Tests fail in rake but pass alone** → Use `bin/rake` not `rake`
- **Testing fixtures is cumbersome** → Use `rux -C fixtures/minitest-success` instead of `cd`

### Project Structure
- `rux/` - Go source (main binary)
- `spec/` - Integration tests (USE THESE as guardrails)
- `default-ruby/` - Example Ruby project for testing
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

### Go Testing Guidelines
- Use testify assertions (`assert` and `require`) for all new Go tests
- `require` for critical assertions that should stop the test
- `assert` for non-critical assertions that can continue
- Only add descriptive messages when the assertion itself isn't self-explanatory (e.g., complex conditions or domain-specific checks)

## Kong CLI Patterns

**IMPORTANT**: When implementing Kong subcommands, be aware that Kong executes commands in reverse order (from deepest subcommand up to parent). Parent commands must check the context to avoid running when a subcommand is invoked. See `docs/development/kong-cli-patterns.md` for critical implementation details.

## GitHub MCP Server Integration

This project includes a GitHub MCP (Model Context Protocol) server configuration for enhanced GitHub integration with Claude Code.

### Features Enabled
The `.mcp.json` configuration enables:
- **Context**: Access repository context and metadata
- **Pull Requests**: Create, review, and manage PRs directly
- **Issues**: Create and manage GitHub issues
- **Repos**: Access repository information and settings

### Usage
Once configured, Claude Code can:
- Create and update PRs and issues
- Review PR changes and provide feedback
- Create and manage issues
- Access repository metadata

## GitHub CLI (`gh`) for Better Control

When searching GitHub repositories or needing more control over the data returned, use the `gh` CLI instead of MCP tools:

### Repository Search Examples
```bash
# Search Go glob libraries with specific fields
gh search repos --language=go --stars=">50" glob --json name,owner,stargazersCount,pushedAt

# Search with custom output format
gh search repos glob --language=go --limit=10 \
  --json name,owner,stargazersCount,pushedAt \
  --jq '.[] | {name, owner: .owner.login, stars: .stargazersCount, updated: .pushedAt}'

# Search code in repositories
gh search code "glob extension:go" --limit=20

# Search issues/PRs
gh search issues "glob" --repo=gobwas/glob --state=open
```

### Direct API Access When Needed
```bash
# Get specific commit info
gh api repos/owner/repo/commits/SHA --jq '{sha: .sha, date: .commit.author.date, message: .commit.message}'

# List branches with just names
gh repo view owner/repo --json defaultBranchRef,refs --jq '.refs.nodes[].name'
```

Key advantages over MCP tools:
- `--json` flag to specify only the fields you need
- Built-in search syntax with proper filters
- `--limit` to control result count
- Cleaner command structure with dedicated subcommands

## Documentation Guidelines

Keep documentation focused on the **current state** of the project:
- Document what exists and works today, not future plans
- Remove inline references to "coming soon", "will support", etc.
- Future plans belong only in `docs/overview/roadmap.md`
- When features are implemented, move them from roadmap to main docs

## Output Formatting

- No ANSI color codes in output (keep it plain text)
- Use simple ASCII for emphasis: `>>>`, `✓`, `✗`
