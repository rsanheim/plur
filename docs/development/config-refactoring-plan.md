# Config Refactoring Plan

## The Good
- Consolidating scattered config logic into `Config` and `ConfigPaths` structs is solid. Much cleaner than having path construction littered everywhere.
- `InitConfigPaths()` being called once at startup is the right approach for directory setup.
- Your instinct to separate "stuff that can be initialized immediately" (`ConfigPaths`) from "stuff that needs CLI context" (`Config`) is spot on.

## The Bad (and Ugly)
- **Classic Go footgun**: You've been bitten by variable shadowing. That `:=` in your Before function created a local `ruxConfig` while the global one stayed nil. This is Go's most beloved/hated feature. ✅ FIXED
- **Global variables**: Yeah, you're right to feel icky about them. They make testing a pain and create hidden dependencies. Every function that touches `ruxConfig` now has an implicit dependency you can't see from its signature.

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
- Store your config in the CLI context and pass it through
- Each command extracts what it needs from context
- Makes dependencies explicit

### 2. Init in Before, Store in App Metadata
```go
app.Before = func(ctx *cli.Context) error {
    config := BuildConfig(ctx)
    ctx.App.Metadata["config"] = config
    return nil
}
```

### 3. Command-specific initialization
- Each command's Action initializes only what it needs
- More boilerplate but very explicit

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
- Kong uses struct tags for CLI parsing
- You can embed your config structs directly in command structs
- No globals needed - Kong instantiates your structs for you

## Snide Remarks Department

- "I don't really know how to best handle init/startup state" - Join the club! Every Go CLI has its own creative interpretation.
- That panic stacktrace is Go's way of saying "Welcome to the language! Here's your first variable shadowing bug, collect all 10 for a free t-shirt!"
- Using globals for config is like using `goto` - everyone says don't do it, but sometimes it's the most pragmatic solution.

## Current Status (2025-06-15)

### Completed ✅
- Fixed the variable shadowing bug that caused nil panics
- Consolidated scattered config logic into `Config` and `ConfigPaths` structs
- `InitConfigPaths()` called once at startup for directory setup
- Separated "immediate initialization" (`ConfigPaths`) from "CLI context dependent" (`Config`)
- Extracted `specFiles` from `BuildConfig` - no longer part of global config
- All tests passing with the refactored configuration

### In Progress 🚧
- Still using global variables (`ruxConfig` and `configPaths`)
- Need to implement App.Metadata approach to eliminate globals
- Watch command and doctor command still depend on global config

### Next Steps
1. Implement App.Metadata approach to eliminate global variables
2. Update all commands to extract config from context instead of globals
3. Consider making ConfigPaths unexported if keeping any globals
4. Add comprehensive tests for config initialization

The real Go lesson here: The language actively tries to trick you with `:=`, but at least the compiler is fast enough that you discover your mistakes quickly!

## Actionable TODOs

### Immediate Tasks
- [x] **Fix variable shadowing bug** - Changed `:=` to `=` to properly assign to global
- [x] **Extract specFiles from BuildConfig** - Spec files are now discovered per-command, not globally
- [ ] **Implement App.Metadata approach** - Replace global `ruxConfig` with storing config in `ctx.App.Metadata["config"]`
- [x] **Fix `rux watch` command parsing** - Watch command now has dedicated handler, doesn't treat 'watch' as spec pattern

### Config Refactoring Tasks
- [x] **Consolidate config initialization** - Created `Config` and `ConfigPaths` structs
- [x] **Separate concerns** - ConfigPaths for immediate init, Config for CLI context
- [ ] **Remove global variables** - Replace `var ruxConfig *Config` with App.Metadata storage
- [ ] **Update command Actions** - Modify all commands to extract config from `ctx.App.Metadata`
- [ ] **Make ConfigPaths unexported** - Change `ConfigPaths` to `configPaths` if keeping any globals
- [ ] **Add initialization documentation** - Document the config initialization flow and lifecycle

### Testing & Validation
- [x] **Test all commands** - Doctor, watch, and main test runner confirmed working
- [x] **Fix failing specs** - Updated specs to work with new config structure
- [ ] **Add config tests** - Unit tests for BuildConfig and ConfigPaths initialization
- [ ] **Test App.Metadata approach** - Ensure new pattern works across all commands

### Documentation
- [ ] **Document App.Metadata pattern** - Add examples of how to access config from commands
- [ ] **Before hook behavior** - Document when Before runs vs command resolution
- [ ] **Migration guide** - If switching approaches, document the changes needed in each command