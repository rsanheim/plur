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

## Phase 3: Replace CommandBuilder (Breaking Change)

* [ ] Migrate RSpecCommandBuilder logic to Task.BuildCommand
* [ ] Migrate MinitestCommandBuilder logic to Task.BuildCommand  
* [ ] Update execution.go to use Task instead of CommandBuilder
* [ ] Update all references to CommandBuilder throughout codebase
* [ ] DELETE command_builder.go entirely
* [ ] DELETE minitest/command.go command building logic
* [ ] Update execution_test.go to test with Task

## Phase 4: Consolidate Test Discovery (Breaking Change)

* [ ] Move `getTestFileSuffix` logic into Task
* [ ] Move `getDefaultPattern` logic into Task
* [ ] Update glob.go FindTestFiles to use Task
* [ ] Update glob.go ExpandGlobPatterns to use Task
* [ ] Simplify DetectTestFramework to return appropriate Task
* [ ] DELETE getTestFileSuffix function
* [ ] DELETE getDefaultPattern function
* [ ] Write discovery tests:
  * [ ] RSpec pattern discovery
  * [ ] Minitest pattern discovery
  * [ ] Mixed framework detection

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