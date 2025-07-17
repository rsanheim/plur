# Plur Watch: Multiple Watchers Implementation Plan

## Problem Statement

The current plur watch implementation attempts to pass multiple directory paths to a single watcher binary instance, but the watcher binary only processes the first path argument. This means directories beyond the first one are not actually being watched.

## Proposed Solution

Implement a multi-watcher architecture where each directory gets its own watcher process, with events aggregated into a single stream.

## Architecture Overview

```
┌─────────────────┐
│   plur-watch     │
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

## Implementation Plan

### Phase 1: Refactor Watcher Structure

#### 1.0 Create `plur-watch` binary

Create a new `plur-watch` binary, which is equivalent to `plur watch`. This will just make it a bit easier to spin up plur-watch for test purposes,
and makes it clear its a seperate piece of plur.  `plur watch` can still exist for now, and the underlying code should be the same. So I think this
is mostly a change to the CLI layer.

#### 1.1 Create WatcherManager
```go
// watch/watcher_manager.go
type WatcherManager struct {
    watchers      []*Watcher
    eventChan     chan Event
    errorChan     chan error
    stopChan      chan struct{}
    config        *Config
}
```

#### 1.2 Modify Watcher for Single Directory
```go
// watch/watcher.go
type Watcher struct {
    directory   string        // Single directory instead of slice
    binaryPath  string
    process     *exec.Cmd
    eventChan   chan Event
    errorChan   chan error
    stopChan    chan struct{}
}
```

### Phase 2: Implement Multi-Process Management

#### 2.1 WatcherManager Methods
```go
func (wm *WatcherManager) Start() error {
    // For each directory, create and start a watcher
    for _, dir := range wm.config.Directories {
        watcher := NewWatcher(dir, wm.config.BinaryPath)
        if err := watcher.Start(); err != nil {
            // Clean up already started watchers
            wm.cleanup()
            return fmt.Errorf("failed to start watcher for %s: %w", dir, err)
        }
        wm.watchers = append(wm.watchers, watcher)
        
        // Aggregate events from this watcher
        go wm.aggregateEvents(watcher)
    }
    return nil
}

func (wm *WatcherManager) aggregateEvents(w *Watcher) {
    for {
        select {
        case event := <-w.Events():
            wm.eventChan <- event
        case err := <-w.Errors():
            wm.errorChan <- err
        case <-wm.stopChan:
            return
        }
    }
}
```

### Phase 3: Update Watch Command

#### 3.1 Modify watchCommand Function
```go
func watchCommand(c *cli.Context) error {
    // ... existing setup code ...
    
    // Create watcher manager instead of single watcher
    manager := watch.NewWatcherManager(&watch.Config{
        Directories:    dirsToWatch,
        DebounceDelay:  debounceDelay,
        BinaryPath:     binaryPath,
    })
    
    if err := manager.Start(); err != nil {
        return err
    }
    defer manager.Stop()
    
    // Listen to aggregated events
    for {
        select {
        case event := <-manager.Events():
            // ... existing event handling ...
        case err := <-manager.Errors():
            // ... existing error handling ...
        case <-timeoutChan:
            // ... existing timeout handling ...
        }
    }
}
```

### Phase 4: Handle Edge Cases

#### 4.1 Process Cleanup
- Ensure all watcher processes are killed on exit
- Handle partial startup failures (some watchers start, others fail)
- Implement graceful shutdown with SIGINT/SIGTERM

#### 4.2 Resource Management
- Set reasonable limits on number of concurrent watchers
- Monitor resource usage (file descriptors, memory)
- Implement backpressure if event queue gets too large

#### 4.3 Error Recovery
- If a watcher process dies, attempt to restart it
- Log which directory's watcher failed
- Continue operation with remaining watchers

### Phase 5: Testing Strategy

#### 5.1 Unit Tests
- Test WatcherManager with mock watchers
- Test event aggregation
- Test error handling and cleanup

#### 5.2 Integration Tests
- Test with multiple real directories
- Verify all directories are actually being watched
- Test process cleanup on various exit scenarios

#### 5.3 Performance Tests
- Measure resource usage with many watchers
- Test behavior with high-frequency file changes
- Verify debouncing works across multiple watchers

## Migration Path

1. Keep existing Watcher interface mostly intact
2. Introduce WatcherManager as a layer above
3. Update watch command to use WatcherManager
4. Maintain backward compatibility where possible

## Alternative Considerations

### Why Not Watch Repository Root?

Watching the entire repository root has significant drawbacks:
- **Performance**: Large directories like `.git`, `node_modules`, `vendor`, `tmp` would generate excessive events
- **Noise**: Many irrelevant file changes would need filtering
- **Resource Usage**: FSEvents/inotify limits could be exceeded
- **Latency**: Event filtering adds processing overhead

### Future Enhancements

1. **Smart Directory Detection**: Automatically find directories to watch based on project type
2. **Configurable Ignore Patterns**: Allow users to exclude certain subdirectories
3. **Watch Profiles**: Pre-configured watch patterns for common project types (Rails, Node, etc.)
4. **Dynamic Watcher Management**: Add/remove watchers as directories are created/deleted

## Implementation Timeline

1. **Week 1**: Implement WatcherManager and refactor Watcher
2. **Week 2**: Update watch command and test basic functionality  
3. **Week 3**: Add error handling, cleanup, and edge cases
4. **Week 4**: Comprehensive testing and documentation

## Success Criteria

- [ ] Multiple directories are successfully watched simultaneously
- [ ] Events from all directories are properly aggregated
- [ ] Process cleanup works reliably
- [ ] Resource usage is reasonable (< 50MB per watcher)
- [ ] All existing tests pass with new implementation
- [ ] New tests verify multi-watcher functionality