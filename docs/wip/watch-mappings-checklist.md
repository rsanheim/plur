# Watch Mappings Checklist

This is the implementation checklist for watch mappings overhaul.
AI agents should use this checklist as they iterate and work on the 
watch mapping overhaul.

See [watch-mappings-prd.md](watch-mappings-prd.md) for the PRD.

## Phaes 1 - Remove all mapping related Go code and tests

* [ ] a good place to start would be top level `Mappings` in main.go
* [ ] other tasks TODO...
* [ ] BEFORE marking complete -- the entire Go build and test suite MUST pass

## Phase 2 - Selectively remove rspec integration specs testing watch mappings

* [ ] Remove watch mapping related RSpec integration specs that are skipped in CI
  * as these were frail and brittle in CI, they are good candidates for first removal
* [ ] Review remaining RSpec integration specs dealing with watch to determine what we remove
* [ ] BEFORE marking complete
  * the entire test suite MUST pass (including ruby and Go via `bin/rake`)
  * entire test suite MUST pass in CI


## Phase 3 - Remove all mapping related docs, config examples, etc

* [ ] if there are comparative analysis docs, we may want to keep them -- we can review those when we get to this point
* [ ] other tasks TODO...
* [ ] BEFORE marking complete -- a search for 'mapping' & `mappingRules`in the code base MUST return no results

## Phase 4 - Verify the watch commands still work as shells to build back on top of

* [ ] `watch run` starts watchers for [src,lib,spec,test] in current project and 
reports on files changed
* [ ] do log levels make sense? is logging output helpful?
* [ ] does `--verbose|-v` work as expected?
* [ ] does `--debug|-d` work as expected?
* [ ] other tasks TODO...

## Phase 5 - Lessons Learned
