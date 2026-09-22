# Rails Parallel Test Setup

`plur rails:init` updates Rails test configuration so each worker uses separate
database names and Redis database indexes based on `TEST_ENV_NUMBER`.

```bash
plur rails:init --dry-run  # Preview configuration diffs
plur rails:init           # Apply changes
plur rails db:test:prepare # Prepare test databases for each worker
```

## Configuration Changes

* `config/database.yml`: appends `<%= ENV['TEST_ENV_NUMBER'] %>` to test database
  names, including multi-database configurations. SQLite database paths are skipped.
* `config/cable.yml`: replaces numeric Redis database indexes in the test section
  with an expression based on `TEST_ENV_NUMBER`.
* The command recognizes existing `TEST_ENV_NUMBER` settings and leaves them alone.

The command also warns about service URLs in test environment files and known
Sidekiq, Elasticsearch, and Searchkick configuration files that may need manual
isolation.

## Limitations

Plur edits these files line by line to preserve ERB. Multiline values, YAML flow
style, and database names inherited only from anchors may require manual edits.
The command warns when it cannot find a test database name to change.

SQLite test databases, environment files, Ruby initializers, and ActiveStorage
configuration are not modified. Set those up separately if your tests need
resources for each worker.
