# Release Process

## Quick Start

```bash
# Create a new release
script/release v0.7.0
```

## Prerequisites

- GitHub CLI (`gh`) installed and authenticated
- Clean working directory on `main` branch
- Push access to the repository

## Examples

```bash
# Patch release (bug fixes)
script/release v0.7.1

# Minor release (new features)
script/release v0.8.0

# Major release (breaking changes)
script/release v1.0.0
```

## What Happens

1. Verifies prerequisites
2. Builds rux to ensure it compiles
3. Updates CHANGELOG.md with PRs since last release
4. Shows summary and asks for confirmation
5. Creates git tag and GitHub release

## Manual Release

If automation fails:

```bash
# Update CHANGELOG.md manually, then:
git commit -am "Update CHANGELOG for vX.Y.Z"
git tag -a vX.Y.Z -m "Release vX.Y.Z"
git push origin main --tags
gh release create vX.Y.Z --title vX.Y.Z --notes "See CHANGELOG.md"
```