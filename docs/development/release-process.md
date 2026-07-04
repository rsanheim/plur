# Release Process

This is the operator-facing runbook for a full Plur release. Use `script/release --help` for command-level reference; use this page for the end-to-end process.

## Overview

Plur releases use `script/release` to prepare changelog entries, validate release state, create and push a version tag, and extract release notes for CI. The actual published release is built by GitHub Actions and GoReleaser after the tag is pushed.

## Prerequisites

* GitHub CLI (`gh`) installed and authenticated.
* Push access to `rsanheim/plur`.
* An up-to-date `main` branch.
* A clean working directory before `script/release push`.
* Release workflow secrets configured in GitHub Actions:
  * `GITHUB_TOKEN` is provided by GitHub Actions.
  * `TAP_GITHUB_TOKEN` updates `rsanheim/homebrew-tap` for non-prerelease tags.

## Version Format

Versions must be semver with a `v` prefix:

```text
vX.Y.Z
vX.Y.Z-rc.1
```

## Release Steps

```bash
# Start from an up-to-date main branch
git checkout main
git pull --ff-only origin main

# Preview and prepare the changelog entry
script/release prepare v0.61.0 --dry-run
script/release prepare v0.61.0

# Review/edit CHANGELOG.md (keep "## Unreleased" at the top), then verify and commit it
bin/rake
git add CHANGELOG.md
git commit -m "Changelog for v0.61.0"

# Push main, create the annotated tag, and trigger GitHub Actions
script/release push v0.61.0
```

## Automated Release Workflow

Pushing a `v*` tag starts `.github/workflows/release.yml`. The workflow:

1. Checks out the full git history.
2. Sets up Go and Ruby.
3. Runs `bundle exec script/release extract-notes VERSION`.
4. Runs `goreleaser release --clean --release-notes=/tmp/notes.md`.
5. Creates or replaces the GitHub release.
6. Uploads release archives and checksums.
7. Updates the Homebrew formula in `rsanheim/homebrew-tap` for non-prerelease tags.

GoReleaser uses `.goreleaser.yml`. That file is release configuration, not the operator runbook.

## Known GoReleaser Warning

GoReleaser currently warns that the `brews` configuration is deprecated. We intentionally ignore this warning.

For Plur, a Homebrew formula remains the simpler and preferred distribution path for a CLI binary installed from release archives. Casks are not a better fit for this release shape. Keep the formula-based `brews` setup unless GoReleaser removes support entirely or Homebrew's CLI packaging guidance changes.

## Platform Artifacts

Each release includes:

* macOS ARM64 (Apple Silicon)
* Linux x86_64
* Linux ARM64
* Windows x86_64 (experimental)
* SHA256 checksums

Archives include `README.md`, `LICENSE`, and `CHANGELOG.md`.

## Verification

After `script/release push VERSION`:

1. Watch the [Release workflow](https://github.com/rsanheim/plur/actions/workflows/release.yml).
2. Confirm the [GitHub release](https://github.com/rsanheim/plur/releases) exists and its notes match `script/release extract-notes VERSION`.
3. Confirm the release contains archives for the supported platforms plus the checksums file.
4. For a stable release, confirm `rsanheim/homebrew-tap` updated `Formula/plur.rb`.
5. Install the release and verify the binary:

```bash
brew upgrade rsanheim/tap/plur
plur --version
plur doctor
```

To verify the shell installer:

```bash
curl -fsSL https://github.com/rsanheim/plur/raw/main/install.sh | PLUR_VERSION=v0.61.0 sh
plur --version
```

## Manual Release

If automation fails:

```bash
git add CHANGELOG.md
git commit -m "Changelog for vX.Y.Z"
git tag -a vX.Y.Z -m "Release vX.Y.Z"
git push origin main --tags

# Or run GoReleaser locally:
goreleaser release --clean
```
