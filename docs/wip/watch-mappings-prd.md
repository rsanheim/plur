# Watch Mappings Overhaul


Our current approach to handling watch mappings is all wrong. We are going to rip out the entirity of the current system and start fresh.

_This document is the high level plan and goal. See [watch-mappings-checklist.md](watch-mappings-checklist.md) for the implementation checklist._

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

After this is all done, we will verify there are no traces left in the current code base. THen we will compile a 'lessons learned' doc, and begin planning for reimplementing a simpler, cleaner, better designed system.

### Definition of Success

All file mapping code and behavior removed from the current code base. The CLI
commands `watch run` and `watch fine` still work as shells to build back on top of
the new system, including the base file watching implemented via edant_watcher combined with our `WatcherManager` and related Go channels/goroutines.

### Questions

**Resolved in Implementation:**
1. ~~How should watch directories be derived after removing mapping system?~~
   - Answer: From Job's WatchMapping.SourceDir() - BUT this is currently broken (see watch-mappings-checklist.md)

2. ~~Should we keep `watch find` command?~~
   - Answer: Yes, as placeholder for future rebuild

3. ~~What happens to existing watch configurations in user projects?~~
   - Answer: Removed mapping sections from TOML configs, users will need to update

**Unresolved (Blocking Watch Functionality):**
1. **Why is WatchMapping.SourceDir() not working?**
   - Error: "no directories to watch found in watch mappings"
   - Need to investigate Job → WatchMapping conversion
   - May need explicit watch directory configuration in job definitions

2. **Should watch directories be explicit in job config or derived?**
   - Current approach (derive from mappings) appears broken
   - May need to add explicit `watch_dirs` field back to Job or use `source_dirs`

3. **How should error messages be updated?**
   - Currently reference removed "watch mappings" system
   - Need to reflect new Job-based architecture