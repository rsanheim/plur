# Snapshot/Golden Testing Approaches: Record-Time vs Compare-Time Transformations

## Overview

This document summarizes how different snapshot/golden testing tools handle the distinction between:
1. **Record-time transformations** - Filtering/sanitizing data when creating the snapshot
2. **Compare-time transformations** - Normalizing/ignoring differences during comparison

## Tool Approaches

### VCR (Ruby) - HTTP Recording Library

**Philosophy**: Strong emphasis on record-time filtering for security

**Record-time features**:
- `filter_sensitive_data` configuration to replace sensitive text before saving
- Environment variable substitution: `c.filter_sensitive_data('<API_KEY>') { ENV['API_KEY'] }`
- Dynamic filtering based on request context
- `define_cassette_placeholder` for reusable placeholders

**Compare-time features**:
- Limited - VCR replays the filtered cassette as-is
- The sensitive text replaces the substitution string during replay

**Key insight**: VCR prioritizes security by ensuring sensitive data never gets written to disk

### Jest (JavaScript) - Snapshot Testing

**Philosophy**: Flexible approach supporting both record and compare-time handling

**Record-time features**:
- Custom snapshot serializers that transform data before saving
- Can create reusable serializers for common patterns

**Compare-time features**:
- Property matchers: `expect.any(String)`, `expect.any(Date)`
- Allows dynamic values to pass without exact matching
- External libraries like `snapshot-serializers` for pattern-based replacements

**Key insight**: Jest provides multiple strategies, letting developers choose based on use case

### Approval Tests Pattern (Language-agnostic)

**Philosophy**: Simple file comparison with external diff tools

**Record-time features**:
- Minimal - typically writes raw output to golden files
- Relies on deterministic test setup

**Compare-time features**:
- Delegates to external diff tools
- Some implementations support "scrubbers" for non-deterministic data
- Often requires manual approval of changes

**Key insight**: Simplicity over features - relies on humans to verify changes

### Insta (Rust) - Modern Snapshot Testing

**Philosophy**: Clear separation between filters and redactions

**Record-time features**:
- **Redactions**: Replace dynamic values in structured data
- Static redactions: `".id" => "[uuid]"`
- Dynamic redactions with callbacks
- Sorted redactions for non-deterministic ordering
- Rounded redactions for floating-point precision

**Compare-time features**:
- **Filters**: Regex-based transformations on final snapshot strings
- Applied after serialization
- Useful for string normalization (paths, whitespace)

**Key insight**: Explicit distinction between structured data handling (redactions) and string processing (filters)

### Go Golden Testing

**Philosophy**: Simple file-based testing with manual update workflow

**Record-time features**:
- `-update` flag to regenerate golden files
- Typically stores raw output
- Relies on deterministic test execution

**Compare-time features**:
- Basic byte-for-byte comparison
- Some libraries support line-ending normalization
- Limited built-in transformation capabilities

**Key insight**: Minimal complexity - transformations typically handled in test code

## Trade-offs Analysis

### Record-Time Transformations

**Pros**:
- Security: Sensitive data never touches disk
- Performance: No runtime transformation overhead
- Simplicity: Snapshots represent actual expected output
- Reproducibility: Anyone can see exactly what's expected

**Cons**:
- Inflexibility: Must regenerate snapshots for acceptable variations
- Maintenance: More frequent snapshot updates
- Loss of information: Can't see original values for debugging

### Compare-Time Transformations

**Pros**:
- Flexibility: Can handle platform differences, timestamps
- Debugging: Original values preserved in snapshots
- Stability: Fewer snapshot updates needed
- Backward compatibility: Can add new normalizations without regenerating

**Cons**:
- Performance: Transformation overhead on every test run
- Complexity: More code to maintain
- Security risk: Sensitive data might be stored
- Hidden behavior: Expected values not immediately visible

## Best Practices Summary

1. **Use record-time filtering for**:
   - Sensitive data (passwords, API keys, PII)
   - Values that should never be stored
   - Security-critical transformations

2. **Use compare-time normalization for**:
   - Platform differences (file paths, line endings)
   - Acceptable variations (whitespace, ordering)
   - Timestamps with known patterns
   - Debugging-friendly transformations

3. **Hybrid approach** (recommended):
   - Record-time: Remove truly sensitive/variable data
   - Compare-time: Handle environmental differences
   - Clear documentation of what happens when

4. **User experience considerations**:
   - Make transformation rules explicit and discoverable
   - Provide clear error messages showing both expected and actual
   - Allow easy snapshot updates with review workflow
   - Support debugging by showing pre-transformation values

## Conclusion

The most sophisticated tools (Insta, Jest with custom serializers) provide clear mechanisms for both approaches, recognizing that different types of dynamic content require different handling strategies. The key is to be intentional about when and why transformations occur, balancing security, maintainability, and debugging needs.