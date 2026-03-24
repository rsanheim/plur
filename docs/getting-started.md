# Getting Started

Everything you need to install plur and run your first parallel test suite.

## Installation

### Homebrew (macOS)

```bash
brew install rsanheim/tap/plur
```

### Shell script (macOS / Linux)

```bash
curl -sSL https://raw.githubusercontent.com/rsanheim/plur/main/install.sh | sh
```

Installs to `~/.local/bin` by default.

Installer options:

* `--help` — show installer usage
* `--install-path PATH` — installation directory override
* `PLUR_INSTALL_PATH` — installation directory (default: `~/.local/bin`)

Pin a specific version:

```bash
curl -sSL https://raw.githubusercontent.com/rsanheim/plur/main/install.sh | sh -s -- v0.5.0
```

Override install path and pin version:

```bash
curl -sSL https://raw.githubusercontent.com/rsanheim/plur/main/install.sh | sh -s -- --install-path "$HOME/.local/bin" v0.5.0
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
# Run all tests with auto-detected parallelism
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
