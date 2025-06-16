# Config Refactoring Plan

## The Good
* Consolidating scattered config logic into `Config` and `ConfigPaths` structs is solid. Much cleaner than having path construction littered everywhere.
* `InitConfigPaths()` being called once at startup is the right approach for directory setup.
* Your instinct to separate "stuff that can be initialized immediately" (`ConfigPaths`) from "stuff that needs CLI context" (`Config`) is spot on.

## The Bad (and Ugly)
* **Classic Go footgun**: You've been bitten by variable shadowing. That `:=` in your Before function created a local `ruxConfig` while the global one stayed nil. This is Go's most beloved/hated feature. ✅ FIXED
* **Global variables**: Yeah, you're right to feel icky about them. They make testing a pain and create hidden dependencies. Every function that touches `ruxConfig` now has an implicit dependency you can't see from its signature.

## The Bug ✅ FIXED

The panic was caused by variable shadowing in `main.go`:

```go
// Line 27 - This created a LOCAL ruxConfig, leaving global nil
ruxConfig, err := BuildConfig(ctx, configPaths)

// Fixed to:
ruxConfig = BuildConfig(ctx, configPaths)

## Better Patterns for CLI Init

Instead of globals, consider these approaches:

### 1. Dependency Injection via Context
* Store your config in the CLI context and pass it through
* Each command extracts what it needs from context
* Makes dependencies explicit

### 2. Init in Before, Store in App Metadata
```go
app.Before = func(ctx *cli.Context) error {
    config := BuildConfig(ctx)
    ctx.App.Metadata["config"] = config
    return nil
}
```

### 3. Command-specific initialization
* Each command's Action initializes only what it needs
* More boilerplate but very explicit

### 4. Struct with methods (my favorite for larger CLIs)
```go
type RuxCLI struct {
    configPaths *ConfigPaths
    config      *Config
}

func (r *RuxCLI) Run() { /* setup and run */ }
func (r *RuxCLI) createApp() *cli.App { /* commands reference r */ }
```

## The Kong Consideration

If you're planning to switch to Kong, it has better patterns built-in:
* Kong uses struct tags for CLI parsing
* You can embed your config structs directly in command structs
* No globals needed - Kong instantiates your structs for you

## Snide Remarks Department

* "I don't really know how to best handle init/startup state" - Join the club! Every Go CLI has its own creative interpretation.
* That panic stacktrace is Go's way of saying "Welcome to the language! Here's your first variable shadowing bug, collect all 10 for a free t-shirt!"
* Using globals for config is like using `goto` - everyone says don't do it, but sometimes it's the most pragmatic solution.

## Current Status (2025-06-15)

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
* Kong CLI is fully functional and can be used with `rux-kong` or `KONG=1 rux`
* Still using global variables (`ruxConfig` and `configPaths`) for backward compatibility
* Both CLIs (urfave/cli2 and Kong) coexist peacefully

### Next Steps (Simplified)
1. ~~**Today:** Update all functions to accept config as parameters (mechanical change)~~ ✅ DONE
2. ~~**Tomorrow:** Complete Kong implementation and test with `KONG=1`~~ ✅ DONE
3. **Prepare Kong for production:**
   * Update integration test suite to test both CLIs
   * Rename `RunCmd` → `SpecCmd` for clarity  
   * Configure default command so `rux` runs specs without subcommand
4. **Switch to Kong as default:**
   * Remove global variables (`ruxConfig`, `configPaths`)
   * Make Kong the default CLI (remove `KONG=1` check)
   * Remove urfave/cli2 dependencies
   * Update all documentation

### Later

* ~~Complete or remove Kong CLI implementation~~ ✅ DONE - Kong CLI is now fully functional!

The real Go lesson here: The language actively tries to trick you with `:=`, but at least the compiler is fast enough that you discover your mistakes quickly!

## Simplified Kong Transition Strategy (2025-06-15)

### Current State
* **2 global variables:** `configPaths` and `ruxConfig`
* **4 commands:** main runner, watch, doctor, version
* **Limited surface area:** Most config usage is in just a few files

### Simple 3-Step Plan

#### Step 1: Pass config as parameters (keep globals for now)
Update function signatures to accept config instead of using globals:

```go
// Before:
func runDoctor() error {
    // uses global ruxConfig
}

// After:
func runDoctor(config *Config) error {
    // uses passed config
}
```

Do this for all functions that use config. The CLI handlers can still use globals to pass them in.

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
* [ ] **Step 3: Prepare for Kong as default**
  * [ ] Update integration test suite to support both CLIs
    * [ ] Add `TEST_KONG_CLI` env var support in spec helper
    * [ ] Update `run_rux` helper to set `KONG=1` when `TEST_KONG_CLI=1`
    * [ ] Run CI tests with both CLI frameworks
  * [ ] Rename `RunCmd` to `SpecCmd` for clarity
  * [ ] Investigate Kong default command syntax
  * [ ] Make `SpecCmd` the default command so `rux` and `rux spec` both run tests
* [ ] **Step 4: Clean up**
  * [ ] Remove globals once Kong is primary
  * [ ] Remove urfave/cli2 code
  * [ ] Update docs and tests

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
* Incomplete Kong CLI implementation that needs completion or removal