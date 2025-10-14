# plur-meta

`plur` is a Go-based parallel test runner for Ruby projects (RSpec and Minitest) designed to outperform existing solutions like turbo_tests and parallel_tests.

## 🚀 Quick Start

```bash
# Build and install plur
bin/rake install

# Run tests on default Ruby library
cd fixtures/projects/default-ruby && plur

# Run tests on default Rails app
cd fixtures/projects/default-rails && plur -n 3
```

## 📦 What's Included

### Core plur Implementation (`plur/`)
- **Go-based CLI** for parallel RSpec execution
- **Database commands** (db:create, db:migrate, db:setup, db:test:prepare)
- **Performance optimized** - 13% faster than turbo_tests

### Test Fixture Projects (`fixtures/projects/*`)
- **default-ruby/**: Simple Ruby library for basic testing and development
  - Pure Ruby project with models, services, and utilities
  - Comprehensive RSpec test suite
  - Used for quick iteration and debugging
- **default-rails/**: Complete Rails 8 application for integration testing
  - Full Rails app with sqlite3 database
  - Parallel database configuration using TEST_ENV_NUMBER
  - Sample models (User, Post) with validations and associations
  - RSpec test suite demonstrating parallel execution

### Reference Implementations
- **references/parallel_tests/**: Study of mature Ruby parallel runner
- **references/turbo_tests/**: Analysis of fast RSpec runner with excellent output

### Testing Infrastructure
- **Ruby integration tests** (`test_plur_integration.rb`)
- **Go unit tests** for all core functionality
- **Benchmarking script** (`script/bench`) for performance comparison
- **Setup automation** (`setup_rails_testing.rb`)

## 📦 Platform Support

### Production Ready
* **macOS ARM64** (Apple Silicon) - Fully supported
* **Linux x86_64** - Fully supported
* **Linux ARM64** - Fully supported

### Not Supported
* **macOS Intel (x86_64)** - Not supported (no upstream watcher binary available)

### Experimental/Alpha
* **Windows x86_64** - Experimental support (never tested in production)

**Note**: Watch mode (`plur watch`) requires platform-specific binaries. All platforms support standard test execution.

## 🎯 Key Features

### Parallel Test Execution
```bash
plur -n 4                    # Run with specific worker count
plur                          # Auto-detect workers (cores-2)
plur --dry-run               # Preview execution plan
```

### Database Management
```bash
plur db:create -n 3          # Create test databases in parallel
plur db:migrate -n 3         # Run migrations across all test DBs
plur db:setup -n 3           # Full database setup
```

### Framework Selection
```bash
plur --use=rspec             # Run RSpec tests
plur --use=minitest          # Run Minitest tests
plur                         # Auto-detect from directory structure
```

Projects with both `spec/` and `test/` directories should use `--use` or set `use = "rspec"` in `.plur.toml`.

### Configuration
Plur supports TOML configuration files for persistent settings:
```toml
# .plur.toml or ~/.plur.toml
workers = 4

[task.rspec]
run = "bin/rspec"

[watch.run]
debounce = 200  # Milliseconds to wait before running tests
```

See `examples/` directory for more configuration examples.

### Environment Variables
- **TEST_ENV_NUMBER**: Worker 0 gets `""`, worker N gets `"N+1"`
- **PARALLEL_TEST_GROUPS**: Total number of workers
- **PARALLEL_TEST_PROCESSORS**: Compatible with parallel_tests

## 📊 Performance Results

Benchmarked on example-project project (24 spec files):

| Command | Time | Relative Performance |
|---------|------|---------------------|
| `plur -n 4` | **9.04s** | Fastest (baseline) |
| `plur` (default) | 10.15s | +12% slower |
| `bundle exec turbo_tests` | 10.18s | +13% slower |

**Result**: plur with optimized worker count is 13% faster than turbo_tests.

## 🧪 Testing

### Run All Tests
```bash
rake                         # Run ALL tests & lints before committing
rake test                    # Run full Ruby test suite
rake test:go                 # Run Go tests only
```

### Test with Default Projects
```bash
# Simple Ruby library
cd fixtures/projects/default-ruby
plur                        # Run all specs

# Rails application
cd fixtures/projects/default-rails
plur db:create -n 3        # Set up databases
plur db:migrate -n 3       # Run migrations
plur -n 3                  # Run RSpec tests in parallel
```

### Benchmarking
```bash
./script/bench              # Benchmark default projects
./script/bench -p fixtures/projects/default-rails    # Benchmark Rails app
./script/bench -p fixtures/projects/default-ruby     # Benchmark Ruby library
```

## 🏗️ Architecture

### Clean Separation
- **`main.go`**: CLI interface and command routing
- **`runner.go`**: Business logic for test execution and database tasks
- **Comprehensive testing**: Both Go unit tests and Ruby integration tests

### Go Implementation Benefits
- **Fast startup**: No Ruby VM overhead
- **Efficient concurrency**: Go goroutines and worker pools
- **Single binary**: Easy distribution and deployment
- **Memory efficient**: Lower resource usage than Ruby alternatives

### Rails Integration
- **Database isolation**: Each worker gets separate test database
- **Environment compatibility**: Works with existing Rails test setups
- **Migration support**: Parallel database setup and migration
- **Standard RSpec**: Uses built-in formatters, no custom setup needed

## 📚 Documentation

- **`docs/project-status.md`**: Complete project overview and status
- **`docs/development/user-guide.md`**: Detailed usage guide and examples
- **`CLAUDE.md`**: Development environment setup for future work

### Viewing Documentation Locally

We use MkDocs Material for browsing documentation. To view the docs locally:

```bash
# Requires mise + uv (both installed automatically if missing)
script/docs
```

The documentation setup uses mise for Python version management and uv for dependency management. All Python dependencies are managed in `docs/pyproject.toml`.

For more documentation commands:
```bash
script/docs help
```

## 🛠️ Development

### Requirements
- Go 1.22+
- Ruby 3.0+ (for Rails testing)

### Project Structure
```
plur/
├── plur/                    # Main Go implementation
├── fixtures/
│   └── projects/
│       ├── default-ruby/   # Simple Ruby library for testing
│       └── default-rails/  # Rails 8 app for testing
├── script/                 # Utility scripts
├── docs/                   # Documentation
├── lib/                    # Ruby libraries and helpers
└── references/
    ├── parallel_tests/     # Reference implementation
    └── turbo_tests/        # Reference implementation
```

This project demonstrates that Go can provide significant performance improvements for Ruby tooling while maintaining full compatibility with existing Rails workflows.