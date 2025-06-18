# Docker Architecture Test Comparison

## Test Environment
- Base image: Ubuntu 24.04
- Ruby version: 3.4.0dev (2024-12-25 master f450108330) +PRISM
- Go version: 1.24.4
- Rux version: v0.7.2-0.20250617080016-3c29b33f4ce7+dirty

## ARM64 (aarch64-linux)
### Platform Details
- Architecture: aarch64
- Ruby platform: aarch64-linux
- Watcher binary: watcher-aarch64-unknown-linux-gnu

### Test Results
- ✅ Rux builds successfully
- ✅ Watcher binary downloads correctly (after fix)
- ✅ Test suite passes: 66 examples, 0 failures
- ✅ Execution time: ~0.44 seconds

## x86_64 (via Rosetta)
### Platform Details
- Architecture: x86_64
- Ruby platform: x86_64-linux
- Watcher binary: watcher-x86_64-unknown-linux-gnu

### Test Results
- ✅ Rux builds successfully
- ✅ Watcher binary downloads correctly
- ✅ Test suite passes: 66 examples, 0 failures
- ✅ Execution time: ~0.44 seconds

## Key Findings

### 1. Watcher Binary Architecture Bug (Fixed)
- **Issue**: Platform detection in `lib/tasks/vendor.rake` had incorrect regex ordering
- **Fix**: Added `/aarch64.*linux/` pattern to catch "aarch64-linux" before generic `/linux/`
- **Result**: Both architectures now download correct binaries

### 2. Build Times
- ARM64: Fast native builds (~1-2 minutes)
- x86_64: Initially slow Ruby compilation via Rosetta (~2-4 minutes)
- Subsequent builds use cache (instant)

### 3. Test Performance
- No significant performance difference between architectures
- Both complete test suite in ~0.44 seconds
- Parallel execution works correctly on both

### 4. Compatibility
- All rux features work identically on both architectures
- No architecture-specific bugs found
- Docker volume mounts work correctly on both

## Recommendations

1. **CI/CD**: Test on both architectures to ensure compatibility
2. **Development**: Either architecture works well for local development
3. **Deployment**: Use native architecture for best performance

## Commands for Testing

Docker operations are now split into dedicated build and run scripts:

```bash
# Build images for different architectures
./script/docker-build --platform linux/arm64
./script/docker-build --platform linux/amd64

# Run tests on specific architecture
./script/docker-run --test --platform linux/arm64
./script/docker-run --test --platform linux/amd64

# Quick commands on specific architecture
./script/docker-run --command "rux doctor" --platform linux/amd64

# Interactive shell for debugging
./script/docker-run --shell --platform linux/arm64

# Run without rebuilding (much faster)
./script/docker-run --command "bin/rake test:ruby"
```

The scripts automatically use Plur configuration for:
- Correct number of cores (RUX_CORES)
- E-dant watcher version
- Proper directory paths
- Build timestamp tracking