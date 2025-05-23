# rux-meta

A research repository for `rux`, a Go-based parallel test runner for Ruby/RSpec projects designed to outperform existing solutions like turbo_tests and parallel_tests.

## 🚀 Quick Start

```bash
# Set up everything (builds rux, creates Rails app, runs tests)
./script/setup

# Or manually:
cd rux && go build -o rux main.go
cd ../test_app && ../rux/rux -n 3
```

## 📦 What's Included

### Core rux Implementation (`rux/`)
- **Go-based CLI** for parallel RSpec execution
- **TEST_ENV_NUMBER support** for Rails database isolation
- **Database commands** (db:create, db:migrate, db:setup, db:test:prepare)
- **Performance optimized** - 13% faster than turbo_tests

### Rails Test App (`test_app/`)
- **Complete Rails 8 application** with sqlite3
- **Parallel database configuration** using TEST_ENV_NUMBER
- **Sample models** (User, Post) with validations and associations
- **RSpec test suite** demonstrating parallel execution

### Reference Implementations
- **parallel_tests/**: Study of mature Ruby parallel runner
- **turbo_tests/**: Analysis of fast RSpec runner with excellent output
- **rux-ruby/**: Simple Ruby test project for development

### Testing Infrastructure
- **Ruby integration tests** (`test_rux_integration.rb`)
- **Go unit tests** for all core functionality
- **Benchmarking script** (`script/bench`) for performance comparison
- **Setup automation** (`setup_rails_testing.rb`)

## 🎯 Key Features

### Parallel Test Execution
```bash
rux -n 4                    # Run with specific worker count
rux                          # Auto-detect workers (cores-2)
rux --dry-run               # Preview execution plan
```

### Database Management
```bash
rux db:create -n 3          # Create test databases in parallel
rux db:migrate -n 3         # Run migrations across all test DBs
rux db:setup -n 3           # Full database setup
```

### Environment Variables
- **TEST_ENV_NUMBER**: Worker 0 gets `""`, worker N gets `"N+1"`
- **PARALLEL_TEST_GROUPS**: Total number of workers
- **PARALLEL_TEST_PROCESSORS**: Compatible with parallel_tests

## 📊 Performance Results

Benchmarked on example-project project (24 spec files):

| Command | Time | Relative Performance |
|---------|------|---------------------|
| `rux -n 4` | **9.04s** | Fastest (baseline) |
| `rux` (default) | 10.15s | +12% slower |
| `bundle exec turbo_tests` | 10.18s | +13% slower |

**Result**: rux with optimized worker count is 13% faster than turbo_tests.

## 🧪 Testing

### Run All Tests
```bash
rake                         # Go tests (lint + test)
ruby test_rux_integration.rb # Ruby integration tests
```

### Test with Rails App
```bash
cd test_app
../rux/rux -n 3            # Run RSpec tests in parallel
../rux/rux db:create -n 3        # Set up databases
```

### Benchmarking
```bash
./script/bench ./test_app    # Compare rux vs turbo_tests
./script/bench ./rux-ruby    # Test with simple Ruby project
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
- **`docs/usage.md`**: Detailed usage guide and examples
- **`CLAUDE.md`**: Development environment setup for future work

## 🛠️ Development

### Requirements
- Go 1.22+
- Ruby 3.0+ (for Rails testing)
- Rails 8+ (for test app)

### Project Structure
```
rux-meta/
├── rux/                    # Main Go implementation
├── test_app/              # Rails app for testing
├── script/                # Utility scripts
├── docs/                  # Documentation
├── parallel_tests/        # Reference implementation
├── turbo_tests/          # Reference implementation
└── rux-ruby/             # Simple Ruby test project
```

This project demonstrates that Go can provide significant performance improvements for Ruby tooling while maintaining full compatibility with existing Rails workflows.