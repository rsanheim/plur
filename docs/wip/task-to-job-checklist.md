# Task-to-Job Migration Checklist

## Overview
Consolidating Task and Job concepts into a unified Job model for both parallel execution (`plur spec`) and watch mode (`plur watch`).

## Phase 1: Move Job and Extend ✓

- [x] Create plur/job/job.go with extended Job struct (add TargetPattern field)
- [x] Move BuildJobCmd and BuildJobAllCmd functions to plur/job/job.go
- [x] Add Job helper methods (GetTargetPattern, GetTargetSuffix, CreateParser, IsMinitestStyle)
- [x] Create plur/passthrough/parser.go for custom jobs
- [x] Move WatchMapping and MultiString to watch/processor.go
- [x] Delete watch/job.go entirely
- [x] Update watch/defaults.toml with target_pattern field
- [x] Update all imports from watch.Job to job.Job across codebase
- [x] Update BuildJobCmd call sites to use array syntax
- [x] Fix watch package tests (defaults_test.go, processor_test.go)
- [x] Verify build and watch mode functionality

**Status**: Complete - All tests passing, build successful

## Phase 2: Job-Based File Discovery ✓

- [x] Add FindFilesFromJob() to plur/glob.go
- [x] Add ExpandPatternsFromJob() to plur/glob.go
- [x] Verify build still works

**Status**: Complete - Functions added and tested

## Phase 3: Switch SpecCmd to Use Job ✓

- [x] Add insertBeforeFiles() helper to runner.go
- [x] Add buildRSpecCommand() to runner.go
- [x] Add buildMinitestCommand() to runner.go
- [x] Update main.go SpecCmd to load Job instead of Task
- [x] Update execution.go TestExecutor to use Job
- [x] Update runner.go RunTestFiles to use Job and framework wrappers
- [x] Update result.go to use Job instead of Task
- [x] Update doctor.go to use job.WatchDirs
- [x] Run full integration test suite for plur spec

**Status**: Complete - Core Job integration done, 25 test failures expected (testing old Task config system)

## Phase 4: Remove Task Dependencies ✓

- [x] Remove FindTestFiles and ExpandGlobPatterns from glob.go
- [x] Update result.go to use job.IsMinitestStyle()
- [x] Remove TaskConfig from main.go config parsing
- [x] Remove mergeTaskConfig and getTaskWithOverrides from main.go
- [x] Update watch.go to use job.WatchDirs instead of task.SourceDirs
- [x] Update watch/mapping_rules.go to use watch/defaults detection
- [x] Run framework-specific command tests

**Status**: Complete - Old Task configuration system removed

## Phase 4.5: Remove Job.WatchDirs (Cleanup) ✓

- [x] Remove WatchDirs field from Job struct
- [x] Update watch.go to derive directories from WatchMapping.SourceDir()
- [x] Update doctor.go to use watch mappings for directory info
- [x] Clean up watch_dirs from defaults.toml
- [x] Simplify deduplication logic (sort + slices.Compact)
- [x] Update tests for alphabetical directory ordering

**Status**: Complete - Watch directories now derived from mappings, single source of truth

## Phase 5: Delete Task Package ✓

- [x] Delete internal/task/task.go and internal/task/task_test.go
- [x] Verify no Task imports remain in codebase
- [x] Run full test suite to verify Task deletion

**Status**: Complete - Task package fully removed, all Go tests passing

## Phase 6: Unify Autodetection ✓

### Analysis & Design ✓
- [x] Deeply analyze current autodetection usage patterns
  - [x] Map all call sites: where is autodetection used? (spec command, watch command, doctor, etc.)
  - [x] Identify responsibility: is autodetection just for watching, or broader?
  - [x] Evaluate coupling: should autodetection be split by concern?
  - [x] Created comprehensive design doc: docs/wip/autodetection-design-phase6.md
- [x] Design clear, non-magical autodetection
  - [x] Documented detection strategies for Ruby, Go, JS/TS, Python, Rust, Zig
  - [x] Designed visibility improvements for future implementation
  - [x] Planned enhanced plur doctor output

### Implementation (Package Extraction) ✓
- [x] Create dedicated plur/autodetect/ package
- [x] Move defaults.go from watch/ to autodetect/
- [x] Move defaults.toml from watch/ to autodetect/
- [x] Move defaults_test.go from watch/ to autodetect/
- [x] Remove unused watch/mapping_rules.go (~86 lines)
- [x] Remove unused watch/mapping_rules_test.go (~174 lines)
- [x] Update package imports in main.go
- [x] Update package imports in watch.go
- [x] Update package imports in doctor.go
- [x] Improve framework selection logic (smart spec/ vs test/ detection)
- [x] Make autodetection permissive for backward compatibility
- [x] Update framework selection tests
- [x] Run autodetection tests for both spec and watch modes

### Future Enhancements (Phase 6.5 - Not Started)
- [ ] Add clear logging/output for autodetection decisions
- [ ] Enhance plur doctor to show detection reasoning
- [ ] Add plur config:show command
- [ ] Add plur config:init --from-autodetect flag

**Status**: Complete - Autodetection extracted to dedicated package, ~260 lines removed

## Final Verification

- [ ] Run final full test suite (bin/rake test)
- [ ] Verify all success criteria are met

## Success Criteria

- [x] ~486 lines removed (internal/task/ + unused watch/mapping_rules files)
- [x] No Task references in codebase
- [x] Dedicated autodetection package (plur/autodetect/)
- [x] Simplified configuration parsing (no task merging)
- [x] All existing tests pass (Go tests + framework selection)
- [x] `plur spec` works with Jobs
- [x] `plur watch` continues to work
- [x] Custom jobs can be defined
- [x] Autodetection provides correct defaults
- [x] RSpec and Minitest parsers work correctly

## Current Progress

**Completed**: 56/60 tasks (93%)
**Phases Complete**: 6/6 (Phase 6.5 enhancements remain)
**Build Status**: ✅ Passing
**Go Tests**: ✅ All passing
**Spec Mode**: ✅ Working (with Job autodetection)
**Watch Mode**: ✅ Working (derives directories from watch mappings)
**Framework Selection**: ✅ All tests passing (8/8)

## Key Changes Made

1. **New job package** (`plur/job/job.go`):
   - Job struct with TargetPattern field (WatchDirs removed - derived from mappings)
   - BuildJobCmd/BuildJobAllCmd for command building
   - Helper methods for pattern extraction and parser creation

2. **Passthrough parser** (`plur/passthrough/parser.go`):
   - Default parser for custom (non-RSpec/Minitest) jobs
   - Forwards output directly without parsing

3. **Framework-specific builders** (`plur/runner.go`):
   - `insertBeforeFiles()` - Insert args before file paths
   - `buildRSpecCommand()` - Add formatter and color flags
   - `buildMinitestCommand()` - Handle multi-file require pattern

4. **Job-based file discovery** (`plur/glob.go`):
   - `FindFilesFromJob()` - Discover files by target pattern
   - `ExpandPatternsFromJob()` - Expand globs with job suffix

5. **Updated defaults** (`watch/defaults.toml`):
   - Added target_pattern to rspec/minitest jobs
   - Added watch_dirs to jobs

6. **SpecCmd Job integration** (`plur/main.go`):
   - SpecCmd.Run() now uses `watch.GetAutodetectedDefaults()` for autodetection
   - Loads jobs from `parent.Job` config map with fallback to autodetected jobs
   - Prioritizes rspec/minitest over other jobs like rubocop
   - Validates jobs have target_pattern before use

7. **TestExecutor Job support** (`plur/execution.go`):
   - TestExecutor.currentJob field replaces currentTask
   - Dry-run uses framework-specific command builders
   - BuildTestSummary and PrintResults use Job

8. **Result formatters** (`plur/result.go`):
   - BuildTestSummary and PrintResults accept Job
   - Uses currentJob.IsMinitestStyle() and currentJob.CreateParser()

9. **Doctor command** (`plur/doctor.go`):
   - Uses job autodetection instead of task.DetectFramework()
   - Shows watch directories from autodetected jobs

10. **Task Dependencies Removed** (Phase 4):
   - Removed FindTestFiles and ExpandGlobPatterns from glob.go
   - Removed TaskConfig struct and Task/Tasks fields from PlurCLI
   - Removed mergeTaskConfig, getTaskWithOverrides, and validateTaskExists
   - Updated WatchRunCmd to use Job autodetection
   - Updated watch.go to use job.WatchDirs instead of task.SourceDirs
   - Updated watch/mapping_rules.go to use watch.AutodetectProfile()
   - Removed all imports of internal/task package from main plur code

11. **Job.WatchDirs Removed** (Phase 4.5 Cleanup):
   - Removed WatchDirs field from Job struct entirely
   - Watch directories now derived from WatchMapping.SourceDir()
   - Updated watch.go and doctor.go to use watch mappings
   - Simplified deduplication with sort + slices.Compact
   - Single source of truth: directories come from watch mapping patterns
   - Removed watch_dirs from defaults.toml job definitions

12. **Task Package Deleted** (Phase 5):
   - Deleted internal/task/task.go (~145 lines)
   - Deleted internal/task/task_test.go (~81 lines)
   - Removed internal/task/ directory entirely
   - Verified no remaining imports of internal/task package
   - All Go tests passing after deletion
   - Total lines removed: ~226 lines

13. **Autodetection Package Extracted** (Phase 6):
   - Created dedicated plur/autodetect/ package
   - Moved defaults.go, defaults.toml, defaults_test.go from watch/ to autodetect/
   - Removed unused watch/mapping_rules.go (~86 lines)
   - Removed unused watch/mapping_rules_test.go (~174 lines)
   - Updated imports in main.go, watch.go, doctor.go to use autodetect package
   - Improved framework selection: smart detection based on spec/ vs test/ directories
   - Made autodetection more permissive (Gemfile OR spec/ OR test/ OR lib/)
   - Fixed framework selection tests for correct behavior
   - Benefits: Autodetection no longer coupled to watch package, clearer dependencies
   - Total lines removed: ~260 lines
   - Created comprehensive design doc: docs/wip/autodetection-design-phase6.md

## Next Steps

Phase 6 complete! Future enhancements (Phase 6.5):
- Add clear logging/output for autodetection decisions
- Enhance plur doctor to show detection reasoning and available jobs
- Add plur config:show command to display effective configuration
- Add plur config:init --from-autodetect flag to export defaults
