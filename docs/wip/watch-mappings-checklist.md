# Watch Mappings Checklist

This is the implementation checklist for watch mappings overhaul.
AI agents should use this checklist as they iterate and work on the
watch mapping overhaul.

See [watch-mappings-prd.md](watch-mappings-prd.md) for the PRD.

## Status Summary

**Phase 1 (Go Code & Tests):** ✅ COMPLETE
**Phase 2 (Ruby Specs):** ✅ COMPLETE
**Phase 3 (Documentation):** ✅ COMPLETE
**Phase 4 (Verification):** ✅ COMPLETE

**Overall:** Ready for merge. All mapping logic removed, all documentation cleaned, all tests passing.

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
