# Go Vendoring and CI Build Issues

## The Issue

CI builds are failing with:
```
watch.go:22:12: pattern vendor/watcher/*: no matching files found
```

This happens because:
1. The `go:embed` directive in `watch.go` embeds watcher binaries at compile time
2. The `.gitignore` excludes the entire `vendor/` directory
3. CI doesn't have the required Linux watcher binary

## How Go Vendoring Works

### Go Modules (Default)
- Dependencies defined in `go.mod` and locked in `go.sum`
- Downloaded to shared cache (`$GOPATH/pkg/mod`)
- No local `vendor/` directory needed

### Vendoring (Optional)
- Run `go mod vendor` to copy Go dependencies to `vendor/`
- Useful for reproducible builds without network access
- **Note**: The watcher binaries are NOT Go dependencies - they're pre-compiled C++ executables

## Solution for CI

### Option 1: Download binaries during CI (Recommended)
Add this to your CI configuration before running tests:

```bash
# Download the platform-specific watcher binary
bin/rake deps:watcher

# Then run tests
bin/rake test:go
```

### Option 2: Commit the binaries
Modify `.gitignore` to include watcher binaries:

```gitignore
# Dependency directories
vendor/
# But keep the watcher binaries that are embedded
!vendor/watcher/
```

Then download all platform binaries locally:
```bash
# Download for your platform
bin/rake deps:watcher

# For CI, you'd need to download Linux binaries on a Linux machine
# or use cross-platform download tools
```

## Platform-Specific Binaries

The code expects these binaries in `vendor/watcher/`:
- `watcher-aarch64-apple-darwin` (macOS ARM64)
- `watcher-x86_64-apple-darwin` (macOS x64)
- `watcher-aarch64-unknown-linux-gnu` (Linux ARM64)
- `watcher-x86_64-unknown-linux-gnu` (Linux x64) ← Needed for most CI

## Quick Fix for CI

Add this to your CI workflow:

```yaml
# Example for GitHub Actions
- name: Download watcher binary
  run: bin/rake deps:watcher

# Example for CircleCI
- run:
    name: Download watcher binary
    command: bin/rake deps:watcher
```

## Testing Locally

To test what CI will experience:
```bash
# Remove local binary
rm -rf rux/vendor/watcher/

# Download for your platform
bin/rake deps:watcher

# Run tests
bin/rake test:go
```