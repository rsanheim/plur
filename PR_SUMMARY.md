# Add Full RSpec-Compatible Colorized Output Support

## Summary

This PR adds support for preserving RSpec's colorized output in rux, ensuring that test results are displayed with the same formatting and ANSI color codes as native RSpec. This significantly improves the user experience by providing familiar, visually enhanced output.

## Motivation

Previously, rux would:
- Always run RSpec with `--no-color`, stripping all color codes
- Recreate failure output in Go, losing RSpec's rich formatting
- Display plain text output that didn't match RSpec's colorized format

Users expect to see the same colorized output they're familiar with from RSpec, including:
- Red F for failures
- Green dots for passing tests
- Colored failure messages with proper highlighting
- Syntax-highlighted code snippets in errors

## Implementation

The solution captures RSpec's fully formatted output directly rather than trying to recreate it:

1. **Enhanced JSON Formatter** (`rux/rspec/formatter.rb`):
   - Added `dump_failures` and `dump_summary` hooks to capture RSpec's formatted output
   - These methods use RSpec's `fully_formatted_failed_examples` and `fully_formatted` methods
   - The formatted output (including ANSI codes) is sent through the JSON stream

2. **Color Flag Support** (`rux/runner.go`, `rux/main.go`):
   - Pass color preference from CLI through to RSpec
   - Use `--force-color --tty` when colors are enabled (forces colors even when piping)
   - Use `--no-color` when explicitly disabled
   - Respect `--color` and `--no-color` flags in any position (before or after file args)

3. **Display Formatted Output** (`rux/result.go`):
   - Use RSpec's formatted output when available
   - Fall back to manual formatting for backwards compatibility
   - Preserve all ANSI color codes in the output

## Benefits

- **Exact RSpec compatibility**: Output matches RSpec's format character-for-character
- **Rich formatting**: Preserves all colors, indentation, and styling
- **Simpler implementation**: No need to reimplement RSpec's complex formatting logic
- **Future-proof**: Automatically supports any formatting changes in future RSpec versions

## Testing

- Existing specs pass with colorized output
- Color output verified manually with failing specs
- Backwards compatibility maintained with fallback formatting

## Example Output

Before (plain text):
```
F

Failures:
  1) Single Failure fails due to strings not matching
     Failure/Error: expected = "All work and no play makes Jack a dull boy"
       expected: "All work and no play makes Jack a dull boy"
            got: "All work and no play makes something something something"
       (compared using ==)
```

After (with colors):
```
F

Failures:

  1) Single Failure fails due to strings not matching
     Failure/Error: expect(actual).to eq(expected)
     
       expected: "All work and no play makes Jack a dull boy"
            got: "All work and no play makes something something something"
     
       (compared using ==)
     # ./spec/single_failure_spec.rb:6:in 'block (2 levels) in <top (required)>'

Finished in 0.01228 seconds (files took 0.03648 seconds to load)
1 example, 1 failure

Failed examples:

rspec ./spec/single_failure_spec.rb:2 # Single Failure fails due to strings not matching
```
(Note: Colors are preserved but not visible in markdown)

## Technical Notes

- The approach is inspired by turbo_tests but implemented more simply
- We capture complete formatted output rather than individual failure formatting
- Multi-worker scenarios work correctly (last worker's output is used for summary)
- No changes needed to the streaming protocol or message parsing