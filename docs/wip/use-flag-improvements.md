# --use Flag Improvements and Follow-up Work

## Overview

This document outlines recommended improvements to make framework selection easier for users, particularly those with non-standard Ruby projects (mixed RSpec/Minitest, Minitest-only, custom frameworks).

## 1. Add `-t` Short Flag

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

## 2. Create Mixed-Framework Test Fixture

### Why
* **Zero test coverage**: No integration tests for projects with both `spec/` and `test/`
* **Real-world scenario**: Many legacy codebases have mixed frameworks
* **Confidence**: Can't refactor detection logic safely without tests

### What
Create `fixtures/projects/mixed-rspec-minitest/` with:
* `spec/` directory with passing RSpec tests
* `test/` directory with passing Minitest tests
* `.plur.toml` with example configuration
* README explaining the fixture's purpose

### Test Scenarios to Cover
* Default behavior (should pick minitest)
* `plur --use=rspec` explicitly runs RSpec
* `plur --use=minitest` explicitly runs Minitest
* Config file `use = "rspec"` changes default
* CLI flag overrides config file setting

### Impact
* Enables safe refactoring of detection logic
* Documents expected behavior for future maintainers
* Prevents regressions when adding command-aware detection

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
Running Minitest tests by default.

To run RSpec instead:
  plur -t rspec              # Use -t flag
  echo 'use = "rspec"' > .plur.toml  # Or set default in config

Run 'plur doctor' to see current configuration.
```

### Implementation Notes
* Message goes to stderr, not stdout
* Only shows once per project (can track in `.plur/` directory)
* Respects `--quiet` or similar flags
* Can be disabled via config: `show_hints = false`

## 4. Integration Test Coverage

### Why
* Current minitest tests only use `--use minitest` flag
* No tests for auto-detection with mixed projects
* No tests for config file `use` setting
* Missing coverage for common user workflows

### Tests Needed

#### Auto-detection Tests
* Single `spec/` directory → detects rspec
* Single `test/` directory → detects minitest
* Both directories → defaults to minitest
* Neither directory → defaults to rspec (backward compat)

#### CLI Flag Tests
* `-t rspec` on mixed project runs RSpec
* `-t minitest` on mixed project runs Minitest
* Invalid task name shows helpful error

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

## 5. Improve Help Text

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

## 6. Update installation.md

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
