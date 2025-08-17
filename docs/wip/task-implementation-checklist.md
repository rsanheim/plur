# Task Implementation Checklist

This checklist tracks the implementation of the new Task system for Plur, which consolidates test framework configuration, command building, and file mapping into a unified architecture.

## Prerequisites ✅ COMPLETED

* [x] **Move GlobalConfig to shared package** - Created `plur/config` package to hold shared configuration types
  * [x] Move GlobalConfig, ConfigPaths, TestFramework types to `plur/config/`
  * [x] Update internal/task to use shared config.GlobalConfig (eliminating duplication)
  * [x] Fix variable shadowing issues throughout codebase
  * [x] Update all imports and test files
  * [x] Verify all tests pass and compilation works

This prerequisite eliminates the config duplication between main and internal/task packages, providing a clean foundation for Task integration.

## Phase 1: Core Task Infrastructure ✅ COMPLETED

* [x] Create `plur/internal/task/` package directory
* [x] Define Task struct in `plur/internal/task/task.go` with all fields from design
  * [x] description field
  * [x] run field  
  * [x] source_dirs field
  * [x] mappings field
  * [x] ignore_patterns field
* [x] Implement `BuildCommand` method on Task
  * [x] Handle RSpec command building
  * [x] Handle Minitest command building
  * [x] Support command override from config/CLI
* [x] Implement `MapFilesToTarget` method on Task
  * [x] Parse mapping patterns with `{{path}}`, `{{name}}`, `{{file}}` tokens
  * [x] Return all matching target files
* [x] Create default RSpec task configuration
* [x] Create default Minitest task configuration
* [x] Write `task_test.go` with tests for:
  * [x] BuildCommand happy path for RSpec
  * [x] BuildCommand happy path for Minitest
  * [x] MapFilesToTarget with various patterns
  * [x] Edge cases for empty/invalid mappings
* [x] **Simplify MappingRule struct** - Remove unnecessary fields
  * [x] Remove Description field (not needed)
  * [x] Remove Type field (all same type)  
  * [x] Remove Priority field (simple order-based processing)
  * [x] Remove hard-coded GetTestPattern/GetTestSuffix methods (defeats data-driven design)

## Phase 2: TOML Config Integration ✅ COMPLETED

* [x] **Use Kong's native features for TOML parsing** - Eliminated all custom TOML loading code
  * [x] Create TaskConfig struct with proper TOML tags for Kong parsing
  * [x] Add `Task map[string]TaskConfig` field to PlurCLI for Kong to populate automatically
  * [x] Convert TaskConfig to task.Task in AfterApply() hook for architecture compatibility
  * [x] Remove entire loadTaskConfigurations() function (100% less custom code)
* [x] Support loading custom tasks from `[task.NAME]` sections via Kong's map parsing
* [x] Handle task override logic (CLI > TOML config > defaults) in getTaskWithOverrides()
* [x] Verify config loading works:
  * [x] Successfully load custom task from with-tasks.toml
  * [x] Test `--type=custom` uses correct command (`echo 'CUSTOM TASK:'`)
  * [x] Maintain all existing functionality and fallback behavior

**Key Achievement**: Zero custom TOML loading code - Kong handles everything natively!

## Phase 3: Replace CommandBuilder (Breaking Change) ✅ COMPLETED

* [x] Migrate RSpecCommandBuilder logic to Task.BuildCommand
* [x] Migrate MinitestCommandBuilder logic to Task.BuildCommand  
* [x] Update execution.go to use Task instead of CommandBuilder
* [x] Update all references to CommandBuilder throughout codebase
* [x] DELETE command_builder.go entirely
* [x] DELETE minitest/command.go command building logic
* [x] Verified with full test suite - 235 examples, 0 failures

## Phase 4: Consolidate Test Discovery (Breaking Change) ✅ COMPLETED

* [x] Move `getTestFileSuffix` logic into Task
* [x] Move `getDefaultPattern` logic into Task
* [x] Update glob.go FindTestFiles to use Task
* [x] Update glob.go ExpandGlobPatterns to use Task
* [x] Simplify DetectTestFramework to return appropriate Task
* [x] DELETE getTestFileSuffix function
* [x] DELETE getDefaultPattern function
* [x] Write discovery tests:
  * [x] RSpec pattern discovery
  * [x] Minitest pattern discovery
  * [x] Mixed framework detection

**Key Achievement**: All test discovery is now data-driven through Task configuration!

## Phase 4.5: Simplify Test Discovery Fields ✅ COMPLETED

* [x] Rename `TestPattern` field to `TestGlob` in Task struct for clearer naming
* [x] Remove `TestSuffix` field from Task struct (redundant with TestGlob)
* [x] Update `GetTestSuffix()` method to derive suffix from TestGlob pattern
  * [x] Extract suffix from patterns like `"spec/**/*_spec.rb"` → `"_spec.rb"`
  * [x] Extract suffix from patterns like `"test/**/*_test.rb"` → `"_test.rb"`
* [x] Update TOML configuration to use `test_glob` instead of `test_pattern`
* [x] Update all references to TestPattern/TestSuffix throughout codebase
* [x] Add tests for suffix extraction from glob patterns
* [x] Verify all existing functionality works with derived suffixes

**Key Achievement**: Eliminate redundant TestSuffix field by deriving it from TestGlob!

## Phase 4.6: Replace --type Flag with --use Flag ✅ COMPLETED

* [x] Replace `--type` flag with `--use` flag throughout codebase
* [x] Add `Use` field to PlurCLI struct for global task configuration
* [x] Add `Use` field to SpecCmd (replacing Type field)
* [x] Add `Use` field to WatchRunCmd (replacing Type field)
* [x] Implement priority-based task selection (CLI --use > global use > auto-detect)
* [x] Remove all Type fields and GetFramework methods from command structs
* [x] Remove ParseFrameworkType function entirely
* [x] Add getFrameworkFromTask helper to convert task names to TestFramework enum
* [x] Fix getTaskWithOverrides to properly merge TOML configs with defaults
* [x] Update all integration tests to use --use instead of --type
* [x] Simplify ExpandGlobPatterns to leverage task TestGlob directly
* [x] Remove unused FindSpecFiles() and FindMinitestFiles() functions
* [x] Clean up main_test.go (framework validation no longer needed)

**Key Achievement**: Clean --use flag interface with proper task configuration merging!

## Phase 5: Watch Consolidation

* [ ] Update watch FileMapper to use Task.MapFilesToTarget
* [ ] Update watch_find.go to use Task mappings
* [ ] Remove duplicate mapping logic from watch_find.go:
  * [ ] DELETE detectPatternFromAlternative function
  * [ ] DELETE createRuleForFile function
  * [ ] Replace with Task-based mapping
* [ ] Ensure watch and watch find use identical mapping logic
* [ ] Update watch/mapping_rules.go to work with Task
* [ ] Write unified mapping tests:
  * [ ] Test lib -> spec mappings
  * [ ] Test app -> spec mappings
  * [ ] Test direct spec file mappings
  * [ ] Test custom mapping patterns

## Phase 6: Final Cleanup (Breaking Changes)

* [ ] DELETE SpecCmd struct and all its methods
* [ ] Update main.go to use Task directly for `plur spec` command
* [ ] Remove all deprecated framework detection functions
* [ ] Update all integration tests to use new Task system
* [ ] Update documentation:
  * [ ] Update CLAUDE.md with new architecture
  * [ ] Update example TOML configs
  * [ ] Document task configuration options
* [ ] Write end-to-end integration tests:
  * [ ] Full RSpec run with custom task
  * [ ] Full Minitest run with custom task
  * [ ] Watch mode with custom mappings

## Notes

- **No backwards compatibility** - we're making clean breaks throughout
- Token syntax will use `{{}}` to avoid conflicts with doublestar glob patterns
- Framework detection based only on directory structure (spec/ vs test/), no Gemfile inspection
- All mapping logic will be centralized in Task, eliminating current duplication