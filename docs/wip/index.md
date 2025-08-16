# Work in Progress Documentation

This directory contains active development documentation.

## Document Structure

### 📋 Strategy & Planning

| Document | Purpose | Last Updated |
|----------|---------|--------------|  
| [interactive-plur.md](interactive-plur.md) | Interactive config building via `plur watch` - learning mode concept | 2025-08-16 |

### 🔬 Technical Deep Dives

| Document | Purpose | Last Updated |
|----------|---------|--------------|  
| [optimize-watch-tests-with-handlers.md](optimize-watch-tests-with-handlers.md) | Hook-based test automation for file changes | 2025-08-16 |
| [consolidate-watch-and-spec-report.md](consolidate-watch-and-spec-report.md) | Consolidate watch and spec report handlers | - |

### 📁 Reference Files

| File | Purpose |
|------|---------|

## Quick Start

1. **Test Automation Hook**: We've implemented a post-tool-use hook that automatically runs tests when files are edited
   - Located at `script/cc-post-tool-use`
   - Configured in `.claude/settings.json`
   - Runs relevant tests and blocks edits if tests fail

2. **Interactive Watch Mode**: `plur watch find` is an experimental diagnostic tool for exploring file-to-test mappings
   - Works well for RSpec projects (~75-85% success rate)
   - Currently doesn't support Test::Unit/Minitest projects
   - Not yet integrated into main `plur watch` command


## Current Status

### ✅ Completed
* **Post-tool-use hook** (`script/cc-post-tool-use`): Automatically runs tests when files are edited via Claude Code
  * Fixed JSON parsing to use correct `tool_input.file_path` structure
  * Added error logging for debugging hook format changes
  * Successfully blocks edits when tests fail (exit code 2)
  * Allows edits when tests pass (exit code 0)

### 🧪 Experimental
* **`plur watch find`**: Diagnostic command for testing file-to-test mapping discovery
  * Validates if mapped spec files exist
  * Searches for alternative specs when default mappings fail  
  * Suggests custom mapping rules based on discovered alternatives
  * Works well for RSpec projects, needs work for Test::Unit/Minitest

### 📝 Next Steps
* Extend hook to support Go tests and other file types
* Improve `plur watch find` to support Test::Unit/Minitest conventions
* Consider integrating `plur watch find` learnings into main `plur watch` mode
* Add confidence scoring to reduce false positive suggestions
