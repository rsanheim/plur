We welcome contributions! This guide will help you get started.

## Development Setup

```bash
# Clone the repository
git clone https://github.com/rsanheim/plur-meta.git
cd plur-meta

# Install dependencies
bundle install
cd plur && go mod vendor

# Build and install
bin/rake install
```

## Development Workflow

1. Make changes
2. Run `bin/rake install` to test globally
3. Run `bin/rake` to run all tests and lints
4. Fix any issues with `bin/rake standard:fix`
5. Commit your changes

## Testing

Always test from outside-in using integration specs:
- `spec/general_integration_spec.rb` - Core functionality
- `spec/parallel_execution_spec.rb` - Parallelism
- `spec/error_handling_spec.rb` - Error cases

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