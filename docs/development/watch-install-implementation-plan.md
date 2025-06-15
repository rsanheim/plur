# Watch Install Implementation Plan

## Overview
Implemented the `rux watch install` command to extract and install the platform-specific watcher binary from embedded resources to `RUX_HOME/bin/[platform-specific-binary]`.

## Changes Made

### 1. Configuration Updates
- Added `BinDir` field to `ConfigPaths` struct in `config.go`
- Updated `InitConfigPaths()` to create the bin directory at `RUX_HOME/bin`

### 2. New Command Structure
- Added `watch install` subcommand in `main.go`
- Subcommand calls `runWatchInstall(ctx)` function

### 3. Core Implementation
- Created `rux/watch/install_binary.go` with `InstallBinary()` function
- Function extracts embedded watcher binary and writes to bin directory
- Always overwrites existing binary (as per requirement)
- Prints installation path relative to RUX_HOME

### 4. Refactored Binary Path Logic
- Updated `getWatcherBinaryPath()` in `watch.go` to look in bin directory
- Removed automatic extraction logic from `getWatcherBinaryPath()`
- Now returns error suggesting to run `rux watch install` if binary not found

### 5. Test Support
- Added generic `run_rux()` method to `RuxWatchHelper` module
- Integration test `spec/watch/watch_install_spec.rb` passes

## Binary Path Changes
- **Old**: Binaries were extracted to `RUX_HOME/.cache/[binary-name]` on demand
- **New**: Binaries are installed to `RUX_HOME/bin/[binary-name]` via explicit command

## Platform Support
The implementation supports the following platforms:
- `watcher-aarch64-apple-darwin` (Apple Silicon Mac)
- `watcher-aarch64-unknown-linux-gnu` (ARM64 Linux)
- `watcher-x86_64-unknown-linux-gnu` (x64 Linux)

Note: Intel Macs (x86_64) are not supported and will return an error.

## Usage
```bash
# Install the watcher binary
rux watch install

# Output:
# installed watcher binary path=bin/watcher-aarch64-apple-darwin
```

## Benefits
1. **Explicit control**: Users can reinstall the binary if needed
2. **Clear error messages**: If binary is missing, user gets helpful message
3. **Better organization**: Binaries in `bin/` instead of `.cache/`
4. **Simpler debugging**: Easy to check if binary is installed

## Testing
The implementation includes a comprehensive integration test that:
- Sets up a temporary RUX_HOME
- Runs `rux watch install`
- Verifies the binary is installed at the correct path
- Checks that the binary has executable permissions