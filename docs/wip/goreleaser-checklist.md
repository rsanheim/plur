# GoReleaser Implementation Checklist

This checklist tracks the implementation of GoReleaser for plur, following the [GoReleaser PRD](./goreleaser-prd.md).

## Current State Summary
* **Current Release Process**: Ruby script (`script/release`) that creates tags, updates changelog, and creates GitHub releases
* **Version Format**: Semver with `v` prefix (e.g., `v0.7.0`)
* **Distribution**: Currently via `go install` only - no pre-built binaries
* **Version Management**: ✅ COMPLETED - Now uses Go's VCS BuildSettings with git describe formatting
* **Last Updated**: 2025-08-27

---

## Phase 1: Local GoReleaser Setup ✅ COMPLETED

### 1.1 Installation & Basic Configuration
- [x] ✅ Install GoReleaser locally (`brew install goreleaser` or download binary)
- [x] ✅ Create `.goreleaser.yml` in plur directory
- [x] ✅ Configure basic project settings:
  - [x] Project name: `plur`
  - [x] Main package: `.` (from plur dir)
  - [x] Binary name: `plur`

### 1.2 Build Configuration
- [x] ✅ Configure builds section:
  ```yaml
  builds:
    - id: plur
      main: .
      binary: plur
      goos: [darwin, linux, windows]
      goarch: [amd64, arm64]
  ```
- [x] ✅ Set up ldflags for version injection:
  ```yaml
  ldflags:
    - -s -w
    - -X main.version={{.Version}}
    - -X main.commit={{.Commit}}
    - -X main.date={{.Date}}
    - -X main.builtBy=goreleaser
  ```
- [x] ✅ Update `plur/version.go` to properly use ldflags variables - COMPLETED
- [x] ✅ Test version output with `plur --version` after build - Shows `0.10.0-dev-327b9f2`

### 1.3 Archive Configuration
- [x] ✅ Configure archive formats:
  - [x] tar.gz for Unix systems (darwin, linux)
  - [x] zip for Windows
- [x] ✅ Set naming template: `plur_{{.Version}}_{{.Os}}_{{.Arch}}`
- [x] ✅ Include necessary files:
  - [x] README.md (copied via before hook)
  - [x] LICENSE (created placeholder, copied via before hook)
  - [x] CHANGELOG.md (copied via before hook)
- [x] ✅ Configure checksums (SHA256)

### 1.4 Local Testing
- [x] ✅ Run `goreleaser build --snapshot --clean`
- [x] ✅ Verify output structure in `dist/` directory
- [x] ✅ Test binaries on available platforms:
  - [ ] darwin/amd64 (Intel Mac) - Not tested yet
  - [x] darwin/arm64 (Apple Silicon) - Tested
  - [x] linux/amd64 (via CircleCI) - Tested in CI
- [x] ✅ Verify version info: `./dist/plur_darwin_arm64_v8.0/plur --version`
- [x] ✅ Confirm artifact naming follows convention

### 1.5 Watcher Binary Packaging ✅
- [x] ✅ **Strategy Decision**: Continue with existing go:embed approach
  - [x] ✅ macOS Intel (x86_64) - NOT SUPPORTED (documented)
  - [x] ✅ Windows - Include as experimental/alpha
  - [x] ✅ macOS ARM64 - Already embedded
  - [x] ✅ Linux ARM64 - Already embedded
  - [x] ✅ Linux x86_64 - Already embedded
- [x] ✅ Implementation tasks:
  - [x] ✅ Download Windows watcher binaries
  - [x] ✅ Update `lib/tasks/vendor.rake` to include Windows platforms
  - [x] ✅ Update `watch/binary.go` to support Windows platforms
  - [x] ✅ Test GoReleaser builds with embedded binaries
  - [x] ✅ Document Windows support as experimental in README

### 1.6 CircleCI Integration ✅
- [x] ✅ Add new job to `.circleci/config.yml`
- [x] ✅ Configure using official CircleCI Go orb's `install-goreleaser` command
- [x] ✅ Test linux/amd64 build in CI
- [x] ✅ Store artifacts for download/verification
- [x] ✅ Add to existing workflow

---

## Phase 2: Enhanced Developer Experience ⏳

### 2.1 Script Integration
- [ ] Create `script/release-goreleaser` as experimental wrapper
- [ ] Add options to `script/release`:
  - [ ] `--dry-run` flag (uses goreleaser snapshot)
  - [ ] `--goreleaser` flag to use new pipeline
- [ ] Maintain backward compatibility with current process

### 2.2 Version Management Enhancement
- [x] ✅ Ensure `plur/version.go` properly handles:
  - [x] ldflags when set (release builds)
  - [x] runtime/debug fallback (development builds)
  - [x] "dev" version for untagged builds
- [x] ✅ Test version display in various scenarios:
  - [x] Local development build - Shows `v0.10.0-8-g327b9f2`
  - [x] GoReleaser snapshot - Shows `0.10.0-dev-327b9f2`
  - [ ] Tagged release build - To be tested

### 2.3 Validation & Testing
- [ ] Create test script to validate artifacts:
  - [ ] Check binary architecture
  - [ ] Verify version string format
  - [ ] Test execution on target platform
- [ ] Document validation process

---

## Phase 3: CI/CD Readiness (Pre-Public) ⏳

### 3.1 GitHub Actions Workflow
- [ ] Create `.github/workflows/goreleaser.yml`:
  ```yaml
  name: goreleaser
  on:
    push:
      tags:
        - 'v*'
  ```
- [ ] Configure workflow:
  - [ ] Checkout with fetch-depth: 0
  - [ ] Set up Go environment
  - [ ] Install GoReleaser
  - [ ] Run goreleaser release
- [ ] Initially disable with `if: false` condition
- [ ] Add required secrets documentation

### 3.2 Release Configuration
- [ ] Configure release section in `.goreleaser.yml`:
  - [ ] GitHub release creation
  - [ ] Release notes from CHANGELOG.md
  - [ ] Draft mode initially
- [ ] Set up artifact upload configuration
- [ ] Configure release name template

### 3.3 Homebrew Formula Preparation
- [ ] Decide on tap structure:
  - [ ] Option A: `rsanheim/homebrew-tap`
  - [ ] Option B: `rsanheim/homebrew-plur`
- [ ] Create tap repository (private initially)
- [ ] Configure GoReleaser brew section (commented out):
  ```yaml
  # brews:
  #   - tap:
  #       owner: rsanheim
  #       name: homebrew-tap
  #     folder: Formula
  #     homepage: https://github.com/rsanheim/plur
  #     description: Fast parallel test runner for Ruby/RSpec
  ```
- [ ] Document formula testing process

### 3.4 Signing & Notarization Prep
- [ ] Research macOS code signing requirements
- [ ] Document certificate requirements
- [ ] Prepare signing configuration (commented):
  ```yaml
  # signs:
  #   - artifacts: checksum
  #     cmd: gpg
  ```
- [ ] Create signing key documentation

---

## Phase 4: Migration & Validation ⏳

### 4.1 Parallel Testing Period
- [ ] Run both release methods for 2-3 releases:
  - [ ] Traditional `script/release`
  - [ ] GoReleaser in snapshot mode
- [ ] Compare outputs:
  - [ ] File sizes
  - [ ] Version strings
  - [ ] Archive contents
- [ ] Document any discrepancies

### 4.2 Documentation Updates
- [ ] Update README.md with new installation methods:
  - [ ] Pre-built binaries from releases
  - [ ] Homebrew (when ready)
  - [ ] Existing `go install` method
- [ ] Create `docs/releasing.md` with:
  - [ ] Release process overview
  - [ ] GoReleaser configuration details
  - [ ] Troubleshooting guide
- [ ] Update CONTRIBUTING.md with release information

### 4.3 Cutover Preparation
- [ ] Create cutover checklist
- [ ] Test full release process locally
- [ ] Verify GitHub token permissions
- [ ] Plan rollback strategy

---

## Phase 5: Public Launch Activation 🚀

### 5.1 Enable GitHub Actions
- [ ] Remove `if: false` condition from workflow
- [ ] Test with a release candidate tag
- [ ] Verify artifacts appear in GitHub release
- [ ] Check download and execution on all platforms

### 5.2 Homebrew Activation
- [ ] Make tap repository public
- [ ] Uncomment brew configuration
- [ ] Test formula generation
- [ ] Verify `brew install rsanheim/tap/plur` works
- [ ] Add tap/install instructions to README

### 5.3 Enhanced Distribution
- [ ] Enable GPG signing if configured
- [ ] Add macOS notarization if available
- [ ] Consider Docker image generation
- [ ] Set up release announcement automation

### 5.4 Monitoring & Iteration
- [ ] Monitor first few releases closely
- [ ] Gather user feedback on installation
- [ ] Track download statistics
- [ ] Iterate based on issues

---

## Success Criteria ✅

### Must Have
- [ ] Single command release (`script/release v1.0.0`)
- [ ] Identical artifacts whether built locally or in CI
- [ ] Multi-platform binaries (darwin, linux, windows)
- [ ] Checksums for all artifacts
- [ ] Automated GitHub release creation

### Should Have
- [ ] Homebrew formula generation
- [ ] Signed binaries
- [ ] Shell completion files
- [ ] Docker images

### Nice to Have
- [ ] macOS notarization
- [ ] Scoop manifest (Windows)
- [ ] AUR package (Arch Linux)
- [ ] Snap package (Ubuntu)

---

## Open Questions 🤔

1. **Homebrew Tap Structure**
   - Single tap for all projects (`homebrew-tap`)?
   - Project-specific tap (`homebrew-plur`)?
   - Decision: _______________

2. **Code Signing Strategy**
   - GPG signing only?
   - Apple Developer certificate?
   - Decision: _______________

3. **Shell Completions**
   - Include in releases?
   - Which shells (bash, zsh, fish)?
   - Decision: _______________

4. **Docker Distribution**
   - Official image?
   - Multi-arch support?
   - Decision: _______________

5. **Version Bump Automation**
   - Keep manual version specification?
   - Add semantic-release?
   - Decision: _______________

---

## Notes & References

* [GoReleaser Documentation](https://goreleaser.com/intro/)
* [Example: Hugo's .goreleaser.yml](https://github.com/gohugoio/hugo/blob/master/.goreleaser.yaml)
* [Example: Terraform's release process](https://github.com/hashicorp/terraform)
* Current release script: `script/release`
* Version management: `plur/version.go`

---

## Progress Tracking

* **Started**: 2025-08-21
* **Phase 1 Complete**: ✅ 100% - 2025-08-27
* **Phase 2 Complete**: ~25% (version management done, script integration pending)
* **Phase 3 Complete**: _________
* **Phase 4 Complete**: _________
* **Go Live**: _________

### Release History with GoReleaser
* First test release: _________
* First production release: _________

### Recent Updates (2025-08-27)
* Created placeholder LICENSE file (pending OSS license selection)
* Modified `.goreleaser.yml` to copy parent directory files via before hooks
* Updated `.gitignore` in plur/ to exclude copied documentation files
* Updated CircleCI config to use official Go orb's `install-goreleaser` command (v1.11.0)
* Verified successful snapshot builds with all documentation files included