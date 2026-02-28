# Release Process

## Quick Start

```bash
# 1. Prepare release (generates changelog)
script/release prepare v0.14.0

# 2. Review and edit CHANGELOG.md if needed

# 3. Commit changelog
git add CHANGELOG.md && git commit -m "Changelog for v0.14.0"

# 4. Push release (tags and triggers GitHub Actions)
script/release push v0.14.0
```

## Prerequisites

* GitHub CLI (`gh`) installed and authenticated
* Clean working directory on `main` branch
* Push access to the repository

## Commands

### `script/release prepare VERSION`

Generates changelog entries by finding PRs merged since the last release:

```bash
script/release prepare v0.14.0
```

This updates `CHANGELOG.md` with PR titles and links. Review and edit as needed before committing.

### `script/release push VERSION`

Tags and pushes to trigger the release:

```bash
script/release push v0.14.0
```

This command:
1. Verifies you're on `main` branch
2. Verifies git status is clean
3. Verifies changelog has entry for version
4. Creates annotated git tag
5. Pushes tag to origin

Pushing the tag triggers GitHub Actions to run GoReleaser.

### `script/release extract-notes VERSION`

Extracts release notes from CHANGELOG.md (used by CI):

```bash
script/release extract-notes v0.14.0
```

### `--dry-run` Flag

All commands support `--dry-run` to see what would happen without executing:

```bash
script/release prepare v0.14.0 --dry-run
script/release push v0.14.0 --dry-run
```

## What Happens After Push

GitHub Actions (`.github/workflows/release.yml`) automatically:

1. Extracts release notes from CHANGELOG.md
2. Runs GoReleaser to build multi-platform binaries
3. Creates GitHub release with artifacts

## Platform Artifacts

Each release includes binaries for:

* macOS ARM64 (Apple Silicon)
* Linux x86_64
* Linux ARM64
* Windows x86_64 (experimental)

All artifacts include SHA256 checksums, README, LICENSE, and CHANGELOG.

## Manual Release (Emergency)

If automation fails:

```bash
# Update CHANGELOG.md manually, then:
git commit -am "Update CHANGELOG for vX.Y.Z"
git tag -a vX.Y.Z -m "Release vX.Y.Z"
git push origin main --tags

# GitHub Actions will handle the rest
# Or run GoReleaser locally:
goreleaser release --clean
```

## Version Format

Versions must be semver with `v` prefix: `vX.Y.Z` (e.g., `v0.14.0`, `v1.0.0-rc1`)
