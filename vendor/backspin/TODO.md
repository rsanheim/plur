# Backspin TODO

Remeber to run the full build after significant changes via `bin/rake` - the default task runs specs and standardrb.

### Rename Dubplate to Record
I want to have a more straight forward name here. Lets rename away from Dubplate to Record.
Update specs as you go.

* [ ] change the object name from `Dubplate` to `Record`
* [ ] change the directory that backspin uses to store the records from `dubplates` to `records`
* [ ] change the top level Backspin.record to be named `Backspin.call` to avoid confusion
* [ ] update docs and CLAUDE.md
* [ ] commit  when everything is passing

### Store `first_recorded_at` in the cassette file

We need to know the first recorded at time to allow automatic re-recording of the record later on.
Add it to the Record object and the record file. Do not implement 're-record' yet.

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