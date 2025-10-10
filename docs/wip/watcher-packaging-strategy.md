# Watcher Binary Packaging Strategy for GoReleaser

## Current State
- Watcher binaries from [e-dant/watcher](https://github.com/e-dant/watcher) are embedded at compile time
- Currently have binaries for:
  - macOS ARM64 (`watcher-aarch64-apple-darwin`)
  - Linux ARM64 (`watcher-aarch64-unknown-linux-gnu`)
  - Linux x86_64 (`watcher-x86_64-unknown-linux-gnu`)
- Missing:
  - macOS x86_64 (Intel Macs)
  - Windows (any architecture)

## Problem
When GoReleaser builds for multiple platforms, we need to ensure each binary contains the correct watcher for its target platform, not the build host's platform.

## Proposed Solution

### Option A: Pre-download All Binaries (Recommended)
1. Download all watcher binaries for all platforms before GoReleaser runs
2. Use build tags or ldflags to select the correct binary at compile time
3. Pros:
   - Simple and reliable
   - All binaries available during build
   - Works with GoReleaser's cross-compilation
4. Cons:
   - Larger repository size (but binaries are small ~70KB each)

### Implementation Steps

#### Phase 1: Complete Binary Collection
```bash
# Add to lib/tasks/vendor.rake
def all_watcher_platforms
  [
    "aarch64-apple-darwin",       # macOS ARM64
    "x86_64-apple-darwin",        # macOS Intel (NEW)
    "aarch64-unknown-linux-gnu",  # Linux ARM64
    "x86_64-unknown-linux-gnu",   # Linux x64
    "x86_64-pc-windows-msvc"      # Windows x64 (NEW)
  ]
end
```

#### Phase 2: Conditional Embedding
Create platform-specific embed files:
```go
// plur/embed_darwin_amd64.go
//go:build darwin && amd64

package main

import _ "embed"

//go:embed embedded/watcher/watcher-x86_64-apple-darwin
var watcherBinary []byte
```

```go
// plur/embed_darwin_arm64.go
//go:build darwin && arm64

package main

import _ "embed"

//go:embed embedded/watcher/watcher-aarch64-apple-darwin
var watcherBinary []byte
```

#### Phase 3: GoReleaser Configuration
Update `.goreleaser.yml`:
```yaml
builds:
  - id: plur
    main: .
    binary: plur
    goos:
      - darwin
      - linux
      - windows
    goarch:
      - amd64
      - arm64
    env:
      - CGO_ENABLED=0
    hooks:
      pre:
        # Ensure correct watcher binary is available
        - cmd: script/ensure-watcher-binary {{ .Os }} {{ .Arch }}
```

### Option B: Download During Build
1. Use GoReleaser hooks to download the correct watcher binary
2. Pros:
   - Smaller repository
   - Always gets latest watcher version
3. Cons:
   - Build depends on external GitHub releases
   - More complex build process
   - Network dependency during builds

### Option C: Build Watcher from Source
1. Include watcher source and build during GoReleaser
2. Pros:
   - Full control over watcher build
3. Cons:
   - Requires C++ toolchain
   - Complex cross-compilation
   - Significantly slower builds

## Recommendation

**Use Option A** with pre-downloaded binaries:

1. **Immediate**: Download missing binaries (macOS x86_64, Windows)
2. **Embed Strategy**: Use build tags to embed only the correct binary for each platform
3. **Verification**: Add tests to ensure correct binary is embedded
4. **Documentation**: Update README with supported platforms

## Missing Platforms

Check if e-dant/watcher provides:
- `x86_64-apple-darwin` (macOS Intel)
- `x86_64-pc-windows-msvc` or similar (Windows)

If not available, we need to either:
- Build them ourselves
- Request them from upstream
- Mark those platforms as unsupported for watch mode

## Testing Strategy

1. Build with GoReleaser for each platform
2. Extract and verify the embedded watcher binary matches the platform
3. Test `plur watch install` on each platform
4. Ensure watch mode works correctly

## Implementation Checklist

- [ ] Check e-dant/watcher releases for all needed binaries
- [ ] Download missing binaries if available
- [ ] Create platform-specific embed files
- [ ] Update build process to use correct embed file
- [ ] Test cross-platform builds with GoReleaser
- [ ] Update documentation with platform support matrix
- [ ] Add CI tests for each platform