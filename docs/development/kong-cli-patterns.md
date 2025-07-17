# Kong CLI Patterns and Gotchas

## Commands vs Subcommands in Kong

**Critical:** In Kong, commands with subcommands are **namespaces only** - they cannot be directly executable!

### The Problem

Unlike urfave/cli, Kong has a strict separation:
- **Commands with subcommands** = namespaces (cannot have Run methods)
- **Leaf commands** = executable (have Run methods)

If you try to make a command both executable AND have subcommands, Kong will error with "expected [subcommand]".

### The Solution

Use Kong's `default:""` tag to specify a default subcommand:

```go
type WatchCmd struct {
    Run     WatchRunCmd     `cmd:"" default:"" help:"Run watch mode"`
    Install WatchInstallCmd `cmd:"" help:"Install the watcher binary"`
}
```

This pattern allows:
- `plur watch` → executes the default `run` subcommand
- `plur watch install` → executes the `install` subcommand  
- `plur watch run` → explicitly executes the `run` subcommand

### Example Structure

```go
type WatchCmd struct {
    Run     WatchRunCmd     `cmd:"" default:"" help:"Run watch mode"`
    Install WatchInstallCmd `cmd:"" help:"Install the watcher binary"`
}

type WatchRunCmd struct {
    // Flags for the run subcommand
    Timeout  int `help:"Exit after specified seconds"`
    Debounce int `help:"Debounce delay in milliseconds"`
}

func (w *WatchRunCmd) Run(parent *PlurCLI) error {
    // Watch logic here
    return runWatchWithConfig(config, w.Timeout, w.Debounce)
}

type WatchInstallCmd struct{}

func (w *WatchInstallCmd) Run(parent *PlurCLI) error {
    // Install logic
    return runWatchInstall(true)
}
```

## Key Learnings

1. **Kong is opinionated**: Commands are either namespaces OR executable, not both
2. **Use `default:""` tag**: This allows a namespace to have default behavior
3. **Flags go on leaf commands**: Place flags on the actual executable commands, not the namespace
4. **No context checking needed**: With proper structure, Kong handles routing automatically