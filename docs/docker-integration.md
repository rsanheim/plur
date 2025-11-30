# Docker Integration Guide

This guide covers all methods for integrating plur with Docker containers, from simple one-off installations to production deployment patterns.

## Quick Start

### Method 1: Self-Install Script (Recommended for Running Containers)

Run this inside any container to install plur:

```bash
# Install latest version
curl -sSL https://raw.githubusercontent.com/rsanheim/plur/main/install.sh | sh

# Install specific version
curl -sSL https://raw.githubusercontent.com/rsanheim/plur/main/install.sh | sh -s -- v1.0.0

# With custom install path
PLUR_INSTALL_PATH=/opt/bin curl -sSL https://raw.githubusercontent.com/rsanheim/plur/main/install.sh | sh
```

### Method 2: Host-Based Installation

Install from your host machine into running containers:

```bash
# Single container
script/install-plur-docker my-container

# Docker Compose service
script/install-plur-docker web -C myproject

# Pattern matching
script/install-plur-docker --pattern "myapp_.*"

# From file list
script/install-plur-docker --file containers.txt

# Parallel installation
script/install-plur-docker --pattern ".*" --parallel
```

## Dockerfile Integration

### Multi-Stage Build (Recommended)

This approach copies plur from an official image without needing to download:

```dockerfile
# Copy plur from a pre-built image
FROM ghcr.io/rsanheim/plur:latest as plur

FROM ubuntu:22.04
# Copy the plur binary
COPY --from=plur /usr/bin/plur /usr/local/bin/plur

# Your application setup
WORKDIR /app
# ... rest of your Dockerfile
```

### Direct Download in Dockerfile

Download and install during image build:

```dockerfile
FROM ubuntu:22.04

# Install plur using the install script
RUN apt-get update && apt-get install -y curl && \
    curl -sSL https://raw.githubusercontent.com/rsanheim/plur/main/install.sh | sh && \
    apt-get remove -y curl && apt-get autoremove -y

# Your application
WORKDIR /app
```

### Using .deb Package (Ubuntu/Debian)

```dockerfile
FROM ubuntu:22.04

# Install dependencies for downloading
RUN apt-get update && apt-get install -y curl jq

# Download and install the .deb package for the correct architecture
RUN ARCH=$(dpkg --print-architecture | sed 's/amd64/x86_64/;s/arm64/arm64/') && \
    DEB_URL=$(curl -s https://api.github.com/repos/rsanheim/plur/releases/latest | \
              jq -r ".assets[] | select(.name | contains(\"linux_${ARCH}.deb\")) | .browser_download_url") && \
    curl -L -o /tmp/plur.deb "$DEB_URL" && \
    dpkg -i /tmp/plur.deb && \
    rm /tmp/plur.deb

# Clean up
RUN apt-get remove -y curl jq && apt-get autoremove -y && rm -rf /var/lib/apt/lists/*
```

### Using Binary Archive

```dockerfile
FROM ubuntu:22.04

# Install curl for downloading
RUN apt-get update && apt-get install -y curl

# Detect architecture and download appropriate binary
RUN ARCH=$(uname -m | sed 's/x86_64/x86_64/;s/aarch64/arm64/') && \
    VERSION=$(curl -s https://api.github.com/repos/rsanheim/plur/releases/latest | grep tag_name | cut -d'"' -f4 | sed 's/v//') && \
    curl -L "https://github.com/rsanheim/plur/releases/download/v${VERSION}/plur_${VERSION}_Linux_${ARCH}.tar.gz" | \
    tar xz -C /tmp && \
    mv /tmp/plur/plur /usr/local/bin/plur && \
    chmod +x /usr/local/bin/plur && \
    rm -rf /tmp/plur

# Clean up
RUN apt-get remove -y curl && apt-get autoremove -y && rm -rf /var/lib/apt/lists/*
```

## Docker Compose Integration

### Using Init Containers

```yaml
version: '3.8'

services:
  # Init container to install plur
  plur-installer:
    image: alpine:latest
    command: |
      sh -c "
        apk add --no-cache curl
        curl -sSL https://raw.githubusercontent.com/rsanheim/plur/main/install.sh | sh
        cp /usr/local/bin/plur /shared/plur
      "
    volumes:
      - plur-bin:/shared

  # Your application container
  app:
    image: ruby:3.2
    depends_on:
      - plur-installer
    volumes:
      - plur-bin:/usr/local/bin:ro
      - ./:/app
    working_dir: /app
    command: plur -n 4

volumes:
  plur-bin:
```

### Using Entrypoint Wrapper

```yaml
version: '3.8'

services:
  app:
    image: ruby:3.2
    volumes:
      - ./:/app
    working_dir: /app
    entrypoint: |
      sh -c "
        if ! command -v plur >/dev/null 2>&1; then
          echo 'Installing plur...'
          curl -sSL https://raw.githubusercontent.com/rsanheim/plur/main/install.sh | sh
        fi
        exec plur -n 4
      "
```

### Post-Start Installation

```yaml
version: '3.8'

services:
  web:
    image: ruby:3.2
    volumes:
      - ./:/app
    working_dir: /app
    command: bundle exec rails server

  test:
    image: ruby:3.2
    volumes:
      - ./:/app
    working_dir: /app
    command: |
      sh -c "
        curl -sSL https://raw.githubusercontent.com/rsanheim/plur/main/install.sh | sh
        plur -n 4
      "
```

## CI/CD Integration

### GitHub Actions

```yaml
name: Test with plur

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    container:
      image: ruby:3.2
    steps:
      - uses: actions/checkout@v3

      - name: Install plur
        run: curl -sSL https://raw.githubusercontent.com/rsanheim/plur/main/install.sh | sh

      - name: Run tests
        run: plur -n 4
```

### GitLab CI

```yaml
test:
  image: ruby:3.2
  before_script:
    - curl -sSL https://raw.githubusercontent.com/rsanheim/plur/main/install.sh | sh
  script:
    - plur -n 4
```

### CircleCI

```yaml
version: 2.1

jobs:
  test:
    docker:
      - image: cimg/ruby:3.2
    steps:
      - checkout
      - run:
          name: Install plur
          command: curl -sSL https://raw.githubusercontent.com/rsanheim/plur/main/install.sh | sh
      - run:
          name: Run tests
          command: plur -n 4
```

## Production Patterns

### Persistent Binary Volume

For production environments where multiple containers need plur:

```yaml
# docker-compose.prod.yml
version: '3.8'

services:
  # One-time setup container
  setup:
    image: alpine:latest
    command: |
      sh -c "
        if [ ! -f /binaries/plur ]; then
          apk add --no-cache curl
          curl -sSL https://raw.githubusercontent.com/rsanheim/plur/main/install.sh | PLUR_INSTALL_PATH=/binaries sh
        fi
      "
    volumes:
      - app-binaries:/binaries

  app:
    image: ruby:3.2
    volumes:
      - app-binaries:/usr/local/bin:ro
      - ./:/app
    depends_on:
      - setup

volumes:
  app-binaries:
    driver: local
```

### Custom Base Image

Create your own base image with plur pre-installed:

```dockerfile
# Dockerfile.base
FROM ruby:3.2

# Install plur
RUN curl -sSL https://raw.githubusercontent.com/rsanheim/plur/main/install.sh | sh

# Set as default test runner
ENV TEST_RUNNER=plur

# Build: docker build -f Dockerfile.base -t myorg/ruby-plur:3.2 .
# Push: docker push myorg/ruby-plur:3.2
```

Then use in your application:

```dockerfile
FROM myorg/ruby-plur:3.2

WORKDIR /app
COPY . .
RUN bundle install

CMD ["plur", "-n", "4"]
```

## Batch Installation Examples

### Install to All Containers

```bash
# Install to all running containers
docker ps --format '{{.Names}}' | xargs -I {} script/install-plur-docker {}

# Or using the pattern feature
script/install-plur-docker --pattern ".*"
```

### Install to Specific Environment

```bash
# Create a file with container names
docker ps --filter "label=environment=staging" --format '{{.Names}}' > staging-containers.txt

# Install to all staging containers
script/install-plur-docker --file staging-containers.txt --parallel
```

### Docker Swarm Services

```bash
# Get all containers for a service
SERVICE="web"
docker service ps $SERVICE --format '{{.Name}}' | \
  xargs -I {} script/install-plur-docker {}
```

## Troubleshooting

### Architecture Detection Issues

If the install script can't detect architecture:

```bash
# Manually specify architecture
ARCH=arm64  # or x86_64
curl -L "https://github.com/rsanheim/plur/releases/latest/download/plur_VERSION_Linux_${ARCH}.tar.gz" | \
  tar xz -C /usr/local/bin plur
```

### Permission Issues

If you get permission denied errors:

```bash
# Option 1: Use sudo in container (if available)
curl -sSL https://raw.githubusercontent.com/rsanheim/plur/main/install.sh | sudo sh

# Option 2: Install to user directory
PLUR_INSTALL_PATH=$HOME/.local/bin curl -sSL https://raw.githubusercontent.com/rsanheim/plur/main/install.sh | sh
export PATH=$HOME/.local/bin:$PATH
```

### Container Doesn't Have curl/wget

```dockerfile
# In Dockerfile, use multi-stage build
FROM alpine:latest as downloader
RUN apk add --no-cache curl
RUN curl -sSL https://raw.githubusercontent.com/rsanheim/plur/main/install.sh | sh

FROM your-base-image
COPY --from=downloader /usr/local/bin/plur /usr/local/bin/plur
```

### Verify Installation

Always verify plur is installed correctly:

```bash
# Check version
docker exec my-container plur --version

# Run doctor command
docker exec my-container plur doctor

# Check all containers
for container in $(docker ps --format '{{.Names}}'); do
  echo "=== $container ==="
  docker exec $container plur --version 2>/dev/null || echo "Not installed"
done
```

## Best Practices

1. **Use Multi-Stage Builds**: Reduces final image size by not including download tools
2. **Version Pin in Production**: Always specify exact versions in production Dockerfiles
3. **Cache Binary Downloads**: Use Docker layer caching or volumes to avoid re-downloading
4. **Health Checks**: Add plur doctor to container health checks
5. **Resource Limits**: Set appropriate CPU/memory limits for parallel test execution

## Security Considerations

1. **Verify Checksums**: For production, verify downloaded binaries against published checksums
2. **Use Official Images**: When possible, use official plur Docker images
3. **Minimize Attack Surface**: Remove download tools (curl/wget) after installation
4. **Run as Non-Root**: Configure containers to run as non-root users

```dockerfile
# Example: Secure installation
FROM ubuntu:22.04

# Install as root
RUN curl -sSL https://raw.githubusercontent.com/rsanheim/plur/main/install.sh | sh && \
    apt-get remove -y curl && apt-get autoremove -y

# Create non-root user
RUN useradd -m -s /bin/bash appuser
USER appuser

WORKDIR /app
# plur is available to non-root user
CMD ["plur", "--version"]
```

## Next Steps

* Review [Installation Guide](installation.md) for non-Docker installation methods
* Check [Configuration](configuration.md) for plur configuration options