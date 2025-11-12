# Watch Mappings Checklist

This is the implementation checklist for watch mappings overhaul.
AI agents should use this checklist as they iterate and work on the
watch mapping overhaul.

See [watch-mappings-prd.md](watch-mappings-prd.md) for the PRD.

## Status Summary

**Phase 1 (Go Code & Tests):** ✅ COMPLETE
**Phase 2 (Ruby Specs):** ❌ FAILED - Watch tests now broken (4 failures)
**Phase 3 (Documentation):** ✅ COMPLETE
**Phase 4 (Verification):** ❌ FAILED - Watch mode broken

**Overall:** NOT ready for merge. Mapping logic removed but broke watch functionality.

**Current Issues (as of 2025-11-12):**
* 4 watch-related test failures in integration specs
* Error: "no directories to watch found in watch mappings"
* Watch mode appears to be broken after mapping removal
* See task-to-job-checklist.md for full test failure details

## Phase 1 - Remove all mapping related Go code and tests

* [x] Remove Mappings field from TaskConfig struct in main.go
* [x] Remove mapping conversion in AfterApply() in main.go
* [x] Remove mapping merging in mergeTaskConfig() in main.go
* [x] Remove MappingRule struct and Task.Mappings field from task.go
* [x] Remove MapFilesToTarget() and applyMapping() methods from task.go
* [x] Remove default mappings from NewRSpecTask() and NewMinitestTask()
* [x] Remove shouldWatchFile() and mapping logic from watch.go
* [x] Replace watch_find.go with minimal stub (now just shows placeholder message)
* [x] Remove all mapping tests from task_test.go (TestMapFilesToTarget_*, TestApplyMapping_*)
* [x] Remove Mappings assertions from main_test.go
* [x] Remove unused doublestar imports from task.go and watch.go
* [x] Entire Go build and test suite PASSES

## Phase 2 - Selectively remove rspec integration specs testing watch mappings

* [x] Remove watch_mapping_integration_spec.rb (entire file was about mappings)
* [x] Remove watch_find_spec.rb (entire file was about mapping discovery)
* [x] Remove file_mapper_spec.rb (entire file was about mapping logic)
* [x] Remove CI-skipped mapping tests from watch_integration_spec.rb:
  * "maps nested lib files correctly" test
  * "Rails-style mappings" describe block with all its tests
* [x] Update remaining watch specs to test new file-change-only behavior:
  * Updated watch_spec.rb to check for "File changed:" instead of "running:"
  * Updated watch_integration_spec.rb tests to expect file change reporting, not test execution
* [x] Entire test suite PASSES (215 examples, 0 failures, 4 pending)


## Phase 3 - Remove all mapping related docs, config examples, etc

* [x] Remove docs/architecture/file-mapping.md (entire 251-line file about mappings)
* [x] Remove mapping sections from examples/plur.toml.example (lines 70-101)
* [x] Clean up mapping references in docs/configuration.md
  * Removed "How to map source files to test files" from task overview
  * Removed "and how to map source files to test files" from task description
  * Removed "(mapped to corresponding specs)" from watch mode file watching
* [x] Clean up fixture configuration files
  * Removed [[task.rspec.mappings]] blocks from fixtures/projects/config-test/with-tasks.toml
  * Removed [[task.minitest.mappings]] blocks from fixtures/projects/config-test/with-tasks.toml
  * Removed [[task.doctor-test.mappings]] blocks from fixtures/projects/config-test/doctor-test.toml
  * Updated comment in with-tasks.toml to remove mapping inheritance reference
* [x] Update watch command help text
  * Removed mapping-related fields from WatchFindCmd struct in watch_find.go
  * Updated watch find help text in main.go from "Find and suggest mappings" to "Placeholder"
* [x] Remaining 'mapping' references are only in historical contexts (git history, this checklist, PRD)

## Phase 4 - Verify the watch commands still work as shells to build back on top of

* [x] `watch run` starts watchers for [src,lib,spec,test] in current project and reports file changes
  * Shows "File changed: <path>" when files are modified
  * Displays interactive prompt with commands ([Enter], reload, exit, Ctrl-C)
  * Respects --timeout flag for testing
* [x] `watch find` shows placeholder message:
  * "The 'watch find' functionality is currently being rebuilt."
  * "This feature will return in a future release with a simpler, cleaner design."
* [x] Logging output is helpful and clean
  * Debug logging shows file events
  * Info logging shows configuration and file changes
  * Verbose and debug flags work as expected

## Phase 5 - Lessons Learned

**What Went Wrong:**
1. **Incomplete migration** - Removed mapping system but didn't fully replace watch directory detection
2. **Insufficient testing during removal** - Tests passing after Phase 1/2 but broke later during Task→Job migration
3. **Coupling issues** - Watch mode was more tightly coupled to mapping system than understood
4. **Missing replacement implementation** - Job-based watch directory derivation not fully implemented

**Impact:**
* Watch mode broken with "no directories to watch found in watch mappings" error
* 4 integration test failures
* Error messages still reference removed "watch mappings" system

**What Needs Fixing:**
* Implement proper watch directory derivation from Job definitions
* Fix error messages to reflect new architecture (no more "watch mappings")
* Ensure WatchMapping.SourceDir() properly derives from job config
* Add tests specifically for watch directory detection in new system

**Lessons for Future Refactors:**
1. Don't mark phases "COMPLETE" until full test suite passes
2. When removing old system, ensure replacement is fully implemented first
3. Run integration tests after EVERY phase, not just at the end
4. Keep better connection between what code does and what tests verify
