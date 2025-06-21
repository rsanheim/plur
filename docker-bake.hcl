# Docker Bake configuration for multi-platform builds
# https://docs.docker.com/build/bake/

variable "EDANT_WATCHER_VERSION" {
  default = "0.13.6"
}

# Default group - builds current platform only  
group "default" {
  targets = ["local"]
}

# Build all platforms with architecture-specific tags
group "all" {
  targets = ["amd64", "arm64"]
}

# Base configuration shared by all targets
target "_base" {
  dockerfile = "Dockerfile"
  context = "."
  args = {
    EDANT_WATCHER_VERSION = "${EDANT_WATCHER_VERSION}"
  }
}

# Build for current platform as 'latest'
target "local" {
  inherits = ["_base"]
  tags = ["rux-test:latest"]
}

# AMD64/x86_64 build with architecture tag
target "amd64" {
  inherits = ["_base"]
  tags = ["rux-test:amd64"]
  platforms = ["linux/amd64"]
}

# ARM64 build with architecture tag
target "arm64" {
  inherits = ["_base"]
  tags = ["rux-test:arm64"]
  platforms = ["linux/arm64"]
}