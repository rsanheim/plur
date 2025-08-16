# Interactive Plur - Implementation Tasks

## TODO Checklist

### 🔴 Critical: Minitest/Test::Unit Support
- [ ] Add support for `test/` directory detection in `FileMapper`
- [ ] Add support for `*_test.rb` file naming convention
- [ ] Auto-detect testing framework (check for `spec/` vs `test/`, `Gemfile` deps)
- [ ] Add configurable test directory and naming patterns to `.plur.toml`
- [ ] Update `findAlternativeSpecs()` to search for both `*_spec.rb` and `*_test.rb`
- [ ] Test with Rails, Sidekiq, Devise, and other minitest projects

### 🟡 High Priority: Improve Alternative Suggestions
- [ ] Add confidence scoring to alternative spec suggestions
  - Higher score for same directory structure
  - Lower score for wildcard matches
  - Filter out suggestions below threshold
- [ ] Reduce false positives for non-Ruby files
  - Skip ERB/template files from spec matching
  - Add file type validation before suggesting alternatives
- [ ] Improve pattern detection to avoid overly generic rules
  - Prefer specific paths over `**` wildcards when possible
  - Validate that suggested patterns don't match too many files
- [ ] Better handling of multiple match scenarios
  - Group by type (unit vs integration)
  - Allow user to select which type to prioritize

### 🟢 Medium Priority: Interactive Mode Enhancements
- [ ] Integrate `plur watch find` logic into main `plur watch` command
  - Add `--learn` mode flag to enable interactive mapping
  - Show suggestions when file changes have no mapping
  - Allow adding rules without restarting watch
- [ ] Improve interactive prompts
  - Show preview of what files would match new rule
  - Allow editing suggested rules before adding
  - Support multi-select for adding multiple rules at once
- [ ] Add runtime config reloading
  - Watch `.plur.toml` for changes
  - Apply new rules without restart
  - Show notification when config reloads

### 🔵 Nice to Have: Advanced Features
- [ ] Support for monorepo structures
  - Detect Rails engines/components
  - Handle project-specific test directories
  - Support workspace-aware mappings
- [ ] Smart suggestions for special files
  - `config/` files → suggest full suite or specific config specs
  - `db/migrate/` → suggest model specs
  - `Gemfile` → suggest dependency-related specs
- [ ] Integration with test coverage data
  - Suggest specs based on code coverage reports
  - Prioritize specs that cover changed lines
- [ ] Machine learning based suggestions
  - Learn from user's accepted/rejected suggestions
  - Build project-specific patterns over time

## Vision & Context

For the full vision and rationale behind interactive plur, see the sections below. The core idea is to help developers build up correct file-to-test mappings interactively while they work, learning from their project's structure and conventions.

## Current Implementation Status (2025-08-16)

### What's Been Built

We've implemented `plur watch find [files]` as a standalone exploration/diagnostic command that:
- **USES THE SAME MAPPING CODE AS `plur watch`** via `FileMapper.MapFileToSpecs()`
- Validates if mapped spec files actually exist
- Searches for alternative specs when default mappings fail
- Suggests custom mapping rules based on discovered alternatives
- Supports interactive mode (`-i`) and dry-run mode (`--dry-run`)
- Uses doublestar for proper `**` glob pattern support

**Important:** This command now correctly uses the same `FileMapper.MapFileToSpecs()` method that `plur watch` uses internally, ensuring consistent behavior between testing mappings and actual watch mode.

### Testing with Real Projects

Tested `plur watch find` with projects in the references directory:

#### Projects with Non-Standard Structures ✅
- **example-project project**: Correctly detected `lib/` → `spec/lib/` misalignment
  - Found alternatives like `spec/lib/example-project/cli_spec.rb` when default expected `spec/example-project/cli_spec.rb`
  - Suggested rule: `lib/**/*.rb` → `spec/lib/{path}/{name}_spec.rb`
  
- **tty-command**: Detected `lib/tty/command/` → `spec/unit/` pattern
  - Found specs in `spec/unit/` instead of mirroring lib structure
  - Suggested rule: `lib/tty/command/**/*.rb` → `spec/**/{name}_spec.rb`

#### Projects Following Conventions ✅
- **parallel_tests**: Standard `lib/` → `spec/` mapping worked perfectly
- **rspec-core**: Standard conventions, all mappings validated correctly

#### Projects with Missing Specs ✅
- **turbo_tests**: Correctly showed no alternatives when specs don't exist
- **example-project errors.rb**: No spec exists, correctly showed nothing (no false positives)

### Issues Discovered

1. **False positives for non-Ruby files**: 
   - ERB templates matched unrelated specs with similar names
   - Example: `lib/templates/command.erb` matched `spec/lib/example-project/command/ship_command_spec.rb`

2. **Pattern suggestions can be too broad**:
   - Sometimes suggests `spec/**/{name}_spec.rb` which is overly generic
   - Could match unrelated files across the project

3. **Multiple matches can be noisy**:
   - Shows integration tests alongside unit tests
   - Example: `lib/example-project/cli.rb` found both unit and integration specs

### Assessment

The approach is **promising but needs refinement** before integration into main watch mode:

**Strengths:**
- Accurately detects misaligned project structures
- Helps developers understand why specs aren't being found
- Useful for discovering project-specific patterns
- No false positives when specs truly don't exist

**Weaknesses:**
- Can suggest unrelated specs for non-Ruby files
- Pattern detection sometimes too generic
- Needs better filtering of match relevance

### Next Steps

1. **Keep as experimental feature** - `plur watch find` remains a diagnostic tool
2. **Refine matching algorithm** - Better relevance scoring, filter false positives
3. **Test with more projects** - Need more edge cases before integrating into watch mode
4. **Consider confidence scoring** - Only suggest alternatives with high confidence
5. **Eventually integrate into watch mode** - Once proven reliable and not noisy

## Extended Testing with Popular Ruby/Rails Projects (2025-08-16)

Tested `plur watch find` on additional popular Ruby/Rails projects to understand its effectiveness:

### Projects That Work Well ✅

#### RSpec-based Projects (High Success Rate)
- **rspec-core**: 73% success rate (8/11 files had existing specs found correctly)
  - Perfect nested directory mapping: `lib/rspec/core/formatters/html_formatter.rb` → `spec/rspec/core/formatters/html_formatter_spec.rb`
  - Correctly identified missing specs for version.rb, flat_map.rb
  
- **Flipper (feature flags)**: 83% success rate (5/6 files mapped correctly)
  - Standard lib → spec mapping worked perfectly
  - Deep nesting handled well: `lib/flipper/adapters/memory.rb` → `spec/flipper/adapters/memory_spec.rb`

- **GraphQL Ruby**: Good mapping with intelligent alternatives
  - When exact match not found, suggested related specs
  - Example: `lib/graphql/execution.rb` found `spec/graphql/execution_error_spec.rb` as alternative

### Projects That Don't Work ❌

#### Test::Unit/Minitest Projects (0% Success Rate)
- **Rails Framework**: Complete failure - uses `test/` directories with `*_test.rb` naming
  - Tested actionpack, activerecord, activesupport - all failed
  - No awareness of Rails' monorepo structure with component-specific test directories
  
- **Sidekiq**: 0% success - uses Test::Unit style
  - Expected `spec/sidekiq/client_spec.rb` but actual is `test/client_test.rb`
  - No fallback to check for `*_test.rb` patterns

- **Devise**: 0% success - uses `test/` directory structure
  - All 6 files tested failed to find their corresponding test files

- **Redis gem**: Convention mismatch detected correctly
  - Tool correctly identified no specs exist, actual tests in `test/` directory

### Critical Insights

1. **Framework Limitation**: `plur watch find` is hardcoded for RSpec conventions only
   - Only looks for `spec/` directories
   - Only searches for `*_spec.rb` files
   - No support for `test/` directories or `*_test.rb` naming

2. **Success Correlation**: 
   - RSpec projects: ~75-85% success rate
   - Test::Unit/Minitest projects: 0% success rate
   - Mixed conventions: Partial success based on RSpec adoption

3. **Directory Structure Sensitivity**:
   - Must run from project root (failed when run from parent directory of monorepo)
   - Correctly respects project boundaries
   - Handles deeply nested structures well when conventions match

4. **No False Positives**: Tool correctly identifies when no specs exist rather than guessing

### Recommendation

`plur watch find` is **excellent for RSpec-based projects** but **completely ineffective** for the significant portion of the Ruby ecosystem using Test::Unit/Minitest (including Rails itself). Before wider adoption, the tool needs:

1. Support for `test/` directory detection
2. Support for `*_test.rb` file patterns  
3. Auto-detection of testing framework in use
4. Configurable test directory and naming patterns

Currently best suited as a diagnostic tool for RSpec-based projects only.

## Related Work

### Post-Tool-Use Hook Implementation (2025-08-16)

We've successfully implemented a Claude Code post-tool-use hook that provides immediate test feedback:

* **Location**: `script/cc-post-tool-use`
* **Configuration**: `.claude/settings.json` with PostToolUse hook matcher for Edit|MultiEdit|Write
* **Functionality**: 
  * Automatically runs tests when files are edited
  * Maps files to their corresponding test files (Ruby specs and Go tests)
  * Blocks edits (exit code 2) when tests fail
  * Allows edits (exit code 0) when tests pass
  * Provides detailed failure output to stderr for debugging

This hook provides a foundation for the "learn mode" concept - we're already intercepting file changes and running tests, so the next step would be to detect when no tests are found and suggest mappings interactively.

See [optimize-watch-tests-with-handlers.md](optimize-watch-tests-with-handlers.md) for the original proposal and implementation details.
