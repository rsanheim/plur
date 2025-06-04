# Backspin TODO

Remember to run the full build after significant changes via `bin/rake` - the default task runs specs and standardrb.

## Pending Tasks

### Add preprocessing/filtering support
* Add ability to filter/preprocess command output before recording
* This would allow normalizing timestamps, dynamic IDs, etc.
* Should work with both `call` and `use_record` methods

### Implement automatic re-recording based on age
* Use the `first_recorded_at` timestamp to automatically re-record stale records
* Add configuration option for max record age
* Add `:auto` mode to `use_record` that re-records if older than threshold

### Add support for more command types
* Add support for backticks (`` `command` ``)
* Add support for %x{command}
* Add support for IO.popen
* Consider a plugin system for adding new command types

### Improve new_episodes mode
* Currently just appends new commands
* Should track which commands have been seen and only record truly new ones
* Consider command matching strategies (exact args vs. pattern matching)

## Completed Tasks ✅

### Clean up where we store the data in the spec suite! ✅
* For unit test (i.e. things that don't remove saved data) we should use "spec/backspin_data", just like any other gem
* For integration tests (i.e. things that have to remove saved records) we should use "./tmp/backspin_data"
* We should never store data in "spec/backspin" - its confusing and I don't know how it keeps creeping back in

### Rename Dubplate to Record ✅
* Changed the object name from `Dubplate` to `Record`
* The location for record files remains "backspin_data" - so by default in an rspec project it would be ./spec/backspin_data
* Changed the top level Backspin.record to be named `Backspin.call` to avoid confusion
* Updated docs and CLAUDE.md

### Store `first_recorded_at` in the cassette file ✅
* Added to the Record object and the record file for future automatic re-recording

### Store the command info in the record file ✅
* Store the command type (Open3.capture3) and the arguments (["echo", "hello"])
* Implemented in the Command object with proper serialization

### Introduce the ability to record `system` calls ✅
* Can now capture system calls like `system("echo hello")`
* Results are serialized in the same format as capture3 (with empty stdout/stderr)
* System calls maintain proper behavior (returning true/false based on exit status)

### Refactor the stub mechanism ✅
* Created Recorder class to encapsulate all stubbing logic
* Recorder is mode-aware and handles different contexts (record, replay, verify)
* All main methods now use Recorder, eliminating code duplication
* Clean separation of concerns between recording logic and the main API


