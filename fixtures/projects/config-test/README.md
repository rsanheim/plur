# Config Test Fixture

This fixture project contains various configuration files for testing plur's config handling:

* `valid.toml` - Valid configuration with basic settings
* `invalid-syntax.toml` - TOML syntax errors
* `type-mismatch.toml` - Wrong data types for fields
* `empty.toml` - Empty file
* `comments-only.toml` - Only comments, no actual config
* `command-specific.toml` - Command-specific sections ([spec], [watch.run])
* `high-debounce.toml` - Unusually high debounce value
* `negative-debounce.toml` - Invalid negative debounce

Used by configuration tests to avoid creating temporary files.