# Config Refactoring Plan

## The Kong Consideration

If you're planning to switch to Kong, it has better patterns built-in:
* Kong uses struct tags for CLI parsing
* You can embed your config structs directly in command structs
* No globals needed - Kong instantiates your structs for you

## Snide Remarks Department

* "I don't really know how to best handle init/startup state" - Join the club! Every Go CLI has its own creative interpretation.
* That panic stacktrace is Go's way of saying "Welcome to the language! Here's your first variable shadowing bug, collect all 10 for a free t-shirt!"
* Using globals for config is like using `goto` - everyone says don't do it, but sometimes it's the most pragmatic solution.

## Current Status (2025-06-16)

### Completed ✅
* Fixed the variable shadowing bug that caused nil panics
* Consolidated scattered config logic into `Config` and `ConfigPaths` structs
* `InitConfigPaths()` called once at startup for directory setup
* Separated "immediate initialization" (`ConfigPaths`) from "CLI context dependent" (`Config`)
* Extracted `specFiles` from `BuildConfig` - no longer part of global config
* All tests passing with the refactored configuration
* Added `BinDir` to `ConfigPaths` for watcher binary storage
* Implemented `rux watch install` command for explicit binary installation
* Refactored all watcher binary logic into `watch/binary.go` for better organization
* Updated `doctor` and `watch` commands to use centralized binary functions

### In Progress 🚧
* Kong is now the default CLI framework!
* Removed urfave/cli2 completely from codebase
* Integration tests updated to work with Kong CLI structure

### Completed During Transition ✅
1. **Updated all functions to accept config as parameters** 
2. **Completed Kong implementation**
3. **Prepared Kong for production:**
   * ✅ Updated integration test suite to work with Kong CLI
   * ✅ Renamed `RunCmd` → `SpecCmd` for clarity  
   * ✅ Configured default command so `rux` runs specs without subcommand
   * ✅ Fixed British spelling support (`--no-colour`)
4. **Switched to Kong as default:**
   * ✅ Removed global variable `configPaths`
   * ✅ Made Kong the default CLI (removed `KONG=1` check)
   * ✅ Removed urfave/cli2 dependencies completely
   * ✅ Fixed integration tests for Kong CLI structure

#### Step 2: Complete the Kong implementation
Finish the existing Kong CLI in `kong.go`:

```go
type RuxCLI struct {
    // Global flags (same as current)
    Auto     bool `help:"Auto bundle install"`
    Verbose  bool `help:"Verbose output"`
    Workers  int  `short:"n" help:"Worker count"`
    // ... etc
    
    // Commands
    Run    RunCmd    `cmd:"" default:"withargs"`
    Watch  WatchCmd  `cmd:""`
    Doctor DoctorCmd `cmd:""`
}

func (cli *RuxCLI) AfterApply() error {
    // Build config once after parsing
    paths := InitConfigPaths()
    config := &Config{
        Auto:        cli.Auto,
        ColorOutput: cli.Color,
        ConfigPaths: paths,
        DryRun:      cli.DryRun,
        WorkerCount: GetWorkerCount(cli.Workers),
    }
    
    // Store in context for commands
    // Kong passes this context to all Run methods
    ctx := kong.GetContext(cli)
    ctx.Bind(config)
    ctx.Bind(paths)
    return nil
}

func (r *RunCmd) Run(ctx *kong.Context) error {
    config := ctx.Value((*Config)(nil)).(*Config)
    return runTests(config, r.Patterns)
}
```

#### Step 3: Switch the entry point
```go
// main.go
func main() {
    if os.Getenv("KONG") == "1" {
        runKongCLI()
    } else {
        runUrfaveCLI() // current implementation
    }
}
```

### Why This Works
* **No abstractions needed** - just pass config as params
* **Kong handles DI naturally** - via context binding
* **Easy testing** - `KONG=1 rux` to try it out
* **Low risk** - keep both CLIs until Kong is proven
* **Fast to implement** - mostly mechanical changes

## Kong Migration: Testing Strategy

### Integration Test Updates
To ensure both CLIs work correctly during the transition:

1. **Use KONG environment variable in tests:**
```ruby
# spec_helper.rb
def run_rux(args, **options)
  env = options[:env] || {}
  # Allow tests to opt-in to Kong CLI
  env['KONG'] = '1' if ENV['TEST_KONG_CLI'] == '1'
  
  cmd = TTY::Command.new(**options.merge(env: env))
  cmd.run("rux #{args}")
end
```

2. **Update CI to test both CLIs:**
```yaml
# .circleci/config.yml or similar
test_matrix:
  - TEST_KONG_CLI: ''   # test urfave/cli2 (default)
  - TEST_KONG_CLI: '1'  # test Kong CLI
```

3. **No changes needed to individual test files** - they'll automatically use Kong CLI when TEST_KONG_CLI=1 is set

### Kong Command Structure Updates

1. **Rename for clarity:**
   * `RunCmd` → `SpecCmd` (clearer that it runs specs/tests)
   
2. **Default command configuration:**
   * Research Kong's default command syntax (likely `default:"withargs"` tag)
   * Configure so both `rux` and `rux spec` run tests
   * Ensure argument handling works the same as current CLI

## Actionable TODOs

### Immediate Tasks (Simplified Kong Transition)
* [x] **Fix variable shadowing bug** - Changed `:=` to `=` to properly assign to global
* [x] **Extract specFiles from BuildConfig** - Spec files are now discovered per-command, not globally
* [x] **Fix `rux watch` command parsing** - Watch command now has dedicated handler, doesn't treat 'watch' as spec pattern
* [x] **Implement `rux watch install`** - Added explicit binary installation command
* [x] **Refactor binary management** - Consolidated all binary logic in `watch/binary.go`
* [x] **Step 1: Update function signatures** - Pass config as parameters instead of using globals
  * [x] `runDoctor()` → `runDoctorWithConfig(config *Config)`
  * [x] `runWatch()` → `runWatchWithConfig(config *Config, timeout, debounce int)`
  * [x] `Execute()` → Already accepts config via `NewTestExecutor(config, specFiles)`
  * [x] Update callers to pass global config (temporary)
* [x] **Step 2: Complete Kong implementation**
  * [x] Finish `RuxCLI` struct with all flags
  * [x] Implement `AfterApply()` for logging initialization
  * [x] Add Run methods for all commands (run, watch, doctor, db:*)
  * [x] Test with `KONG=1` env var
  * [x] Create `rux-kong` wrapper script
  * [x] Symlink to `~/go/bin/rux-kong` for global access
  * [x] Add CLI framework info to doctor output
* [x] **Step 3: Prepare for Kong as default**
  * [x] Update integration test suite to support both CLIs
    * [x] Add `KONG` env var support in spec helper
    * [x] Update `run_rux` helper to set `KONG=1` when `TEST_KONG_CLI=1`
    * [x] Fixed all integration tests for Kong CLI
  * [x] Fix watch subcommand structure using Kong's `default:""` pattern
  * [x] Rename `RunCmd` to `SpecCmd` for clarity
  * [x] Investigate Kong default command syntax - uses `default:""` tag
  * [x] Make `SpecCmd` the default command so `rux` and `rux spec` both run tests
* [x] **Step 4: Clean up**
  * [x] Remove global variable `configPaths` 
  * [x] Remove urfave/cli2 code completely
  * [x] Update integration tests for Kong

### Config Refactoring Tasks
* [x] **Consolidate config initialization** - Created `Config` and `ConfigPaths` structs
* [x] **Separate concerns** - ConfigPaths for immediate init, Config for CLI context
* [ ] **Remove global variables** - Replace `var ruxConfig *Config` with App.Metadata storage
* [ ] **Update command Actions** - Modify all commands to extract config from `ctx.App.Metadata`
* [ ] **Make ConfigPaths unexported** - Change `ConfigPaths` to `configPaths` if keeping any globals
* [ ] **Add initialization documentation** - Document the config initialization flow and lifecycle

### Testing & Validation
* [x] **Test all commands** - Doctor, watch, and main test runner confirmed working
* [x] **Fix failing specs** - Updated specs to work with new config structure
* [x] **Add watch install spec** - Integration test for binary installation
* [x] **Update watch specs** - Fixed binary path expectations for new bin directory
* [ ] **Add config tests** - Unit tests for BuildConfig and ConfigPaths initialization
* [ ] **Test App.Metadata approach** - Ensure new pattern works across all commands

### Documentation
* [x] **Document watch install implementation** - Created implementation plan documentation
* [x] **Document binary refactoring** - Created summary of binary management changes
* [ ] **Document App.Metadata pattern** - Add examples of how to access config from commands
* [ ] **Before hook behavior** - Document when Before runs vs command resolution
* [ ] **Migration guide** - If switching approaches, document the changes needed in each command

## Recent Accomplishments (2025-06-16)

### Complete Migration to Kong CLI ✅
* **Removed urfave/cli2 entirely:**
  - Deleted all urfave/cli2 imports and code
  - Removed `file_mapper_cmd.go` (dev command not in Kong)
  - Cleaned up go.mod dependencies
  - Fixed all Go formatting issues

* **Fixed British spelling support:**
  - Added separate `Colour` field with negatable flag
  - Syncs to `Color` field in `AfterApply` method
  - All colorization tests pass

* **Updated integration tests:**
  - Fixed watch helper to use 'watch run' subcommand
  - Updated db commands from colon format to space format
  - Skipped file_mapper tests (command removed)
  - All tests passing!

### Kong CLI Subcommand Pattern Discovery
* Discovered that Kong treats commands differently than urfave/cli:
  - Commands with subcommands are **namespaces only** and cannot be directly executable
  - Must use `default:""` tag to specify default subcommand behavior
* Fixed `watch` command structure:
  - Created `WatchRunCmd` as default subcommand with `default:""` tag
  - Moved flags to the leaf command where they belong
  - Now supports both `rux watch` (uses default) and `rux watch install`
* Documented Kong patterns in `docs/development/kong-cli-patterns.md`

## Recent Accomplishments (2025-06-15)

### Watch Binary Management Refactoring
* Created `watch/binary.go` consolidating all binary-related functions
* Removed duplicate code from `watcher.go` and `watch.go`
* Implemented `rux watch install` command for explicit binary installation
* Changed binary location from `.cache/bin/` to `bin/` for better organization
* All binary logic now follows DRY principles with centralized platform detection

### Code Review Findings
Identified dead code to be cleaned up:
* Unused caching variables in `runner.go` (lines 21-23)
* Unused `GetVersion()` function in `version.go`
* Unused logger wrapper functions (`LogError`, `LogWarn`, etc.)
* ~~Incomplete Kong CLI implementation that needs completion or removal~~ ✅ COMPLETED!

## TODO: Summary of Current State (2025-06-16)

### What's Done ✅
* **Kong is now the default and only CLI framework!**
* Removed urfave/cli2 completely from the codebase
* Fixed all integration tests to work with Kong CLI
* Removed global variable `configPaths` (only `ruxConfig` remains in main.go)
* British spelling support working (`--no-colour`)
* All tests passing!

### What's Left to Do 📝
1. **Remove last global variable** - `ruxConfig` in main.go (line 15)
   - This is only used by urfave code which is now gone
   - Can be safely removed

2. **Update documentation:**
   - Main README to reflect Kong as the CLI
   - Remove references to `KONG=1` env var
   - Update CLI usage examples

3. **Clean up dead code** (from code review findings above)

4. **Consider CI updates:**
   - Remove any urfave-specific CI config
   - Ensure all CI tests use the Kong CLI

5. **Update rux-kong wrapper:**
   - Either remove it or update docs to clarify it's no longer needed

### The Migration is 95% Complete! 🎉
The hard work is done. Kong is working, tests are passing, and urfave is gone. Just need some final cleanup and docs updates.