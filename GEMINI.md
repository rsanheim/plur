# Plur: A Go-Based Parallel Test Runner for Ruby

## Project Overview

This project, `plur`, is a high-performance, Go-based test runner designed to execute Ruby (RSpec and Minitest) test suites in parallel. The primary goal of `plur` is to provide a faster alternative to existing tools like `turbo_tests` and `parallel_tests`. The core of the application is a Go binary that manages test execution, while the project also includes Ruby code for integration testing and Rake tasks.

**Key Technologies:**

*   **Go:** The core test runner is written in Go for its performance and concurrency benefits.
*   **Ruby:** The project is designed to test Ruby projects, and it includes Ruby-based test fixtures and helper code.
*   **RSpec & Minitest:** `plur` supports both RSpec and Minitest, two of the most popular testing frameworks in the Ruby ecosystem.
*   **TOML:** Configuration is handled through TOML files (`.plur.toml`), allowing for project-specific and global settings.

**Architecture:**

*   **Go CLI (`plur/`):** The main application is a Go command-line interface that handles test discovery, parallel execution, and database management.
*   **Ruby Integration (`lib/`, `spec/`):** The project includes a Ruby library and RSpec tests to ensure seamless integration with Ruby projects.
*   **Fixture Projects (`fixtures/projects/`):** Sample Ruby and Rails projects are included for testing and demonstration purposes.
*   **Rake Tasks (`Rakefile`):** Rake tasks are used to automate building, testing, and linting for both the Go and Ruby parts of the project.

## Building and Running

### Installation

To build and install the `plur` binary:

```bash
bin/rake install
```

### Running Tests

To run tests using `plur`:

```bash
# Run tests in the current directory
plur

# Run tests with a specific number of workers
plur -n 4
```

### Development Tasks

The `Rakefile` provides several tasks for development:

*   **Run all tests and linters:**
    ```bash
    rake
    ```
*   **Run Go tests:**
    ```bash
    rake test:go
    ```
*   **Run Ruby integration tests:**
    ```bash
    rake test
    ```
*   **Lint Go and Ruby code:**
    ```bash
    rake lint
    ```

## Development Conventions

*   **Linting:**
    *   Go code is linted with `go fmt` and `go vet`.
    *   Ruby code is linted with `standard`.
*   **Testing:**
    *   Go code has unit tests in the `plur/` directory.
    *   Ruby integration tests are located in the `spec/` directory.
*   **Configuration:** Project-specific configuration is managed in a `.plur.toml` file.
*   **Documentation:** The `docs/` directory contains project documentation, which can be viewed locally using `script/docs`.
