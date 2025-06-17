#!/bin/bash
set -euo pipefail

GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

print_step() {
    echo -e "\n${GREEN}>>> $1${NC}"
}

print_error() {
    echo -e "${RED}ERROR: $1${NC}" >&2
}

# Verify environment
print_step "Verifying environment..."
echo "Go version: $(go version)"
echo "Ruby version: $(ruby --version)"
echo "Bundle version: $(bundle --version)"
echo "Working directory: $(pwd)"
echo "Architecture: $(uname -m)"

# Install Ruby dependencies
print_step "Installing Ruby dependencies..."
bundle install

# Build rux
print_step "Building rux for Linux..."
cd rux
go mod download
go mod vendor
cd ..
bin/rake install

# Add Go bin to PATH
export PATH="/go/bin:$PATH"

# Verify rux installation
print_step "Verifying rux installation..."
which rux
rux --version

# Test watcher installation
print_step "Testing watcher binary extraction..."
rux doctor

# Run tests on fixture projects
print_step "Running tests on fixture projects..."

for project in fixtures/projects/*/; do
    if [ -d "$project" ] && [ -f "$project/Gemfile" ]; then
        project_name=$(basename "$project")
        echo -e "\n${BLUE}Testing project: $project_name${NC}"
        
        cd "$project"
        
        # Install dependencies
        bundle install --quiet
        
        # Run rux
        if rux --dry-run; then
            echo -e "${GREEN}✓ $project_name: dry-run passed${NC}"
            
            # Run actual tests if spec directory exists
            if [ -d "spec" ]; then
                if timeout 30s rux -n 2; then
                    echo -e "${GREEN}✓ $project_name: tests passed${NC}"
                else
                    echo -e "${RED}✗ $project_name: tests failed${NC}"
                fi
            fi
        else
            echo -e "${RED}✗ $project_name: dry-run failed${NC}"
        fi
        
        cd - > /dev/null
    fi
done

# Run rux's own test suite
print_step "Running rux test suite..."
cd /workspace
bin/rake test:ruby

print_step "All tests complete!"