## Rux and Docker

I'd like to add the ability to test and exercise Rux inside a Docker container for testing 
on linux architecture, as well as to test file system events on mounted volumes.

As such, we need a docker container that matches the following:

* latest Alpine image
* standard ruby 3.4 installed
* mounted volume for ./rux (so we can build rux from source with the matching architecture)
* mounted volume for ./fixtures/projects (so we have all test projects bidirectionally mounted to the container)
* ...maybe we just mount the entire repo if that is easier? it may make sense to mount everything EXCEPT `references`
* a small script that we can run on the docker container to build rux, install the watcher binary, and then run the various projects in fixtures/projects
* anything else to make debugging /dev easier on this image as issues arise

## Questions & Considerations

1. **Go toolchain**: Should we include Go in the container since we need to build rux from source? Or use multi-stage build?
   - yup, lets install the latest go - 1.24.4 I think ? single stage is fine
2. **Ruby version**: Ruby 3.4 is bleeding edge - should we use 3.3.x for stability or stick with 3.4?
   - ruby 3.4 is good for now!
3. **Volume mounting strategy**: I'd recommend mounting the entire repo except `references/` and `vendor/` - simpler and allows access to specs
   * sounds great
4. **Base image**: Alpine is great for size, but sometimes causes issues with native extensions. Should we also test on Debian/Ubuntu?
   * lets do Ubuntu
5. **Development tools**: What debugging tools would be helpful? (vim, less, curl, git, strace for fs events?)
   * vim, less, hyperfine, and strace sound good to start

## Notes

* I have docker and docker-compose installed, though my actual runtime is Orbstack (it responds to the standard docker commands)

## Proposed Approach

1. Create a `Dockerfile` with:
   * Multi-stage build (Go builder stage + Ruby runtime stage)
   * Or single stage with both Go and Ruby
   * Essential build tools and debugging utilities
   * NOTE: we may want to be able to do multi-arch builds at some point, to do Linux/aarch64, Linux/x86_64

2. Create `docker-compose.yml` for easy volume management:
   * Mount entire repo except `references/` and `vendor/`
   * Set working directory
   * Environment variables for test execution

3. Create `scripts/docker-test.sh` that:
   * Builds rux for Linux
   * Installs watcher binary
   * Runs test suite against fixture projects
   * Optionally runs specific tests

## Checklist

* [x] Create Dockerfile with Ubuntu + Ruby 3.4 + Go 1.24.4
* [x] Create docker-compose.yml for volume mounting
* [x] Create docker-test.sh script
* [x] Test rux build process in container
* [x] Test watcher installation in container
* [x] Run fixture project tests
* [x] Document any Linux-specific issues found
* [ ] Add CI integration (optional)
* [ ] Verify full build works on Mac (host) and docker image
* [x] Update this document, add findings and follow up
* [ ] Commit

## Findings

### Successfully Implemented
1. **Multi-architecture Docker support** - Dockerfile handles both amd64 and arm64
2. **Volume mounting strategy** - Entire repo mounted except references/ and vendor/
3. **Test script with options**:
   - `--build` - Build image only
   - `--shell` - Interactive shell
   - `--run <cmd>` - Run specific command
   - `--ps` - Show container status
   - `--logs` - Show container logs
   - Default: Run full test suite

### Issues Found
1. **Watcher binary architecture mismatch** - Downloads x86_64 binary on arm64 system
   - Platform detection in `lib/tasks/vendor.rake` correctly identifies `aarch64-linux`
   - But still downloads x86_64 binary: "https://github.com/e-dant/watcher/releases/download/0.13.6/x86_64-unknown-linux-gnu.tar"
   - Rux doctor reports: "watcher binary not found at /root/.rux/bin/watcher-aarch64-unknown-linux-gnu"
   - The rake task saves binary as `watcher-x86_64-unknown-linux-gnu` but rux expects `watcher-aarch64-unknown-linux-gnu`

2. **Version warning** - docker-compose.yml version attribute is obsolete
   - ✅ Fixed - removed `version: '3.8'` line

3. **Root user warnings** - Bundler warns about running as root
   - Currently disabled user restriction to avoid permission issues
   - Could revisit with proper volume permissions

### Improvements Made
1. **Created standalone test script** - `scripts/docker-test-runner.sh`
   - Extracted from inline script for better maintenance
   - Synced via volume mount

2. **Enhanced docker-test.sh with platform support**
   - `--platform linux/amd64` - Build/run x86_64 image
   - `--platform linux/arm64` - Build/run ARM64 image
   - Falls back to docker-compose for default platform

3. **Multi-architecture Dockerfile**
   - Handles both amd64 and arm64 architectures
   - Installs correct hyperfine and Go binaries

### x86_64 Test Results (Completed)
1. **Build successful** - x86_64 image builds via Rosetta (~2-4 minutes)
2. **Watcher binary** - Downloads correct x86_64 binary after fix
3. **Test suite** - All tests pass: 66 examples, 0 failures
4. **Performance** - No significant difference from ARM64

### Architecture Comparison Summary
* Both ARM64 and x86_64 work correctly after watcher binary fix
* Test execution times are identical (~0.44 seconds)
* No architecture-specific issues found
* See `docker-architecture-comparison.md` for detailed results

### Next Steps
1. ✅ Fixed watcher binary architecture detection bug in vendor.rake
2. Add GitHub Actions workflow for Docker testing
3. ✅ Tested x86_64 build successfully
4. Consider using pre-built Ruby images to speed up builds

## Docker Optimization Analysis

### Current Docker Build Performance
* **Always rebuilds** - No caching logic in docker-test.sh
* **Build times**:
  * ARM64 (native): ~1-2 minutes
  * x86_64 (Rosetta): ~2-4 minutes (Ruby compilation is slow)
* **Image size**: 1.32GB

### Official Ruby Image Analysis

Investigated using official Ruby Docker images instead of building from source:

#### Available Options:
* **ruby:3.4** (full): ~1GB, includes gcc/make/bundler
* **ruby:3.4-slim**: ~180MB, minimal (no build tools)
* **Ruby version**: 3.4.4 (newer than our 3.4.0)

#### Pros of Official Ruby Base:
1. **70% faster builds** - No Ruby compilation needed
2. **Maintained** - Regular security updates
3. **Multi-arch** - Native amd64/arm64 images
4. **Better caching** - Smaller, focused layers

#### Cons/Challenges:
1. **Fixture projects conflict** - Official image expects single app at `/usr/src/app`
2. **Bundler paths** - Uses `BUNDLE_PATH="/usr/local/bundle"` globally
3. **Missing tools** - No Go, hyperfine, strace, vim
4. **Version difference** - 3.4.4 vs our 3.4.0

#### Multi-Project Bundler Issue:
Our structure has multiple independent Ruby projects:
* `/workspace/default-ruby/Gemfile`
* `/workspace/spec/Gemfile`
* `/workspace/Gemfile` (root)

Official Ruby image assumes single project, which conflicts with our test setup.

### Recommendation:
Use `ruby:3.4` as base with these modifications:
1. Override bundler config for multi-project support
2. Add Go installation (~100MB)
3. Add development tools layer
4. Skip ONBUILD variants (they enforce single-app structure)

This would reduce Dockerfile from 88 to ~40 lines and cut build time significantly.

### Smart Build Strategy
Implement caching logic in docker-test.sh:
1. Add `--no-build` flag to skip rebuilds
2. Check image freshness vs Dockerfile timestamp
3. Add `--rebuild` flag to force fresh builds
4. Use Docker build cache more effectively
