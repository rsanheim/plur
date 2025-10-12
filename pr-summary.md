# Automated Releases, Docker Improvements, and Package Manager Preparation

## Summary

Enhances the release process with automation and draft support, improves Docker installation, and prepares for future package manager distribution.

## Key Changes

### Release Process Automation
* **Automated releases** - Added `--automated` flag to skip prompts for CI/CD
* **Draft releases** - New `--draft` flag (default) for safer releases
* **CircleCI integration** - Test releases on `test-releases/*` branches
* **Git configuration** - Auto-configures git for automated tagging
* **Go 1.23 update** - Updated CI to use latest Go version

### Docker Improvements
* **Universal installer** (`install.sh`) - Self-installation script for containers
* **Batch installation** - Enhanced `install-plur-docker` with `--pattern` and `--parallel`
* **Comprehensive guide** (`docs/docker-integration.md`) - All Docker patterns
* **Legacy cleanup** - Removed 700+ lines of outdated Rux-era Docker code

### Package Manager Preparation
* **Research document** - Analysis of Go CLI distribution strategies
* **Homebrew tap guide** - Setup instructions for custom tap
* **GitHub Actions workflow** - Disabled (.disabled) for future use
* **Documentation reorganization** - Moved from `docs/wip/` to `docs/development/releases/`

## Test Plan

* [ ] Test automated release: Push to `test-releases/*` branch
* [ ] Verify draft release creation in GitHub UI
* [ ] Test `install.sh` in Docker: `curl -sSL .../install.sh | sh`
* [ ] Test batch installation: `install-plur-docker --pattern ".*" --parallel`
* [ ] Confirm GitHub Actions workflow doesn't run (disabled extension)