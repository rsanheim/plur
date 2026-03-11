# Kong CLI Patterns and Gotchas

[Kong CLI](https://github.com/alecthomas/kong) has some quirks around default subcommands and how it structures things.
This is a little doc on lessons learned, and may be out of date depending on how much Kong has been updated 
since last review.

## Commands vs Subcommands in Kong (i.e. default:"withargs")

In Kong, commands with subcommands are **namespaces only** - they cannot be directly executable!

### The Problem

Kong has a strict separation:
- **Commands with subcommands** = namespaces (cannot have Run methods)
- **Leaf commands** = executable (have Run methods)

If you try to make a command both executable AND have subcommands, Kong will error with "expected [subcommand]".

### The Solution

Use Kong's `default:"withargs"` tag to specify a default subcommand:

```go
type WatchCmd struct {
    Run     WatchRunCmd     `cmd:"" default:"withargs" help:"Run watch mode"`
    Install WatchInstallCmd `cmd:"" help:"Install the watcher binary"`
}
```

#### Kong Default Tag Options

* `default:"withargs"` - Activates subcommand even when flags are passed (recommended)
* `default:"1"` - Only activates subcommand when no arguments are provided
* `default:""` - **Invalid**, does not create a default (avoid)

This pattern allows:
- `plur watch` → executes the default `run` subcommand
- `plur watch --timeout=60` → executes `run` with flags (only works with `withargs`)
- `plur watch install` → executes the `install` subcommand
- `plur watch run` → explicitly executes the `run` subcommand

### Example Structure

```go
type WatchCmd struct {
    Run     WatchRunCmd     `cmd:"" default:"withargs" help:"Run watch mode"`
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

## Known Limitations with Default Subcommands

### Help Text Doesn't Show Default Subcommand Flags

**Issue:** When using `default:"withargs"`, the parent command's help text doesn't display the default subcommand's flags, even though they work functionally.

**Example:**
```bash
# This works - timeout flag is accepted
$ plur watch --timeout=60

# But help doesn't show the timeout flag
$ plur watch -h
# Shows: watch <command> [flags]
# Missing: --timeout, --debounce flags from WatchRunCmd

# You have to explicitly check the subcommand help
$ plur watch run -h
# Shows: --timeout, --debounce flags
```

**Why This Happens:**
* Kong's help system shows the parent command structure when subcommands exist
* The `default:"withargs"` tag makes flags work functionally but doesn't modify help output
* This is an unintentional limitation in Kong, not a deliberate design choice

**Workaround:**
* Document this behavior in user-facing docs
* Consider adding a note in your command help text mentioning the default subcommand

**Related Kong Issues:**
* [#33](https://github.com/alecthomas/kong/issues/33) - Help/usage printing issues with default commands
* [#217](https://github.com/alecthomas/kong/issues/217) - Flags not passed to default command (led to `withargs` implementation)
* [#188](https://github.com/alecthomas/kong/pull/188) - PR that added `default:"withargs"` support
* [#561](https://github.com/alecthomas/kong/issues/561) - Recent request for root commands with subcommands
