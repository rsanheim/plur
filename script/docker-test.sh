#!/bin/bash
set -euo pipefail

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}=== Rux Docker Test Runner ===${NC}"

# Function to print step headers
print_step() {
    echo -e "\n${GREEN}>>> $1${NC}"
}

# Function to print errors
print_error() {
    echo -e "${RED}ERROR: $1${NC}" >&2
}

# Parse command line arguments
RUN_SPECIFIC=""
BUILD_ONLY=false
SHELL_ONLY=false
PLATFORM=""

while [[ $# -gt 0 ]]; do
    case $1 in
        --build)
            BUILD_ONLY=true
            shift
            ;;
        --shell)
            SHELL_ONLY=true
            shift
            ;;
        --run)
            RUN_SPECIFIC="$2"
            shift 2
            ;;
        --platform)
            PLATFORM="$2"
            shift 2
            ;;
        --ps)
            print_step "Docker container status..."
            docker-compose ps
            exit 0
            ;;
        --logs)
            print_step "Docker container logs..."
            docker-compose logs $2
            exit 0
            ;;
        *)
            echo "Usage: $0 [--build] [--shell] [--run <command>] [--platform <platform>] [--ps] [--logs [service]]"
            echo "  --build           Only build the Docker image"
            echo "  --shell           Start an interactive shell"
            echo "  --run <cmd>       Run a specific command"
            echo "  --platform <arch> Build/run for specific platform (linux/amd64, linux/arm64)"
            echo "  --ps              Show container status"
            echo "  --logs [service]  Show container logs"
            echo ""
            echo "Examples:"
            echo "  $0 --platform linux/amd64  # Run tests on x86_64"
            echo "  $0 --platform linux/arm64  # Run tests on ARM64"
            exit 1
            ;;
    esac
done

# Set platform args if specified
PLATFORM_ARGS=""
if [ -n "$PLATFORM" ]; then
    PLATFORM_ARGS="--platform $PLATFORM"
    print_step "Using platform: $PLATFORM"
fi

# Build the Docker image
print_step "Building Docker image..."
if [ -n "$PLATFORM" ]; then
    docker buildx build $PLATFORM_ARGS -t rux-test:latest .
else
    docker-compose build
fi

if [ "$BUILD_ONLY" = true ]; then
    echo -e "${GREEN}Build complete!${NC}"
    exit 0
fi

if [ "$SHELL_ONLY" = true ]; then
    print_step "Starting interactive shell..."
    if [ -n "$PLATFORM" ]; then
        docker run --rm -it $PLATFORM_ARGS -v $(pwd):/workspace:rw -v /workspace/references -v /workspace/vendor -w /workspace rux-test:latest bash
    else
        docker-compose run --rm rux-test
    fi
    exit 0
fi

if [ -n "$RUN_SPECIFIC" ]; then
    print_step "Running: $RUN_SPECIFIC"
    if [ -n "$PLATFORM" ]; then
        docker run --rm $PLATFORM_ARGS -v $(pwd):/workspace:rw -v /workspace/references -v /workspace/vendor -w /workspace rux-test:latest bash -c "$RUN_SPECIFIC"
    else
        docker-compose run --rm rux-test bash -c "$RUN_SPECIFIC"
    fi
    exit 0
fi

# Default: Run full test suite
print_step "Running full test suite in Docker..."

# Run using the standalone script
if [ -n "$PLATFORM" ]; then
    docker run --rm $PLATFORM_ARGS -v $(pwd):/workspace:rw -v /workspace/references -v /workspace/vendor -w /workspace rux-test:latest /workspace/scripts/docker-test-runner.sh
else
    docker-compose run --rm rux-test /workspace/scripts/docker-test-runner.sh
fi

echo -e "\n${GREEN}Docker tests complete!${NC}"