# Getting Started

Quick start guide for using Rux.

## Prerequisites

- Ruby project with RSpec tests
- Go 1.21+ (for building from source)

## Quick Install

```bash
# From source
git clone https://github.com/rsanheim/rux-meta.git
cd rux-meta
bin/rake install

# Verify installation
rux --version
```

## First Run

```bash
# Run all tests with auto-detected parallelism
rux

# Run with specific number of workers
rux -n 4

# See what would run without executing
rux --dry-run
```

For detailed installation options, see [Installation](installation.md).
For comprehensive usage information, see [Usage](usage.md).
