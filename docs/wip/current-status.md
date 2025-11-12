# Current Status: Task-to-Job Migration
**Branch:** `new-cli-who-dis`
**Date:** 2025-11-12
**Status:** ❌ INCOMPLETE - 10 test failures blocking completion

## Quick Summary

The Task-to-Job migration has completed all implementation phases (6/6) but broke several key features. While the architectural changes are in place and unit tests pass, integration tests reveal critical regressions.

**What Works:**
* ✅ Go unit tests (all passing)
* ✅ Framework selection (8/8 tests passing)
* ✅ Fixture tests (bin/rake test:default_ruby)
* ✅ Task package deleted (~226 lines removed)
* ✅ Autodetection extracted to dedicated package (~260 lines removed)
* ✅ Basic `plur spec` functionality (simple cases work)

**What's Broken:**
* ❌ Watch mode (4 test failures)
* ❌ -C/--change-dir flag (3 test failures)
* ❌ Custom job command quoting (1 test failure)
* ❌ RSpec output formatter (1 test failure)
* ❌ Error messaging (1 test failure)

## Test Failure Details

**Total: 10 failures out of 219 examples, plus 4 pending**

### Category 1: Watch Mode (4 failures)
**Files:** `configuration_integration_spec.rb`, `command_specific_config_spec.rb`

```
1. configuration_integration_spec.rb:103 - Watch mode configuration
2. command_specific_config_spec.rb:47 - Watch debounce configuration
3. command_specific_config_spec.rb:57 - Watch debounce setting
4. All watch tests fail with: "no directories to watch found in watch mappings"
```

**Root Cause:** Watch directory derivation broken after mapping removal. Job-based watch directory detection not fully implemented.

**Impact:** `plur watch` command is broken

### Category 2: -C Flag (3 failures)
**File:** `change_dir_config_spec.rb:113,121,129`

```
All three formats fail:
* plur -C fixtures/projects/config-test spec
* plur -C=fixtures/projects/config-test spec
* plur --change-dir=fixtures/projects/config-test spec
```

**Root Cause:** Config loading or directory change logic broken when using -C flag

**Impact:** Cannot run plur from outside project directory - critical regression

### Category 3: Job Command Building (1 failure)
**File:** `configuration_integration_spec.rb:167`

```
Expected: echo 'CUSTOM TASK:'  (with single quotes preserved)
Actual:   echo CUSTOM TASK:    (quotes stripped)
```

**Root Cause:** Job.BuildCmd() not preserving quoted strings in commands

**Impact:** Custom jobs with quoted arguments won't work correctly

### Category 4: Error Messaging (1 failure)
**File:** `configuration_integration_spec.rb:196`

```
When using nonexistent job with watch command:
Expected: "job 'nonexistent' not found"
Actual:   "no directories to watch found in watch mappings"
```

**Root Cause:** Error handling hitting watch directory issue before job validation

**Impact:** Confusing error messages referencing removed "watch mappings" system

### Category 5: Output Formatter (1 failure)
**File:** `output_performance_spec.rb:32`

```
Expected: RSpec dots/F pattern matching [.F]+
Actual:   Empty string (no output)
```

**Root Cause:** RSpec formatter not producing output or output aggregation broken

**Impact:** Users may not see test progress indicators

## What Needs Fixing

### Priority 1: Critical Regressions
1. **Fix -C flag** (3 tests) - Basic functionality broken
2. **Fix watch mode** (4 tests) - Major feature broken
3. **Fix job command quoting** (1 test) - Custom jobs broken

### Priority 2: Output & Error Handling
4. **Fix output formatter** (1 test) - User experience issue
5. **Fix error messages** (1 test) - Confusing messaging

### Priority 3: Cleanup
6. **Update doctor golden test** - Environment-specific failure (minor)
7. **Resolve pending tests** - 4 pending tests to address or document

## Documentation Status

All WIP documentation has been updated to reflect actual current state:

* ✅ **task-to-job-checklist.md** - Updated with test failures, realistic completion status
* ✅ **watch-mappings-checklist.md** - Updated to show Phase 2/4 failed, added lessons learned
* ✅ **watch-mappings-prd.md** - Filled in empty Questions section with unresolved issues
* ✅ **current-status.md** - This file, comprehensive current state summary

## Architectural Changes Completed

Despite test failures, the architectural refactoring is complete:

1. **Task package deleted** - `internal/task/` removed entirely
2. **Job package created** - `plur/job/job.go` with unified Job model
3. **Autodetect package** - `plur/autodetect/` for framework detection
4. **Passthrough parser** - `plur/passthrough/parser.go` for custom jobs
5. **Framework builders** - RSpec and Minitest command builders in runner.go
6. **Job-based discovery** - FindFilesFromJob(), ExpandPatternsFromJob()
7. **Spec command** - Uses Job instead of Task throughout
8. **Watch mappings removed** - Old mapping system deleted (but broke watch)

## Next Steps

1. **Fix the 10 test failures** - Address each category systematically
2. **Verify watch directory derivation** - Implement proper Job → watch dirs logic
3. **Test thoroughly** - Run `bin/rake test` after each fix
4. **Update error messages** - Remove "watch mappings" references
5. **Consider Phase 6.5** - Decide whether to implement or defer visibility enhancements

## Branch Information

```
Current Branch: new-cli-who-dis
Latest Commit: c108e51b "fix integration specs"
Main Branch: 3 commits behind
```

**Files Changed:** ~61 files
**Insertions:** +5777 lines
**Deletions:** -1291 lines
**Net Change:** +4486 lines (mostly documentation)

## Key Learnings

1. **Don't mark phases "COMPLETE" until tests pass** - Several checklists claimed completion while tests were failing
2. **Run integration tests after EVERY phase** - Unit tests passing doesn't mean integration works
3. **Ensure replacements are fully implemented** - Removing old system (watch mappings) broke functionality because replacement wasn't complete
4. **Keep documentation honest** - False completion markers hid real issues

## Related Documents

* `task-to-job-checklist.md` - Full implementation checklist with phase details
* `task-to-job-migration-plan.md` - Original design document and rationale
* `watch-mappings-checklist.md` - Watch mappings removal checklist (broke watch)
* `watch-mappings-prd.md` - Watch mappings removal PRD with unresolved questions
* `autodetection-design-phase6.md` - Autodetection design (Phase 6.5 not started)
