# Release Process

This document describes how to create a new release of rux.

## Prerequisites

1. You must have the GitHub CLI (`gh`) installed and authenticated
2. You must be on the `main` branch with a clean working directory
3. You must have push access to the repository

## Creating a Release

To create a new release:

```bash
script/release v0.7.0
```

## Release Steps

The release script will:

1. **Verify prerequisites**
   - Ensure you're on the `main` branch
   - Ensure git working directory is clean
   - Verify new version is greater than current version

2. **Build verification**
   - Build rux to ensure it compiles without errors

3. **Update changelog**
   - Find all PRs merged since the last release
   - Add them to CHANGELOG.md with the new version and date

4. **Confirm release**
   - Show a summary of changes
   - Ask for confirmation before proceeding

5. **Perform release**
   - Commit changelog updates
   - Create and push git tag
   - Rebuild rux with the new version tag
   - Push to GitHub
   - Create GitHub release

## Versioning

We use semantic versioning (vX.Y.Z):

- **MAJOR** (X): Breaking changes
- **MINOR** (Y): New features, backwards compatible
- **PATCH** (Z): Bug fixes, backwards compatible

## Useful Commands

```bash
# Show current version
bin/rake release:version

# Dry run - see what would be released
bin/rake release:dry_run

# Test the release system without releasing
script/test-release
```

## Manual Release (if automation fails)

If the automated release fails, you can complete it manually:

1. Update CHANGELOG.md with version and PRs
2. Commit: `git commit -am "Update CHANGELOG for vX.Y.Z"`
3. Tag: `git tag -a vX.Y.Z -m "Release vX.Y.Z"`
4. Push: `git push origin main --tags`
5. Create GitHub release: `gh release create vX.Y.Z --title vX.Y.Z --notes "See CHANGELOG.md"`

## Troubleshooting

- **"Not on main branch"**: Switch to main with `git checkout main`
- **"Unclean working directory"**: Commit or stash changes
- **"Version not greater"**: Ensure new version > current version
- **"Build failed"**: Fix compilation errors in rux/
- **"gh command not found"**: Install GitHub CLI from https://cli.github.com/