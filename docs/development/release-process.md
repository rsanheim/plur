# Release Process

## Quick Start

```bash
# Create a new release (interactive, creates draft)
script/release v0.7.0

# Create and publish immediately
script/release v0.7.0 --no-draft

# Automated release (for CI/CD)
script/release v0.7.0 --automated --draft
```

## Prerequisites

- GitHub CLI (`gh`) installed and authenticated
- GoReleaser installed (`brew install goreleaser`)
- Clean working directory on `main` branch
- Push access to the repository
- GITHUB_TOKEN for automated releases (in CI)

## Release Options

### Manual Releases (Developer Machine)

```bash
# Standard release (interactive, creates draft)
script/release v0.7.1

# Publish immediately (no draft)
script/release v1.0.0 --no-draft

# Extract changelog notes only
script/release v0.7.0 --extract-notes
```

### Automated Releases (CI/CD)

```bash
# Automated draft release (no prompts)
script/release v0.7.0 --automated --draft

# Automated published release
script/release v0.7.0 --automated --no-draft

# CI example with build number
script/release v0.10.4-$BUILD_NUM-test --automated --draft
```

## What Happens

### Interactive Mode (default)
1. Verifies prerequisites
2. Builds plur to ensure it compiles
3. Updates CHANGELOG.md with PRs since last release
4. Shows summary and **asks for confirmation**
5. Commits changelog updates
6. Creates git tag
7. Pushes tag to GitHub
8. Runs GoReleaser to:
   - Build multi-platform binaries
   - Generate checksums
   - Create GitHub release with artifacts
   - Upload release notes from CHANGELOG

### Automated Mode (`--automated`)
Same as above but:
- **Skips confirmation prompt**
- Logs all actions
- Suitable for CI/CD pipelines

## Release Workflows

### Production Releases (from main branch)
Production releases are created manually from the main branch:

```bash
# On main branch, create a production release
git checkout main
git pull origin main

# Interactive release (creates draft for review)
script/release v0.11.0

# Or publish immediately
script/release v0.11.0 --no-draft
```

### Test Releases (CI validation)
Test releases are automated via CircleCI on `test-releases/*` branches:

1. Create a test branch:
   ```bash
   git checkout -b test-releases/validate-v0.11
   git push origin test-releases/validate-v0.11
   ```

2. CircleCI automatically creates test releases with format:
   ```
   v0.10.4-{BUILD_NUMBER}-test
   ```

3. These are **draft releases** used for:
   - Validating the release pipeline
   - Testing multi-platform binaries
   - Verifying GoReleaser configuration
   - **NOT for production use**

### CircleCI Configuration
```yaml
workflows:
  release:
    jobs:
      - release:
          context: github-releases
          filters:
            branches:
              only: /test-releases.*/  # Test branches only
```

### Required Environment Variables
- `GITHUB_TOKEN`: Personal access token with repo permissions
- Must be stored in CircleCI context or project settings

## Release Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--automated` | Skip confirmation prompts (for CI) | Interactive |
| `--draft` | Create draft release | Yes |
| `--no-draft` | Publish immediately | No |
| `--extract-notes` | Only extract changelog notes | No |

## Manual Release (Emergency)

If automation fails:

```bash
# Update CHANGELOG.md manually, then:
git commit -am "Update CHANGELOG for vX.Y.Z"
git tag -a vX.Y.Z -m "Release vX.Y.Z"
git push origin main --tags

# GoReleaser will use existing tag
cd plur
goreleaser release --clean

# Or create GitHub release manually
gh release create vX.Y.Z --title vX.Y.Z --notes "See CHANGELOG.md"
```

## Platform Artifacts

Each release includes binaries for:
- macOS ARM64 (Apple Silicon)
- Linux x86_64
- Linux ARM64
- Windows x86_64 (experimental)

All artifacts include:
- SHA256 checksums
- README, LICENSE, and CHANGELOG
- Compressed archives (tar.gz for Unix, zip for Windows)