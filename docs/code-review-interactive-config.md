# Code Review: Interactive Config Branch

**Date:** August 16, 2025  
**Branch:** interactive-config  
**Reviewer:** Pragmatic Code Reviewer (Collette)  
**Focus:** Watch subsystem enhancements, framework detection, and file mapping improvements

## Executive Summary

This branch introduces significant enhancements to the plur test runner's watch subsystem, adding comprehensive Minitest/Test::Unit support and improving the file-to-test mapping capabilities. The implementation is functionally solid with good test coverage, but there are architectural concerns around type consistency and framework detection that should be addressed to prevent future technical debt.

### Overall Assessment: **GOOD** (Ready to merge with minor fixes)

**Strengths:**
- Well-tested functionality with comprehensive integration tests
- User-friendly watch find command with actionable suggestions
- Clean separation of concerns in the watch package
- Thoughtful error handling and user feedback

**Main Concerns:**
- Inconsistent framework type handling (enum vs string)
- Silent configuration parsing failures
- Some code duplication in framework detection

## Architecture Review

### Framework Detection Design

The current architecture has a **split personality** regarding framework types:

```go
// Main package uses enum
type TestFramework string
const (
    FrameworkRSpec    TestFramework = "rspec"
    FrameworkMinitest TestFramework = "minitest"
)

// Watch package uses strings
func LoadMappingConfig(configPath string, framework string) (*MappingConfig, error)
```

This creates unnecessary type conversions throughout:
- `watch.go`: `string(framework)` conversions
- `watch_find.go`: Framework detection then string conversion
- `doctor.go`: Framework detection then string conversion

**Recommendation:** Use the TestFramework type consistently throughout all packages.

### File Mapping Architecture

The file mapping system is well-designed with:
- Clear separation between builtin and custom rules
- Priority-based rule evaluation
- Template variable expansion

However, the glob matching implementation is complex:
```go
func matchDoubleWildcard(pattern, filePath string) bool {
    // Complex regex conversion logic
    regexPattern := regexp.QuoteMeta(pattern)
    regexPattern = strings.ReplaceAll(regexPattern, `\*\*`, `.*`)
    // ... more replacements
}
```

This could be simplified using a dedicated glob library or pre-compiled patterns.

## Code Quality Assessment

### Positive Patterns

1. **Excellent Test Organization**
   ```ruby
   # spec/integration/plur_spec/watch_find_spec.rb
   context "with complex project structure" do
     context "when mapping exists and spec exists" do
       # Clear, nested contexts
     end
   end
   ```

2. **User-Friendly Output**
   ```go
   fmt.Printf("✓ %s → %s (exists)\n", file, spec)
   fmt.Printf("✗ %s → %s (mapping exists but spec not found)\n", file, spec)
   ```

3. **Good Error Context**
   ```ruby
   # script/cc-post-tool-use
   $stderr.puts "Error parsing JSON: #{e.message}"
   $stderr.puts "Input was: #{raw_input[0..200]}..."
   ```

### Areas for Improvement

1. **Silent Configuration Failures**
   ```go
   // config_loader.go - line 57-60
   if err := toml.Unmarshal(data, &fullConfig); err != nil {
       // Silently returns defaults instead of warning user
       return config, nil
   }
   ```
   This hides configuration problems from users.

2. **Duplicated Framework Detection**
   ```go
   // Three separate implementations:
   // 1. config.go: DetectTestFramework()
   // 2. mapping_rules.go: detectFramework()
   // 3. Inline detection in various files
   ```

3. **Complex Nested Conditions**
   ```go
   // detectPatternFromAlternative - deeply nested if/else
   if framework == FrameworkMinitest {
       if strings.HasPrefix(sourceFile, "lib/") {
           if strings.Contains(specDir, "/lib/") {
               // ...
           }
       }
   }
   ```

## Potential Issues and Risks

### Critical Issues
**None identified** - The code handles error cases appropriately and has no critical bugs.

### Medium Priority Issues

1. **Configuration Parsing Silence**
   - **Risk:** Users may have invalid TOML that silently fails
   - **Fix:** Add warning log when parsing fails
   ```go
   if err := toml.Unmarshal(data, &fullConfig); err != nil {
       logger.LogDebug("Failed to parse config file, using defaults", "error", err)
       return config, nil
   }
   ```

2. **Unbounded Test Execution**
   - **Risk:** Hook script could hang on long-running tests
   - **Fix:** Add timeout to cc-post-tool-use script
   ```ruby
   Timeout::timeout(30) do
     system(test_command)
   end
   ```

### Low Priority Issues

1. **Magic Numbers**
   ```go
   DefaultDebounceDelay = 100 * time.Millisecond  // Why 100ms?
   Priority: 60  // What does 60 mean?
   ```
   Consider constants with meaningful names.

2. **Incomplete Error Messages**
   ```go
   return fmt.Errorf("failed to update config: %w", err)
   // Could include config path for context
   ```

## Testing Analysis

### Test Coverage Assessment

**Excellent Coverage:**
- Integration tests cover real-world scenarios
- Both success and failure paths tested
- Edge cases like missing directories handled

**Good Test Patterns:**
```ruby
# Clear test structure
it "finds standard lib to spec mapping" do
  output = run_watch_find("lib/standard_mapper.rb")
  expect(output).to include("✓ lib/standard_mapper.rb → spec/standard_mapper_spec.rb (exists)")
end
```

### Missing Test Cases

1. **Concurrent file changes** - What happens with rapid successive changes?
2. **Symlink handling** - How does the mapper handle symlinked files?
3. **Permission errors** - File exists but not readable
4. **Large config files** - Performance with many custom rules

## Recommendations

### Immediate (Before Merge)

1. **Fix silent config parsing failure**
   ```go
   // Add warning log
   if err := toml.Unmarshal(data, &fullConfig); err != nil {
       logger.LogWarn("Config parsing failed, using defaults", "file", configPath, "error", err)
       return config, nil
   }
   ```

2. **Add script timeout**
   ```ruby
   # script/cc-post-tool-use
   require 'timeout'
   begin
     Timeout::timeout(30) do
       system(test_command)
     end
   rescue Timeout::Error
     $stderr.puts "Test execution timed out after 30 seconds"
   end
   ```

### Short Term (Next PR)

1. **Unify Framework Types**
   - Use TestFramework enum throughout
   - Remove string conversions
   - Single source of truth for framework detection

2. **Centralize Framework Detection**
   ```go
   // Create framework/detector.go
   package framework
   
   func Detect(projectPath string) TestFramework {
       // Single implementation
   }
   ```

3. **Simplify Glob Matching**
   - Consider using github.com/bmatcuk/doublestar
   - Pre-compile patterns for performance
   - Add pattern validation

### Long Term

1. **Performance Optimization**
   - Cache compiled regex patterns
   - Lazy load configuration
   - Profile large project performance

2. **Enhanced Diagnostics**
   - Add `plur watch diagnose` command
   - Show why a file doesn't match any rules
   - Visualize rule priority and conflicts

## Conclusion

This is a **solid, production-ready implementation** that successfully adds Minitest support and improves the watch subsystem. The code is well-tested and handles user interactions thoughtfully. 

The main architectural concern—inconsistent framework type handling—doesn't affect correctness but will increase maintenance burden over time. Address the silent config parsing issue before merging, then tackle the type unification in a follow-up PR.

The watch find command is particularly well done, providing clear, actionable feedback to users. The integration tests are comprehensive and well-structured. With the recommended fixes, this will be a valuable addition to plur.

### Merge Recommendation: **APPROVED** with minor fixes

Fix the config parsing warning, optionally add the script timeout, then ship it. The framework type unification can be handled in a cleanup PR without blocking this valuable functionality.