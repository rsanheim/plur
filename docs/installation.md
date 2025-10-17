# Installation

## Prerequisites

- Ruby 2.7+ with RSpec or Minitest
- Go 1.21+ (only if building from source)

## Installation Methods

### Pre-built Binaries (Coming Soon)

Binary releases will be available for:
- macOS (Intel & Apple Silicon)
- Linux (x86_64)
- Windows (x86_64)

### From Source

```bash
# Clone the repository
git clone https://github.com/rsanheim/plur-meta.git
cd plur-meta

# Install using rake (recommended)
bin/rake install

# This builds and installs plur to your $GOPATH/bin
# Make sure $GOPATH/bin is in your PATH
```

### Using Go Install

```bash
# Install directly with go
go install github.com/rsanheim/plur-meta/plur@latest
```

## Verify Installation

```bash
# Check version
plur --version

# Run doctor command to verify setup
plur doctor
```

## Troubleshooting

### Common Issues

**"command not found: plur"**
- Ensure `$GOPATH/bin` is in your PATH
- Run `echo $PATH` to verify
- Add to your shell profile: `export PATH=$GOPATH/bin:$PATH`

**"cannot load such file -- backspin"**
- Run `bundle install` in your project root
- Backspin is required for golden testing features

**Build failures**
- Ensure Go 1.21+ is installed: `go version`
- Try cleaning and rebuilding: `bin/rake clean install`

## Next Steps

- See [Getting Started](getting-started.md) for a quick introduction
- See [Usage](usage.md) for detailed command documentation
- See [Configuration](configuration.md) for customization options