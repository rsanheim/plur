# Getting Started

Quick start guide for using Plur.

## Prerequisites

- Ruby project with RSpec or Minitest tests
- Go 1.25+ (for building from source)

## Quick Install

```bash
# From source
git clone https://github.com/rsanheim/plur.git
cd plur
go install github.com/goreleaser/goreleaser/v2@latest
bin/rake install

# Verify installation
plur --version
```

## First Run

```bash
# Run all tests with auto-detected parallelism
plur

# Run with specific number of workers
plur -n 4

# See what would run without executing
plur --dry-run
```

## Multiple Frameworks

If your project has both `spec/` (RSpec) and `test/` (Minitest) directories:

```bash
# Select framework with --use flag
plur --use=rspec
plur --use=minitest

# Or set default in .plur.toml
echo 'use = "minitest"' > .plur.toml
```

See [Configuration](configuration.md#job-configuration) for more details.

## Next Steps

For detailed installation options, see [Installation](installation.md).
For comprehensive usage information, see [Usage](usage.md).
