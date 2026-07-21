# Plur Watch Mode

## Overview

`plur watch` provides automatic test/spec execution when files change. Imagine [guard](https://github.com/guard/guard), but much faster, zero-config (by default), and no gem/ruby setup necessary. It's designed to be a "one stop shop" - just run `plur watch` in any Ruby project and get instant feedback as you code.

It uses a [fast, lean embedded C++ watcher](https://github.com/e-dant/watcher) to monitor file changes and trigger test/spec execution, using the best platform-specific fsevent library. (FSEvents, inotify, ReadDirectoryChangesW, etc.)

## Usage

```bash
# Start watching for file changes
plur watch

# Dry run to see what would be watched
plur watch --dry-run

# Set custom debounce delay (milliseconds)
plur watch --debounce 250
```

### What Gets Watched

By default, plur watch monitors:

- `spec/**/*_spec.rb` - Test files (runs the changed spec)
- `lib/**/*.rb` - Library files (runs corresponding spec)
- `app/**/*.rb` - Rails app files (runs corresponding spec)

Default watch mappings do not include helper files such as
`spec/spec_helper.rb` or `spec/rails_helper.rb`. Add a project-specific
`[[watch]]` rule if helper changes should run tests.

### File Mapping Examples

| Changed File | Runs |
|--------------|------|
| `lib/foo.rb` | `spec/foo_spec.rb` |
| `lib/foo/bar.rb` | `spec/foo/bar_spec.rb` |
| `app/models/user.rb` | `spec/models/user_spec.rb` |
| `app/controllers/posts_controller.rb` | `spec/controllers/posts_controller_spec.rb` |
| `spec/models/user_spec.rb` | `spec/models/user_spec.rb` (itself) |

### Global Exclusions

By default, events from certain directories are ignored to reduce noise:

* `.git/**` - Git internal files
* `node_modules/**` - JavaScript dependencies

These patterns are applied globally before any watch rules are evaluated. You can customize them in `.plur.toml` with the `watch-ignore` option:

```toml
watch-ignore = [".git/**", "node_modules/**", "vendor/**", ".bundle/**"]
```

Or customize a single watch session with the repeatable `--ignore` flag:

```bash
plur watch --ignore ".git/**" --ignore "node_modules/**" --ignore "vendor/**" --ignore ".bundle/**"
```

Setting either `watch-ignore` or `--ignore` replaces the defaults entirely - include `.git/**` and `node_modules/**` if you still want them ignored.

## Architecture

### Multi-Process Design

Watch mode uses a multi-process architecture. Before spawning watchers, directories are
filtered to remove overlaps (e.g., if watching `.`, subdirectories like `lib/` are removed
to prevent duplicate events):

```
┌─────────────────┐
│   plur watch    │
└────────┬────────┘
         │
  filterWatchDirectories()
  (remove overlaps, validate paths)
         │
┌────────▼────────┐
│ WatcherManager  │
└────────┬────────┘
         │
   ┌─────┴─────┬─────────┐
   │           │         │
┌──▼──┐    ┌──▼──┐  ┌──▼──┐
│  .  │ or │ lib │  │spec │  (Filtered directories → Watcher Processes)
└──┬──┘    └──┬──┘  └──┬──┘
   │          │        │
   └──────────┴───┬────┘
                  │
           ┌──────▼──────┐
           │Event Channel│  (Aggregated Events)
           └──────┬──────┘
                  │
           ┌──────▼──────┐
           │  Debouncer  │
           └──────┬──────┘
                  │
           ┌──────▼──────┐
           │ Test Runner │
           └─────────────┘
```

### Key Components

1. **WatcherManager**: Orchestrates multiple watcher processes, aggregating their events into a single stream
2. **Watcher**: Wrapper around the external C++ watcher binary, one per directory
3. **Planner**: Matches changed files against watch mappings and renders the targets each job runs
4. **Debouncer**: Batches rapid changes to prevent duplicate test runs
5. **Embedded Binary**: Platform-specific watcher binaries embedded at compile time

### Event Processing

1. File system change detected by C++ watcher process
2. JSON event emitted via stdout
3. Watcher parses and forwards to WatcherManager
4. Events filtered by file type and effect, then admitted by the planner (paths outside the project or matching ignore patterns are dropped)
5. Debouncer batches changes (default 30ms window)
6. Planner maps the batched files to job runs via watch mappings
7. Each job run executes, streaming output to the terminal

### Platform Support

Embedded watcher binaries via [e-dant/watcher](https://github.com/e-dant/watcher) auto-installed for.

- macOS ARM64 (Apple Silicon)
- Linux x86_64
- Linux ARM64
- Windows x86_64 (experimental)

Binaries are extracted on first use to `~/.plur/bin/` (or `$PLUR_HOME/bin/`).

## Implementation Details

### Binary Management

The watcher uses [e-dant/watcher](https://github.com/e-dant/watcher), a high-performance C++ file watcher. Platform-specific binaries are embedded in the plur executable using Go's `embed` package and extracted on demand.

### Process Lifecycle

- Each watcher process is kept via standard *nix pipes
- Graceful shutdown on SIGINT/SIGTERM
- Automatic cleanup ensures no zombie processes

### Event Types

Plur triggers a test run on `create` and `modify` events (see the effect-type
filter in `cmd_watch.go`); events with other effect types are skipped.

This means watch mode is driven by **content** changes, not timestamps. A bare
`touch` that only bumps a file's modification time is not treated as a change, so
it does *not* trigger a run. That is deliberate: modern editors, formatters, build
tools, and sync agents churn file timestamps constantly, and reacting to every
mtime bump would make watch mode far too noisy. Real edits always change content
and are always picked up.

### Debouncing

* Default 30ms delay to batch related changes
* Prevents test runs from overlapping file saves
* Configurable via `--debounce` flag

## Known Issues and Limitations

### Concurrent Output
When multiple file changes occur rapidly, concurrent test runs can execute, leading to:

- Interleaved output from different test runs
- Multiple "plur> " prompts appearing
- Generally "janky" terminal experience

This is a known issue currently. The functionality works correctly despite the output confusion.

### Current Limitations

- Serial test execution only (no parallel mode in watch)
- Limited to Ruby/Rails conventions by default (custom mappings available via `[[watch]]` config)

See [Watch Configuration](../configuration.md#watch-configuration) for custom file mapping options.

## Technical Decision Log

### Why e-dant/watcher?

- Go alternatives have troubled macOS history, and fsnotify would require CGO
- C++ binary works "out of the box" on all platforms
- Excellent performance and low resource usage
