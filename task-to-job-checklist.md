# Task-to-Job Migration Checklist

## Overview
Consolidating Task and Job concepts into a unified Job model for both parallel execution (`plur spec`) and watch mode (`plur watch`).

## Phase 1: Move Job and Extend ✓

- [x] Create plur/job/job.go with extended Job struct (add TargetPattern and WatchDirs fields)
- [x] Move BuildJobCmd and BuildJobAllCmd functions to plur/job/job.go
- [x] Add Job helper methods (GetTargetPattern, GetTargetSuffix, CreateParser, IsMinitestStyle)
- [x] Create plur/passthrough/parser.go for custom jobs
- [x] Move WatchMapping and MultiString to watch/processor.go
- [x] Delete watch/job.go entirely
- [x] Update watch/defaults.toml with target_pattern and watch_dirs fields
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

## Phase 3: Switch SpecCmd to Use Job

- [x] Add insertBeforeFiles() helper to runner.go
- [x] Add buildRSpecCommand() to runner.go
- [x] Add buildMinitestCommand() to runner.go
- [ ] Update main.go SpecCmd to load Job instead of Task
- [ ] Update execution.go TestExecutor to use Job
- [ ] Update runner.go RunTestFiles to use Job and framework wrappers
- [ ] Update doctor.go to use job.WatchDirs
- [ ] Run full integration test suite for plur spec

**Status**: In Progress - Helper functions complete, core migration pending

## Phase 4: Remove Task Dependencies

- [ ] Remove FindTestFiles and ExpandGlobPatterns from glob.go
- [ ] Update result.go to use job.IsMinitestStyle()
- [ ] Remove TaskConfig from main.go config parsing
- [ ] Remove mergeTaskConfig and getTaskWithOverrides from main.go
- [ ] Update watch.go to use job.WatchDirs instead of task.SourceDirs
- [ ] Update watch/mapping_rules.go to use watch/defaults detection
- [ ] Run framework-specific command tests

**Status**: Not Started

## Phase 5: Delete Task Package

- [ ] Delete internal/task/task.go and internal/task/task_test.go
- [ ] Verify no Task imports remain in codebase
- [ ] Run full test suite to verify Task deletion

**Status**: Not Started

## Phase 6: Unify Autodetection

- [ ] Consolidate autodetection to use watch/defaults.go everywhere
- [ ] Remove duplicate framework detection logic from main.go
- [ ] Run autodetection tests for both spec and watch modes

**Status**: Not Started

## Final Verification

- [ ] Run final full test suite (bin/rake test)
- [ ] Verify all success criteria are met

## Success Criteria

- [ ] ~226 lines removed (internal/task/ directory deleted)
- [ ] No Task references in codebase
- [ ] Single autodetection system (watch/defaults.go)
- [ ] Simplified configuration parsing (no task merging)
- [ ] All existing tests pass
- [ ] `plur spec` works with Jobs
- [ ] `plur watch` continues to work
- [ ] Custom jobs can be defined
- [ ] Autodetection provides correct defaults
- [ ] RSpec and Minitest parsers work correctly

## Current Progress

**Completed**: 17/38 tasks (45%)
**Phases Complete**: 2/6
**Build Status**: ✅ Passing
**Go Tests**: ✅ All passing
**Watch Mode**: ✅ Working

## Key Changes Made

1. **New job package** (`plur/job/job.go`):
   - Job struct with TargetPattern and WatchDirs
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

## Next Steps

Continue with Phase 3.4: Update main.go SpecCmd to load Job instead of Task, then proceed through remaining phases to complete the migration.
