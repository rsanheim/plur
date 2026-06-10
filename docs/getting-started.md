# Getting Started

Get up and running with plur for running parallel tests or watch mode.

Plur works with Ruby projects that use RSpec, Minitest, or both. No Ruby gem installation is needed — plur is a standalone binary.

## Installation

### Homebrew (macOS)

```bash
brew install rsanheim/tap/plur
```

### Shell script (macOS / Linux)

```bash
curl -fsSL https://github.com/rsanheim/plur/raw/main/install.sh | sh
```

The script detects your platform, downloads the latest release, verifies its checksum, and installs the binary. It installs to `~/.local/bin`, or `/usr/local/bin` if `~/.local/bin` doesn't exist.

Two environment variables configure it:

* `PLUR_VERSION` — release tag to install (default: latest release)
* `PLUR_INSTALL_PATH` — install directory

```bash
curl -fsSL https://github.com/rsanheim/plur/raw/main/install.sh | PLUR_VERSION=v0.60.0 sh
curl -fsSL https://github.com/rsanheim/plur/raw/main/install.sh | PLUR_INSTALL_PATH=/usr/local/bin sh
```

### Manual binary download

Download the latest release for your platform from [GitHub Releases](https://github.com/rsanheim/plur/releases), extract the archive, and place the `plur` binary somewhere on your PATH.

Available platforms:

* macOS ARM64 (Apple Silicon)
* Linux x86_64
* Linux ARM64
* Windows x86_64 (experimental)

## Verify

```bash
plur --version

# Check your environment for common issues
plur doctor
```

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
