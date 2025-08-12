# Watcher Linux Binaries Issue

**Date:** 2025-08-12
**Status:** 🔴 Broken - Linux watcher binaries not embedded  
**Impact:** `plur watch install` fails in all Linux Docker containers

## Problem Summary

The `plur watch` command fails to install the watcher binary in Linux containers because the Linux watcher binaries are not embedded in the plur executable, despite documentation claiming Linux support.

### Error Message
```
[example-service] root@271f735d9557:/app# plur watch install
01:07:14 - ERROR - Command failed error=watcher binary not embedded for this platform: 
open embedded/watcher/watcher-aarch64-unknown-linux-gnu: file does not exist
```

### Current State

**What exists:**
* Only macOS ARM64 watcher binary: `plur/embedded/watcher/watcher-aarch64-apple-darwin`
* No Linux binaries present

**What's expected (per documentation):**
* `watcher-aarch64-apple-darwin` (macOS ARM64) ✅ Present
* `watcher-x86_64-apple-darwin` (macOS x64) ❌ Missing  
  **NOTE** we don't care or want x86 for mac, we should remove it from docs
* `watcher-aarch64-unknown-linux-gnu` (Linux ARM64) ❌ Missing
* `watcher-x86_64-unknown-linux-gnu` (Linux x64) ❌ Missing

## Root Cause Analysis

### Two Disconnected Build Systems

1. **`vendor:build` task** (`lib/tasks/vendor.rake`)
   ```ruby
   # Only downloads for CURRENT platform
   def watcher_platform
     case RUBY_PLATFORM
     when /aarch64-darwin/, /arm64-darwin/
       "aarch64-apple-darwin"
     when /linux.*aarch64/, /linux.*arm64/, /aarch64.*linux/
       "aarch64-unknown-linux-gnu"
     when /linux/
       "x86_64-unknown-linux-gnu"
     else
       raise "Unsupported platform: #{RUBY_PLATFORM}"
     end
   end
   
   namespace :vendor do
     desc "Build vendored dependencies"
     task build: [watcher_binary_path]  # Only builds ONE binary
   end
   ```

2. **`build:linux` task** (`lib/tasks/multi_platform.rake`)
   * Cross-compiles plur Go binary for Linux (amd64/arm64)
   * Embeds whatever is in `plur/embedded/watcher/` at compile time
   * Since only macOS binary exists, Linux builds lack Linux watcher support

3. Confusing names: `vendor:build` actually just downloads the compiled watcher release, 
it doesn't actually "build" anything.

### The Missing Link

* Main `build` task depends on `vendor:build`
* `vendor:build` only downloads the watcher for the **build machine's platform**
* No task exists to download **all platform** watcher binaries
* Documentation assumes all binaries would be present

## Solution

1) Rename `vendor:build` to `vendor:download` to make it explicit that it just downloads
vendor'ed depedencies, it doesn't actually "build" anything

2) Update any refrences from `vendor:build` to `vendor:download`

3) Change `vendor:download` to `vendor:download:current` to be explicit that it
downloads the current machine platform.

4) Update main `build` to depend on `vendor:download:current`

When running `bin/rake build`, we are typically building for local dev, tests, or for CI tests - for this
case, we only care about the current machine platform.

5) Add `vendor:download:all` Task

Add a new task inside the `vendor` namespace that downloads desired watcher binaries.
This should build on and reuse the existing `vendor:download` code, but just do it for 
all desired platforms.

6) Update `vendor:clean` to clean all downloaded watcher binaries

7) Update `script/install-docker` to use `build:all` instead of `build:linux`

This should fix the issue having watcher binaries downloaded for linux amd64 and aarch64/arm64,
so that we can embed the correct binary for the targeted docker container. It also will 
ensure we have all binaries built for plur itself for supported platforms.

8) Update `docs/architecture/plur-watch-architecture.md` to reflect the new build process

9) Run full test suite and linters

## Testing Requirements

After implementing the fix:

1. **Local verification:**
   ```bash
   bin/rake vendor:clean
   bin/rake vendor:download:all
   ls -la plur/embedded/watcher/  # Should show binaries for all supported platforms
   ```

2. **Multi-arch build verification:**
   ```bash
   bin/rake build:all
   ls -la dist/  # Should show binaries for all supported platforms with correct names
   ```

2. **Linux container test:**
   ```bash
   ./script/install-docker install example-service -C app-compose -p vendor/gems/bin
   docker exec app-compose_example-service_1 plur watch install  # Should succeed
   ```

3. **Cross-platform verification:**
   * Test on Linux ARM64 container
   * Test on Linux x64 container  
   * Verify macOS still works

## Impact on Docker Installation

Once fixed, the enhanced `script/install-docker` with shared volume support will work properly:
* Plur installed to `vendor/gems/bin` (shared volume)
* Symlinks created in `/usr/local/bin`
* `plur watch install` will successfully install platform-appropriate watcher

## Architecture Notes

* Go's `runtime.GOARCH` reports `arm64` on ARM systems
* Linux `uname -m` reports `aarch64` for ARM64
* The code correctly handles both as aliases
* Binary naming convention: `watcher-{arch}-{vendor}-{os}-{abi}`

## References

* `docs/architecture/plur-watch-architecture.md` - Claims Linux support exists
* `lib/tasks/vendor.rake` - Current platform-specific download logic
* `lib/tasks/multi_platform.rake` - Cross-compilation build tasks
* `plur/watch/binary.go` - Runtime platform detection and binary mapping