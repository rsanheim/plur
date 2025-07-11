Welcome to Rux! Rux is a fast, friendly test runner & watcher for Ruby.

## Documentation Structure

### Root Level

- `project-status.md` - Overall project status, architecture overview, and performance results
- `rux-optimization-plan.md` - Active optimization planning (needs update with file grouping results)
- `rux-output-and-design.md` - Design philosophy and logging strategy

### `/architecture`
Technical implementation details and system design:
- `rux-watch-architecture.md` - Watch mode implementation details

### `/development`
Developer guides and setup:
- `user-guide.md` - End-user documentation (installation, usage, troubleshooting)
- `go-vendoring-and-ci.md` - CI/CD setup and binary vendoring strategies

### `/research`
Research, analysis, and design explorations:
- `backspin-filter-vs-match-research.md` - Comparison of transformation approaches
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
- **Watch Mode Details**: See `architecture/rux-watch-architecture.md`