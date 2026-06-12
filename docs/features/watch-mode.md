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
- `spec/spec_helper.rb` - Triggers all specs
- `spec/rails_helper.rb` - Triggers all specs

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

These patterns are applied globally before any watch rules are evaluated. You can customize this via the `watch-ignore` config option:

```toml
# .plur.toml
watch-ignore = [".git/**", "node_modules/**", "vendor/**", ".bundle/**"]
```

Setting `watch-ignore` replaces the defaults entirely - include `.git/**` and `node_modules/**` if you still want them ignored.

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

The watcher detects:

- `create` - New files
- `modify` - Content changes (metadata-only changes like `touch` are ignored)
- `destroy` - Deleted files
- `rename` - Renamed files

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
