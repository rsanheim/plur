# CLI Comparison: urfave/cli vs Kong

## Overview

This document analyzes the usage of `*cli.Context` in the plur codebase and compares urfave/cli's Context approach with Kong's struct-based approach.

## urfave/cli Context Usage in Plur

### 1. Flag Access Methods

The most common usage is accessing command-line flags through type-specific methods:

#### Boolean Flags
- `ctx.Bool("dry-run")` - Dry run mode
- `ctx.Bool("auto")` - Auto bundle install
- `ctx.Bool("json")` - JSON output mode
- `ctx.Bool("color")` - Force color output
- `ctx.Bool("no-color")` - Disable color output
- `ctx.Bool("trace")` - Performance tracing
- `ctx.Bool("verbose")` - Verbose output
- `ctx.Bool("debug")` - Debug output

#### Integer Flags
- `ctx.Int("n")` / `ctx.Int("workers")` - Number of parallel workers
- `ctx.Int("timeout")` - Timeout in seconds (watch mode)
- `ctx.Int("debounce")` - Debounce delay in milliseconds

#### String Flags
- `ctx.String("runtime-dir")` - Custom runtime directory path

### 2. Argument Access Methods

For positional arguments (non-flag arguments):

- `ctx.NArg()` - Returns the number of positional arguments
- `ctx.Args().Slice()` - Converts arguments to a string slice

### 3. Usage Patterns by Command

#### Main Command (running tests)
```go
// Check for trace mode
if ctx.Bool("trace") { ... }

// Get runtime directory
if runtimeDir := ctx.String("runtime-dir"); runtimeDir != "" { ... }

// Check dry-run mode
dryRun := ctx.Bool("dry-run")

// Get worker count
workerCount := GetWorkerCount(ctx.Int("n"))

// Check color preferences
shouldUseColor(ctx) // Helper function that checks multiple flags

// Check auto-install flag
if ctx.Bool("auto") { ... }

// Get spec files from arguments
if ctx.NArg() > 0 {
    specFiles, err = ExpandGlobPatterns(ctx.Args().Slice())
}
```

#### Watch Command
```go
// Get timeout value
timeout := ctx.Int("timeout")

// Get debounce delay
debounceMs := ctx.Int("debounce")
```

#### Doctor Command
- Only receives context but doesn't use any flags/arguments directly
- The verbose flag is handled by the global Before hook

#### File Mapper Command
```go
// Get files from arguments
files := ctx.Args().Slice()
```

#### Database Commands
```go
// Get worker count
workerCount := GetWorkerCount(ctx.Int("n"))

// Check dry-run mode
dryRun := ctx.Bool("dry-run")
```

### 4. Global Behavior

There's a global `Before` hook that runs before any command:
```go
Before: func(ctx *cli.Context) error {
    // Initialize logging globally before any command runs
    debug := ctx.Bool("debug") || os.Getenv("PLUR_DEBUG") == "1"
    InitLogger(ctx.Bool("verbose"), debug)
    return nil
}
```

### 5. Context Responsibilities Summary

The `*cli.Context` in plur is responsible for:
1. **Flag retrieval** - Getting boolean, integer, and string flag values
2. **Argument access** - Retrieving positional arguments as strings
3. **Argument counting** - Checking how many positional arguments were provided
4. **No direct command info** - The context doesn't appear to be used to get command names
5. **No output writing** - The context isn't used for writing output (uses fmt/os directly)
6. **No error handling** - Errors are returned from action functions, not set on context

## Kong CLI Approach

### 1. Kong Context Structure

Unlike urfave/cli which uses a single Context object, Kong uses:
```go
type Context struct {
    *Kong         // Embedded Kong instance
    Path []Path   // Trace through parsed nodes
    Args []string // Original command-line arguments
    Error error   // Error during trace, if any
}
```

### 2. Command Structure in Kong

Kong uses a **struct-based approach** with dependency injection:

```go
// urfave/cli approach:
func (c *MyCommand) Action(ctx *cli.Context) error {
    workers := ctx.Int("workers")
    pattern := ctx.String("pattern")
    // Access flags via ctx methods
}

// Kong approach - struct fields:
type MyCommand struct {
    Workers int    `short:"n" help:"Number of workers"`
    Pattern string `help:"File pattern to watch"`
    Paths   []string `arg:"" optional:"" help:"Paths to watch"`
}

// Flexible method signatures:
func (c *MyCommand) Run() error { }
func (c *MyCommand) Run(globals *GlobalOptions) error { }
func (c *MyCommand) Run(ctx *kong.Context) error { }
```

### 3. Accessing Flags and Arguments in Kong

In Kong, flags and arguments are **struct fields**, not accessed through methods:

```go
// Define command with flags as struct fields
type WatchCmd struct {
    Workers   int      `short:"n" help:"Number of workers"`
    Pattern   string   `help:"File pattern to watch"`
    Paths     []string `arg:"" optional:"" help:"Paths to watch"`
}

// Access directly in Run method
func (w *WatchCmd) Run() error {
    fmt.Printf("Workers: %d\n", w.Workers)
    fmt.Printf("Pattern: %s\n", w.Pattern)
    // w.Workers, w.Pattern etc are populated by Kong
}
```

## Key Differences

| Feature | urfave/cli | Kong |
|---------|------------|------|
| **Context Object** | Required for all actions | Optional - can inject any dependencies |
| **Flag Access** | `ctx.String("flag")` runtime lookup | Direct struct field access |
| **Argument Access** | `ctx.Args().Get(0)` | Struct fields with `arg:""` tag |
| **Command Structure** | Function-based Actions | Method-based with `Run()` |
| **Type Safety** | Runtime string lookups | Compile-time type checking |
| **Default Values** | Set in flag definition | Set in struct tags |
| **Validation** | Manual in action function | Automatic via validation tags |
| **Error Handling** | Typos cause runtime errors | Compiler catches typos |

## Migration Example

### Converting from urfave/cli to Kong

```go
// urfave/cli style
func watchAction(ctx *cli.Context) error {
    workers := ctx.Int("workers")
    pattern := ctx.String("pattern")
    timeout := ctx.Int("timeout")
    debounce := ctx.Int("debounce")
    paths := ctx.Args().Slice()
    
    // Implementation...
}

// Kong style
type WatchCmd struct {
    Workers  int      `short:"n" default:"4" help:"Number of workers"`
    Pattern  string   `default:"**/*_spec.rb" help:"File pattern"`
    Timeout  int      `help:"Exit after N seconds"`
    Debounce int      `default:"100" help:"Debounce delay in ms"`
    Paths    []string `arg:"" optional:"" help:"Paths to watch"`
}

func (w *WatchCmd) Run() error {
    // All values are already parsed and available as w.Workers, etc
    // No string lookups needed
}
```

## Migration Impact Analysis

### Advantages of Migration

1. **Type Safety** - Compile-time checking instead of runtime string lookups
2. **Better IDE Support** - Autocomplete and refactoring work with struct fields
3. **Cleaner Code** - No need for repetitive `ctx.Bool()` calls
4. **Validation** - Built-in validation through struct tags
5. **Flexibility** - Can inject any dependencies, not just context

### Migration Challenges

1. **Structural Change** - Need to convert function-based commands to struct-based
2. **Global Hooks** - Need to adapt the `Before` hook pattern
3. **Testing** - Test code that expects `*cli.Context` needs updating
4. **Build System** - May need to update build/vendor processes

### Limited Context Usage Makes Migration Easier

The good news is that plur uses Context in a very limited way:
- Only for flag/argument retrieval
- No complex Context manipulation
- No use of advanced Context features
- No context-based state management

This means we can create a straightforward mapping:
1. Each `ctx.Bool/Int/String()` call → struct field
2. `ctx.Args().Slice()` → struct field with `arg:""` tag
3. `ctx.NArg()` → `len(cmd.Args)`

## Incremental Migration Strategy

Based on this analysis, here's the recommended approach:

### Phase 1: Create Abstractions
1. Define interface for command context access
2. Wrap urfave/cli.Context to implement interface
3. Update commands to use interface instead of concrete type

### Phase 2: Parallel Implementation
1. Implement Kong backend for same interface
2. Use build tags or feature flags to switch implementations
3. Start with simplest command (doctor) as proof of concept

### Phase 3: Gradual Migration
1. Migrate commands one by one
2. Test thoroughly at each step
3. Keep both implementations working during transition

### Phase 4: Complete Migration
1. Remove urfave/cli dependency
2. Remove abstraction layer if no longer needed
3. Optimize for Kong's native patterns

## Conclusion

While urfave/cli and Kong have different approaches to command-line parsing, the limited and straightforward usage of Context in plur makes migration feasible. Kong's struct-based approach offers better type safety and cleaner code, making it a worthwhile migration target.