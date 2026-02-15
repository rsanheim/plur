# Config Test Fixture

This fixture project contains various configuration files for testing plur's config handling:

* `valid.toml` - Valid configuration with basic settings
* `invalid-syntax.toml` - TOML syntax errors
* `type-mismatch.toml` - Wrong data types for fields
* `empty.toml` - Empty file
* `comments-only.toml` - Only comments, no actual config
* `command-specific.toml` - Job override + custom watch mapping
* `job-without-use.toml` - Job config without use= (selected via --use)
* `minitest.toml` - Minitest job config
* `runner-jobs.toml` - Runner jobs resolver fixture
* `with-tasks.toml` - Multiple jobs for run command selection
* `doctor-test.toml` - Doctor output fixture

Used by configuration tests to avoid creating temporary files.
