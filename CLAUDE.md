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

### Kong CLI (Experimental)
```bash
./rux-kong               # Use Kong-based CLI (sets KONG=1 automatically)
./rux-kong --help        # Show Kong CLI help and available flags
./rux-kong -n 4          # Run with Kong CLI using 4 workers
./rux-kong watch         # Watch mode with Kong CLI

# Kong CLI is experimental - tracks the same functionality but with
# cleaner argument parsing via github.com/alecthomas/kong
```

### Common Fixes
- **"cannot load such file -- backspin"** → `bundle install` at root
- **"go: inconsistent vendoring"** → `cd rux && go mod vendor`
- **"watcher binary not found"** → Binary is embedded and extracted to ~/.cache/rux/bin/
- **Tests fail in rake but pass alone** → Use `bin/rake` not `rake`

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

## GitHub MCP Server Integration

This project includes a GitHub MCP (Model Context Protocol) server configuration for enhanced GitHub integration with Claude Code.

### Setup Requirements
1. **Docker** must be installed and running
2. **GitHub Personal Access Token** configured in 1Password at `op://Private/github-mcp-rux-meta/credential`
3. **1Password CLI** (`op`) must be installed and authenticated

### Features Enabled
The `.mcp.json` configuration enables:
- **Context**: Access repository context and metadata
- **Pull Requests**: Create, review, and manage PRs directly
- **Issues**: Create and manage GitHub issues
- **Repos**: Access repository information and settings

### Usage
Once configured, Claude Code can:
- Create PRs with `gh pr create` command
- Review PR changes and provide feedback
- Create and manage issues
- Access repository metadata

Note: The MCP server configuration is project-scoped and shared with all team members via `.mcp.json`.

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