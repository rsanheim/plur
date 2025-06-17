# gow - Go Watch Mode Cheat Sheet

`gow` is a file watcher for Go that automatically reruns commands when files change.

## Installation
```bash
go install github.com/mitranim/gow@latest
```

## Basic Usage
```bash
gow <gow_flags> <cmd> <cmd_flags> <cmd_args ...>
```

## Common Commands

### Testing
```bash
# Watch and run all tests
gow -c -v test ./...

# Run specific package tests with verbose output
gow -c -v test -v ./rspec

# Run tests with coverage
gow -c -v test -cover ./...

# Run tests with race detection
gow -c -v test -race ./...
```

### Development
```bash
# Watch and run main.go
gow -c -v run .

# Run with arguments
gow -c -v run . arg1 arg2

# Watch and build
gow -c -v build .

# Watch and vet
gow -c -v vet ./...
```

## Key Flags

| Flag | Default | Description |
|------|---------|-------------|
| `-c` | - | Clear terminal on restart |
| `-s` | - | Soft-clear (keeps scrollback) |
| `-v` | - | Verbose logging |
| `-r` | - | Enable hotkeys (raw mode) |
| `-l` | - | Lazy mode: restart only when subprocess not running |
| `-p` | - | Postpone first run until file change |
| `-e` | go,mod | Extensions to watch (multi-flag) |
| `-w` | . | Directories to watch (multi-flag) |
| `-i` | - | Ignored directories (multi-flag) |

## Hotkeys (with `-r` flag)

| Key | Action |
|-----|--------|
| `Ctrl+C` | Kill subprocess, repeat to exit gow |
| `Ctrl+R` | Restart subprocess |
| `Ctrl+T` | Kill subprocess, repeat to exit gow |
| `Ctrl+\` | Kill with SIGQUIT, repeat to exit |
| `Ctrl+-` | Print currently running command |
| `Ctrl+H` | Print hotkey help |

## Advanced Examples

### Custom File Extensions
```bash
# Watch .go, .mod, and .sql files
gow -c -v -e=go -e=mod -e=sql test ./...
```

### Specific Directories
```bash
# Watch only src directory, ignore vendor and .git
gow -c -v -w=src -i=vendor -i=.git test ./...
```

### With Test Flags
```bash
# Run specific test with count
gow -c -v test -v -count=1 -run=TestSpecificFunction ./...

# Disable test caching
gow -c -v test -count=1 ./...
```

### Lazy Mode
```bash
# Only restart when previous run completes
gow -c -v -l test ./...
```

## Tips

1. **Use `-r` for interactive development** to enable hotkeys
2. **Use `-c` to clear terminal** for cleaner output
3. **Combine with `-v` for verbose logging** to see what files triggered restarts
4. **Use `-l` for long-running tests** to avoid interrupting in-progress runs
5. **Exit gow quickly** by pressing Ctrl+C twice within 1 second

## Comparison with Other Tools

- **vs `go test`**: gow adds file watching and auto-restart
- **vs `entr`**: gow is Go-specific with better integration
- **vs `reflex`**: gow has simpler configuration and hotkey support
- **vs `air`**: gow is lighter weight, no config file needed