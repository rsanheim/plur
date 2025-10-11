# Work in Progress Documentation

This directory contains active development documentation.

## Document Structure

### 📋 Strategy & Planning

| Document | Purpose | Last Updated |
|----------|---------|--------------|
| [interactive-plur.md](interactive-plur.md) | Interactive config building via `plur watch` - learning mode concept | 2025-08-16 |
| [watcher-packaging-strategy.md](watcher-packaging-strategy.md) | Strategy for packaging watcher binaries across platforms | 2025-08-27 |

### 🔬 Technical Deep Dives

| Document | Purpose | Last Updated |
|----------|---------|--------------|
| [consolidate-watch-and-spec-report.md](consolidate-watch-and-spec-report.md) | Consolidate watch and spec report handlers | - |

### ✅ Recently Completed (Moved)

| Document | New Location | Status |
|----------|-------------|--------|
| GoReleaser Implementation | [Release Infrastructure](../development/releases/index.md) | Completed Oct 2025 |

### 📁 Reference Files

| File | Purpose |
|------|---------|

## Quick Start

1. **Interactive Watch Mode**: `plur watch find` is an experimental diagnostic tool for exploring file-to-test mappings
   - Works well for RSpec projects (~75-85% success rate)
   - Currently doesn't support Test::Unit/Minitest projects
   - Not yet integrated into main `plur watch` command



## Current Status

### 🧪 Experimental
* **`plur watch find`**: Diagnostic command for testing file-to-test mapping discovery
  * Validates if mapped spec files exist
  * Searches for alternative specs when default mappings fail  
  * Suggests custom mapping rules based on discovered alternatives
  * Works well for RSpec projects, needs work for Test::Unit/Minitest

### 📝 Next Steps
* Improve `plur watch find` to support Test::Unit/Minitest conventions
* Consider integrating `plur watch find` learnings into main `plur watch` mode
* Add confidence scoring to reduce false positive suggestions
