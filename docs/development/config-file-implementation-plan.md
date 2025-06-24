# Configuration File Implementation Plan

## Overview
Add TOML configuration file support to rux to allow persistent configuration and fix failing specs that expect `--command` flag support.

## Goals
1. ✅ Add `--command` CLI flag to override the default `bundle exec rspec` command
2. Support TOML configuration files that map directly to CLI flags
3. Allow example-project project to use `bin/rspec` via config file to improve performance

## Background
- Current performance: rux is ~47% slower than turbo_tests on example-project project (9.57s vs 6.53s)
- ✅ Failing specs in `spec/rux_cli_spec.rb` expect `--command` flag support - **FIXED**
- Kong's Configuration loader can populate CLI struct fields directly from config files
- No need for separate config structs - reuse existing CLI struct

## References
* ruby-lsp has done a lot of work on discovering `TestStyle` and the right command to run tests - https://github.com/Shopify/ruby-lsp/blob/main/lib/ruby_lsp/listeners/test_style.rb

## Progress Update (2025-06-21)

### Phase 1: CLI Flag Support ✅ COMPLETED

1. **Updated SpecCmd struct in main.go** ✅
   - Added `Command string` field with help text and default "bundle exec rspec"
   - Made it spec-command specific, not a global flag

2. **Updated Config struct in config.go** ✅
   - Added `SpecCommand string` field
   - Passed from `SpecCmd.Command` to config in `Run()` method

3. **Updated execution.go and runner.go** ✅
   - Modified `buildRSpecArgs()` to use `config.SpecCommand`
   - Updated `RunSpecFile()` to accept and use the spec command
   - Added debug logging to show the actual command being executed

4. **Fixed failing tests** ✅
   - Updated test syntax to use `rux spec --command=...` (command-specific)
   - Fixed test assertions to check stderr for debug output
   - All tests in `spec/rux_cli_spec.rb` now pass

### Key Implementation Details
- Command is always set (defaults to "bundle exec rspec")
- No conditional logic needed - just split the command string and use it
- Debug mode shows the actual command being executed in stderr
- Works with any command: `--command=rspec`, `--command="bin/rspec"`, etc.

## Implementation Phases

### Phase 1: CLI Flag Support (Fix failing specs)

1. **Update RuxCLI struct in main.go**
   - Add `Command string` field with help text and kong tag:
     ```go
     Command string `help:"Override default test command (default: bundle exec rspec)" default:""`
     ```

2. **Update Config struct in config.go**
   - Add `SpecCommand string` field
   - In SpecCmd.Run(), pass CLI command to config:
     ```go
     SpecCommand: parent.Command, // defaults to empty, meaning use default
     ```

3. **Update execution.go**
   - Modify `buildRSpecArgs()` to check if `config.SpecCommand` is set
   - If empty, use default "bundle exec rspec"
   - If set, parse and use the provided command
   - Handle splitting command strings like "bin/rspec" vs "bundle exec rspec"

### Phase 2: TOML Configuration Support

4. **Add dependency to go.mod**
   ```
   github.com/alecthomas/kong-toml v0.2.0
   ```

5. **Import kong-toml in main.go**
   ```go
   import kongtoml "github.com/alecthomas/kong-toml"
   ```

6. **Update main.go Kong parse**
   ```go
   ctx := kong.Parse(&cli,
       kong.Name("rux"),
       kong.Description("A fast Go-based test runner for Ruby/RSpec"),
       kong.Configuration(kongtoml.Loader, ".rux.toml", "~/.rux.toml"),
       kong.Vars{
           "cache_dir": configPaths.CacheDir,
       })
   ```

7. **TOML file format**
   - Keys map directly to CLI struct fields (converted to hyphen-case)
   - Example `.rux.toml`:
     ```toml
     command = "bin/rspec"
     workers = 4
     color = true
     trace = false
     ```

### Phase 3: Testing & Documentation

8. **Create example config**
   - `references/example-project/.rux.toml`:
     ```toml
     # Override the default test command
     command = "bin/rspec"
     ```

9. **Run tests**
   - Verify CLI flag `--command` works
   - Test TOML config loading
   - Test precedence (CLI flags override config file)
   - Run failing specs to ensure they pass

10. **Update documentation**
    - Add configuration section to CLAUDE.md
    - Document TOML format and available options
    - Explain field name mapping and precedence

## Files to Modify

1. `rux/go.mod` - Add kong-toml dependency
2. `rux/main.go` - Add Command field to RuxCLI, add Configuration loader
3. `rux/config.go` - Add SpecCommand field to Config
4. `rux/execution.go` - Use configurable command
5. `references/example-project/.rux.toml` - Example config file
6. `CLAUDE.md` - Documentation updates

## Benefits
- No duplicate structs to maintain
- All CLI flags automatically available in config files
- Kong handles parsing and validation
- Simple, clean implementation
- Easy to extend - just add new CLI fields

## Expected Outcomes
1. Failing specs in `spec/rux_cli_spec.rb` will pass
2. Any CLI flag can be set via TOML config
3. example-project project can use `bin/rspec` via config file
4. Performance improvement for example-project project (~47% faster)

## Testing Checklist
- [x] `--command='bin/rspec'` CLI flag works
- [x] `.rux.toml` with `command = "bin/rspec"` works
- [x] CLI flag overrides config file
- [x] All specs in `spec/rux_cli_spec.rb` pass
- [x] Benchmark completed - No performance improvement found (see below)

## Phase 2: TOML Configuration Support ✅ COMPLETED

1. **Added kong-toml import** ✅
   - Added `kongtoml "github.com/alecthomas/kong-toml"` import
   
2. **Added Configuration loader** ✅
   - Updated Kong parse with `kong.Configuration(kongtoml.Loader, ".rux.toml", "~/.rux.toml")`
   - Supports both local and home directory config files
   
3. **Created example config** ✅
   - Created `references/example-project/.rux.toml` with `command = "bin/rspec"`
   - Added helpful comments explaining available options
   
4. **Tested configuration loading** ✅
   - Local `.rux.toml` file properly overrides defaults
   - Global `~/.rux.toml` file works as fallback
   - CLI flags properly override config files
   - All existing tests pass

## Summary

Both Phase 1 (CLI flag support) and Phase 2 (TOML configuration support) are now complete! The implementation successfully:

1. Allows overriding the default `bundle exec rspec` command via `--command` flag
2. Supports TOML configuration files in the current directory and home directory
3. Follows proper precedence: CLI flags > local config > global config > defaults
4. Maintains backward compatibility - all existing tests pass
5. Uses Kong's built-in configuration loading, avoiding duplicate structs

The example-project project can now create a `.rux.toml` file with `command = "bin/rspec"` to customize the test command.

## Benchmark Results

Benchmark performed on 2025-06-21 comparing `bundle exec rspec` vs `bin/rspec`:

```
bundle exec rspec: 9.130s (±0.038s)
bin/rspec:         9.145s (±0.045s)
Improvement:       -0.2% (essentially no difference)
```

The expected 47% performance improvement was not realized because the `bin/rspec` file in the example-project project is a standard Bundler binstub that still loads Bundler, making it functionally equivalent to `bundle exec rspec`. To achieve performance improvements, a true standalone RSpec executable would be needed that bypasses Bundler's overhead.