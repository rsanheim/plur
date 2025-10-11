# Release Infrastructure

This directory contains documentation for plur's release infrastructure, which combines GoReleaser with custom Ruby scripting to provide both professional Go releases and developer-friendly ergonomics.

## Documentation

### Process & Usage
- [Release Process](../release-process.md) - How to create releases (manual and automated)

### Implementation
- [GoReleaser Implementation Checklist](./goreleaser-checklist.md) - Complete implementation tracking with current status
- [GoReleaser Implementation Summary](./goreleaser-implementation-summary.md) - Technical summary of what was built
- [GoReleaser PRD](./goreleaser-prd.md) - Original product requirements document

## Current Status

As of October 2025, the release infrastructure is fully operational with:

✅ **Completed Features**
- Multi-platform binary builds (macOS ARM64, Linux x86_64/ARM64, Windows experimental)
- Automated releases via CircleCI
- Draft/published release control
- CHANGELOG generation from PR history
- GoReleaser integration with custom wrapper script
- HTTPS authentication for CI/CD tag pushing

📊 **Recent Releases**
- Multiple successful test releases (v0.10.4-*-test series)
- Automated draft releases working in CircleCI
- Full release pipeline tested and verified

## Quick Reference

### Production Releases (from main branch)
```bash
# Interactive draft release (default)
git checkout main
script/release v1.0.0

# Publish immediately
script/release v1.0.0 --no-draft
```

### Test Releases (CI validation only)
Test releases run automatically on `test-releases/*` branches:
```bash
# Create test branch
git checkout -b test-releases/validate-binaries
git push origin test-releases/validate-binaries

# CI automatically creates:
# v0.10.4-{BUILD_NUMBER}-test (draft)
```

**Note**: Production releases are always manual from main branch for full control. CI automation is only for test release validation.

## Architecture

The release system combines:
1. **script/release** - Ruby wrapper script providing UX and orchestration
2. **lib/plur/release.rb** - Ruby class handling changelog, git operations
3. **GoReleaser** - Industry-standard Go release tool for builds and artifacts
4. **CircleCI** - CI/CD platform for automated releases

This hybrid approach provides the best of both worlds:
- Professional Go ecosystem standards and artifacts
- Friendly, Ruby-like developer experience
- Flexibility to run locally or in CI/CD
- Smooth path to open source release