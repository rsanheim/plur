# Plur CLI Verbosity Levels - Design Document

## Current State Analysis

### Existing Implementation

Plur currently has two flags for controlling output verbosity:
- `--verbose`: Enable verbose output for debugging
- `--debug` / `-d`: Enable debug output (includes verbose), also settable via PLUR_DEBUG env var

**Issues with Current Implementation:**
1. Both verbose and non-verbose modes set the same log level (Info)
2. Relies on a `VerboseMode` boolean flag rather than proper log levels
3. No quiet mode for CI/scripting scenarios
4. Confusing overlap between verbose and debug modes

### Current Output Types

**Verbose Output (`LogVerbose` calls):**
- Worker assignments and file distribution
- Runtime vs size-based grouping decisions
- Worker start/finish status

**Debug Output (`Logger.Debug` calls):**
- Command execution details
- Directory changes
- Minitest line parsing (`[ParseLine]` prefix)
- File change events in watch mode
- Spec file mapping decisions

## Proposed Design

### 1. Progressive Verbosity Levels

Following Unix CLI conventions and modern best practices (2025):

| Flag | Level | slog Level | Description |
|------|-------|------------|-------------|
| `-q, --quiet` | ERROR only | 8 | Only errors, suppress normal output |
| *(default)* | WARN + ERROR | 4 | Warnings and errors only |
| `-v, --verbose` | INFO + above | 0 | Informational messages, progress updates |
| `-vv` | DEBUG + above | -4 | Detailed debugging information |
| `-vvv, --debug` | TRACE + above | -8 | Low-level tracing (parser output, etc.) |

### 2. slog Integration

#### Custom Log Levels
```go
const (
    LevelTrace = slog.Level(-8)  // New custom level for deepest debugging
    LevelDebug = slog.LevelDebug  // -4
    LevelInfo  = slog.LevelInfo   // 0
    LevelWarn  = slog.LevelWarn   // 4
    LevelError = slog.LevelError  // 8
)
```

#### Level Mapping Logic
```go
// Map verbosity count to slog level
func getLogLevel(verbosity int, quiet bool) slog.Level {
    if quiet {
        return LevelError
    }
    switch verbosity {
    case 0:
        return LevelWarn  // Default
    case 1:
        return LevelInfo  // -v
    case 2:
        return LevelDebug // -vv
    default:
        return LevelTrace // -vvv or higher
    }
}
```

### 3. Output Organization by Level

#### Quiet Mode (-q)
- Test failures
- Fatal errors
- Exit codes

#### Default Mode
- Warnings about missing runtime data
- Deprecation warnings
- Error messages

#### Verbose Mode (-v)
- Test execution summary (e.g., "Using runtime-based grouped execution")
- Worker assignments
- Progress indicators
- Timing information

#### Debug Mode (-vv)
- Command execution details
- File discovery process
- Test framework detection
- Worker-level details

#### Trace Mode (-vvv / --debug)
- Line-by-line parser output (`[ParseLine]`)
- Raw test output processing
- Detailed file system operations
- Channel communications

### 4. Implementation Changes

#### CLI Structure
```go
type PlurCLI struct {
    // Remove current flags:
    // Verbose bool
    // Debug bool
    
    // Add new flags:
    Verbosity int  `short:"v" type:"counter" help:"Increase verbosity (-v, -vv, -vvv)"`
    Quiet     bool `short:"q" help:"Quiet mode - only show errors"`
    Debug     bool `help:"Enable trace-level debugging (same as -vvv)" env:"PLUR_DEBUG"`
}
```

#### Logger Refactoring
1. Remove `VerboseMode` boolean
2. Remove `LogVerbose()` function - convert all calls to `Logger.Info()`
3. Add `Logger.Trace()` for lowest-level output
4. Update `InitLogger()` to use level-based approach

#### Backward Compatibility
- Keep `--debug` flag as alias for `-vvv`
- Honor `PLUR_DEBUG` environment variable
- Existing `LogVerbose` calls become `Logger.Info`

### 5. User-Facing Examples

```bash
# Quiet mode - only errors
plur -q

# Default - warnings and errors
plur

# Verbose - see worker assignments and progress
plur -v

# Debug - see command execution and file operations  
plur -vv

# Trace - see parser output and internal details
plur -vvv
# or
plur --debug
```

### 6. Security Considerations

**Verbose logs may contain:**
- Full file paths
- Environment variables
- Command-line arguments
- Test framework credentials

**Recommendations:**
- Document security implications in help text
- Add warning when using -vv or higher in CI environments
- Consider redacting sensitive patterns in output

### 7. Benefits

1. **Standards Compliance**: Follows established Unix/POSIX conventions
2. **Progressive Disclosure**: Users control information detail
3. **Cleaner Architecture**: Level-based instead of boolean flags
4. **Better Debugging**: Clear hierarchy of debug information
5. **CI-Friendly**: Quiet mode for cleaner CI logs
6. **Intuitive**: Matches user expectations from other CLI tools

## Migration Path

### Phase 1: Add New System (Backward Compatible)
1. Implement new verbosity counter flag
2. Map old flags to new system
3. Add deprecation notices

### Phase 2: Update Logging Calls
1. Convert `LogVerbose` → `Logger.Info`
2. Convert appropriate `Logger.Debug` → `Logger.Trace`
3. Remove `VerboseMode` boolean

### Phase 3: Documentation
1. Update help text
2. Create examples in README
3. Add verbosity guide to docs

### Phase 4: Remove Old System (Major Version)
1. Remove old boolean flags
2. Clean up legacy code
3. Update all tests

## References

- [Command Line Interface Guidelines](https://clig.dev/)
- [slog package documentation](https://pkg.go.dev/log/slog)
- [Unix Utility Conventions](https://pubs.opengroup.org/onlinepubs/9699919799/basedefs/V1_chap12.html)
- [CLI verbosity levels - Ubuntu Community Hub](https://discourse.ubuntu.com/t/cli-verbosity-levels/26973)