# Multi-Platform Builds

Cross-compile plur for Linux containers from macOS.

## Quick Start

```bash
# Build Linux binaries (amd64 & arm64)
bin/rake build:linux

# Install on Docker container
script/install-plur-docker CONTAINER_NAME

# Install on docker-compose managed container
script/install-plur-docker SERVICE_NAME -C COMPOSE_PREFIX

# Use plur in container
docker exec CONTAINER_NAME plur
```

## Build Tasks

```bash
bin/rake build:linux    # Linux amd64 & arm64
bin/rake build:all      # All platforms (macOS & Linux)
bin/rake build:list     # Show built binaries
bin/rake build:clean    # Remove dist/ directory
```

Binaries are created in `dist/` with embedded version info via ldflags.

## Docker Installation

The `install-plur-docker` script:
- Auto-detects container architecture
- Copies appropriate binary to `/usr/local/bin/plur`
- Verifies installation

```bash
# Auto-detect architecture (bare container name)
script/install-plur-docker my-container

# With docker-compose prefix (adds _1 suffix automatically)
script/install-plur-docker my-service -C docker-compose

# Specify binary
script/install-plur-docker my-container dist/plur-linux-arm64
```

## Technical Details

- Uses `GOOS/GOARCH` for cross-compilation
- `CGO_ENABLED=0` for static binaries
- `-s -w` ldflags for smaller size (~4.3MB)
- Version embedded from git tags