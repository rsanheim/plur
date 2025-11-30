# Analysis: Moving Go Code to Top Level

## Overview

This document analyzes the feasibility and trade-offs of moving the Go code from the `plur/` subdirectory to the repository root, effectively converting this from a "meta" repository to a standard Go repository.

## Current Structure

The repository is currently organized as a polyglot project:

```
plur/
├── bin/              # Ruby binstubs
├── lib/              # Ruby rake tasks
├── spec/             # Ruby integration tests  
├── fixtures/         # Test fixtures
├── docs/             # Documentation
├── plur/              # Go source code (self-contained)
│   ├── go.mod
│   ├── main.go
│   ├── logger/
│   ├── rspec/
│   └── watch/
├── Rakefile          # Build orchestration
├── Gemfile           # Ruby dependencies
└── plur.rb           # Build configuration
```

## Key Conflicts with Moving to Top Level

### 1. Directory Name Conflicts
- `spec/` exists at both levels (Ruby integration tests vs Go unit tests)
- `tmp/` exists at both levels  
- `vendor/` conflicts between Ruby and Go conventions

### 2. Mixed Language Complexity
- Standard Go tools expect Go files at repository root
- Ruby tools expect their structure (Gemfile, spec/, lib/)
- No clear convention for organizing polyglot projects

### 3. Build System Dependencies
- All rake tasks hardcode `plur/` paths
- `plur.rb` configuration assumes nested structure
- Integration tests depend on specific binary locations

## Trade-offs Analysis

### Pros of Moving to Top Level

1. **Standard Go Layout**: Aligns with Go community expectations
2. **Simpler Import Paths**: No subdirectory in module path
3. **Better Go Tooling Support**: Some tools work better with standard layout
4. **Clearer Identity**: Repository becomes clearly a "Go project"

### Cons of Moving to Top Level

1. **Root Directory Clutter**: Mixes Go source files with Ruby tooling
2. **Loss of Separation**: Currently, Go code is cleanly isolated
3. **Significant Refactoring**: All build scripts need rewriting
4. **Harder Distribution**: Can't easily extract just the Go code
5. **Testing Complexity**: Ruby and Go test files intermixed

## Packaging Implications

### Current Structure Benefits
- Can package `plur/` directory as standalone Go module
- Clear boundary between implementation (Go) and testing framework (Ruby)
- Easy to vendor or distribute just the Go code

### Top-Level Structure Impacts
- Entire repository becomes the Go module
- Harder to separate concerns for distribution
- Ruby dependencies become part of the Go project
- More complex `.gitignore` and build configurations

## Recommendation

**Keep the current structure**. The existing layout provides good separation of concerns:

1. **Go code is self-contained**: The `plur/` directory is a complete Go module
2. **Ruby provides testing infrastructure**: Integration tests and tooling live outside
3. **Clean boundaries**: Easy to understand what's implementation vs tooling
4. **Distribution-friendly**: Can package just the Go code when needed

If we rename to "plur", the repository could be:
- `plur` - Just the Go code (move `plur/` contents here)
- `plur-tools` - Keep a separate repo for integration testing and tooling

This would give us the best of both worlds: a clean Go repository and a separate testing/tooling repository.

## Alternative: Monorepo Structure

If keeping everything together, consider a more explicit monorepo structure:

```
plur/
├── go/               # All Go code
│   ├── go.mod
│   └── ...
├── ruby/             # All Ruby code
│   ├── Gemfile
│   └── ...
├── docs/
└── scripts/          # Shared build scripts
```

This makes the polyglot nature explicit while keeping clear boundaries.