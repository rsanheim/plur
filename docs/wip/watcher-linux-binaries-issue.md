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

### The Missing Link

* Main `build` task depends on `vendor:build`
* `vendor:build` only downloads the watcher for the **build machine's platform**
* No task exists to download **all platform** watcher binaries
* Documentation assumes all binaries would be present

## Solution

### Short-term Fix: Add `vendor:all` Task

Create a new rake task that downloads all platform watcher binaries:

```ruby
# lib/tasks/vendor.rake

namespace :vendor do
  desc "Download watcher binaries for all platforms"
  task :all do
    platforms = [
      "aarch64-apple-darwin",
      "x86_64-apple-darwin", 
      "aarch64-unknown-linux-gnu",
      "x86_64-unknown-linux-gnu"
    ]
    
    platforms.each do |platform|
      download_watcher_for_platform(platform)
    end
  end
  
  def download_watcher_for_platform(platform)
    # Download from: https://github.com/e-dant/watcher/releases/download/0.13.6/#{platform}.tar
    # Extract to: plur/embedded/watcher/watcher-#{platform}
  end
end
```

### Updated Build Process

1. **Before building for distribution:**
   ```bash
   bin/rake vendor:all        # Download ALL watcher binaries
   bin/rake build:linux       # Build Linux binaries with embedded watchers
   ```

2. **Update main build dependency:**
   ```ruby
   # For CI/distribution builds
   task build: ["vendor:all", "lint:go"] do
     # ...
   end
   ```

### Long-term Improvements

1. **Add CI check** to ensure all platform binaries are present before release
2. **Update documentation** to clarify build requirements
3. **Consider git-lfs** or similar for storing binary assets
4. **Add `vendor:verify` task** to check all binaries are present and valid

## Testing Requirements

After implementing the fix:

1. **Local verification:**
   ```bash
   bin/rake vendor:clean
   bin/rake vendor:all
   ls -la plur/embedded/watcher/  # Should show 4 binaries
   ```

2. **Linux container test:**
   ```bash
   bin/rake build:linux
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