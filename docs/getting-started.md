# Getting Started

Quick start guide for using Plur.

## Prerequisites

- Ruby project with RSpec tests
- Go 1.21+ (for building from source)

## Quick Install

```bash
# From source
git clone https://github.com/rsanheim/plur-meta.git
cd plur-meta
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

For detailed installation options, see [Installation](installation.md).
For comprehensive usage information, see [Usage](usage.md).
