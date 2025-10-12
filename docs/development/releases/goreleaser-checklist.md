# GoReleaser Implementation Checklist

This checklist tracks the implementation of GoReleaser for plur, following the [GoReleaser PRD](./goreleaser-prd.md).

## Current State Summary
* **Test Release Process**: Automated via CircleCI on `test-releases/*` branches
* **Production Release Process**: Manual from main branch using `script/release`
* **Version Format**: Semver with `v` prefix (e.g., `v0.7.0`)
* **Test Releases**: Draft releases with `-test` suffix for validation
* **Production Releases**: Clean versions from main branch (manual control)
* **CI/CD**: ✅ COMPLETED - CircleCI for test releases only (NOT GitHub Actions)
* **Last Updated**: 2025-10-11

## 🎯 Next Steps (Phase 4)
1. **Validate Draft Test Releases**: Download and test binaries from existing draft releases
2. **Test Each Platform**: Verify macOS ARM64, Linux x86_64/ARM64, Windows binaries
3. **Clean Up Test Releases**: Remove v0.10.4-*-test drafts after validation
4. **First Production Release**: Run `script/release v0.11.0` from main branch

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

## Phase 2: Enhanced Developer Experience ✅ COMPLETED

### 2.1 Script Integration
- [x] ✅ Enhanced `script/release` to use GoReleaser internally
- [x] ✅ Added `--extract-notes` flag to extract changelog entries for CI
- [x] ✅ Added `--automated` flag for CI/CD (skips confirmation prompts)
- [x] ✅ Added `--draft`/`--no-draft` flags to control release visibility
- [x] ✅ Integrated PR-based changelog with GoReleaser release notes
- [x] ✅ Disabled GoReleaser auto-changelog (we provide our own)
- [x] ✅ Maintained backward compatibility - same `script/release v0.x.x` command

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

## Phase 3: CI/CD Readiness ✅ COMPLETED

### 3.1 CircleCI Integration (Test Releases)
- [x] ✅ Enhanced CircleCI config for test release validation:
  - [x] Added `release` job with GoReleaser
  - [x] Configured to run on `test-releases/*` branches (for testing only)
  - [x] Added Git user configuration for commits
  - [x] Implemented HTTPS authentication with GITHUB_TOKEN
  - [x] Using `github-releases` context for token storage
- [x] ✅ Test release workflow (NOT for production):
  - [x] Creates test releases with `-test` suffix
  - [x] Validates the entire release pipeline
  - [x] Tests multi-platform binary generation
  - [x] All test releases are drafts for validation
- [x] ✅ Production releases are done manually from main branch

### 3.2 Release Configuration
- [x] ✅ Configure release section in `.goreleaser.yml`:
  - [x] GitHub release creation
  - [x] Release notes from CHANGELOG.md
  - [x] Draft mode configurable via `--draft` flag
- [x] ✅ Set up artifact upload configuration
- [x] ✅ Configure release name template
- [x] ✅ Successfully created multiple test releases (v0.10.4-*-test)



---

## Phase 4: Draft Release Validation ⏳ NEXT

### 4.1 Validate Existing Draft Releases
- [ ] Download and test binaries from draft releases:
  - [ ] macOS ARM64 binary
  - [ ] Linux x86_64 binary
  - [ ] Linux ARM64 binary
  - [ ] Windows x86_64 binary (experimental)
- [ ] Verify each binary:
  - [ ] Correct architecture (`file` command)
  - [ ] Version string (`plur --version`)
  - [ ] Basic execution (`plur doctor`)
  - [ ] Checksums match
- [ ] Test archive contents:
  - [ ] README.md included
  - [ ] LICENSE included
  - [ ] CHANGELOG.md included
- [ ] Clean up test releases after validation

### 4.2 Binary Distribution Testing
- [ ] Test installation methods:
  - [ ] Direct download and extract
  - [ ] Verify PATH setup instructions
  - [ ] Document any platform-specific issues
- [ ] Create installation script for easy setup
- [ ] Test on fresh systems/containers

### 4.3 Documentation Updates
- [ ] Update README.md with binary installation:
  - [ ] Direct download instructions
  - [ ] Platform-specific notes
  - [ ] Keep existing `go install` method
- [ ] Create troubleshooting guide for common issues
- [ ] Document checksum verification process

---

## Phase 5: Production Binary Releases 🚀

### 5.1 First Production Release (from main branch)
- [ ] Create first non-draft production release:
  - [ ] Work from main branch (NOT test-releases/*)
  - [ ] Use clean semantic version (e.g., v0.11.0)
  - [ ] Run manually: `script/release v0.11.0`
  - [ ] Consider using `--no-draft` for immediate publication
- [ ] Verify production release:
  - [ ] All platform binaries present
  - [ ] Download links working
  - [ ] Checksums correct
  - [ ] Release notes accurate
- [ ] Production release process:
  - [ ] No CI automation for production releases
  - [ ] Manual execution ensures control
  - [ ] Can be draft or immediate based on flags

### 5.2 Homebrew Formula Setup
- [ ] Create Homebrew tap repository:
  - [ ] `rsanheim/homebrew-tap` or `rsanheim/homebrew-plur`
  - [ ] Initially private for testing
- [ ] Configure GoReleaser brew section:
  ```yaml
  brews:
    - repository:
        owner: rsanheim
        name: homebrew-tap
      folder: Formula
      homepage: https://github.com/rsanheim/plur
      description: Fast parallel test runner for Ruby/RSpec
  ```
- [ ] Test formula generation with next release
- [ ] Verify `brew tap rsanheim/tap && brew install plur`
- [ ] Make tap public when ready

### 5.3 Distribution Enhancements
- [ ] Add installation script:
  ```bash
  curl -sSL https://raw.githubusercontent.com/rsanheim/plur/main/install.sh | sh
  ```
- [ ] Consider additional package managers:
  - [ ] Scoop (Windows)
  - [ ] AUR (Arch Linux)
  - [ ] asdf plugin
- [ ] Docker image (optional)

### 5.4 Release Process Refinement
- [ ] Set up release cadence (e.g., monthly)
- [ ] Create release checklist template
- [ ] Monitor download statistics
- [ ] Gather user feedback on installation
- [ ] Document known issues and solutions

---

## Success Criteria ✅

### Must Have
- [x] ✅ Single command release (`script/release v1.0.0`)
- [x] ✅ Identical artifacts whether built locally or in CI
- [x] ✅ Multi-platform binaries (darwin, linux, windows)
- [x] ✅ Checksums for all artifacts
- [x] ✅ Automated GitHub release creation

### Should Have (Next Priority)
- [ ] Homebrew formula generation
- [ ] Production (non-draft) releases
- [ ] Installation script

### Nice to Have
- [ ] Shell completion files
- [ ] Docker images
- [ ] Scoop manifest (Windows)
- [ ] AUR package (Arch Linux)
- [ ] asdf plugin

---

## Open Questions 🤔

1. **Homebrew Tap Structure**
   - Single tap for all projects (`homebrew-tap`)?
   - Project-specific tap (`homebrew-plur`)?
   - Decision: _______________

2. **Shell Completions**
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
* **Phase 2 Complete**: ✅ 100% - 2025-10-09
* **Phase 3 Complete**: ✅ 100% - 2025-10-11
* **Phase 4 Complete**: _________
* **Go Live**: _________

### Release History with GoReleaser
* First test release: v0.10.4-1568-test (2025-10-11)
* Multiple CI test releases: v0.10.4-{1618-1624}-test
* First production release: _________

### Recent Updates (2025-10-11)
* **Phase 3 Complete**: Full CI/CD automation with CircleCI
* Added `--automated` flag for CI/CD (skips confirmation prompts)
* Added `--draft`/`--no-draft` flags to control release visibility
* Implemented HTTPS authentication using GITHUB_TOKEN for tag pushing
* Fixed option parsing in `script/release` (extract version before parsing)
* Updated Go version to 1.25.2 across all configs
* Successfully created multiple automated draft releases in CircleCI
* Git authentication configured to use token instead of read-only SSH key
* **Removed GitHub Actions**: Not using GitHub Actions, sticking with CircleCI
* **Updated Phase 4**: Now focuses on validating existing draft releases
* **Updated Phase 5**: Focuses on production releases and Homebrew distribution
* **Clarified Release Strategy**:
  - Test releases: Automated via CI on `test-releases/*` branches
  - Production releases: Manual from main branch for full control
  - Changed branch filter from `release-test.*` to `test-releases.*`

### Previous Updates (2025-10-09)
* **Phase 2 Complete**: Integrated PR-based changelog with GoReleaser
* Enhanced `script/release` to use GoReleaser internally (same UX, professional output)
* Added `--extract-notes` flag for CI/automation use
* Disabled GoReleaser auto-changelog (we provide our own from PR tracking)
* Maintained backward compatibility - same command `script/release v0.x.x`
* Updated `.gitignore` to exclude `.goreleaser-notes.md` temp file

### Previous Updates (2025-08-27)
* Created placeholder LICENSE file (pending OSS license selection)
* Modified `.goreleaser.yml` to copy parent directory files via before hooks
* Updated `.gitignore` in plur/ to exclude copied documentation files
* Updated CircleCI config to use official Go orb's `install-goreleaser` command (v3.0.3)
* Verified successful snapshot builds with all documentation files included