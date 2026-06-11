# Getting Started

Get up and running with plur for running parallel tests or watch mode.

Plur works with Ruby projects that use RSpec, Minitest, or both. No Ruby gem installation is needed — plur is a standalone binary.

## Installation

### Homebrew (macOS)

```bash
brew install rsanheim/tap/plur
```

### Direct install (all platforms)

The direct install script detects your platform, downloads the latest release, verifies its checksum, and installs the binary to either `~/.local/bin` or `/usr/local/bin`, whichever is found first. 

```bash
curl -fsSL https://github.com/rsanheim/plur/raw/main/install.sh | sh
```

You can override default install dir and/or version with env vars PLUR_VERSION and PLUR_INSTALL_PATH:

```bash
# Install version 0.60.0
curl -fsSL https://github.com/rsanheim/plur/raw/main/install.sh | PLUR_VERSION=v0.60.0 sh
# Install to /usr/local/bin
curl -fsSL https://github.com/rsanheim/plur/raw/main/install.sh | PLUR_INSTALL_PATH=/usr/local/bin sh
```

### Manual GitHub Releases download (all platforms)

Download the latest release for your platform from [GitHub Releases](https://github.com/rsanheim/plur/releases), extract the archive, and place the `plur` binary somewhere on your PATH.

Available platforms:

* macOS ARM64 (Apple Silicon)
* Linux x86_64
* Linux ARM64
* Windows x86_64 (experimental)

## First Run

```bash
# Run all specs or tests with the default worker count
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

## Watch Mode

See [Watch Mode](features/watch-mode.md) for more details.

```bash
# Watch for changes and re-run tests
plur watch

# Install the watcher binary if needed
plur watch install
```

## Next Steps

* [Usage](usage.md) — full command reference
* [Configuration](configuration.md) — `.plur.toml` options and examples
* [Watch Mode](features/watch-mode.md) — auto-run tests on file changes
