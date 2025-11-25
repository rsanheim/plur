# GoReleaser Implementation Checklist

This checklist tracks the implementation of GoReleaser for plur, following the [GoReleaser PRD](../archive/2025-10-goreleaser-prd.md).

## Current State Summary
* **Release Process**: Two-step with GitHub Actions
  * `script/release prepare v0.14.0` - Generate changelog
  * `script/release push v0.14.0` - Tag and push (triggers GitHub Actions)
* **Version Format**: Semver with `v` prefix (e.g., `v0.13.0`)
* **Latest Release**: v0.13.0 (2025-11-25)
* **Status**: ✅ Production releases working via GitHub Actions
* **Last Updated**: 2025-11-25

---

## Phase 1: Local GoReleaser Setup ✅ COMPLETED

### 1.1 Installation & Basic Configuration
* [x] Install GoReleaser locally (`brew install goreleaser` or download binary)
* [x] Create `.goreleaser.yml` in plur directory
* [x] Configure basic project settings:
  * [x] Project name: `plur`
  * [x] Main package: `.` (from plur dir)
  * [x] Binary name: `plur`

### 1.2 Build Configuration
* [x] Configure builds section:
  ```yaml
  builds:
    - id: plur
      main: .
      binary: plur
      goos: [darwin, linux, windows]
      goarch: [amd64, arm64]
  ```
* [x] Set up ldflags for version injection:
  ```yaml
  ldflags:
    - -s -w
    - -X main.version={{.Version}}
    - -X main.commit={{.Commit}}
    - -X main.date={{.Date}}
    - -X main.builtBy=goreleaser
  ```
* [x] Update `plur/version.go` to properly use ldflags variables
* [x] Test version output with `plur --version` after build

### 1.3 Archive Configuration
* [x] Configure archive formats:
  * [x] tar.gz for Unix systems (darwin, linux)
  * [x] zip for Windows
* [x] Set naming template: `plur_{{.Version}}_{{.Os}}_{{.Arch}}`
* [x] Include necessary files:
  * [x] README.md (copied via before hook)
  * [x] LICENSE (created placeholder, copied via before hook)
  * [x] CHANGELOG.md (copied via before hook)
* [x] Configure checksums (SHA256)

### 1.4 Local Testing
* [x] Run `goreleaser build --snapshot --clean`
* [x] Verify output structure in `dist/` directory
* [x] Test binaries on available platforms:
  * [x] darwin/arm64 (Apple Silicon)
  * [x] linux/amd64 (via CircleCI)
* [x] Verify version info: `./dist/plur_darwin_arm64_v8.0/plur --version`
* [x] Confirm artifact naming follows convention

### 1.5 Watcher Binary Packaging
* [x] **Strategy Decision**: Continue with existing go:embed approach
  * [x] macOS Intel (x86_64) - NOT SUPPORTED (documented)
  * [x] Windows - Include as experimental/alpha
  * [x] macOS ARM64 - Already embedded
  * [x] Linux ARM64 - Already embedded
  * [x] Linux x86_64 - Already embedded
* [x] Implementation tasks:
  * [x] Download Windows watcher binaries
  * [x] Update `lib/tasks/vendor.rake` to include Windows platforms
  * [x] Update `watch/binary.go` to support Windows platforms
  * [x] Test GoReleaser builds with embedded binaries
  * [x] Document Windows support as experimental in README

### 1.6 CircleCI Integration
* [x] Add `test-goreleaser` job to `.circleci/config.yml`
* [x] Configure using official CircleCI Go orb's `install-goreleaser` command
* [x] Test linux/amd64 build in CI
* [x] Store artifacts for download/verification
* [x] Add to existing workflow

---

## Phase 2: Enhanced Developer Experience ✅ COMPLETED

### 2.1 Script Integration (Updated 2025-11-25)
* [x] Refactored `script/release` to subcommands:
  * [x] `prepare VERSION` - Generate changelog entries (doesn't commit)
  * [x] `push VERSION` - Verify, commit, tag, and push
  * [x] `extract-notes VERSION` - Extract release notes for CI
* [x] Integrated PR-based changelog with GoReleaser release notes
* [x] Disabled GoReleaser auto-changelog (we provide our own)
* [x] GitHub Actions handles GoReleaser execution (not local)

### 2.2 Version Management Enhancement
* [x] Ensure `plur/version.go` properly handles:
  * [x] ldflags when set (release builds)
  * [x] runtime/debug fallback (development builds)
  * [x] "dev" version for untagged builds
* [x] Test version display in various scenarios:
  * [x] Local development build
  * [x] GoReleaser snapshot
  * [x] Tagged release build (verified with v0.11.0+)

---

## Phase 3: CI/CD Readiness ✅ COMPLETED

### 3.1 GoReleaser in CI
* [x] `test-goreleaser` job validates builds on every push (CircleCI)
* [x] Snapshot builds tested in CI
* [x] Artifacts stored for verification

### 3.2 GitHub Actions Release Workflow (Added 2025-11-25)
* [x] Created `.github/workflows/release.yml`
* [x] Tag-triggered GoReleaser execution (`v*` tags)
* [x] Extracts release notes from CHANGELOG.md
* [x] Builds and publishes multi-platform binaries automatically

### 3.3 Release Configuration
* [x] Configure release section in `.goreleaser.yml`:
  * [x] GitHub release creation
  * [x] Release notes from CHANGELOG.md
* [x] Set up artifact upload configuration
* [x] Configure release name template

---

## Phase 4: Production Release Validation ✅ COMPLETED

Production releases (v0.11.0, v0.12.0, v0.13.0) have validated the entire release pipeline:

### 4.1 Binary Validation
* [x] Download and test binaries from releases:
  * [x] macOS ARM64 binary
  * [x] Linux x86_64 binary
  * [x] Linux ARM64 binary
  * [x] Windows x86_64 binary (experimental)
* [x] Verify each binary:
  * [x] Correct architecture
  * [x] Version string
  * [x] Basic execution
  * [x] Checksums match

### 4.2 Archive Contents
* [x] README.md included
* [x] LICENSE included
* [x] CHANGELOG.md included

### 4.3 Release Process
* [x] First production release: v0.11.0 (2025-10-17)
* [x] Latest production release: v0.13.0 (2025-11-25)
* [x] All platform binaries present
* [x] Download links working
* [x] Release notes accurate

---

## Phase 5: Distribution Enhancements 🚀 (Future)

### 5.1 Homebrew Formula Setup
* [ ] Create Homebrew tap repository:
  * [ ] `rsanheim/homebrew-tap` or `rsanheim/homebrew-plur`
* [ ] Configure GoReleaser brew section
* [ ] Test formula generation
* [ ] Verify `brew tap rsanheim/tap && brew install plur`

### 5.2 Additional Distribution
* [ ] Installation script (`curl | sh`)
* [ ] Scoop manifest (Windows)
* [ ] Shell completions (bash, zsh, fish)
* [ ] Docker image (optional)

---

## Success Criteria ✅

### Must Have (COMPLETED)
* [x] Single command release (`script/release v1.0.0`)
* [x] Identical artifacts whether built locally or in CI
* [x] Multi-platform binaries (darwin, linux, windows)
* [x] Checksums for all artifacts
* [x] Automated GitHub release creation
* [x] Production (non-draft) releases

### Nice to Have (Future)
* [ ] Homebrew formula generation
* [ ] Installation script
* [ ] Shell completion files
* [ ] Docker images
* [ ] Scoop manifest (Windows)

---

## Future Cleanup Tasks

* [ ] Delete ~100 draft test releases (v0.10.4-*-test) from GitHub
  * These were created during initial testing and are no longer needed
  * Run: `gh release list --json tagName -q '.[].tagName' | grep test | xargs -I {} gh release delete {} --yes`

## Completed Cleanup

* [x] Removed `lib/tasks/multi_platform.rake` - obsolete now that goreleaser handles builds
* [x] Added `--dry-run` flag to `script/release` for safe testing
* [x] Deleted `docs/development/multi-platform-builds.md` - documented obsolete rake tasks

---

## Notes & References

* [GoReleaser Documentation](https://goreleaser.com/intro/)
* [GoReleaser GitHub Actions](https://goreleaser.com/ci/actions/)
* Release script: `script/release` (subcommands: prepare, push, extract-notes)
* GitHub Actions workflow: `.github/workflows/release.yml`
* Version management: `plur/version.go`
* GoReleaser config: `plur/.goreleaser.yml`

---

## Progress Tracking

* **Started**: 2025-08-21
* **Phase 1 Complete**: 2025-08-27
* **Phase 2 Complete**: 2025-10-09
* **Phase 3 Complete**: 2025-10-11
* **Phase 4 Complete**: 2025-11-25
* **Production Releases**: v0.11.0 (2025-10-17), v0.12.0 (2025-10-31), v0.13.0 (2025-11-25)

### Update History

**2025-11-25 (GitHub Actions)**
* Added GitHub Actions workflow for tag-triggered releases
* Refactored `script/release` to subcommands (prepare/push/extract-notes)
* GoReleaser now runs in GitHub Actions, not locally

**2025-11-25**
* Marked Phase 4 as complete - production releases validate the pipeline
* Removed test release CI automation (unused)
* Updated status to reflect v0.11.0-v0.13.0 production releases
* Simplified checklist structure

**2025-10-11**
* Phase 3 Complete: Added CircleCI test release automation
* Added `--automated` and `--draft` flags to script/release

**2025-10-09**
* Phase 2 Complete: Integrated PR-based changelog with GoReleaser

**2025-08-27**
* Phase 1 Complete: Local GoReleaser setup and testing
