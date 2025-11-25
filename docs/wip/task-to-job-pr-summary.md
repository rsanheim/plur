# Task-to-Job Migration PR Summary

Timestamp: 2025-11-21T10:15:00Z
Author: Collette

## Overview
This PR successfully completes the architectural migration from a dual Task/Job system to a unified Job model, removing 486 lines of legacy code while improving test coverage and configuration clarity.

## Key Changes

### Unified Job Model
* **Before**: Separate Task and Job abstractions with overlapping responsibilities
* **After**: Single Job type handling both parallel execution and watch mode
* **Impact**: -486 lines of Task code, cleaner mental model

### Configuration Structure
```toml
# Old (task-based)
[task.rspec]
cmd = ["bundle", "exec", "rspec"]
target = "spec/**/*_spec.rb"

# New (job-based)
[job.rspec]
cmd = ["bundle", "exec", "rspec", "{{target}}"]
target_pattern = "spec/**/*_spec.rb"
```

### Test Results
* **All 217 tests passing** (previously 10 failures)
* **No regressions** in parallel execution performance
* **4 pending specs** for future features

## What's Working Well

### Clean Package Separation
* `plur/job/` - Core job management
* `plur/autodetect/` - Framework detection
* `plur/watch/` - File watching logic
* Clear boundaries, focused responsibilities

### Smart Defaults
* Convention-based patterns (jobs named "rspec" get RSpec patterns)
* Framework autodetection when no config exists
* Backward-compatible behavior for existing workflows

### Template System
* `{{target}}` substitution in commands
* Token-based watch mappings for flexible file transformations
* Supports both simple and complex use cases

## Critical Issues to Address

### 1. Overengineering in Core Systems
The Job templating and watch token systems add complexity without proportional value:
* `{{target}}` replacement logic spans 30+ lines for a simple substitution
* Watch tokens use full text/template package for path mapping
* **Recommendation**: Simplify to positional arguments or direct appending

### 2. Function Length
* `runWatchWithConfig`: 294 lines (plur/watch.go:110-404)
* `DetectFramework`: 68 lines of branching (autodetect/defaults.go:110-178)
* **Recommendation**: Decompose into smaller, focused functions

### 3. Documentation Debt
* Main docs still reference "task" configuration
* No migration guide for config changes
* Missing examples for common patterns
* **Recommendation**: Update docs before release

### 4. Unnecessary Abstractions
* Deep configuration copying on every default access
* **Recommendation**: Inline or remove these abstractions

## Performance Considerations

### Current Issues
* Configuration validation on every file change in watch mode
* Deep copying of default profiles on each access
* Redundant pattern compilation

### Quick Wins
* Validate configuration once at startup
* Cache compiled patterns
* Use original defaults without copying

## Architecture Assessment

### Strengths
* **Successful unification** of Task/Job concepts
* **Clean break** from legacy code (no deprecated paths)
* **Strong test coverage** validating the migration

### Opportunities
* **Reduce abstraction layers** - Many interfaces for single implementations
* **Simplify configuration** - Too many ways to achieve the same result
* **Improve visibility** - Users can't see what plur detected/decided

## Recommended Next Steps

### Immediate (Before Release)
1. Update documentation to reflect Job configuration
2. Simplify `{{target}}` templating to basic string append
3. Fix performance issues (validation, copying)

### Short Term
1. Decompose large functions (especially watch.go)
2. Add visibility commands (`plur config:show`, enhanced doctor)

### Consider for Future
1. Replace autodetection with required explicit config
2. Remove watch token templating in favor of simple globs
3. Collapse job package into main runner code

## Migration Impact

Since you're the only user, no migration path needed. This is an opportunity to:
* Break compatibility freely
* Choose the simplest implementation
* Remove all backward compatibility code

## Bottom Line

The migration succeeds in its primary goal of unifying the Task/Job model. The code is functional and well-tested. However, it's overengineered for a single-user CLI tool. The path forward should focus on aggressive simplification: removing abstractions, requiring explicit configuration, and choosing boring, obvious implementations over clever flexibility.

**Grade: B+** - Solid execution of the migration, but missed opportunities for simplification.

The PR is ready to merge with documentation updates. Future work should focus on reducing complexity and code volume.