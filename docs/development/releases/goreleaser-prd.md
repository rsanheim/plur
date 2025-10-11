# PRD: Implement GoReleaser for plur

> **Status**: ✅ IMPLEMENTED (October 2025)
>
> This PRD has been successfully implemented. See [goreleaser-checklist.md](./goreleaser-checklist.md) for implementation details and [release-process.md](../release-process.md) for usage documentation.

## Executive Summary
Implement GoReleaser to establish a standard Go release pipeline that will enable a smooth transition to open source while preserving the developer-friendly experience like our current `script/release` workflow. This creates a flexible, industry-standard foundation that can run locally during private development and seamlessly transition to GitHub Actions when public, all while maintaining the simple UX that makes releasing effortless.

## Goals
- **Enable seamless OSS transition** - Build release infrastructure that works identically whether private or public
- **Adopt Go ecosystem standards** - Align with how successful Go projects like Hugo, Terraform, and esbuild handle releases
- **Create location-agnostic releases** - Can build locally or on GHA
- **Preserve developer-friendly UX** - Keep `script/release` as the simple, human-friendly interface that wraps the complex machinery
- **Establish professional distribution** - Match the quality and polish expected from modern Go CLI tools
- **Go standard tagging / building** - Use GoReleaser's tagging and building capabilities to build the standard things people expect from Go binaries

## Problem Statement
While `script/release` provides an excellent developer experience, the underlying implementation needs to evolve to Go ecosystem standards before going public. We need the industrial-strength capabilities of GoReleaser (cross-compilation, checksums, Homebrew formulas) wrapped in the simplicity that makes our current process delightful to use. The solution must work identically whether run on a developer's laptop today or in GitHub Actions tomorrow.

## Success Metrics
- Developer still has a simple single script to release (we can wrap with ruby like script/release if necessary)
- Identical release artifacts whether built locally or in CI
- Follows Go release patterns recognized by the community  
- Zero configuration changes needed when transitioning to public
- Release process remains under 5 minutes with single command

## Implementation Philosophy

### The Best of Both Worlds
- **GoReleaser**: Industry-standard heavy lifting (builds, artifacts, distribution)
- **script/release**: Developer-friendly orchestration and UX
- **Result**: Professional Go releases with Ruby-like developer ergonomics

## Implementation Phases

### Phase 1: Local GoReleaser Setup 
**Goal:** Establish Go-standard release pipeline that runs locally

- Configure GoReleaser following Go ecosystem conventions
- Ensure all configuration is portable (no hardcoded paths/secrets)
- Test the full pipeline locally with snapshot releases
- Validate that artifacts match professional Go project standards
- Add a circleci task to build artifacts and test on appropriate platforms

### Phase 2: Enhanced Developer Experience
**Goal:** Wrap GoReleaser with familiar, friendly tooling

- Enhance `script/release` to orchestrate GoReleaser
- Hide complexity while exposing useful options
- Provide clear feedback and progress indicators
- Handle common errors with helpful messages
- Keep the "it just works" feeling

### Phase 3: CI/CD Readiness (Pre-Public)
**Goal:** Ensure seamless transition to automated releases

- Create GitHub Actions workflow (initially disabled)
- Test that local and CI builds produce identical artifacts
- Verify the same commands work in both environments
- Prepare for authenticated operations (Homebrew tap updates)
- Document the "flip the switch" moment for going public

### Phase 4: Public Launch Activation
**Goal:** Go live with professional, automated distribution

- Enable GitHub Actions on public repo
- Activate Homebrew formula generation
- Enable changelog publication
- Monitor and iterate based on user feedback

## Technical Requirements

### Portability First
- Configuration must work unchanged locally and in CI
- No environment-specific assumptions
- Secrets handled through environment variables
- Relative paths only

### Go Ecosystem Alignment  
- Follow naming conventions from successful Go projects
- Use standard artifact formats expected by Go developers
- Include standard files (README, LICENSE, completions)
- Generate checksums and signatures as expected

### Developer Experience
- Single command releases remain the default
- Clear progress output and error messages  
- Graceful fallbacks when tools are missing
- Helper scripts for common operations
- Excellent documentation for contributors

## Migration Strategy

### Keep What Developers Love
- `script/release` can remain the primary interface, at least to start
- Changelog generation stays PR-based
- Version tags keep current format
- No breaking changes to workflow

### Add What Go Ecosystem Expects
- Professional artifact naming
- Multi-platform binary distribution
- Checksums
- Homebrew formula generation
- Standard Go project structure

### Enable Gradual Adoption
- Start with GoReleaser in snapshot mode
- Run parallel to verify compatibility
- Gradually migrate responsibilities
- Always maintain local-first capability

## Why This Matters

When plur goes public, the release process becomes the project's first impression of professionalism. Go developers expect certain standards - predictable artifacts, checksums, easy installation. Ruby developers expect simplicity and great UX. This approach delivers both: a release process that feels as friendly as Ruby tooling while meeting the technical standards of the Go ecosystem.

The ability to run the same release process locally during development and in CI when public means contributors can test and understand the full pipeline without special access. This democratizes the release process and builds contributor confidence.