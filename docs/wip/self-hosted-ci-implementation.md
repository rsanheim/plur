# Self-Hosted CI Implementation Plan

*Status:* In Progress
*Created:* 2026-01-14
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
│  │ (macOS Sequoia, 4 CPU, 8GB RAM)                       │  │
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
* [ ] Reference repos testing works (RuboCop ✅, Discourse blocked on node version)
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
| `test-macos-arm64` | ✅ Passing | Basic build, lint, Go tests |
| `test-ruby-integration-macos` | ✅ Passing | Full Ruby integration (default-ruby, default-rails, Ruby specs) |
| `test-reference-repos` | ❌ Failing | Blocked (see Phase 7) |

## Phase 5: Host Automation (Deferred)

**Status**: Not started - deferring until core workflow is stable

Options:
* Manual start script: `~/bin/start-plur-runner-vm`
* Host LaunchAgent for auto-start at login

## Phase 6: Cleanup Host Runner ✅

**Status**: Complete

* [x] Stop host runner
* [x] Remove host runner config and plist

## Phase 7: Reference OSS Repos Testing 🚧

**Status**: In Progress - Discourse blocked on node version conflict

### Progress

* [x] Git SSH config fixed (cleared URL rewriting in job)
* [x] RuboCop tests passing with plur
* [ ] Discourse tests blocked on node version (see below)
* [ ] rspec-core tests (runs after Discourse)

### Current Blocker: Discourse Node Version

Discourse requires node 20/22/24 but the VM has node v25.3.0 installed via Homebrew from previous runs. Even though we install node@22.12.0 via mise, the Homebrew node takes precedence in PATH.

```
ERR_PNPM_UNSUPPORTED_ENGINE Unsupported environment (bad pnpm and/or Node.js version)
Your Node version is incompatible with "mktemp@2.0.2".
Expected version: 20 || 22 || 24
Got: v25.3.0
```

**Debugging approach**: Split Discourse step into Setup/Verify/Test:
* Setup: Install postgres, redis, node@22.12.0 via mise, pnpm@10 via mise
* Verify: Print PATH, `mise ls`, `which node`, versions (debug step)
* Test: bundle, pnpm install, rake tasks, plur

**Root cause investigation needed**:
* Check Verify step output in CircleCI UI to see PATH order
* Likely need to either uninstall Homebrew node from VM, or ensure mise shims come before `/opt/homebrew/bin` in PATH

### Reference Repos

| Repo | Type | Status | Notes |
|------|------|--------|-------|
| rubocop/rubocop | Pure Ruby | ✅ Passing | No services needed |
| rspec/rspec-core | Pure Ruby | ⏳ Pending | Runs after Discourse |
| discourse/discourse | Rails | ❌ Blocked | Node version conflict |

### Discourse Requirements

From `package.json`:
* `node >= 20` (specifically needs 20, 22, or 24 - not 25)
* `pnpm ^10` (packageManager: pnpm@10.28.0)

Discourse's `db:migrate` triggers `assets:precompile:asset_processor` which runs:
```
pnpm -C=frontend/asset-processor node build.js
```

This requires node and pnpm to be properly installed and in PATH.

### Job Configuration

The `test-reference-repos` job:
* Installs PostgreSQL 17 and Redis via Homebrew (services)
* Installs node@22.12.0 and pnpm@10 via mise (versioned tools)
* Clones repos with `--depth 1` for speed
* Runs plur on each repo's spec directory
* Skips Discourse system tests (need Playwright)

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
