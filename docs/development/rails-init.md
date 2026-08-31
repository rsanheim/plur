# rails:init -- Parallel Test Setup for Rails

## Goal

Most Rails apps aren't configured for parallel testing out of the box. The `plur rails:init` command bridges that gap by automating the most common configuration changes needed to run specs across multiple workers without collisions.

The core problem: when N workers run specs simultaneously, they all hit the same test database (and Redis, Elasticsearch, etc.) unless the app is configured to namespace those resources per-worker using `TEST_ENV_NUMBER`.

This command should make the "zero to parallel" experience as painless as possible for Rails teams adopting plur.

## What v1 Does

1. *database.yml transformation* -- Finds the `test:` section and appends `<%= ENV['TEST_ENV_NUMBER'] %>` to every `database:` line. Handles single-db and multi-db (Rails 6+) configs. Skips sqlite3.

2. *cable.yml transformation* -- If the test section uses a redis adapter, replaces the hardcoded DB number with a `TEST_ENV_NUMBER`-based ERB expression.

3. *Warning scans* -- Checks for files that likely need manual parallel isolation: `.env.test`, sidekiq configs, elasticsearch/searchkick initializers. Prints warnings without modifying these files.

4. *Next steps* -- After making changes, prints the follow-up commands (`plur rails db:test:prepare`)

Supports `--dry-run` (shows diffs without writing) and is idempotent (detects already-configured projects).

## Current Limitations

### Regex-based YAML parsing

The biggest architectural limitation. We parse `database.yml` and `cable.yml` line-by-line with regexes because these files can contain ERB, which means we can't use a real YAML parser. This works well for the common cases but has known gaps:

* Doesn't handle database names that span multiple lines or use YAML flow style
* Doesn't handle cases where the database name is inherited from `&default` anchor with no explicit `database:` line in the test section -- we warn instead of trying to inject one
* Can't reason about YAML structure deeply (e.g., distinguishing a `database:` key nested 3 levels deep from one nested 2 levels)

For v1 this is fine because most real-world `database.yml` files are extremely consistent in format. 

### No sqlite3 support

Deliberately skipped. Apps using sqlite3 for testing are typically small/new and unlikely to benefit from parallel testing. Postgres and MySQL are the real targets.

### No .env file modification

`.env.test` and `.env.test.local` files are detected and warned about but not modified - too much risk with touching these.
### No initializer modification

Ruby initializers for sidekiq, elasticsearch, searchkick, etc. are warned about but not touched. These are full Ruby files with arbitrary logic -- regex transformation would be fragile and dangerous.

### No config/storage.yml handling

ActiveStorage disk paths *can* collide in parallel testing, but it's uncommon in practice (most test suites use fake/stub storage or the collision is harmless). Low priority.

## Future Directions

### Ruby parser-based approach

The nuclear option for handling initializers and complex configs. Instead of regex, use a Ruby parser (tree-sitter, or shell out to a Ruby script) to do AST-aware transformations. This would let us safely modify files like:

* `config/initializers/sidekiq.rb` -- wrap Redis URL config with `TEST_ENV_NUMBER` awareness
* `config/initializers/elasticsearch.rb` -- add index prefix based on worker number
* `spec/rails_helper.rb` or `spec/support/` -- add `parallelize_setup` blocks

Trade-offs:
* *Pro*: Much safer than regex for Ruby files, can handle arbitrary formatting
* *Pro*: Could generate idiomatic Ruby code that users actually want to keep
* *Con*: Significant complexity increase -- need to decide on parser (tree-sitter Go bindings? shelling out to Ruby?)
* *Con*: Diminishing returns -- most apps only have 1-2 services that need parallel isolation beyond the database

A middle ground: instead of parsing and transforming Ruby, generate a new file (e.g., `config/initializers/plur_parallel.rb`) that wraps the common cases. This sidesteps the parsing problem entirely.

### Other datastores

Things we could detect and offer guidance or auto-configuration for:

* *Memcached* -- namespace per worker. Usually configured in `config/environments/test.rb` via `config.cache_store`
* *MongoDB* (Mongoid) -- separate database name per worker, similar to database.yml pattern but in `config/mongoid.yml`
* *S3/ActiveStorage* -- bucket prefix or path prefix per worker

Most of these follow the same pattern: find the config, append `TEST_ENV_NUMBER` to a name/path/prefix. The detection is easy; the transformation varies by config format.
