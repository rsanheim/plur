# Rspec Package

## We have a lot of mixed concerns throughout the Runner and other files.

Lets break out an `rspec` package in the `plur` Go CLI (./plur) that contains responsibilities for running and processing rspec files.
This will help us divide up responsibilities, refine interfaces, and paves the way for perf work 
as well as adding the abililtiy to handle minitest and other test runners down the road.

Do not prematurely generalize, but lets start refactoring towards something cleaner. Keep in mind right now plur _only_ supports rspec.
Other test runners are down the road quite a bit.

Questions:
* any duplication we should clean up first? 
* Or interfaces to extract?
* Is RuntimeTracker general enough for other test runners? Or assume its rspec focused for now?

## Proposed Structure / type break out

* [x] Update claude.md with key tasks from @Rakefile for build/running/etc
* [x] Condense claude.md & remove any extraneous info

### Phase 1: Analysis & Planning
* [x] Audit current Runner responsibilities and identify rspec-specific code
* [x] Map out dependencies between Runner, RspecJSON, Formatter, and Result types
* [x] Identify interfaces that could be extracted (e.g., TestRunner, TestResult, TestFormatter)
* [x] Review RuntimeTracker to determine if it's generic enough for other test frameworks

### Phase 2: Create rspec Package Structure
* [x] Create plur/rspec/ directory
* [x] Move RspecJSON message types to rspec/json_output.go
* [x] Move streaming message types to rspec/streaming.go
* [x] Move formatter logic to rspec/formatter.go
* [ ] Extract RSpec-specific runner logic to rspec/runner.go (deferred - runner logic still mixed)
* [ ] Create rspec/config.go for RSpec-specific configuration (deferred - not needed yet)

### Phase 3: Define Interfaces
* [ ] Create TestRunner interface in main package
* [ ] Create TestResult interface for generic result handling
* [ ] Implement RspecRunner that satisfies TestRunner interface
* [ ] Update main.go to use TestRunner interface instead of concrete types

### Phase 4: Refactor & Clean Up
* [x] Update tests to work with new package structure
* [x] Remove old rspec files after verification
* [x] Ensure RuntimeTracker works cleanly with new structure
* [ ] Remove duplication between runner.go and rspec package (partially done)
* [ ] Update error handling to be more modular (deferred)

## Additional Questions

1. **Interface Design**: Should we create a generic `TestFramework` interface now, or wait until we actually add support for minitest/other frameworks?
- if it is very simple, yeah we could add that now. I'm not sure what the responsibilities are though?

2. **RuntimeTracker Coupling**: The RuntimeTracker currently saves to `~/.cache/plur/runtime.json`. Should this be:
   - Made framework-agnostic (e.g., `~/.cache/plur/rspec-runtime.json`)?
   - Keep as-is since plur only supports RSpec currently?
- I think we should keep it as-is for now.

3. **Configuration**: Should RSpec-specific flags (like format options) be:
   - Moved to a subcommand structure (`plur rspec --format`)?
   - Kept at top level for backward compatibility?
- Keep it as is for now.

4. **Error Types**: Should we create RSpec-specific error types or keep using generic errors?
- Nope - Keep it as is for now.

5. **Formatter Abstraction**: The current formatter handles both progress dots and JSON. Should we:
   - Split these into separate formatter types?
   - Create a formatter interface that both implement?
- The unified formatter was a concious decision for perf and simplicity....lets leave that until we really need to refactor it.

6. **Test Discovery**: File glob patterns (`*_spec.rb`) are RSpec-specific. Should this be:
   - Parameterized through the TestRunner interface?
   - Hardcoded in the rspec package for now?
- Leave this as is as well.

## Notes on Current Duplication

Looking at the codebase, potential duplication to address:
- JSON parsing logic scattered between runner.go and rspec_json.go
- Result aggregation logic mixed with runner responsibilities
- Formatter creation tightly coupled to runner execution

## Completed Work Summary (2025-05-28)

Successfully extracted RSpec-specific code into a dedicated `plur/rspec` package:

1. **Created rspec package** with three main components:
   - `json_output.go` - RSpec JSON output types and parsing (formerly rspec_json.go)
   - `streaming.go` - Streaming JSON message handling (formerly json_message.go)
   - `formatter.go` - RSpec formatter management with embedded Ruby formatter

2. **Updated all imports and references** throughout the codebase:
   - Updated runner.go to use rspec.StreamingResults, rspec.ParseStreamingMessage, etc.
   - Updated result.go to use rspec.FailureDetail and formatting functions
   - Updated runtime_tracker.go to accept rspec.Example
   - Fixed test files to use new package references

3. **Maintained backward compatibility**:
   - All integration tests passing
   - No changes to CLI interface or behavior
   - Runtime tracking continues to work as before

4. **Deferred for future work**:
   - Extracting RSpec-specific runner logic (runner still has mixed responsibilities)
   - Creating TestRunner interface (premature without second test framework)
   - RSpec-specific configuration file (not needed yet)
   - Further modularization of error handling

This refactoring improves code organization and prepares for potential future support of other test frameworks while maintaining the current RSpec-only focus.
