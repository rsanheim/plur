# Rspec Package

## We have a lot of mixed concerns throughout the Runner and other files.

Lets break out an `rspec` package in the `rux` Go CLI (./rux) that contains responsibilities for running and processing rspec files.
This will help us divide up responsibilities, refine interfaces, and paves the way for perf work 
as well as adding the abililtiy to handle minitest and other test runners down the road.

Do not prematurely generalize, but lets start refactoring towards something cleaner. Keep in mind right now rux _only_ supports rspec.
Other test runners are down the road quite a bit.

Questions:
* any duplication we should clean up first? 
* Or interfaces to extract?
* Is RuntimeTracker general enough for other test runners? Or assume its rspec focused for now?

## Proposed Structure / type break out

* [ ] Update claude.md with key tasks from @Rakefile for build/running/etc
* [ ] Condense claude.md & remove any extraneous info

### Phase 1: Analysis & Planning
* [ ] Audit current Runner responsibilities and identify rspec-specific code
* [ ] Map out dependencies between Runner, RspecJSON, Formatter, and Result types
* [ ] Identify interfaces that could be extracted (e.g., TestRunner, TestResult, TestFormatter)
* [ ] Review RuntimeTracker to determine if it's generic enough for other test frameworks

### Phase 2: Create rspec Package Structure
* [ ] Create rux/rspec/ directory
* [ ] Move RspecJSON message types to rspec/messages.go
* [ ] Extract RSpec-specific runner logic to rspec/runner.go
* [ ] Move formatter logic to rspec/formatter.go
* [ ] Create rspec/config.go for RSpec-specific configuration

### Phase 3: Define Interfaces
* [ ] Create TestRunner interface in main package
* [ ] Create TestResult interface for generic result handling
* [ ] Implement RspecRunner that satisfies TestRunner interface
* [ ] Update main.go to use TestRunner interface instead of concrete types

### Phase 4: Refactor & Clean Up
* [ ] Update tests to work with new package structure
* [ ] Remove duplication between runner.go and rspec package
* [ ] Ensure RuntimeTracker works cleanly with new structure
* [ ] Update error handling to be more modular

## Additional Questions

1. **Interface Design**: Should we create a generic `TestFramework` interface now, or wait until we actually add support for minitest/other frameworks?
- if it is very simple, yeah we could add that now. I'm not sure what the responsibilities are though?

2. **RuntimeTracker Coupling**: The RuntimeTracker currently saves to `~/.cache/rux/runtime.json`. Should this be:
   - Made framework-agnostic (e.g., `~/.cache/rux/rspec-runtime.json`)?
   - Keep as-is since rux only supports RSpec currently?
- I think we should keep it as-is for now.

3. **Configuration**: Should RSpec-specific flags (like format options) be:
   - Moved to a subcommand structure (`rux rspec --format`)?
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
