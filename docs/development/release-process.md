# Release Process

How to do a release of Plur.

## Overview

Plur releases use `script/release` to prepare changelog entries, validate release state, create and push a version tag, and extract release notes for CI. The actual published release is built by GitHub Actions and GoReleaser after the tag is pushed.

## Prerequisites

* GitHub CLI (`gh`) installed and authenticated.
* Push access to `rsanheim/plur`.
* An up-to-date `main` branch.
* A clean working directory before `script/release push`.

## Version Format

Versions must be semver with a `v` prefix, i.e. `vX.Y.Z` or `vX.Y.Z-rc.1`.

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

Pushing a `v*` tag starts `.github/workflows/release.yml`. The workflow handles the goreleaser build, GH release, archives, and [homebrew formula update](https://github.com/rsanheim/homebrew-tap).

## Known GoReleaser Warning

GoReleaser currently warns that the `brews` configuration is deprecated. We intentionally ignore this warning. Formulas are fine for CLIs, and casks just add complexity.

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
