# Interactive Plur

One of my 'big idea' goals with Plur is to allow a developer to use `plur watch` to build up an interactive
config and mapping of files to tests while they are running.  So here is one simple example:

User starts `plur watch` and saves the file `app/services/user_publisher.rb`

plur does not find a direct mapping (as it would be spec/services/user_publisher_spec.rb only with our simple mapping rules), and tells the user that, but in presents the user with options of other specs that could be run:

* spec/services/user_publisher_spec.rb
* spec/integrations/user_publisher_spec.rb
* spec/*user_publisher**.rb

and so on.  

The basic idea is to allow a developer to add rules to their config interactively based on files saved that 
_do not_ have any matching files to run....and there are some pretty easy heuristics we can apply to handle
90% of cases for this sort of thing. I think. 

Some contraints:
* we should not try to be too clever -- providing a general glob rule based on a file saved is a good start
* if a user saves 'app/models/user.rb', and there is a typiocal matching spec 'spec/models/user_spec.rb', we should not try to provide suggestions
* if we provide suggestions, we should tell the user what specs _would_ match if they added that rule to make the mathcing work
* we should provide enough feedback to the user to help them understand how mapping rules work with plur, and how they can tweak them later to get more specific or correct

Additionally, we should allow watch to be in two different modes that can be toggled by the user:
* 'learn' mode: where plur will suggest rules to add to the config based on the files saved
* 'standard' mode: where plur will run just what is in the config file as prescribed

by default `plur watch` will be in the standard mode, but I think the learn mode could be an attractive offering for more complicated test suites...and help us build out our mapping rules to suit the many varieties of test - to lib spec rules.

### Implementation Plan

* Remember that we have a the `plur watch run --timeout [seconds]` option that will exit after the timeout is reached.  This is a good way to ensure that the watch mode does not run forever when running it locally or building etsts around. We already have rspec integration tests that use this now.
* Consider how the config is loaded and how we can change it at runtime (for live feedback), and also write it back to the file system to save valid rules for the user
* Consider how to make this user friendly: we want to explain what the rules currently are (and why), and then explain what plur is suggesting, and then explain changes plur may make to the runtime rules and the config saved on disk.
* a broader goal is to help developers think thru what test files are important when a certain file changes....and providing input and guidance to help them build the correct set of matching rules that respond to file change events. This may mean running specs that match a simple glob pattern, or if someone saves "config/application.rb", we can suggest just running the entire suite or maybe running "spec/config/application_spec.rb" if it exists.  

### Example mappings to explore with our default rules and what may come up:

* 'app/services/foo_service.rb'
User saves a service file, and in a rails project does it try to find a spec/services/foo_service_spec.rb? Is that in default rules?

* 'lib/[something]/foo.rb'

This is a ruby gem or library case - the user saves a top level file in the library -- does it look for spec/foo_spec.rb?  If it does, and there are no matches, does it then look for spec/lib/foo_spec.rb?  Or perhaps spec/lib/[something]/foo_spec.rb?

## Current Implementation Status (2025-08-16)

### What's Been Built

We've implemented `plur watch find [files]` as a standalone exploration/diagnostic command that:
- Validates if mapped spec files actually exist
- Searches for alternative specs when default mappings fail
- Suggests custom mapping rules based on discovered alternatives
- Supports interactive mode (`-i`) and dry-run mode (`--dry-run`)
- Uses doublestar for proper `**` glob pattern support

**Important:** This is NOT yet integrated into the main `plur watch` command - it's a separate tool for testing the concept.

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
