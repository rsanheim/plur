# Backspin TODO

Remeber to run the full build after significant changes via `bin/rake` - the default task runs specs and standardrb.

### Clean up where we store the data in the spec suite! ✅

* For unit test (i.e. things that don't remove saved data) we should use "spec/backspin_data", just like any other gem
* For integration tests (i.e. things that have to remove saved records) we should use "./tmp/backspin_data"
* We should never store data in "spec/backspin" - its confusing and I don't know how it keeps creeping back in

### Rename Dubplate to Record ✅
I want to have a more straight forward name here. Lets rename away from Dubplate to Record.
Update specs as you go.

* [x] change the object name from `Dubplate` to `Record`
* [x] the location for record files can remain "backspin_data" - so by default in an rspec project it would be ./spec/backspin_data
* [x] change the top level Backspin.record to be named `Backspin.call` to avoid confusion
* [x] update docs and CLAUDE.md
* [ ] commit when everything is passing

### Store `first_recorded_at` in the cassette file ✅

We need to know the first recorded at time to allow automatic re-recording of the record later on.
Add it to the Record object and the record file. Do not implement 're-record' yet.

### Store the command info in the record file ✅

i.e. store the command type (Open3.capture3) and the arguments (["echo", "hello"])

I think we already have this mostly in the Command object, its mostly a matter of serializing it back and forth.

### Introduce the ability to record `system` calls
* we want to capture other system calls like `system("echo hello")`
* make the result and serialization format the same as capture3, even though this may mean adapting the results from system to match the more detailed output from capture3
* consider splitting this out, along side the capture3 stuff, into new objects that follow the same sort of pattern
* for context, we will want to add other command types later.

### Add preprocessing/filtering support to recording methods

Currently, Backspin only supports custom matchers during verification, not during recording. This makes it difficult to normalize dynamic content (like timestamps) before saving to golden files.

Proposed API additions:

1. Add `filter` or `preprocessor` option to `record`:
   ```ruby
   Backspin.record("example", filter: ->(result) {
     result["stdout"] = normalize_timestamps(result["stdout"])
     result
   })
   ```

2. Add same option to `use_dubplate`:
   ```ruby
   Backspin.use_dubplate("example", record: :once, filter: ->(result) {
     # Normalize before saving
   })
   ```

3. Consider adding common built-in filters:
   ```ruby
   Backspin.record("example", filter: [:timestamps, :paths])
   # Or
   Backspin.record("example", filter: Backspin::Filters.timestamps)
   ```

### Use Cases
- Normalize execution times in test output
- Remove absolute paths that vary between environments
- Scrub dynamic IDs or timestamps
- Make golden files more portable across environments