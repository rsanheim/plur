# CLAUDE.md

## Project: Plur - Fast parallel test runner & watcher

### Development

```bash
# Daily workflow commands (in order of frequency):
bin/rake                      # Run full build -> lint, install, tests
bin/rake install              # Build & install a global binary sys/contianer wide
bin/rake test                 # Run full Ruby test suite
bin/rake standard:fix         # Fix Ruby lint issues
```

Notes:
- `bin/rake install` works as-is; no PATH/GOPATH tweaking is required.
- For a single spec file, use `bin/rspec spec/path/to/file_spec.rb`.
- Install tools from top-level `.mise.toml`: `mise install --yes`

### bin/rake build vs bin/rake install

* **bin/rake build** - standard local go build via `go build` (creates `./plur`)

* **bin/rake install** - Production dev install using `goreleaser build` (installs to `$GOPATH/bin/plur`)
  * Version is correctly embedded via ldflags at build time
  * Always shows consistent version regardless of CWD
  * Use this for real verification with other local repos

## Internal Planning Repo

- The private planning repo for this project is `plur-internal` - it lives at "../plur-internal" releative to this repo.
- GitHub: `https://github.com/rsanheim/plur-internal`
- Git remote: `git@github.com:rsanheim/plur-internal.git`
- Keep public product docs in this repo's `docs/` tree focused on current user-facing behavior.
- Put planning docs, research notes, WIP writeups, marketing drafts, and internal design material in `../plur-internal`.

## Quick Reference

### Plur Commands
```bash
plur                      # Run tests (default: 4 workers)
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

Configuration precedence (highest to lowest): CLI flags > environment variables (e.g. `PLUR_WORKERS`) > config files (`.plur.toml` local > `~/.plur.toml` global) > built-in defaults. Env vars beat config files (locked in by #88); see [docs/configuration.md](docs/configuration.md).

See [Configuration Documentation](docs/configuration.md#job-configuration) for full details on creating custom jobs.

### Common Fixes
- **"watcher binary not found"** → Run `plur watch install` to install the binary
- **Testing fixtures is cumbersome** → Use `plur -C fixtures/minitest-success` instead of `cd`
- **Tests fail in rake but pass alone** → Use `bin/rake` not `rake`

### Temporary Files
- **ALWAYS use plur project root `./tmp` directory** for temporary files, never `/tmp` or subproject tmp dirs

### Framework Detection (Updated Behavior)
When both `spec/` and `test/` directories exist, Plur defaults to RSpec
- **Rationale**: RSpec is typically the primary framework in projects with both directories
- **Override**: Use `plur --use=minitest` or set `use = "minitest"` in `.plur.toml`

### Architecture Notes
- Worker pool with goroutines
- Runtime-based test distribution (tracks execution times)
- Channel-based output aggregation (no lock contention)
- We are removing top-level `*.go` files; new Go files must live under `internal/` or another appropriate package, not the repository root.

### Development Cycle
1. Make changes
2. `bin/rake install` - Test globally
3. `bin/rake` - Run everything
4. Fix issues with `bin/rake standard:fix`
5. `git add -A && git commit`

### Git Operations
Keep git operations simple:
* Prefer new commits over amending or rebasing
* Avoid `--force` pushes unless absolutely necessary
* Don't squash commits unnecessarily - commit history is useful

## Testing from Outside-In

ALWAYS use integration specs as guardrails - we have many, see `spec/integration`.

Run all specs via: `bin/rake test` or target specific: `bundle exec rspec spec/[file]`

### Go Testing Guidelines
- Use testify assertions (`assert` and `require`) for all new Go tests
- `require` for critical assertions that should stop the test
- `assert` for non-critical assertions that can continue
- Only add descriptive messages when the assertion itself isn't self-explanatory (e.g., complex conditions or domain-specific checks)

### Race Detection

Enable Go's race detector for debugging concurrent code issues:

```bash
PLUR_RACE=1 bin/rake test:go    # Run Go tests with race detection
PLUR_RACE=1 bin/rake build      # Build race-enabled binary
PLUR_RACE=1 bin/rake install    # Install race-enabled binary
PLUR_RACE=1 bin/rake            # Run everything with race detection
plur doctor                      # Shows "Race Detector: true/false"
```

Race-enabled binaries run 2-20x slower with 5-10x more memory. Use for debugging, not daily use.

### Benchmarking Across Versions

Use `script/bench-git` to compare plur performance across git refs for a given Ruby project. See `script/bench-git --help` for details.

```bash
script/bench-git --refs v0.15.0 v0.14.0 main -p ~/src/oss/rspec-core
```

## GitHub CLI
Prefer the `gh` CLI for searching GitHub or getting info about related repos, issues, etc:
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
- When features are implemented, move them from roadmap to main docs

## ⚠️ No Backward Compatibility Without Explicit Instruction

**NEVER** keep old code around for backward compatibility UNLESS explicitly instructed to do so. This includes:
- No deprecated aliases or wrapper functions, no 'backwards compatibility' comments or code paths
- No maintaining old method names or interfaces
