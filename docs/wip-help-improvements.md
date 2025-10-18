# WIP: Help Output Improvements for Plur

## Issues to Address

### Core Help Functionality
* [x] Make `plur help` work (currently requires `plur -h`)
* [x] Sort flags in alphabetical order
* [x] Add short flag for `--verbose` (e.g., `-v`)

### Flag Organization
* [x] Move `--auto` flag from top-level to `plur spec` subcommand
* [x] Consider if `--auto` should also be available for `plur watch run` (decision: no, only for spec)
* [x] Review other flags for proper placement in command hierarchy

## Implementation Notes

### 0. Kong Version Check
**Current:** v1.11.0
**Latest:** v1.12.1 (released July 2025)

**Changes in v1.12.0 and v1.12.1:**
* Fix: Don't require a Run() method on dynamic commands
* Various dependency updates
* License update for levenshtein method
* Minor tinygo compatibility tweak

**Recommendation:** The changes are minimal and mostly maintenance-related. Upgrading is low-risk but not critical for our help improvements. Consider upgrading as part of this work for best practices.

### Research Findings

#### 1. `plur help` Support
Kong doesn't have a built-in "help" command - it uses the `-h/--help` flag mechanism. Currently, `plur help` tries to find a file named "help" which fails.

**Solution Options:**
- Add a custom `HelpCmd` command that triggers help display
- Intercept "help" as first argument before Kong parsing

#### 2. Flag Ordering
Kong displays flags in the order they're defined in the struct. No built-in sorting mechanism found.

**Solution Options:**
- Reorder struct fields manually (simplest)
- Use Kong's `AutoGroup` option to group related flags
- Implement custom HelpPrinter (most complex)

#### 3. Short Flag for Verbose
Simple fix - add `short:"v"` tag to the Verbose field in PlurCLI struct.

#### 4. `--auto` Flag Location
Currently defined at top-level PlurCLI but only used in:
- `SpecCmd.Run()` (line 97 in main.go)
- `executeDryRun()` for display purposes
- NOT used in watch command

**Solution:**
- Move to SpecCmd struct
- Optionally add to WatchRunCmd if auto-install makes sense there

## Testing Checklist

* [x] Verify `plur help` displays main help
* [x] Verify `plur help <subcommand>` works for all subcommands
* [x] Verify flag order is alphabetical
* [x] Verify `-v` works as shorthand for `--verbose`
* [x] Verify `--auto` is only available on `plur spec` subcommand
* [x] Update integration tests for new help behavior (Backspin golden file updated)

## Summary of Changes

All improvements have been successfully implemented:

1. **Kong Upgrade**: v1.11.0 → v1.12.1
2. **`plur help` Support**: Intercepts "help" argument and converts to "-h" flag
3. **Short Flag**: Added `-v` for `--verbose`
4. **Flag Reorganization**:
   - Moved `--auto` from top-level to `plur spec` subcommand only
   - Sorted all flags alphabetically in help output
   - Removed `Auto` from GlobalConfig and doctor output
5. **All Tests Pass**: 238 examples, 0 failures, 7 pending

### Files Modified
* `plur/go.mod`, `plur/go.sum` - Kong upgrade
* `plur/main.go` - Help command, flag reorganization, Auto flag moved
* `plur/doctor.go` - Removed Auto from output
* `plur/config/config.go` - No changes needed (Auto kept for execution.go)
* `fixtures/backspin/plur_doctor_golden.yml` - Updated golden file