# Release Process

This is the canonical release runbook for Plur. Keep release procedure details here, and point other docs or command help back to this page instead of duplicating the workflow.

## Quick Start

```bash
# Start from an up-to-date main branch
git checkout main
git pull --ff-only origin main

# Preview and prepare the changelog entry
script/release prepare v0.61.0 --dry-run
script/release prepare v0.61.0

# Review/edit CHANGELOG.md, then verify and commit it
bin/rake
git add CHANGELOG.md
git commit -m "Changelog for v0.61.0"

# Push main, create the annotated tag, and trigger GitHub Actions
script/release push v0.61.0
```

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

## Commands

### `script/release prepare VERSION`

Generates a changelog entry by finding PRs merged since the latest GitHub release:

```bash
script/release prepare v0.61.0
```

The command verifies local `main` matches `origin/main`, reads the latest release tag from GitHub, scans commit messages since that tag for PR numbers, fetches PR titles and URLs with `gh`, and updates `CHANGELOG.md`.

Review the generated changelog before committing. Remove duplicate or noisy entries, add important user-facing context, and keep `## Unreleased` at the top.

### `script/release push VERSION`

Pushes `main`, creates an annotated tag, and pushes the tag:

```bash
script/release push v0.61.0
```

This command:

1. Verifies the current branch is `main`.
2. Verifies git status is clean.
3. Verifies `CHANGELOG.md` has an entry for the version.
4. Pushes `main` to `origin`.
5. Creates an annotated git tag.
6. Pushes the tag to `origin`.

The tag push triggers the active release workflow: `.github/workflows/release.yml`.

### `script/release extract-notes VERSION`

Extracts release notes from `CHANGELOG.md`:

```bash
script/release extract-notes v0.61.0
```

This command is used by GitHub Actions and is useful for checking exactly what release notes GoReleaser will receive.

### `--dry-run`

Use `--dry-run` to see what a command would do without writing the changelog or creating/pushing tags:

```bash
script/release prepare v0.61.0 --dry-run
script/release push v0.61.0 --dry-run
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

GoReleaser uses `.goreleaser.yml`. That file is release configuration, not a second runbook; update this page when the operator-facing release process changes.

## Known GoReleaser Warnings

GoReleaser currently warns that the `brews` configuration is deprecated. We intentionally ignore this warning.

For Plur, a Homebrew formula remains the simpler and preferred distribution path for a CLI binary installed from release archives. Casks are not a better fit for this release shape. Keep the formula-based `brews` setup unless GoReleaser removes support entirely or Homebrew's CLI packaging guidance changes.

## Platform Artifacts

Each release includes binaries for:

* macOS ARM64 (Apple Silicon)
* Linux x86_64
* Linux ARM64
* Windows x86_64 (experimental)

Archives include `README.md`, `LICENSE`, and `CHANGELOG.md`. Each release also includes SHA256 checksums.

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

For the shell installer, pin the version being verified:

```bash
curl -fsSL https://github.com/rsanheim/plur/raw/main/install.sh | PLUR_VERSION=v0.61.0 sh
plur --version
```

## Manual Release (Emergency)

If automation fails:

```bash
# Update CHANGELOG.md manually, then:
git add CHANGELOG.md
git commit -m "Changelog for vX.Y.Z"
git tag -a vX.Y.Z -m "Release vX.Y.Z"
git push origin main --tags

# GitHub Actions will handle the rest
# Or run GoReleaser locally:
goreleaser release --clean
```
