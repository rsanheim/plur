# Go File Pattern Matching Libraries Research

## Executive Summary

After researching several Go libraries for file pattern matching to replace plur's hand-rolled implementation in `plur watch`, **doublestar (github.com/bmatcuk/doublestar)** emerges as the clear winner. It offers excellent globstar support, is actively maintained, performant, and maps well to TOML configuration.

## Requirements

Based on the analysis in `docs/wip/consolidate-watch-and-spec-report.md` and `docs/architecture/file-mapping.md`, we need:

* **Globstar pattern matching** (e.g., `**/*_spec.rb`)
* **Easy testability**
* **Fast performance**
* **Active maintenance** (as of August 2025)
* **Clean API** that maps well to TOML configuration

## Library Comparison

### 1. doublestar (github.com/bmatcuk/doublestar) ⭐ RECOMMENDED

**Version**: v4.9.1 (July 2024)  
**Maintenance**: Actively maintained  
**GitHub Stars**: ~500  

#### Pros
* ✅ Full globstar support (`**/*_spec.rb` works perfectly)
* ✅ Zero memory allocations for pattern matching
* ✅ Performance-focused v4 rewrite
* ✅ Excellent test coverage with benchmarks
* ✅ No external dependencies
* ✅ Used by GitLab Runner (production proven)
* ✅ Clean API with configuration options

#### Cons
* None significant for our use case

#### Code Example
```go
import "github.com/bmatcuk/doublestar/v4"

// Pattern matching
matched, err := doublestar.Match("**/*_spec.rb", "test/models/user_spec.rb")

// File globbing with callback (most performant)
err := doublestar.GlobWalk(os.DirFS("."), "**/*_spec.rb", 
    func(path string, d fs.DirEntry) error {
        // Run test for this file
        return nil
    })

// With options
matches, err := doublestar.FilepathGlob("**/*.rb", 
    doublestar.WithCaseInsensitive())
```

#### TOML Configuration Mapping
```toml
[watch]
patterns = ["**/*_spec.rb", "**/*_test.rb"]
exclude = ["vendor/**", "tmp/**"]
case_insensitive = false
fail_on_io_errors = false
```

### 2. go-zglob (github.com/mattn/go-zglob)

**Version**: v0.0.4+ (September 2024)  
**Maintenance**: Active  
**GitHub Stars**: ~160  

#### Pros
* ✅ Full globstar support
* ✅ Simple API
* ✅ Maintained by mattn (Google Dev Expert)
* ✅ Follow symlinks option

#### Cons
* ❌ Fewer configuration options
* ❌ Less performant than doublestar
* ❌ Smaller community/less battle-tested

#### Code Example
```go
import "github.com/mattn/go-zglob"

// Basic usage
matches, err := zglob.Glob("**/*_spec.rb")

// Pattern matching without filesystem
matched, err := zglob.Match("**/*_spec.rb", "app/models/user_spec.rb")
```

### 3. glob (github.com/gobwas/glob)

**Version**: v0.2.3  
**Maintenance**: Appears unmaintained  
**GitHub Stars**: ~900  

#### Pros
* ✅ Very fast when patterns are compiled once
* ✅ Good API design

#### Cons
* ❌ **No recent updates** (issues from 2021-2023 unaddressed)
* ❌ Globstar requires manual separator specification
* ❌ Limited configuration options
* ❌ Maintenance concerns

#### Code Example
```go
import "github.com/gobwas/glob"

// Must specify separator for filesystem paths
g := glob.MustCompile("**/*_spec.rb", '/')
matched := g.Match("spec/models/user_spec.rb")
```

### 4. filepath.Match (Standard Library)

**Maintenance**: Part of Go stdlib  

#### Pros
* ✅ No external dependencies
* ✅ Always maintained

#### Cons
* ❌ **No globstar support** (`**` not supported)
* ❌ Would require building globstar on top
* ❌ Limited pattern syntax

## Recommendation

**Use doublestar (github.com/bmatcuk/doublestar/v4)** for the following reasons:

1. **Perfect feature match**: Full globstar support with the exact syntax we need
2. **Performance**: Zero-allocation matching and v4 performance rewrite
3. **Maintenance**: Actively maintained with recent releases
4. **Production proven**: Used by GitLab Runner
5. **Clean integration**: Maps well to TOML configuration
6. **Testing**: Excellent test coverage and benchmarks

## Implementation Plan

### Phase 1: Replace FileMapper
```go
// Current hand-rolled implementation
type FileMapper struct {
    // Manual pattern matching
}

// New implementation with doublestar
type FileMapper struct {
    patterns []string
    matcher  *doublestar.Matcher
}

func (fm *FileMapper) MapFileToSpecs(changedFile string) []string {
    var specs []string
    for _, pattern := range fm.patterns {
        if matched, _ := doublestar.Match(pattern, changedFile); matched {
            specs = append(specs, fm.resolveSpec(changedFile))
        }
    }
    return specs
}
```

### Phase 2: Configuration Integration
```toml
# .plur.toml
[watch]
patterns = [
    "**/*_spec.rb",
    "**/*_test.rb",
    "spec/spec_helper.rb"
]

exclude = [
    "vendor/**",
    "tmp/**",
    "node_modules/**"
]

[watch.mapping]
# Map source to test patterns
"lib/**/*.rb" = "spec/**/*_spec.rb"
"app/**/*.rb" = "spec/**/*_spec.rb"
```

### Phase 3: Enhanced Features
* Use `GlobWalk` for efficient file discovery
* Add pattern caching for performance
* Support case-insensitive matching option
* Add pattern validation on config load

## Migration Benefits

1. **Reduced maintenance**: Remove custom pattern matching code
2. **Better performance**: Leverage optimized library
3. **More features**: Gain advanced pattern syntax
4. **Configuration**: Enable user-customizable patterns
5. **Testing**: Easier to test with well-defined library behavior

## Next Steps

1. Add `github.com/bmatcuk/doublestar/v4` to go.mod
2. Refactor `FileMapper` to use doublestar
3. Add configuration support for custom patterns
4. Update documentation with new pattern syntax
5. Add tests for new pattern matching behavior