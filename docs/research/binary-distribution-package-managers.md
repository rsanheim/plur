# Binary Distribution and Package Manager Research for plur

Date: October 2025
Author: Rob Sanheim

## Executive Summary

This document presents research findings on distributing plur binaries through package managers, particularly Homebrew, using GoReleaser's automation capabilities. With GoReleaser already configured for releases, adding package manager support is a natural next step that will significantly improve user adoption and installation experience.

## Current Landscape of Go CLI Distribution (2025)

### Popular Go CLIs and Their Distribution Methods

| Project | Homebrew | Scoop | Snap | AUR | Custom Tap | Notes |
|---------|----------|--------|------|-----|------------|-------|
| **GitHub CLI (gh)** | ✓ (core) | ✓ | ✓ | ✓ | No | Uses GoReleaser, in homebrew-core |
| **Lazygit** | ✓ (core) | ✓ | ✓ | ✓ | No | Uses GoReleaser, wide platform support |
| **GoReleaser** | ✓ (cask) | ✓ | ✓ | ✓ | ✓ | Own tap: goreleaser/tap |
| **Buf** | ✓ | ✓ | - | - | ✓ | Protocol buffer tooling |
| **Cobra** | - | - | - | - | - | Library, not distributed as binary |

### Key Findings

1. **Homebrew Strategy**: Projects typically follow one of two paths:
   * **Homebrew Core**: Established projects (gh, lazygit) get accepted into homebrew-core
   * **Custom Tap**: New/smaller projects maintain their own tap for immediate control

2. **GoReleaser Adoption**: Most successful Go CLIs use GoReleaser for automation
   * Handles multi-platform builds
   * Automates formula/package generation
   * Integrates with CI/CD pipelines

3. **Multi-Platform Support**: Successful projects support 3-5 package managers
   * macOS: Homebrew (primary)
   * Windows: Scoop, Chocolatey
   * Linux: apt/yum via nFPM, Snap, AUR

## GoReleaser's Package Manager Capabilities

### Homebrew Support

GoReleaser provides first-class Homebrew support through the `brews` configuration:

**Features:**
* Automatic formula generation
* SHA256 checksum inclusion
* Version management with tags
* Support for versioned formulas (foo@1.0)
* Cross-repository publishing

**Configuration Example:**
```yaml
brews:
  - repository:
      owner: rsanheim
      name: homebrew-tap
      token: "{{ .Env.GITHUB_TOKEN }}"
    folder: Formula
    homepage: https://github.com/rsanheim/plur
    description: Fast parallel test runner for Ruby/RSpec
    license: MIT
    commit_author:
      name: goreleaserbot
      email: bot@goreleaser.com
    test: |
      system "#{bin}/plur", "--version"
    install: |
      bin.install "plur"
```

### Windows Package Managers

**Scoop:**
```yaml
scoops:
  - repository:
      owner: rsanheim
      name: scoop-bucket
    folder: bucket
    homepage: https://github.com/rsanheim/plur
    description: Fast parallel test runner for Ruby/RSpec
    license: MIT
```

**WinGet:**
```yaml
winget:
  - name: plur
    publisher: rsanheim
    license: MIT
    homepage: https://github.com/rsanheim/plur
    short_description: Fast parallel test runner for Ruby/RSpec
    repository:
      owner: rsanheim
      name: winget-pkgs
      branch: "{{.ProjectName}}-{{.Version}}"
      pull_request:
        enabled: true
        base:
          owner: microsoft
          name: winget-pkgs
          branch: master
```

### Linux Package Managers

**nFPM (deb, rpm, apk):**
```yaml
nfpms:
  - id: plur
    package_name: plur
    vendor: rsanheim
    homepage: https://github.com/rsanheim/plur
    maintainer: Rob Sanheim <rsanheim@gmail.com>
    description: Fast parallel test runner for Ruby/RSpec
    license: MIT
    formats:
      - deb
      - rpm
      - apk
    bindir: /usr/bin
```

**Snapcraft:**
```yaml
snapcrafts:
  - name: plur
    summary: Fast parallel test runner for Ruby/RSpec
    description: |
      plur is a high-performance parallel test runner for Ruby/RSpec
      that significantly speeds up test suite execution.
    grade: stable
    confinement: classic
    license: MIT
    apps:
      plur:
        command: plur
```

**AUR (Arch Linux):**
```yaml
aurs:
  - name: plur-bin
    homepage: https://github.com/rsanheim/plur
    description: Fast parallel test runner for Ruby/RSpec
    maintainers:
      - Rob Sanheim <rsanheim@gmail.com>
    license:
      - MIT
    private_key: "{{ .Env.AUR_KEY }}"
    git_url: ssh://aur@aur.archlinux.org/plur-bin.git
```

## Automation Strategies

### GitHub Actions Integration

**Required Secrets:**
* `GITHUB_TOKEN`: For GitHub releases (default available)
* `TAP_GITHUB_TOKEN`: For cross-repo tap updates
* `AUR_KEY`: For AUR publishing (if enabled)

**Workflow Example:**
```yaml
name: Release
on:
  push:
    tags:
      - 'v*'
jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
      - name: Run GoReleaser
        uses: goreleaser/goreleaser-action@v5
        with:
          version: latest
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          TAP_GITHUB_TOKEN: ${{ secrets.TAP_GITHUB_TOKEN }}
```

### Token Requirements

For cross-repository publishing (e.g., to homebrew-tap):
1. Create a Personal Access Token with:
   * `repo` scope (full control of private repositories)
   * `workflow` scope (if updating GitHub Actions)
2. Add as repository secret (`TAP_GITHUB_TOKEN`)
3. Reference in GoReleaser config: `token: "{{ .Env.TAP_GITHUB_TOKEN }}"`

## Recommendations for plur

### Phase 1: Homebrew Tap (Immediate)

1. **Create Repository**: `github.com/rsanheim/homebrew-tap`
2. **Enable Configuration**: Uncomment and refine the existing Homebrew configuration in `.goreleaser.yml`
3. **Test Release**: Create a test release to verify tap updates

**Benefits:**
* Immediate availability to macOS users
* Full control over formula
* No approval process required

### Phase 2: Additional Package Managers (Next Release)

**Priority Order:**
1. **Scoop** (Windows) - Large Windows developer audience
2. **nFPM** (Linux) - Covers deb/rpm ecosystems
3. **AUR** (Arch) - Active Ruby developer community

**Optional:**
* Snapcraft - Requires additional setup but provides auto-updates
* Chocolatey - Alternative Windows option

### Phase 3: Homebrew Core (Future)

Once plur gains traction:
1. Submit formula to homebrew-core
2. Requires meeting quality/popularity thresholds
3. Provides broader visibility and trust

## Installation Experience Comparison

### Current (Manual Download):
```bash
# Download binary
curl -L https://github.com/rsanheim/plur/releases/download/v1.0.0/plur_1.0.0_Darwin_arm64.tar.gz | tar xz
# Move to PATH
sudo mv plur /usr/local/bin/
```

### With Homebrew Tap:
```bash
brew tap rsanheim/tap
brew install plur
```

### Future (Homebrew Core):
```bash
brew install plur
```

## Best Practices

### Repository Naming
* Homebrew taps should start with `homebrew-` (e.g., `homebrew-tap`)
* Scoop buckets conventionally use `scoop-` prefix
* This enables shorter tap commands: `brew tap rsanheim/tap`

### Version Management
* Use semantic versioning consistently
* GoReleaser supports versioned formulas (plur@1.0)
* Allows users to pin specific versions

### Testing
* Include test blocks in package configurations
* Minimum: version check (`plur --version`)
* Consider: basic functionality test

### Documentation
* Update README with installation instructions for each platform
* Create dedicated installation guide
* Include troubleshooting section

## Security Considerations

### Binary Verification
* GoReleaser generates SHA256 checksums automatically
* Consider implementing:
  * GPG signing for releases
  * SLSA provenance (like GitHub CLI)
  * Sigstore/cosign integration

### Token Security
* Use repository secrets, never commit tokens
* Rotate tokens periodically
* Use fine-grained permissions where possible

## Examples from the Wild

### GitHub CLI's Approach
* Started with custom tap
* Graduated to homebrew-core
* Maintains wide platform support
* Uses SLSA attestation for security

### Lazygit's Success
* Direct inclusion in homebrew-core
* Supports 10+ package managers
* Simple GoReleaser configuration
* Focus on user experience

### GoReleaser's Own Distribution
* Maintains custom tap (goreleaser/tap)
* Distributed as Homebrew cask
* Comprehensive platform coverage
* Dog-foods own features

## Conclusion

GoReleaser provides robust support for package manager distribution, making it straightforward to expand plur's installation options. Starting with a Homebrew tap offers immediate benefits with minimal complexity, while the infrastructure supports future expansion to other platforms.

The existing `.goreleaser.yml` already includes commented Homebrew configuration, indicating this was planned. With the release automation now in place, enabling package manager distribution is the logical next step to improve user adoption and experience.

## Resources

* [GoReleaser Homebrew Documentation](https://goreleaser.com/customization/homebrew/)
* [GoReleaser nFPM Documentation](https://goreleaser.com/customization/nfpm/)
* [GoReleaser Scoop Documentation](https://goreleaser.com/customization/scoop/)
* [GitHub CLI .goreleaser.yml](https://github.com/cli/cli/blob/trunk/.goreleaser.yml)
* [Lazygit .goreleaser.yml](https://github.com/jesseduffield/lazygit/blob/master/.goreleaser.yml)
* [Creating Homebrew Formulas with GoReleaser](https://bindplane.com/blog/creating-homebrew-formulas-with-goreleaser)
* [Distribute Go CLI with GoReleaser and Homebrew](https://dev.to/aurelievache/learning-go-by-examples-part-9-use-homebrew-goreleaser-for-distributing-a-golang-app-44ae)