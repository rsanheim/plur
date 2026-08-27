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

### 3. Planner (`watch/plan.go`)

* Decides what a file change does; shared by `plur watch` and `plur watch find` so both agree on behavior
* `Admit` normalizes paths to be CWD-relative, rejecting paths outside the project and paths matching global ignore patterns
* `Plan` matches watch rules against changed paths, renders target templates, skips targets that do not exist on disk, and merges deduplicated targets into per-job runs
* Built from validated runtime config, so planning cannot fail at runtime

### 4. Debouncer (`watch/debouncer.go`)

* Prevents duplicate test runs when multiple files change rapidly
* Configurable delay (default 30ms)
* Batches and deduplicates file paths before processing
* Timer resets on each new event within the delay window

### 5. Job Execution (`watch/execute.go`)

* `ExecuteJob` runs each `JobRun` from the plan, streaming output to the terminal
* `JobRun.Command` builds the argv (job command plus targets) and environment; execution and display both start here so what plur prints is exactly what it runs

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
Main Watch Loop (cmd_watch.go)
    ↓
Event Filtering (path type, effect type)
    ↓
Planner.Admit() (relativize path, reject outside-CWD, ignore patterns)
    ↓
Debouncer.Debounce() (batch changes, deduplicate files)
    ↓
Planner.Plan() (match rules, render targets, merge job runs)
    ↓
watch.ExecuteJob() (run each job)
    ↓
promptChan (coordinate prompt output)
```

## Multi-Process Design

### Why Filter Overlapping Directories?

The embedded [e-dant/watcher](https://github.com/e-dant/watcher) binary watches a directory
**recursively** and emits events for **all** file changes within that tree. It has no
built-in ignore or exclusion capability - every change is reported.

This means if we start two watchers on overlapping paths (e.g., `.` and `lib/`), a change
to `lib/foo.rb` would trigger events from *both* watchers, causing duplicate test executions.

To prevent this, plur filters the directory list before spawning watchers:

1. **Security validation**: Directories must be within project root (rejects symlinks escaping to `/`)
2. **Symlink deduplication**: Multiple paths resolving to same directory are consolidated
3. **Subdirectory filtering**: If a parent directory is watched, child directories are removed

### Filtering Examples

| Input directories | After filtering | Watchers |
|-------------------|-----------------|----------|
| `[., lib, spec]` | `[.]` | 1 (lib/spec are subdirs of .) |
| `[lib, spec, app]` | `[lib, spec, app]` | 3 (siblings, no overlap) |
| `[lib, lib/foo]` | `[lib]` | 1 (lib/foo is subdir of lib) |

### Process Spawning

After filtering, one watcher process is spawned per remaining directory:

- `spec/` → watcher process 1
- `lib/` → watcher process 2
- `app/` → watcher process 3 (if exists)

All events are aggregated into a single channel for unified processing in Go, where the
actual file matching against watch patterns occurs.

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
- Stored in `~/.plur/bin/`
- Currently supports:
  - macOS arm64 (`watcher-aarch64-apple-darwin`)
  - Linux arm64 (`watcher-aarch64-unknown-linux-gnu`)
  - Linux x64 (`watcher-x86_64-unknown-linux-gnu`)

### Build Process

The watcher binaries are downloaded from the [e-dant/watcher](https://github.com/e-dant/watcher) releases and embedded into the plur binary:

1. **Development builds** (`bin/rake build`): Downloads only the current platform's watcher binary via `vendor:download:current`
2. **Cross-platform builds** (`bin/rake build:all`): Downloads all platform binaries via `vendor:download:all` before compilation

The downloaded binaries are stored in `embedded/watcher/` and embedded into the Go binary at compile time.

## Configuration

### Debounce Delay

* Default: 30ms
* Configurable via `--debounce` flag
* Example: `plur watch --debounce 250`

### Timeout

- For testing/CI: `--timeout` flag sets automatic exit
- Example: `plur watch --timeout 60` (exits after 60 seconds)

## File Mapping Rules

File-to-target mapping is driven by watch mappings: built-in defaults
(`internal/runtime/defaults.toml`) merged with user `[[watch]]` config.
Built-in examples:

1. **Direct spec mapping** (`lib-to-spec`): `lib/foo.rb` → `spec/foo_spec.rb`
2. **Rails conventions** (`app-to-spec`): `app/models/user.rb` → `spec/models/user_spec.rb`
3. **Spec files run themselves** (`spec-files`): `spec/foo_spec.rb` → `spec/foo_spec.rb`
4. **Go packages** (`go-source`): `pkg/foo.go` → `go test ./pkg/`

Rendered targets that do not exist on disk are skipped.

## Signal Handling

* **SIGINT** (Ctrl+C) and **SIGTERM**: Graceful shutdown, cleanly stops all watcher processes
* **SIGHUP**: Triggers process reload (re-exec with same arguments)
* Terminal state is reset before reload to handle jobs that may leave terminal in raw mode

## Reload Functionality

The `reload` command and SIGHUP re-exec plur with the same arguments after
stopping its watcher processes and restoring the terminal state.

## Goroutine Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           MAIN PROCESS (plur watch)                         │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌─────────────────┐                                                        │
│  │ Main Goroutine  │  runWatchWithConfig() - main select loop               │
│  │ (cmd_watch.go)  │  Owns: sigChan, timeoutChan, promptChan                │
│  └────────┬────────┘                                                        │
│           │                                                                 │
│           │ select {                                                        │
│           │   case <-stdinChan:        // user commands                     │
│           │   case <-manager.Events(): // file changes → debouncer          │
│           │   case <-manager.Errors(): // watcher errors                    │
│           │   case <-sigChan:          // SIGINT/SIGTERM/SIGHUP             │
│           │   case <-timeoutChan:      // timeout (if set)                  │
│           │   case <-promptChan:       // display prompt                    │
│           │ }                                                               │
│           │                                                                 │
│  ┌────────▼────────┐                                                        │
│  │ stdin Goroutine │  bufio.Scanner on os.Stdin                             │
│  │                 │  Sends to: stdinChan                                   │
│  └─────────────────┘                                                        │
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │                      Event Processing                               │    │
│  │                                                                     │    │
│  │  ┌─────────────────┐    ┌─────────────────────┐                    │    │
│  │  │    Debouncer    │───▶│  Planner / Execute  │                    │    │
│  │  │                 │    │                     │                    │    │
│  │  │ * Batch files   │    │ * Match watch rules │                    │    │
│  │  │ * Deduplicate   │    │ * Render targets    │                    │    │
│  │  │ * 30ms delay    │    │ * Execute job runs  │                    │    │
│  │  └─────────────────┘    └─────────────────────┘                    │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
│                                                                             │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │                      WatcherManager                                 │    │
│  │                                                                     │    │
│  │  Owns: wm.stopChan, wm.eventChan, wm.errorChan, wm.wg              │    │
│  │                                                                     │    │
│  │  ┌─────────────────────┐    ┌─────────────────────┐                │    │
│  │  │ aggregateEvents     │    │ aggregateEvents     │   (1 per dir)  │    │
│  │  │ goroutine           │    │ goroutine           │                │    │
│  │  │ Waits: wm.stopChan  │    │ Waits: wm.stopChan  │                │    │
│  │  └──────────┬──────────┘    └──────────┬──────────┘                │    │
│  │             │                          │                           │    │
│  │  ┌──────────▼──────────┐    ┌──────────▼──────────┐                │    │
│  │  │      Watcher        │    │      Watcher        │                │    │
│  │  │                     │    │                     │                │    │
│  │  │ ┌─────────────────┐ │    │ ┌─────────────────┐ │                │    │
│  │  │ │ lifecycle       │ │    │ │ lifecycle       │ │                │    │
│  │  │ │ goroutine       │ │    │ │ goroutine       │ │                │    │
│  │  │ │ Waits: stopChan │ │    │ │ Waits: stopChan │ │                │    │
│  │  │ └─────────────────┘ │    │ └─────────────────┘ │                │    │
│  │  │ ┌─────────────────┐ │    │ ┌─────────────────┐ │                │    │
│  │  │ │ readEvents      │ │    │ │ readEvents      │ │                │    │
│  │  │ │ goroutine       │ │    │ │ goroutine       │ │                │    │
│  │  │ └─────────────────┘ │    │ └─────────────────┘ │                │    │
│  │  │ ┌─────────────────┐ │    │ ┌─────────────────┐ │                │    │
│  │  │ │ readErrors      │ │    │ │ readErrors      │ │                │    │
│  │  │ │ goroutine       │ │    │ │ goroutine       │ │                │    │
│  │  │ └─────────────────┘ │    │ └─────────────────┘ │                │    │
│  │  │                     │    │                     │                │    │
│  │  │   ┌───────────┐     │    │   ┌───────────┐     │                │    │
│  │  │   │ watcher   │     │    │   │ watcher   │     │  (C++ binary)  │    │
│  │  │   │ subprocess│     │    │   │ subprocess│     │                │    │
│  │  │   └───────────┘     │    │   └───────────┘     │                │    │
│  │  └─────────────────────┘    └─────────────────────┘                │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Shutdown & Reload Paths

### Normal Exit (SIGINT/SIGTERM or "exit" command)

```
Signal or command received
    │
    ▼
Main loop returns nil
    │
    ▼
defer manager.Stop() executes
    │
    ▼
manager.Stop():
  1. close(stopChan)     → signals aggregateEvents goroutines
  2. cleanup()           → calls w.Stop() on each watcher
  3. wg.Wait()           → waits for aggregate goroutines
  4. close channels
    │
    ▼
w.Stop():
  1. close(stopChan)     → signals lifecycle goroutine
  2. <-done              → waits for subprocess cleanup
    │
    ▼
lifecycle goroutine:
  stdinPipe.Close()
  process.Kill()
  process.Wait()
  close(done)
    │
    ▼
Process exits cleanly
```

### Reload (SIGHUP or `reload` command)

```
Reload triggered
    │
    ▼
reload(manager) called
    │
    ▼
manager.Stop()           → synchronous cleanup (waits for subprocesses)
    │
    ▼
resetTerminal()          → stty sane (restore terminal state)
    │
    ▼
syscall.Exec(...)        → replaces process image
    │
    ▼
New plur process starts fresh
```
