# Watch Mappings Checklist

This is the implementation checklist for watch mappings overhaul.

See [watch-mappings-prd.md](watch-mappings-prd.md) for the PRD.

## Phaes 1 - remove all mapping related Go code and tests

* [ ] a good place to start would be top level [mapping] config, structs, wiring, etc

## Phase 2 - selectively remove rspec integration specs testing watch mappings

## Phase 3 - remove all mapping related docs, config examples, etc

## Phase 4 - verify the watch commands still work as shells to build back on top of

* [ ] `watch run` starts watchers for [src,lib,spec,test] in current project and 
reports on files changed
* [ ] do log levels make sense? is logging output helpful?
* [ ] does `--verbose|-v` work as expected?
* [ ] does `--debug|-d` work as expected?
