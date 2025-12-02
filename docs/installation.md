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
git clone https://github.com/rsanheim/plur.git
cd plur

# Install using rake (recommended)
bin/rake install

# This builds and installs plur to your $GOPATH/bin
# Make sure $GOPATH/bin is in your PATH
```

### Using Go Install

```bash
go install github.com/rsanheim/plur@latest
```

## Verify Installation

```bash
plur --version

# Run doctor command to verify setup
plur doctor
```

## Next Steps

- See [Getting Started](getting-started.md) for a quick introduction
- See [Usage](usage.md) for detailed command documentation
- See [Configuration](configuration.md) for customization options