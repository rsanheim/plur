# Task Implementation Checklist

This checklist tracks the implementation of the new Task system for Plur, which consolidates test framework configuration, command building, and file mapping into a unified architecture.

## Phase 1: Core Task Infrastructure

* [ ] Create `plur/internal/task/` package directory
* [ ] Define Task struct in `plur/internal/task/task.go` with all fields from design
  * [ ] description field
  * [ ] run field  
  * [ ] source_dirs field
  * [ ] mappings field
  * [ ] ignore_patterns field
* [ ] Implement `BuildCommand` method on Task
  * [ ] Handle RSpec command building
  * [ ] Handle Minitest command building
  * [ ] Support command override from config/CLI
* [ ] Implement `MapFilesToTarget` method on Task
  * [ ] Parse mapping patterns with `{{path}}`, `{{name}}`, `{{file}}` tokens
  * [ ] Return all matching target files
* [ ] Create default RSpec task configuration
* [ ] Create default Minitest task configuration
* [ ] Write `task_test.go` with tests for:
  * [ ] BuildCommand happy path for RSpec
  * [ ] BuildCommand happy path for Minitest
  * [ ] MapFilesToTarget with various patterns
  * [ ] Edge cases for empty/invalid mappings

## Phase 2: TOML Config Integration

* [ ] Add Task struct tags for TOML parsing
* [ ] Extend Kong config to load task definitions from TOML
* [ ] Support loading custom tasks from `[task.NAME]` sections
* [ ] Handle task override logic (CLI > local config > global config > defaults)
* [ ] Write config loading tests:
  * [ ] Load custom task from TOML
  * [ ] Override default task settings
  * [ ] Invalid task configuration handling

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