# Watch Architecture: Overlapping Directory Problem

## The Bug

When a file is modified, `plur watch` executes tests twice for certain configurations.

**Observed behavior:**
```
01:44:54 - INFO  - Executing job job="rspec" targets="[spec/plur/benchmark_spec.rb]"
01:44:55 - INFO  - Executing job job="rspec" targets="[spec/plur/benchmark_spec.rb]"
```

Same file, same targets, ~1 second apart.

## Root Cause

**Multiple watchers monitoring overlapping directories.**

Given this `.plur.toml`:
```toml
[[watch]]
source = "**/*_spec.rb"    # SourceDir = "."
jobs = ["rspec"]

[[watch]]
source = "lib/**/*.rb"     # SourceDir = "lib"
targets = ["spec/{{match}}_spec.rb"]
jobs = ["rspec"]
```

Plur starts TWO watcher processes:

* Watcher 1: monitoring `.` (project root)
* Watcher 2: monitoring `lib`

When `lib/plur/benchmark.rb` changes:

* Watcher 1 sees it (lib is under .)
* Watcher 2 sees it (lib is its root)

Both emit events → both trigger test execution.

## Original Design Decision

**Why multiple watchers?**

The embedded `e-dant/watcher` binary has NO global ignore capability. To avoid watching noisy directories like `.git/`, `node_modules/`, `vendor/`, etc., the design chose to start watchers only on directories that contain source files of interest.

```go
// watch.go:114-126
var watchDirs []string
for _, mapping := range watches {
    dir := mapping.SourceDir()  // Extract prefix from glob pattern
    watchDirs = append(watchDirs, dir)
}
```

**Trade-off made:** More focused watching (less noise) at the cost of potential overlap.

## Complete Event Flow

```
Filesystem Change
    ↓
e-dant/watcher binary (one per directory)
    ↓
JSON event via stdout
    ↓
Watcher.readEvents() parses JSON → Event struct
    ↓
WatcherManager.aggregateEvents() forwards to unified channel
    ↓  ← NO DEDUPLICATION HERE
Main watch loop (watch.go:271+)
    ↓
Event filtering (PathType, EffectType)
    ↓
watch.FindTargetsForFile() maps file → targets
    ↓
EventProcessor.ProcessPath() matches patterns, checks excludes
    ↓
executeJob() runs tests
```

**Critical gap:** No deduplication between events from different watcher processes.

## Code Paths Involved

### 1. Watch Directory Selection
*File: `plur/watch.go:114-126`*

```go
var watchDirs []string
for _, mapping := range watches {
    dir := mapping.SourceDir()
    if _, err := os.Stat(dir); err == nil {
        watchDirs = append(watchDirs, dir)
    }
}
sort.Strings(watchDirs)
watchDirs = slices.Compact(watchDirs)  // Only removes consecutive duplicates
```

**Problem:** `slices.Compact` only removes exact duplicates. `[. lib]` stays as-is because they're different strings, even though `lib` is a subdirectory of `.`.

### 2. SourceDir Calculation
*File: `plur/watch/watch_mapping.go:18-22`*

```go
func (w WatchMapping) SourceDir() string {
    base, _ := doublestar.SplitPattern(w.Source)
    return base
}
```

Examples:

* `lib/**/*.rb` → `lib`
* `**/*_spec.rb` → `.`
* `app/models/**/*.rb` → `app/models`

### 3. Watcher Process Creation
*File: `plur/watch/watcher_manager.go:42-65`*

```go
for _, dir := range wm.config.Directories {
    watcher := NewWatcher(singleDirConfig, wm.binaryPath)
    watcher.Start()
    wm.watchers = append(wm.watchers, watcher)
    go wm.aggregateEvents(watcher)  // Independent goroutine per watcher
}
```

Each directory gets its own watcher process and aggregation goroutine.

### 4. Event Aggregation (No Deduplication)
*File: `plur/watch/watcher_manager.go:96-118`*

```go
func (wm *WatcherManager) aggregateEvents(w *Watcher) {
    for {
        select {
        case event := <-w.Events():
            wm.eventChan <- event  // Direct forward, no filtering
        case <-wm.stopChan:
            return
        }
    }
}
```

Events from all watchers merge into single channel without deduplication.

### 5. Main Event Loop
*File: `plur/watch.go:271-329`*

Processes events as they arrive. No tracking of recently-processed files.

## Existing Deduplication (Insufficient)

### Within-Job Target Deduplication
*File: `plur/watch/processor.go:69-72, 113-126`*

```go
// Deduplicates targets within a single file change event
for jobName := range results {
    results[jobName] = deduplicate(results[jobName])
}
```

This only deduplicates targets for a SINGLE event, not across events.

### Debouncer (Exists but Unused)
*File: `plur/watch/debouncer.go`*

A complete debouncer implementation exists but is NOT wired into the event loop:
```go
// watch.go:110-112
debounceDelay := time.Duration(watchCmd.Debounce) * time.Millisecond
logger.Logger.Debug("Debounce delay", "ms", watchCmd.Debounce)
// ← debouncer never created or used
```

## Possible Solutions

### Option A: Filter Overlapping Directories at Startup

**Approach:** Before starting watchers, remove directories that are subdirectories of other directories in the list.

```go
func filterSubdirectories(dirs []string) []string {
    sort.Strings(dirs)  // Ensures parents come before children
    result := []string{}
    for _, dir := range dirs {
        isSubdir := false
        for _, existing := range result {
            if strings.HasPrefix(dir, existing+string(filepath.Separator)) {
                isSubdir = true
                break
            }
        }
        if !isSubdir {
            result = append(result, dir)
        }
    }
    return result
}
```

**Pros:**
* Simple, fixes the immediate bug
* Minimal code change

**Cons:**
* Defeats original design goal - now watching `.git`, `node_modules`, etc.
* May be noisy/slow on large projects
* User loses ability to watch only specific subdirectories

### Option B: Deduplicate Events in WatcherManager

**Approach:** Track recently-seen file paths in `aggregateEvents()` and skip duplicates within a time window.

```go
type WatcherManager struct {
    // ...
    recentEvents map[string]time.Time
    recentMu     sync.Mutex
}

func (wm *WatcherManager) aggregateEvents(w *Watcher) {
    for event := range w.Events() {
        wm.recentMu.Lock()
        if lastSeen, exists := wm.recentEvents[event.PathName]; exists {
            if time.Since(lastSeen) < wm.config.DebounceDelay {
                wm.recentMu.Unlock()
                continue  // Skip duplicate
            }
        }
        wm.recentEvents[event.PathName] = time.Now()
        wm.recentMu.Unlock()
        wm.eventChan <- event
    }
}
```

**Pros:**
* Preserves focused watching (original design goal)
* Handles any source of duplicate events
* Works with debounce delay configuration

**Cons:**
* More complex
* Memory overhead for tracking recent events
* Need to clean up stale entries periodically

### Option C: Wire Up Existing Debouncer

**Approach:** Use the existing `Debouncer` in the main event loop.

```go
debouncer := watch.NewDebouncer(debounceDelay)

// In event loop:
case event := <-manager.Events():
    // ... validation ...
    debouncer.Debounce([]string{path}, func(files []string) {
        for _, f := range files {
            result, _ := watch.FindTargetsForFile(f, jobs, watches)
            // ... execute jobs ...
        }
    })
```

**Pros:**
* Uses existing code
* Batches rapid changes together
* Already handles the timing logic

**Cons:**
* Debouncer batches by FILE, not by event source
* Two events for same file from different watchers would still both be processed (just batched together)
* May not fully solve the problem without additional deduplication

### Option D: Single Watcher + Application-Level Filtering

**Approach:** Always watch from project root, filter unwanted paths in Go code.

```go
// Always start single watcher at project root
watchDirs = []string{"."}

// In event loop, check against global excludes
globalExcludes := []string{".git/**", "node_modules/**", "vendor/**", "tmp/**"}
if isExcludedByGlobal(path, globalExcludes) {
    continue
}
```

**Pros:**
* Simple mental model - one watcher
* No overlap possible
* Application has full control over filtering

**Cons:**
* Watcher binary still receives ALL events (just filtered in Go)
* May have performance implications on large codebases
* e-dant/watcher may still emit events we'll discard

### Option E: Hybrid - Dedupe + Focused Watching

**Approach:** Keep multiple watchers for performance, but deduplicate at aggregation level.

Combine Options A and B:

1. Filter obvious overlaps (parent/child directories)
2. Still deduplicate events in aggregation as safety net

**Pros:**
* Best of both worlds
* Handles edge cases
* Maintains performance benefits of focused watching

**Cons:**
* Most complex implementation
* Two layers of logic to maintain

## Recommendation

**Start with Option B (Deduplicate Events in WatcherManager)** because:

1. Preserves the original design intent (focused watching)
2. Fixes the bug regardless of how overlap occurs
3. Works with the existing debounce delay configuration
4. Can be enhanced later if needed

**Implementation priority:**
1. Add event deduplication in `WatcherManager.aggregateEvents()`
2. Use existing debounce delay for the dedup window
3. Add periodic cleanup of stale entries
4. Add debug logging to track deduplicated events

## Industry Context: File Watcher Comparison

### No Watcher Has Built-in Ignore

The lack of ignore/filter capability in e-dant/watcher is **industry-standard**, not a limitation unique to it.

| Feature | [e-dant/watcher](https://github.com/e-dant/watcher) | [notify-rs](https://github.com/notify-rs/notify) (Rust) | [fsnotify](https://github.com/fsnotify/fsnotify) (Go) |
|---------|---------------|------------------|---------------|
| **Built-in Ignore/Filter** | NO | NO | NO |
| **Native Recursive Watching** | YES | YES | NO (manual) |
| **Language** | C++ | Rust | Go |
| **Codebase Size** | ~1,579 LOC | ~2,799 LOC | Larger |

### Relevant GitHub Issues

**fsnotify (Go):**
* [Issue #18: User-space recursive watcher](https://github.com/fsnotify/fsnotify/issues/18) - Open since June 2014. "If you monitor a directory with FSEvents then the monitor is recursive; there is no non-recursive option." Platform inconsistency is the core challenge.
* [Issue #41: Removing recursive watches](https://github.com/fsnotify/fsnotify/issues/41) - Users must track the full tree of watches themselves.
* [Issue #21: Subtree watch on Windows](https://github.com/fsnotify/fsnotify/issues/21) - Proposal for `w.Add("dir/...")` syntax.
* [Issue #223: Recursive Directory Watcher](https://github.com/fsnotify/fsnotify/issues/223) - More recent discussion of the same limitation.

**notify-rs (Rust):**
* [Issue #204: Filtered recursive watchers](https://github.com/notify-rs/notify/issues/204) - Proposes `RecursiveFiltered(Box<Filter>)` enum variant. Was on 5.0 milestone, then removed (Aug 2022). Still open.
* [Issue #291: User-defined filtering](https://github.com/notify-rs/notify/issues/291) - Maintainers "dislike adding more callbacks as we really don't want to block in the core watcher implementation."

**Why no built-in filtering?**
1. Filter callbacks could block the watcher's event loop
2. Different apps need different filtering semantics (gitignore vs glob vs regex)
3. It's simpler/safer to filter events after receiving them

### Path Capacity: Platform Limits

#### Linux

**inotify (kernel < 5.9 or non-root):**
* Default `max_user_watches`: **8,192** (can be increased via sysctl)
* Each watch consumes ~1KB kernel memory (64-bit)
* Shared across ALL applications per user
* Common to increase to 65,536 or 524,288 for development

```bash
# Check current limit
cat /proc/sys/fs/inotify/max_user_watches

# Increase temporarily
sudo sysctl fs.inotify.max_user_watches=524288

# Increase permanently
echo "fs.inotify.max_user_watches=524288" | sudo tee /etc/sysctl.d/90-inotify.conf
```

**fanotify (kernel ≥ 5.9 with root):**
* **No per-path limits** - watches entire mount/filesystem
* e-dant/watcher uses fanotify when available (Linux 5.9+, root privileges)
* Falls back to inotify otherwise
* Note: fanotify requires admin privileges as of kernel 5.12

**e-dant/watcher advantage:** Uses fanotify on modern Linux, avoiding inotify limits entirely.

#### macOS

**FSEvents (default on macOS):**
* **No known limitations** - scales to 500GB+ filesystems
* Native recursive watching
* Directory-level granularity

**kqueue (BSD fallback):**
* Requires one file descriptor per watched item
* Default per-process limit: **256** (extremely low!)
* System-wide limit: ~10,000 on typical BSD
* **Not recommended for large trees**

**e-dant/watcher uses FSEvents on macOS**, avoiding kqueue limitations.

### e-dant/watcher Performance

From project documentation:

* Handles **100,000+ paths** before stuttering (with "warthog" platform-independent adapter)
* "Overhead of detecting and sending an event is an order of magnitude less than filesystem operations"
* Binary size: 50-80KB
* Minimal cache misses during operation

### Implication for plur

Given e-dant/watcher's ability to handle 100k+ paths efficiently on modern systems:

* **Linux (fanotify)**: No practical path limit with root privileges
* **Linux (inotify)**: 8k default, easily increased
* **macOS (FSEvents)**: No practical limit

**A single watcher on `.` with application-level filtering is viable** even for large repos. The original multi-watcher design to avoid `.git`/`node_modules` may be unnecessary overhead given e-dant/watcher's performance characteristics.

## Related Internal Issues

* `docs/architecture/watch-concurrent-output.md` - Documents concurrent test execution issue (separate but related)
* Debouncer exists but is unused - should be wired up as part of this fix

## Files to Modify

* `plur/watch/watcher_manager.go` - Add deduplication logic
* `plur/watch.go` - Wire up debouncer (optional, complementary fix)
* Tests: `plur/watch/watcher_manager_test.go` (new or existing)

## References

* [e-dant/watcher](https://github.com/e-dant/watcher) - The embedded watcher binary
* [notify-rs/notify](https://github.com/notify-rs/notify) - Rust file watcher
* [fsnotify/fsnotify](https://github.com/fsnotify/fsnotify) - Go file watcher
* [watchexec inotify limits](https://watchexec.github.io/docs/inotify-limits.html) - Detailed inotify limit documentation
* [fanotify(7) man page](https://man7.org/linux/man-pages/man7/fanotify.7.html) - Linux fanotify documentation
* [JetBrains inotify limits](https://intellij-support.jetbrains.com/hc/en-us/articles/15268113529362-Inotify-Watches-Limit-Linux) - Practical guidance on increasing limits
