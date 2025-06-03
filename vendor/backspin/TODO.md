# Backspin TODO

## API Improvements

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