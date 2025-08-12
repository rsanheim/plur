# Plur Watch Architecture

## Overview

The `plur watch` command provides automatic test execution when files change, using a multi-process architecture for efficient file system monitoring.

## Key Components

### 1. WatcherManager (`watch/watcher_manager.go`)
- Central orchestrator that manages multiple watcher processes
- Creates one watcher process per directory (spec, lib, app)
- Aggregates events from all watchers into a single event stream
- Handles graceful shutdown and process cleanup

### 2. Watcher (`watch/watcher.go`)
- Wrapper around the external watcher binary (C++ fsnotify implementation)
- Each instance monitors a single directory
- Communicates via JSON events over stdout/stderr
- Keeps process alive via stdin pipe

### 3. FileMapper (`watch/file_mapper.go`)
- Maps source files to their corresponding test files
- Supports Rails conventions (app/models → spec/models)
- Handles special cases (spec_helper.rb runs all specs)

### 4. Debouncer (`watch/debouncer.go`)
- Prevents duplicate test runs when multiple files change rapidly
- Configurable delay (default 100ms)
- Batches related changes together

## Event Flow

```
File System Change
    ↓
Watcher Binary (C++ process) detects change
    ↓
JSON Event via stdout → Watcher.readEvents()
    ↓
Watcher.eventChan
    ↓
WatcherManager.aggregateEvents() (goroutine per watcher)
    ↓
WatcherManager.eventChan (unified stream)
    ↓
Main Watch Loop (watch.go)
    ↓
Event Filtering (file type, effect type, should watch)
    ↓
FileMapper.MapFileToSpecs() (relative path → spec files)
    ↓
Debouncer.Debounce() (batch changes, prevent duplicates)
    ↓
runSpecsOrDirectory() (execute tests)
```

## Multi-Process Design

The key insight that led to this architecture: the watcher binary only processes the first path argument, so we spawn one process per directory:

- `spec/` → watcher process 1
- `lib/` → watcher process 2  
- `app/` → watcher process 3 (if exists)

All events are aggregated into a single channel for unified processing.

## Event Types

The watcher binary emits JSON events with the following structure:
```json
{
  "path_type": "file",
  "path_name": "/path/to/file.rb",
  "effect_type": "modify",
  "effect_time": 1749085414193312000,
  "associated": null
}
```

### Path Types
- `"watcher"` - Internal watcher lifecycle events (live/die)
- `"file"` - File system changes
- `"dir"` - Directory changes (ignored)
- `"other"` - Other fs objects (ignored)

### Effect Types
- `"create"` - File created
- `"modify"` - File modified (content change)
- `"destroy"` - File deleted
- `"rename"` - File renamed

**Important**: Metadata-only changes (like `touch`) do NOT trigger events. Only actual content modifications are detected.

## Platform Support

- Uses pre-compiled watcher binaries for each platform
- Binaries are embedded in the plur executable and extracted on first use
- Stored in `~/.cache/plur/bin/`
- Currently supports:
  - macOS arm64 (`watcher-aarch64-apple-darwin`)
  - Linux arm64 (`watcher-aarch64-unknown-linux-gnu`)
  - Linux x64 (`watcher-x86_64-unknown-linux-gnu`)

### Build Process

The watcher binaries are downloaded from the [e-dant/watcher](https://github.com/e-dant/watcher) releases and embedded into the plur binary:

1. **Development builds** (`bin/rake build`): Downloads only the current platform's watcher binary via `vendor:download:current`
2. **Cross-platform builds** (`bin/rake build:linux`, `bin/rake build:all`): Downloads all platform binaries via `vendor:download:all` before compilation
3. **Docker installation**: Uses `build:linux` which ensures all Linux watcher variants are embedded

The downloaded binaries are stored in `plur/embedded/watcher/` and embedded into the Go binary at compile time.

## Configuration

### Debounce Delay
- Default: 100ms
- Configurable via `--debounce` flag
- Example: `plur watch --debounce 250`

### Timeout
- For testing/CI: `--timeout` flag sets automatic exit
- Example: `plur watch --timeout 60` (exits after 60 seconds)

## File Mapping Rules

1. **Direct spec mapping**: `lib/foo.rb` → `spec/foo_spec.rb`
2. **Rails conventions**: `app/models/user.rb` → `spec/models/user_spec.rb`
3. **Nested files**: `lib/foo/bar.rb` → `spec/foo/bar_spec.rb`
4. **Special cases**:
   - `spec_helper.rb` → runs all specs in `spec/`
   - `rails_helper.rb` → runs all specs in `spec/`

## Signal Handling

- Gracefully handles SIGINT (Ctrl+C) and SIGTERM
- Cleanly shuts down all watcher processes
- Ensures no zombie processes are left behind