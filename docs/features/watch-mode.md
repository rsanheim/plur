# Plur Watch Mode

## Overview

Plur watch mode provides automatic test execution when files change, replacing tools like Guard with a zero-configuration, fast file watcher. It's designed to be a "one stop shop" - just run `plur watch` in any Ruby project and get instant feedback as you code.

### Key Features

- **Zero Configuration**: Works out of the box with Ruby/Rails conventions
- **Fast & Efficient**: Native file system events via embedded C++ watcher
- **Multi-Directory Monitoring**: Watches `spec/`, `lib/`, and `app/` simultaneously
- **Smart File Mapping**: Automatically maps source files to their tests
- **Debounced Execution**: Prevents duplicate runs from rapid changes
- **No Dependencies**: Single binary with embedded watcher - no Gemfile changes needed

## Usage

### Basic Usage

```bash
# Start watching for file changes
plur watch
```

### Command Options

```bash
# Dry run to see what would be watched
plur watch --dry-run

# Set custom debounce delay (milliseconds)
plur watch --debounce 250

# Auto-exit after timeout (useful for CI)
plur watch --timeout 60

```

### What Gets Watched

By default, plur watch monitors:
- `spec/**/*_spec.rb` - Test files (runs the changed spec)
- `lib/**/*.rb` - Library files (runs corresponding spec)
- `app/**/*.rb` - Rails app files (runs corresponding spec)

Special files:
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

## Architecture

### Multi-Process Design

Watch mode uses a multi-process architecture where each directory gets its own watcher process:

```
┌─────────────────┐
│   plur watch     │
└────────┬────────┘
         │
┌────────▼────────┐
│ WatcherManager  │
└────────┬────────┘
         │
   ┌─────┴─────┬─────────┬─────────┐
   │           │         │         │
┌──▼──┐    ┌──▼──┐  ┌──▼──┐  ┌──▼──┐
│spec │    │ lib │  │ app │  │test │  (Individual Watcher Processes)
└──┬──┘    └──┬──┘  └──┬──┘  └──┬──┘
   │          │         │         │
   └──────────┴────┬────┴─────────┘
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
3. **FileMapper**: Maps source files to test files using Ruby/Rails conventions  
4. **Debouncer**: Batches rapid changes to prevent duplicate test runs
5. **Embedded Binary**: Platform-specific watcher binaries embedded at compile time

### Event Processing

1. File system change detected by C++ watcher process
2. JSON event emitted via stdout
3. Watcher parses and forwards to WatcherManager
4. Events filtered by file type and effect
5. FileMapper determines which specs to run
6. Debouncer batches changes (default 100ms window)
7. Test runner executes specs using existing plur infrastructure

### Platform Support

Embedded watcher binaries for:
- macOS ARM64 (Apple Silicon)
- macOS x86_64 (Intel)
- Linux ARM64
- Linux x86_64

Binaries are extracted on first use to `~/.cache/plur/bin/`.

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

- Default 100ms delay to batch related changes
- Prevents test runs from overlapping file saves
- Configurable via `--debounce` flag

## Known Issues and Limitations

### Concurrent Output
When multiple file changes occur rapidly, concurrent test runs can execute, leading to:
- Interleaved output from different test runs
- Multiple "plur> " prompts appearing
- Generally "janky" terminal experience

This is a known issue currently. The functionality works correctly despite the output confusion.

### Current Limitations
- Serial test execution only (no parallel mode in watch)
- No support for custom file mappings
- Limited to Ruby/Rails conventions
- No ignore patterns beyond built-in exclusions

## Troubleshooting

### Common Issues

**"watcher binary not found"**
- Binary should auto-extract to `~/.cache/plur/bin/`
- Check permissions on cache directory
- Run `plur doctor` for diagnostics

**Tests not running on file change**
- Verify file is not in .gitignore
- Check that spec file exists at expected location
- Use `plur --debug watch` to see file system events
- Note: metadata-only changes (touch) don't trigger events

### Debug Commands

```bash
# Check watcher status and installation
plur doctor

# See file system events
plur --debug watch

# See what files would be watched
plur watch --dry-run

# Verbose output for debugging
plur watch --verbose
```

## Technical Decision Log

### Why e-dant/watcher?
- Go alternatives have complex macOS support issues
- fsnotify would require CGO, adding complexity
- C++ binary works "out of the box" on all platforms
- Excellent performance and low resource usage

### Why Multiple Processes?
- The watcher binary only processes the first path argument
- Spawning one process per directory was simpler than patching
- Allows independent monitoring with unified event stream
- Clean process isolation and error handling

### Why Embed Binaries?
- Single binary distribution - no runtime downloads
- No network dependencies or version conflicts  
- Simpler installation - just copy plur binary
- Follows Go best practices for self-contained tools