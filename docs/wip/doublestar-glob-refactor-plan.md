# Plan: Simplify and Improve glob.go with doublestar

## Overview

Replace the hand-rolled glob pattern matching in `plur/glob.go` with the battle-tested `doublestar` library to simplify code, improve performance, and add advanced pattern matching features.

## Current Implementation Analysis

The current `glob.go` has several issues:
* **Complex custom logic** - 250+ lines with hand-rolled `**` pattern handling (`expandDoubleStarGlob`, lines 165-234)
* **Limited pattern support** - No support for brace expansion `{a,b}`, limited `**` handling
* **Error-prone implementation** - Manually splits patterns, walks directories, complex path matching logic
* **Inconsistent behavior** - Different code paths for `**` patterns vs regular globs
* **Performance unknown** - No benchmarks to measure glob expansion performance

## Benefits of Using doublestar

The `github.com/bmatcuk/doublestar/v4` library will provide:
* **Simplified code** - Replace 250+ lines with ~100 lines
* **Advanced patterns** - Full globstar support, brace expansion `{alt1,alt2}`, character classes
* **Better reliability** - Production-proven (used by GitLab Runner)
* **Performance** - Zero-allocation pattern matching, optimized v4 rewrite
* **Maintainability** - Offload complex pattern matching to a well-tested library

## Implementation Plan

### Phase 1: Benchmark Current Implementation

Create benchmarks to measure current performance:

```go
// plur/glob_bench_test.go
func BenchmarkExpandGlobPatterns_Simple(b *testing.B) {
    // Benchmark: plur --dry-run spec/*_spec.rb
}

func BenchmarkExpandGlobPatterns_Recursive(b *testing.B) {
    // Benchmark: plur --dry-run spec/**/*_spec.rb
}

func BenchmarkExpandGlobPatterns_Multiple(b *testing.B) {
    // Benchmark: plur --dry-run spec/models/**/*_spec.rb spec/controllers/**/*_spec.rb
}

func BenchmarkExpandGlobPatterns_Complex(b *testing.B) {
    // Benchmark: plur --dry-run spec/{models,controllers,services}/**/*_spec.rb
}
```

Run benchmarks and capture baseline metrics:
```bash
cd plur
go test -bench=BenchmarkExpandGlob -benchmem -benchtime=10s > ../docs/wip/glob-benchmark-before.txt
```

### Phase 2: Add doublestar Dependency

```bash
cd plur
go get github.com/bmatcuk/doublestar/v4@v4.9.1
go mod vendor  # if using vendoring
```

### Phase 3: Refactor glob.go

#### New simplified structure:

```go
package main

import (
    "fmt"
    "os"
    "path/filepath"
    "strings"
    
    "github.com/bmatcuk/doublestar/v4"
)

// FindTestFiles - unchanged interface
func FindTestFiles(framework TestFramework) ([]string, error) {
    pattern := getDefaultPattern(framework)
    return doublestar.FilepathGlob(pattern)
}

// ExpandGlobPatterns - simplified with doublestar
func ExpandGlobPatterns(patterns []string, framework TestFramework) ([]string, error) {
    var allFiles []string
    seenFiles := make(map[string]bool)
    
    for _, pattern := range patterns {
        // Check if it's a directory
        if info, err := os.Stat(pattern); err == nil && info.IsDir() {
            // Expand directory to find all test files
            suffix := getTestFileSuffix(framework)
            pattern = filepath.Join(pattern, "**", "*"+suffix)
        }
        
        // Use doublestar for all pattern matching
        matches, err := doublestar.FilepathGlob(pattern)
        if err != nil {
            return nil, fmt.Errorf("invalid pattern %q: %v", pattern, err)
        }
        
        // Filter and dedupe
        for _, match := range matches {
            if isTestFile(match, framework) && !seenFiles[match] {
                allFiles = append(allFiles, match)
                seenFiles[match] = true
            }
        }
    }
    
    if len(allFiles) == 0 {
        return nil, fmt.Errorf("no test files found matching provided patterns")
    }
    
    return allFiles, nil
}
```

### Phase 4: Run Benchmarks After Refactor

```bash
cd plur
go test -bench=BenchmarkExpandGlob -benchmem -benchtime=10s > ../docs/wip/glob-benchmark-after.txt
```

Compare results to validate performance improvements.

### Phase 5: Validate with Existing Tests

Ensure all existing tests pass:
```bash
bin/rake test:default_ruby  # Quick validation
bin/rake test               # Full test suite
bundle exec rspec spec/integration/plur_spec/glob_support_spec.rb  # Specific glob tests
```

### Phase 6: Test New Pattern Features

Verify enhanced patterns work:
```bash
# Test brace expansion (not currently supported)
plur --dry-run 'spec/{models,controllers}/**/*_spec.rb'

# Test complex patterns
plur --dry-run '**/user*_spec.rb'

# Test case-insensitive matching (future feature)
```

## Expected Outcomes

### Code Simplification
* Remove ~150 lines of complex pattern matching code
* Eliminate `expandDoubleStarGlob` function entirely
* Cleaner, more maintainable implementation

### Performance Metrics
Track these metrics before/after:
* Time to expand `spec/**/*_spec.rb` pattern
* Memory allocations during pattern expansion
* Time for complex patterns with multiple `**`

### New Capabilities
Users will be able to use:
* `plur spec/models/**/*_spec.rb` - all specs under models recursively
* `plur spec/foo/**` - all specs under foo directory
* `plur 'spec/{models,controllers}/**/*_spec.rb'` - multiple directories with brace expansion
* `plur '**/user*_spec.rb'` - find patterns anywhere in the tree

## Risk Assessment

### Low Risk
* doublestar is well-tested and production-proven
* We maintain the same external API
* All existing patterns continue to work

### Mitigation
* Comprehensive benchmark comparison
* Full test suite validation
* Can easily revert if issues arise

## Success Criteria

1. ✅ All existing tests in `glob_support_spec.rb` pass
2. ✅ Benchmarks show equal or better performance
3. ✅ Code reduction of at least 100 lines
4. ✅ Support for brace expansion patterns works
5. ✅ No regression in existing functionality

## Timeline

* **Day 1**: Create benchmarks, measure baseline
* **Day 2**: Implement doublestar refactor
* **Day 3**: Test, benchmark, validate
* **Day 4**: Documentation updates if needed

## Future Enhancements

After successful implementation, consider:
* Add case-insensitive matching option
* Support for negation patterns (`!vendor/**`)
* Configuration for custom glob patterns in `.plur.toml`
* Apply same improvements to `plur watch` file mapping