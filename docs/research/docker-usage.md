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

* [ ] Create Dockerfile with Alpine + Ruby 3.4 + Go
* [ ] Create docker-compose.yml for volume mounting
* [ ] Create docker-test.sh script
* [ ] Test rux build process in container
* [ ] Test watcher installation in container
* [ ] Run fixture project tests
* [ ] Document any Linux-specific issues found
* [ ] Add CI integration (optional)
