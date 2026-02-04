# Installation

## Prerequisites

- Ruby 3.0+ with RSpec or Minitest
- Go 1.25+ (only if building from source)

## Installation Methods

### Pre-built Binaries

Binary releases are available via GitHub Releases for:
* macOS Apple Silicon (ARM64)
* Linux x86_64
* Linux ARM64

Download from the [GitHub Releases page](https://github.com/rsanheim/plur/releases) for this repository.

### From Source

(requires Go 1.25+) You can build the binary from source and install globally to your $GOPATH/bin.

```bash
git clone https://github.com/rsanheim/plur.git
cd plur

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