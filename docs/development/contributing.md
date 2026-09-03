We welcome contributions! This guide will help you get started.

## Development Setup

```bash
# Clone the repository
git clone https://github.com/rsanheim/plur.git
cd plur

# Install dependencies
bundle install
go mod vendor

# Build and install
bin/rake install
```

## Development Workflow

1. Make changes
2. Run `bin/rake install` to test globally
3. Run `bin/rake` to run all tests and lints
4. Fix any issues with `bin/rake standard:fix`
5. Commit your changes

## Profiling a Run

The hidden `--dev-profile DIR` flag (or `PLUR_DEV_PROFILE=DIR`) makes any plur command write Go runtime profiles at exit: `cpu.pprof`, `heap.pprof`, `goroutine.txt`, and `goroutineleak.txt` under `DIR/plur-<pid>/`. Inspect the binary profiles with `go tool pprof`; the goroutine files are plain text, and a leak-free run has a `goroutineleak.txt` that starts with `goroutineleak profile: total 0`.

## Pull Request Process

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Ensure all tests pass
5. Submit a pull request

## Code Style

- Go: Follow standard Go conventions
- Ruby: Use StandardRB (enforced by `bin/rake`)
- Keep changes focused and atomic
