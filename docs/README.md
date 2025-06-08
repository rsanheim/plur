# Rux Documentation

This directory contains all documentation for the Rux project.

## Structure

### Root Level
- `project-status.md` - Overall project status, architecture overview, and performance results
- `rux-optimization-plan.md` - Active optimization planning (needs update with file grouping results)
- `rux-output-and-design.md` - Design philosophy and logging strategy

### `/architecture`
Technical implementation details and system design:
- `rux-watch-architecture.md` - Watch mode implementation details
- `performance-tracing.md` - Guide to using rux's built-in tracing

### `/development`
Developer guides and setup:
- `user-guide.md` - End-user documentation (installation, usage, troubleshooting)
- `go-vendoring-and-ci.md` - CI/CD setup and binary vendoring strategies

### `/research`
Research, analysis, and design explorations:
- `backspin-api-analysis.md` - Analysis of backspin's filter/matcher API design
- `backspin-filter-vs-match-research.md` - Comparison of transformation approaches
- `snapshot-testing-approaches-summary.md` - Snapshot testing patterns across languages
- `file-mapping-config-formats.md` - Config format comparison for watch mappings

### `/_archive`
Completed spikes and historical planning documents (dated):
- `2025-05-28-rspec-package.md` - Completed rspec package extraction
- `2025-05-28-single-stream-full-migration.md` - Completed JSON formatter migration
- `2025-06-03-spike-into-adding-watcher.md` - Initial watcher implementation spike
- `2025-06-04-rux-watch-multiple-watchers-plan.md` - Multi-process watcher architecture
- `2025-06-03-watcher-output-darwin.txt` - Raw watcher output samples

## Quick Links

- **Getting Started**: See `development/user-guide.md`
- **Architecture Overview**: See `project-status.md`
- **Performance Debugging**: See `architecture/performance-tracing.md`
- **Watch Mode Details**: See `architecture/rux-watch-architecture.md`