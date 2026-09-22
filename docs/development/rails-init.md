# Rails Parallel Test Setup

`plur rails:init` updates common Rails test configuration to isolate resources
using `TEST_ENV_NUMBER`.

```bash
plur rails:init --dry-run  # Preview configuration diffs
plur rails:init            # Apply changes
plur rails db:test:prepare # Prepare test databases for each worker
```

## Configuration Changes

* `config/database.yml`: appends `<%= ENV['TEST_ENV_NUMBER'] %>` to test database
  names, including multi-database configurations. SQLite database paths are skipped.
* `config/cable.yml`: replaces numeric Redis database indexes in the test section
  with an expression based on `TEST_ENV_NUMBER`.
* Existing `TEST_ENV_NUMBER` configuration is recognized, so rerunning the command
  does not append it again.

The command also warns about service URLs in test environment files and known
Sidekiq, Elasticsearch, and Searchkick configuration files that may need manual
isolation.

## Limitations

Transformations are line-based to accommodate ERB. Multiline values, YAML flow
style, and database names inherited only from anchors may require manual edits.
The command warns when it cannot find a test database name to change.

SQLite test databases, environment files, Ruby initializers, and ActiveStorage
configuration are not modified. Configure resource isolation for those separately.
