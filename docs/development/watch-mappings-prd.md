# Watch Mappings Overhaul

Our current approach to handling watch mappings is all wrong. We are going to rip out the entirity 
of the current system and start fresh.

## Current Problems

The design is currently a mess: `mappings` are coupled to `Task` and `TaskConfig`, which 
is combining 'watch' config with a more general task config. Also, the mapping was 
too difficult and cumbersome to configure and use.

Also, the wiring between mappings and Kong CLI config never quite worked right, and 
had too many layers of indirection.

We didn't effectively leverage doublestar to match files, so we wrote too much of our own code.

Overall it was too complicated and fragile to maintain.

### Solution

We are going to rip out the entirity of the current mapping system to have a clean 
slate to build on. So all [mapping] config, logic, and related data types will be removed.
This includes:

* any watch mapping related configuration
* all mapping logic in `task.go` and related Go tests
* mapping_rules.go and tests
* `watch run` will remain and will just display files changed from the underlying
edant_watcher library - this will give us an opporuntity to fine tune and refine
logging and the overall file watch system, before we add back the test runner.
* `watch fine` logic -- the CLI command can stay, but we will build back the logic behind it.
* _any_ related documentation, Go tests, example configs, etc.
* For rspec tests, we may leave a few that are built from the outside in and 
don't verify implementation details. Any that are too complicated or tied ot the 
current mapping system can be removed.

### Questions
