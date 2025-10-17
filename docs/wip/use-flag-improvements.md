# --use Flag Improvements and Follow-up Work

## STATUS UPDATE (2025-10-17)

### ✅ COMPLETED ITEMS:
- **Added `-t` short flag** - Both spec and watch commands now support `-t` for task selection
- **Improved help text** - Changed to clear "Task to run (rspec/minitest/custom)"
- **Hidden implementation details** - Global `--use` and `--task` flags now have `hidden:""` tags
- **Fail-fast validation** - Invalid task names now error immediately with sorted list of available tasks
  - Example: `task 'nonexistent' not found. Available tasks: custom, minitest, rspec`
- **Code refactoring** - Extracted mergeTaskConfig() helper to reduce duplication
- **Updated installation.md** - Now mentions "Ruby 2.7+ with RSpec or Minitest"
- **Fixed custom task name preservation** - Custom tasks now show their configured name (e.g., "watch") instead of auto-detected framework name (e.g., "rspec") while still inheriting defaults
  - mergeTaskConfig now preserves Name field from custom task
  - getTaskWithOverrides simplified to always use auto-detect as base for custom tasks
  - Added comprehensive Go tests for custom task inheritance behavior
  - Example: `[task.watch]` with `run = "bin/rspec"` now shows `task=watch` in debug output
- **Created mixed-framework test fixture** - `fixtures/projects/mixed-rspec-minitest/` with both RSpec and Minitest tests
  - Tests all framework selection scenarios (auto-detect, explicit, config file, override)
  - 4x faster than dynamic test creation (0.28s vs 1.1s)
  - Includes README with usage examples

### 🚧 REMAINING WORK:
- [ ] Add framework detection hint when both spec/ and test/ exist

---

## Overview

This document outlines recommended improvements to make framework selection easier for users, particularly those with non-standard Ruby projects (mixed RSpec/Minitest, Minitest-only, custom frameworks).

## Analysis: Current Flag Issues (2025-10-14)

### Problem 1: Duplicate --use in Help ✅ DONE

**Status:** RESOLVED - Global --use and --task flags now hidden with `hidden:""` tags

When running `plur spec -h`, users see:
```
--use=""                Default task configuration to use
-u, --use=""            Task configuration to use
```

**Root Cause:**
- `PlurCLI.Use` (main.go:196): Global `--use=""` with no short flag
- `SpecCmd.Use` (main.go:27): Command-level `-u, --use=""`
- Kong shows all parent flags in subcommand help, creating duplication

**Current Behavior:**
- Priority: `SpecCmd.Use` → `PlurCLI.Use` → auto-detect
- Both commands (`spec` and `watch`) have their own `Use` field with `-u`
- Global `Use` acts as fallback if command-level not specified

### Problem 2: --task Flag Leaking to CLI Help

The `--task=KEY=VALUE` flag appears in help output but is marked "config file only".

**Root Cause:**
- `PlurCLI.Task` (main.go:199): Used by Kong to parse `[task.NAME]` TOML sections
- No `hidden:""` tag, so it appears in help
- Confuses users - looks like a CLI flag but is really for config parsing

### Why -t Was Initially Avoided

**Initial Concern:** RSpec uses `-t` for `--tag` (filtering tests by tag)

**Analysis Result:** **Not a Real Conflict**
- Plur doesn't pass arbitrary flags through to rspec
- `Task.BuildCommand()` (task.go:33) constructs commands from scratch
- No mechanism for users to pass rspec flags like `-t`
- parallel_tests uses `-t` for test type (aligns with our goal)

**Conclusion:** `-t` is safe to use and available (no conflicts with `-h`, `-d`, `-C`, `-n`)

### Kong Tag Reference

Confirmed Kong supports:
- `hidden:""` - Hides from help but still parseable (used for `Colour` flag)
- `kong:"-"` - Completely ignored by Kong (used for internal fields)

## Design Options

### Option A: Command-Level -t Only (Recommended)

**Changes:**
1. Change `SpecCmd.Use` from `short:"u"` to `short:"t"`
2. Change `WatchRunCmd.Use` from `short:"u"` to `short:"t"`
3. Add `hidden:""` to `PlurCLI.Use` (hide global flag from help)
4. Add `hidden:""` to `PlurCLI.Task` (hide config parsing detail)

**Result:**
```bash
plur spec -t rspec       # Clean, obvious
plur watch -t minitest   # Consistent
plur -t rspec spec       # Still works via hidden global flag
```

**Pros:**
- Clean help output, no duplication
- Matches parallel_tests convention (`-t` for test type)
- Shorter and more intuitive than `-u`
- Hides implementation details (--task, global --use)

**Cons:**
- Breaking change for anyone using `-u` (small impact, recently added)
- Different from current documented examples

### Option B: Global -t with Hidden Command Flags

**Changes:**
1. Add `short:"t"` to `PlurCLI.Use`
2. Add `hidden:""` to `SpecCmd.Use` and `WatchRunCmd.Use`
3. Add `hidden:""` to `PlurCLI.Task`

**Result:**
```bash
plur -t rspec            # Works globally
plur -t rspec spec       # Also works
plur spec -u rspec       # Hidden but still works
```

**Pros:**
- Global flag is more versatile
- Backward compatible (`-u` still works, just hidden)

**Cons:**
- Flag must come before command (`plur -t rspec spec`, not `plur spec -t rspec`)
- Less discoverable for command-specific use

### Option C: Keep Both Visible with -t

**Changes:**
1. Add `short:"t"` to `PlurCLI.Use`
2. Change `SpecCmd.Use` from `short:"u"` to `short:"t"`
3. Add `hidden:""` to `PlurCLI.Task`

**Result:**
Users see both flags but with clearer naming

**Pros:**
- Maximum flexibility
- Both patterns work

**Cons:**
- Still has duplication in help
- Doesn't solve the original problem

### Recommendation: Option A

**Rationale:**
- Cleanest UX with no duplicate flags in help
- Command-level placement is most natural: `plur spec -t rspec`
- Matches how users think: "run the spec command with this task"
- Easy deprecation path: remove command flags later if we want global
- Hides implementation details users don't need to see

## 1. Add `-t` Short Flag ✅ DONE

**Status:** IMPLEMENTED - Both SpecCmd and WatchRunCmd now have `short:"t"` flags

### Why
* **Ease of use**: `plur -t rspec` is much faster to type than `plur --use=rspec`
* **Familiarity**: Matches `parallel_tests` convention (`-t, --type`)
* **Migration friendly**: Helps users transitioning from parallel_tests
* **Semantic clarity**: `-t` for "test type" or "task" makes intuitive sense

### What
Add short flag alias to existing `--use` flag:
```go
Use string `short:"t" help:"Task to use (rspec/minitest/custom)" default:""`
```

### Impact
* **Code change**: One line in `plur/main.go:196`
* **Documentation**: Update all 4 docs we just modified to show `-t` examples
* **Testing**: Verify `-t rspec` works identically to `--use=rspec`

### Benefits
* Lower barrier to entry for new users
* Faster daily workflow for power users
* Better alignment with Ruby ecosystem conventions

## 2. Create Mixed-Framework Test Fixture ✅ DONE

**Status:** COMPLETED - Created `fixtures/projects/mixed-rspec-minitest/`

### What We Built
* `spec/example_spec.rb` - 3 passing RSpec examples
* `test/example_test.rb` - 3 passing Minitest tests
* Gemfile with both rspec and minitest gems
* Gemfile.lock checked in (no bundle install in tests)
* README with usage examples and purpose

### Test Coverage
Updated `spec/integration/plur_spec/framework_selection_spec.rb` to use the fixture:
* ✅ Default behavior (picks RSpec when both exist)
* ✅ `plur spec -t rspec` explicitly runs RSpec
* ✅ `plur spec -t minitest` explicitly runs Minitest
* ✅ Config file `use = "minitest"` changes default
* ✅ CLI flag overrides config file setting
* ✅ All 8 scenarios pass in 0.28s (4x faster than dynamic creation)

### Benefits Achieved
* Safe refactoring of detection logic with test coverage
* Easy to review and debug (real files on disk)
* Faster test runs (no bundle install each time)
* Documents expected behavior for maintainers

## 3. Add Framework Detection Hint

### Why
* **Discoverability**: Users don't know `--use` exists until they read docs
* **Just-in-time help**: Show guidance exactly when it's needed
* **Reduce friction**: Help users before they get frustrated

### When to Show
* When both `spec/` and `test/` directories exist
* Only on first run (or when no config file exists)
* Can be suppressed with config file setting

### Example Message
```
Note: Both spec/ and test/ directories detected.
Running RSpec tests by default.

To run Minitest instead:
  plur -t minitest              # Use -t flag
  echo 'use = "minitest"' > .plur.toml  # Or set default in config

Run 'plur doctor' to see current configuration.
```

### Implementation Notes
* Message goes to stderr, not stdout
* Only shows once per project (can track in `.plur/` directory)
* Respects `--quiet` or similar flags
* Can be disabled via config: `show_hints = false`

## 4. Integration Test Coverage ✅ DONE

**Status:** Complete test coverage for task selection, validation, inheritance, and framework detection

**Completed:**
- ✅ Integration tests for non-existent task errors (spec and watch commands)
- ✅ Unit tests for validateTaskExists() helper
- ✅ Go unit tests for custom task inheritance (TestGetTaskWithOverrides)
  - Tests verify custom tasks inherit mappings, source_dirs, test_glob from auto-detected framework
  - Tests verify custom task name is preserved during merge
- ✅ Integration tests for mixed framework projects (framework_selection_spec.rb)
  - Tests both spec/ and test/ directories exist (8 scenarios)
  - Tests config file `use` setting
  - Tests CLI flag overriding config file
  - Tests single framework detection (spec only, test only, neither)

### Why
* Current minitest tests only use `--use minitest` flag
* No tests for auto-detection with mixed projects
* No tests for config file `use` setting
* Missing coverage for common user workflows

### Tests Needed

#### Auto-detection Tests
* Single `spec/` directory → detects rspec
* Single `test/` directory → detects minitest
* Both directories → defaults to rspec
* Neither directory → defaults to rspec (backward compat)

#### CLI Flag Tests
* `-t rspec` on mixed project runs RSpec (needs mixed fixture)
* `-t minitest` on mixed project runs Minitest (needs mixed fixture)
* ✅ Invalid task name shows helpful error (DONE - see configuration_integration_spec.rb:169)

#### Config File Tests
* `use = "rspec"` makes RSpec default
* `use = "minitest"` makes Minitest default
* CLI flag overrides config file

#### Edge Cases
* Explicit directory pattern: `plur spec test/`
* Custom task names: `plur -t custom-task`
* Both spec and test empty directories

### Test Location
Add to existing `spec/integration/plur_spec/` or create new:
`spec/integration/plur_spec/framework_selection_spec.rb`

## 5. Improve Help Text ✅ DONE

**Status:** IMPLEMENTED - Help text now shows "Task to run (rspec/minitest/custom)"

### Why
* Current text is vague: "Default task configuration to use"
* Doesn't explain what "task" means to new users
* No examples of valid values

### Current
```
--use=""    Default task configuration to use
```

### Proposed
```
-t, --use=TASK    Task to use (rspec/minitest/custom)
```

### Benefits
* Immediately clear what the flag does
* Shows common values (rspec, minitest)
* Hints that custom tasks are possible
* Shorter and more actionable

## 6. Update installation.md ✅ DONE

**Status:** COMPLETED - Now says "Ruby 2.7+ with RSpec or Minitest"

### Why
* Currently says "Ruby 2.7+ with RSpec"
* Implies RSpec is required
* Inconsistent with other docs we just updated

### Change
**From**: `- Ruby 2.7+ with RSpec`
**To**: `- Ruby 2.7+ with RSpec or Minitest`

### Impact
* One line change in docs/installation.md
* Aligns with updated README and getting-started docs

## Implementation Priority

### Phase 1: Quick Wins (< 1 hour)
1. Add `-t` short flag (1 line of code)
2. Update help text (1 line of code)
3. Update installation.md (1 line of docs)
4. Update all docs to show `-t` instead of `--use`

### Phase 2: Test Infrastructure (2-3 hours)
1. Create mixed-framework fixture
2. Add integration tests for framework selection
3. Document fixture in fixtures/README

### Phase 3: UX Polish (1-2 hours)
1. Implement detection hint message
2. Add hint suppression via config
3. Test hint behavior across scenarios

## Success Metrics

* Users with mixed projects can get started in < 5 minutes
* Zero "how do I run RSpec?" support questions
* Test coverage for all framework selection paths
* Documentation shows `-t` flag prominently

## Notes

* All changes maintain backward compatibility
* No breaking changes to existing behavior
* Focus on discoverability and ease of use
* Leverage existing task system (no new concepts)

## References

* parallel_tests README: `references/parallel_tests/Readme.md`
* Current task detection: `plur/internal/task/task.go:314-328`
* Analysis document: `docs/research/spec-test-command-analysis.md`
