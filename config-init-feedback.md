# Config Initialization Feedback

## The Good
- Consolidating scattered config logic into `Config` and `ConfigPaths` structs is solid. Much cleaner than having path construction littered everywhere.
- `InitConfigPaths()` being called once at startup is the right approach for directory setup.
- Your instinct to separate "stuff that can be initialized immediately" (`ConfigPaths`) from "stuff that needs CLI context" (`Config`) is spot on.

## The Bad (and Ugly)
- **Classic Go footgun**: You've been bitten by variable shadowing. That `:=` in your Before function created a local `ruxConfig` while the global one stayed nil. This is Go's most beloved/hated feature.
- **Global variables**: Yeah, you're right to feel icky about them. They make testing a pain and create hidden dependencies. Every function that touches `ruxConfig` now has an implicit dependency you can't see from its signature.

## The Bug

The panic is caused by variable shadowing in `main.go`:

```go
// Line 27 - This creates a LOCAL ruxConfig, leaving global nil
ruxConfig, err := BuildConfig(ctx, configPaths)

// Should be:
var err error
ruxConfig, err = BuildConfig(ctx, configPaths)
```

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

## Bottom Line

Your refactoring direction is good - consolidating config is the right move. Just:
1. Fix the shadowing bug (change `:=` to `=` after declaring `err`)
2. Consider moving away from globals before it spreads further
3. If keeping globals for now, at least make them unexported (`configPaths` not `ConfigPaths`)
4. Add a comment above the globals explaining your shame and future refactoring plans

The real Go lesson here: The language actively tries to trick you with `:=`, but at least the compiler is fast enough that you discover your mistakes quickly!