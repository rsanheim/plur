# Getting Started

Get up and running with plur for running parallel tests or watch mode.

## Installation

### Homebrew (macOS)

```bash
brew install rsanheim/tap/plur
cd [my-project]
plur --dry-run # preview what would run (no actual test execution)
plur -n 4     # run tests across four cores
plur          # run tests with the default 4 workers
plur watch    # watch for changes and run tests automatically
```

### Shell script (macOS / Linux)

You can use `install.sh` with the following options:

* `--install-path PATH` — installation directory override
* `--version VERSION` — install a specific release tag

```bash
curl -sSL https://raw.githubusercontent.com/rsanheim/plur/main/install.sh | sh
curl -sSL https://raw.githubusercontent.com/rsanheim/plur/main/install.sh | sh -s -- --install-path "/usr/local/bin"
curl -sSL https://raw.githubusercontent.com/rsanheim/plur/main/install.sh | sh -s -- --version v0.5.0
```

### Manual binary download

Download the latest release for your platform from [GitHub Releases](https://github.com/rsanheim/plur/releases), extract the archive, and place the `plur` binary somewhere on your PATH.

Available platforms:

* macOS ARM64 (Apple Silicon)
* Linux x86_64
* Linux ARM64

## Verify

```bash
plur --version

# Check your environment for common issues
plur doctor
```

## Prerequisites

Plur works with Ruby projects that use RSpec, Minitest, or both. No Ruby gem installation is needed — plur is a standalone binary.

## First Run

```bash
# Run all tests with the default worker count
plur

# Run with a specific number of workers
plur -n 4

# Preview what would run without executing anything
plur --dry-run
```

## Multiple Frameworks

If your project has both `spec/` (RSpec) and `test/` (Minitest) directories, use the `--use` flag to select:

```bash
plur --use=rspec
plur --use=minitest
```

Or set a default in your config file:

```toml
# .plur.toml
use = "minitest"
```

## Next Steps

* [Usage](usage.md) — full command reference
* [Configuration](configuration.md) — `.plur.toml` options and examples
* [Watch Mode](features/watch-mode.md) — auto-run tests on file changes
