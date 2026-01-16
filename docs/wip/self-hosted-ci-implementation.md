# Self-Hosted CI Implementation Plan

*Status:* Complete
*Created:* 2026-01-14
*Completed:* 2026-01-15
*Related:* [self-hosted-ci.md](self-hosted-ci.md) (research & reference)

## Overview

Implement a secure, VM-isolated CircleCI self-hosted runner using Tart on Mac Studio. The runner executes inside a macOS VM, providing hypervisor-level isolation from the host.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│ Mac Studio (Host)                                           │
│                                                             │
│  ┌───────────────────────────────────────────────────────┐  │
│  │ Tart VM: plur-runner                                  │  │
│  │ (macOS Sequoia, 6 CPU, 16GB RAM)                      │  │
│  │                                                       │  │
│  │  ┌─────────────────────────────────────────────────┐  │  │
│  │  │ CircleCI Machine Runner                         │  │  │
│  │  │ * Resource class: rsanheim/mac-studio           │  │  │
│  │  │ * Token via 1Password (op read)                 │  │  │
│  │  │ * Runs as LaunchAgent                           │  │  │
│  │  └─────────────────────────────────────────────────┘  │  │
│  │                                                       │  │
│  │  Tools: mise → ruby 4, go 1.25, python 3              │  │
│  │  Plur: cloned from git, built with bin/rake install   │  │
│  └───────────────────────────────────────────────────────┘  │
│                                                             │
│  Host only runs: Tart (VM management)                       │
└─────────────────────────────────────────────────────────────┘
```

## Success Criteria

* [x] Tart VM boots and is accessible via SSH from host
* [x] VM has mise, Ruby 4, Go 1.25, Python 3 installed and working
* [x] CircleCI machine runner runs inside VM and claims jobs
* [x] Plur builds and tests pass when triggered from CircleCI
* [x] Reference repos testing works (RuboCop ✅, Discourse ✅ 99.9%, rspec-expectations ✅)
* [ ] VM startup is automated (host launchd or manual script)
* [x] Setup is documented and reproducible

## Decision Log

| Decision | Choice | Rationale |
|----------|--------|-----------|
| VM persistence | Long-running (not ephemeral) | Simpler for v1; ephemeral can be added later |
| Tool installation | mise (not brew/manual) | Consistent with host setup, easy version management |
| Token storage | Config file in VM | Simple; token retrieved from host's 1Password during setup |
| Runner mode | Continuous (not single-task) | Standard runner behavior |

## Phase 1: Tart VM Base Setup ✅

**Status**: Complete

* [x] Clone base macOS image: `tart clone ghcr.io/cirruslabs/macos-sequoia-base:latest plur-runner`
* [x] Configure VM resources: `tart set plur-runner --cpu 4 --memory 8192`
* [x] Start VM and complete initial setup (enable Remote Login/SSH)
* [x] Verify SSH access from host
* [x] Add SSH key for passwordless access (`~/.ssh/ci-vm-key`)
* [x] Run `script/ci-host-setup` to install all dev tools
  * Verified: Ruby 4.0.1, Go 1.25.5, Python 3.14.2, Bundler 4.0.4, circleci-runner

## Phase 2: VM Development Environment ✅

**Status**: Complete

Use `script/ci-host-setup` to install all development tools. The script handles:
* Xcode Command Line Tools
* Homebrew
* mise (version manager)
* Ruby 4, Go 1.25, Python 3 (via mise, from `.mise.toml`)
* Bundler

Validation: `ssh admin@$(tart ip plur-runner) "ruby --version && go version && python --version && bundler --version"`

## Phase 3: CircleCI Runner in VM ✅

**Status**: Complete

* [x] Install runner via Homebrew (handled by `script/ci-host-setup`)
* [x] Clear quarantine attribute (handled by `script/ci-host-setup`)
* [x] Get token from host's 1Password and create config
* [x] Test runner starts manually
* [x] Set up LaunchAgent for auto-start
* [x] Verify runner is running and claiming jobs

## Phase 4: CircleCI Workflow Update ✅

**Status**: Complete - Core jobs passing

Jobs in `.circleci/config.yml`:

| Job | Status | Description |
|-----|--------|-------------|
| `build-and-test-go-macos` | ✅ Passing | Basic build, lint, Go tests |
| `test-ruby-integration-macos` | ✅ Passing | Full Ruby integration (default-ruby, default-rails, Ruby specs) |
| `test-reference-repos-macos` | ✅ Passing | Reference repos (RuboCop, Discourse, rspec-expectations) |

## Phase 5: Host Automation (Deferred)

**Status**: Not started - deferring until core workflow is stable

Options:
* Manual start script: `~/bin/start-plur-runner-vm`
* Host LaunchAgent for auto-start at login

## Phase 6: Cleanup Host Runner ✅

**Status**: Complete

* [x] Stop host runner
* [x] Remove host runner config and plist

## Phase 7: Reference OSS Repos Testing ✅

**Status**: Complete - All reference repos working

### Results

| Repo | Type | Status | Notes |
|------|------|--------|-------|
| rubocop/rubocop | Pure Ruby | ✅ Passing | No services needed |
| discourse/discourse | Rails | ✅ 99.9% (4051/4055) | 4 failures are ImageMagick config differences |
| rspec/rspec-expectations | Pure Ruby | ✅ Passing | Coverage disabled for parallel runs |

### Fixes Applied (2026-01-15)

1. **PATH precedence fix**: Removed redundant `PATH="/opt/homebrew/bin:$PATH"` that was overriding mise shims
2. **Node version**: Bumped to node@22.13.0 for @faker-js/faker compatibility
3. **PostgreSQL**: Added pgvector extension for Discourse AI plugin
4. **Tools**: Added coreutils (for `timeout` command) and imagemagick

### Discourse Test Notes

The 4 failing tests (out of 4055) are `OptimizedImage.crop` specs that expect specific file sizes. These fail due to different ImageMagick compression defaults between Homebrew and Discourse's Docker-based CI. Discourse's own CI also has intermittent failures.

### Job Configuration

The `test-reference-repos` job installs via Homebrew:
* PostgreSQL 17, pgvector, Redis (services)
* coreutils, imagemagick (tools)

And via mise:
* node@22.13.0, pnpm@10 (versioned tools)

## Limitations & Constraints

### One Job at a Time

**Each CircleCI machine runner agent executes one job at a time.** This is architectural, not configurable.

Our current workflow has three jobs targeting `rsanheim/mac-studio`:
* `build-and-test-go-macos`
* `test-ruby-integration-macos` (depends on build-and-test-go-macos)
* `test-reference-repos-macos` (depends on build-and-test-go-macos)

When a build triggers, `build-and-test-go-macos` runs first. Once it passes, both downstream jobs become eligible and one will queue while the other runs.

This is expected behavior with a single runner agent. See [self-hosted-ci.md](self-hosted-ci.md#concurrency--job-queuing) for scaling options if this becomes a bottleneck.

### VM Resources

The VM is configured with 6 CPU cores and 16GB RAM, shared across all CI workloads. Resource-intensive jobs (like Discourse tests) may benefit from having the runner to themselves rather than competing with other jobs.

## Troubleshooting

### VM Won't Start (Keychain Error)

macOS 15+ requires unlocked keychain. Connect via Screen Sharing first, or:
```bash
security unlock-keychain -p '' login.keychain
```

### Runner Can't Authenticate

Regenerate token from host:
```bash
TOKEN=$(op read 'op://Private/circle ci self hosted runner/credential')
ssh admin@$(tart ip plur-runner) "sed -i '' \"s/auth_token:.*/auth_token: $TOKEN/\" ~/Library/Preferences/com.circleci.runner/config.yaml"
```

### SSH Connection Refused

Enable Remote Login in VM: System Settings → General → Sharing → Remote Login → On

### VM IP Changes

Use `tart ip plur-runner` to get current IP. Consider adding to `/etc/hosts` for stable access.

## Files Created/Modified

| Location | File | Purpose |
|----------|------|---------|
| Repo | `script/ci-host-setup` | Automated VM setup script |
| Repo | `.mise.toml` | Tool versions (Ruby 4, Go 1.25, Python 3) |
| Repo | `.circleci/config.yml` | Updated workflow with self-hosted jobs |
| Host | `~/bin/start-plur-runner-vm` | VM startup script (optional) |
| VM | `~/Library/Preferences/com.circleci.runner/config.yaml` | Runner config |
| VM | `~/Library/LaunchAgents/com.circleci.runner.plist` | Runner auto-start |

## References

* [self-hosted-ci.md](self-hosted-ci.md) - Research and security model details
* [Tart Quick Start](https://tart.run/quick-start/)
* [CircleCI Machine Runner Installation](https://circleci.com/docs/install-machine-runner-3-on-macos/)
* [mise Getting Started](https://mise.jdx.dev/getting-started.html)
