# Rux Meta

This repository contains the research and development for `rux`, a fast Go-based test runner for Ruby/RSpec projects.

## Goal

Create a CLI tool called `rux` that provides:
- **Parallel test execution** out of the box
- **Interleaved output** similar to turbo_tests
- **Speed** through Go's performance characteristics
- **Simple binary distribution** without Ruby dependencies

## Why Go?

- Fast compilation and execution
- Easy cross-platform binary distribution
- No Ruby runtime dependency for the runner itself
- Learning opportunity for Go development

## Research Dependencies

This repo includes two existing Ruby test runners for study:

### `parallel_tests/`
A mature parallel test runner for Ruby frameworks. Provides insights into:
- Test file grouping strategies
- Process management
- Output handling across multiple workers

### `turbo_tests/`
A fast RSpec runner that acts as a **shim on top of parallel_tests**. It leverages parallel_tests' core functionality while adding its own process/thread management for excellent interleaved output. Key features to study:
- Real-time output streaming
- JSON-based communication between processes
- Clean result formatting
- How to build on existing parallel execution foundations

## Current Status

🚧 **In Development** - Currently studying the existing implementations to design the Go version.

## Structure

```
rux-meta/
├── rux/           # Go implementation (main project)
├── parallel_tests/ # Reference: parallel execution patterns
├── turbo_tests/   # Reference: interleaved output implementation
└── README.md      # This file
```

The `rux/` directory contains the actual Go implementation, while the other directories serve as reference implementations for understanding proven approaches to parallel Ruby test execution.