# MkDocs Migration to docs/ with mise + uv

## Overview

Migrate Python/MkDocs setup from repo root to `docs/` subdirectory using mise + uv for cleaner project organization.

## Current State

### Root Directory Files (to be removed/moved)
* `requirements.txt` - 8 MkDocs dependencies
* `mkdocs.yml` - MkDocs configuration
* `.venv/` - Virtual environment
* `script/activate-python` - venv activation script
* `script/docs` - MkDocs server script
* `script/check-links` - Link validation script

### Dependencies to Migrate
```txt
mkdocs>=1.5.3
mkdocs-material[imaging]>=9.5.0
pymdown-extensions>=10.5
mkdocs-material-extensions>=1.3
mkdocs-awesome-pages-plugin>=2.9.2
mkdocs-git-revision-date-localized-plugin>=1.2.6
mkdocs-panzoom-plugin @ git+https://github.com/PLAYG0N/mkdocs-panzoom.git
linkcheckmd>=1.4.0
```

### Environment
* Python: 3.14.0 (to be installed via mise)
* uv: installed
* mise: 2025.10.7 (installed)

## Target State

### docs/ Directory Structure
```
docs/
├── .mise.toml              # Python version + uv integration
├── pyproject.toml          # Dependencies (replaces requirements.txt)
├── uv.lock                 # Lock file (auto-generated)
├── .venv/                  # Virtual environment (mise-managed)
├── mkdocs.yml              # MkDocs config (moved from root)
├── generate_pages_list.py  # Existing utility
├── stylesheets/            # Existing assets
└── [markdown files]        # Existing docs
```

### Root Directory Scripts (updated)
* `script/docs` - Updated to run from docs/ directory
* `script/check-links` - Updated to run from docs/ directory
* `script/activate-python` - REMOVED (mise handles this)

## Implementation Plan

### Phase 1: Set up mise + uv in docs/

#### Step 1.1: Create docs/.mise.toml
```toml
[tools]
python = "3.14.0"

[settings]
python.uv_venv_auto = true
```

This tells mise to automatically create and manage a `.venv` using uv when entering the docs/ directory.

#### Step 1.2: Initialize uv project in docs/
```bash
cd docs
uv init --no-readme --no-workspace
```

This creates:
* `pyproject.toml` with basic structure
* `.python-version` file

#### Step 1.3: Update pyproject.toml metadata
Edit `docs/pyproject.toml` to include:
```toml
[project]
name = "plur-docs"
version = "0.1.0"
description = "Documentation for Plur parallel test runner"
requires-python = ">=3.14"
dependencies = [
    "mkdocs>=1.5.3",
    "mkdocs-material[imaging]>=9.5.0",
    "pymdown-extensions>=10.5",
    "mkdocs-material-extensions>=1.3",
    "mkdocs-awesome-pages-plugin>=2.9.2",
    "mkdocs-git-revision-date-localized-plugin>=1.2.6",
    "mkdocs-panzoom-plugin @ git+https://github.com/PLAYG0N/mkdocs-panzoom.git",
    "linkcheckmd>=1.4.0",
]
```

#### Step 1.4: Sync dependencies
```bash
cd docs
# mise will automatically install Python 3.14.0 and create .venv via uv
uv sync  # Install all dependencies
```

#### Step 1.5: Verify setup works
```bash
cd docs
uv run mkdocs --version
uv run python generate_pages_list.py
```

### Phase 2: Move mkdocs.yml

#### Step 2.1: Move configuration file
```bash
mv mkdocs.yml docs/mkdocs.yml
```

**Note:** No changes to `mkdocs.yml` needed - it already has `docs_dir: docs` which is now relative to the config file's location.

#### Step 2.2: Update mkdocs.yml docs_dir
Since mkdocs.yml is now inside docs/, update the docs_dir:
```yaml
docs_dir: .
```

All other paths in mkdocs.yml remain the same (they're relative to docs_dir).

#### Step 2.3: Test MkDocs works from new location
```bash
cd docs
uv run mkdocs build
uv run mkdocs serve
```

Verify at http://localhost:8000 that all pages load correctly.

### Phase 3: Update Scripts

#### Step 3.1: Update script/docs
Replace entire file:
```bash
#!/usr/bin/env bash
set -euo pipefail

# Script info
readonly SCRIPT_NAME=$(basename "$0")
readonly SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
readonly PROJECT_ROOT=$(cd "${SCRIPT_DIR}/.." && pwd)
readonly DOCS_DIR="${PROJECT_ROOT}/docs"

# Show usage information
show_help() {
  cat <<EOF
Plur Documentation CLI

To serve docs locally run:

script/docs

Usage: ${SCRIPT_NAME} [COMMAND]

Commands (_note that the command will default to 'serve' if not provided_)

  serve       Start development server at http://localhost:8000 (default)
  build       Build documentation to site/
  clean       Remove site/ directory
  clean-build Clean and rebuild documentation
  help        Show this help message

Examples:
  ${SCRIPT_NAME}                    # Start dev server
  ${SCRIPT_NAME} build              # Build static site
  ${SCRIPT_NAME} clean-build        # Clean rebuild

Environment:
  Uses mise + uv for Python dependency management.
  Configuration in docs/.mise.toml and docs/pyproject.toml

EOF
}

# Check prerequisites
check_requirements() {
  if ! command -v mise &> /dev/null; then
    cat <<EOF
Error: mise is not installed.

Install it with:
  curl https://mise.run | sh

Then reload your shell.
EOF
    exit 1
  fi

  if ! command -v uv &> /dev/null; then
    cat <<EOF
Error: uv is not installed.

Install it with:
  curl -LsSf https://astral.sh/uv/install.sh | sh

EOF
    exit 1
  fi
}

# Setup Python environment
setup_environment() {
  cd "${DOCS_DIR}"

  # mise automatically creates .venv via uv when we cd here (uv_venv_auto setting)
  # Just ensure dependencies are installed
  if ! uv run mkdocs --version &> /dev/null 2>&1; then
    echo "Installing MkDocs and dependencies..."
    uv sync
  fi
}

# Main script logic
main() {
  check_requirements
  setup_environment

  # Parse command
  local command="${1:-serve}"

  cd "${DOCS_DIR}"

  case "${command}" in
    clean)
      echo "Cleaning build artifacts..."
      rm -rf site/
      echo "✓ Cleaned site directory"
      ;;

    build)
      echo "Building documentation..."
      uv run mkdocs build
      echo "Documentation built in docs/site/"
      ;;

    clean-build)
      echo "Cleaning and building documentation..."
      rm -rf site/
      uv run mkdocs build
      echo "✓ Documentation rebuilt in docs/site/"
      ;;

    serve)
      echo "Starting MkDocs server..."
      echo "Documentation will be available at http://localhost:8000"
      echo "Press Ctrl+C to stop"
      uv run mkdocs serve
      ;;

    help|--help|-h)
      show_help
      ;;

    *)
      echo "Error: Unknown command '${command}'"
      echo ""
      show_help
      exit 1
      ;;
  esac
}

# Run main function
main "$@"
```

#### Step 3.2: Update script/check-links
Replace entire file:
```bash
#!/usr/bin/env bash
set -euo pipefail

readonly SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
readonly PROJECT_ROOT=$(cd "${SCRIPT_DIR}/.." && pwd)
readonly DOCS_DIR="${PROJECT_ROOT}/docs"

# Check if mise and uv are installed
if ! command -v mise &> /dev/null; then
    echo "Error: mise is not installed."
    echo "Install it with: curl https://mise.run | sh"
    exit 1
fi

if ! command -v uv &> /dev/null; then
    echo "Error: uv is not installed."
    echo "Install it with: curl -LsSf https://astral.sh/uv/install.sh | sh"
    exit 1
fi

cd "${DOCS_DIR}"

# Ensure dependencies are installed
if ! uv run mkdocs --version &> /dev/null; then
    echo "Installing MkDocs and dependencies..."
    uv sync
fi

echo ">>> Checking markdown links..."

# First, run MkDocs build with strict validation
echo ""
echo ">>> Running MkDocs validation..."
if uv run mkdocs build --strict > /tmp/mkdocs-build.log 2>&1; then
    echo "✓ MkDocs validation passed"
else
    echo "✗ MkDocs validation failed"
    echo ""
    echo "MkDocs errors:"
    grep -E "WARNING|ERROR" /tmp/mkdocs-build.log || cat /tmp/mkdocs-build.log
    exit_code=1
fi

# Run linkcheckmd on the docs directory
# Note: linkcheckmd may report directory links as issues when they're actually valid
echo ""
echo ">>> Running linkcheckmd..."
if uv run python -m linkcheckmd . 2>&1 | grep -v "seconds to check links"; then
    # Check if there were actual errors (not just the summary line)
    if [ ${PIPESTATUS[0]} -eq 0 ]; then
        echo "✓ linkcheckmd passed"
    else
        echo "✗ linkcheckmd found broken links"
        echo "Note: Directory links (e.g., 'architecture/') may be falsely reported as broken"
        exit_code=1
    fi
else
    echo "✓ linkcheckmd passed"
fi

# Summary
echo ""
if [ "${exit_code:-0}" -eq 0 ]; then
    echo "✓ All link checks passed!"
else
    echo "✗ Link validation failed. Please fix the broken links above."
    exit 1
fi
```

#### Step 3.3: Delete script/activate-python
```bash
rm script/activate-python
```

mise handles environment activation automatically when you `cd` into `docs/`.

### Phase 4: Update .gitignore

#### Step 4.1: Update Python section
Current `.gitignore` has at the end:
```
# MkDocs
site/

# Python
.venv/
__pycache__/
*.pyc
```

Update to:
```
# MkDocs
site/
docs/site/

# Python
.venv/
docs/.venv/
__pycache__/
*.pyc
uv.lock
```

### Phase 5: Clean Up Root Directory

#### Step 5.1: Remove old files
```bash
# Remove requirements.txt
rm requirements.txt

# Remove root .venv directory
rm -rf .venv
```

#### Step 5.2: Verify no Python artifacts remain in root
```bash
ls -la | grep -E "\.py$|\.venv|__pycache__|requirements"
# Should return nothing
```

### Phase 6: Testing

#### Step 6.1: Test docs serving
```bash
script/docs
```

Visit http://localhost:8000 and verify:
* All pages load
* Navigation works
* Search works
* Images/diagrams render
* Code blocks have syntax highlighting

#### Step 6.2: Test docs building
```bash
script/docs build
```

Verify `docs/site/` is created with complete static site.

#### Step 6.3: Test link checking
```bash
script/check-links
```

Should complete without errors.

#### Step 6.4: Test from fresh clone
```bash
cd /tmp
git clone file:///Users/rsanheim/src/oss/plur plur-test
cd plur-test
script/docs
```

This ensures the setup works without any local artifacts.

### Phase 7: Update Documentation

#### Step 7.1: Update README.md
Find this section:
```markdown
### Viewing Documentation Locally

We use MkDocs Material for browsing documentation. To view the docs locally:

```bash
# requires `uv` - servces documentation at http://localhost:8000
script/serve-docs
```

Replace with:
```markdown
### Viewing Documentation Locally

We use MkDocs Material for browsing documentation. To view the docs locally:

```bash
# Requires mise + uv (both installed automatically if missing)
script/docs
```

The documentation setup uses mise for Python version management and uv for dependency management. All Python dependencies are managed in `docs/pyproject.toml`.

For more documentation commands:
```bash
script/docs help
```

#### Step 7.2: Update CLAUDE.md if needed
Check if any references to requirements.txt exist and update them to mention:
* Python/MkDocs configuration is in `docs/.mise.toml` and `docs/pyproject.toml`
* Use `script/docs` for all documentation tasks

#### Step 7.3: Create docs/README.md
```markdown
# Documentation

This directory contains all documentation for Plur.

## Local Development

### Quick Start
```bash
# From project root
script/docs
```

Visit http://localhost:8000

### Requirements
* mise - Python version management
* uv - Python dependency management

Both are installed automatically by `script/docs` if missing.

### Configuration Files
* `.mise.toml` - Python version and auto-venv setup
* `pyproject.toml` - Python dependencies
* `mkdocs.yml` - MkDocs configuration

mise automatically creates and activates `.venv` using uv when you cd into docs/.

### Adding Dependencies
```bash
cd docs
uv add package-name
uv sync
```

### Building Documentation
```bash
script/docs build        # Build to docs/site/
script/docs clean-build  # Clean rebuild
```

### Checking Links
```bash
script/check-links
```

## Directory Structure
* `architecture/` - Architecture decisions and designs
* `development/` - Development guides and processes
* `features/` - Feature documentation
* `overview/` - Project overview and roadmap
* `research/` - Research and analysis documents
* `wip/` - Work in progress documentation
* `archive/` - Archived historical documents
```

### Phase 8: Commit Changes

#### Step 8.1: Stage all changes
```bash
git add -A
```

#### Step 8.2: Review changes
```bash
git status
git diff --cached
```

Verify:
* Removed: `requirements.txt`, `script/activate-python`, `mkdocs.yml` (from root)
* Modified: `script/docs`, `script/check-links`, `.gitignore`, `README.md`
* Added: `docs/.mise.toml`, `docs/pyproject.toml`, `docs/mkdocs.yml`, `docs/README.md`

#### Step 8.3: Commit
```bash
git commit -m "$(cat <<'EOF'
Migrate Python/MkDocs setup to docs/ subdirectory

* Move all Python configuration to docs/ for cleaner project structure
* Use mise for Python version management (3.14.0)
* Use uv with pyproject.toml instead of requirements.txt
* Update script/docs and script/check-links to work with new layout
* Remove script/activate-python (mise handles activation)
* Keep all functionality working: serve, build, link checking

Configuration now in:
- docs/.mise.toml - Python version + venv config
- docs/pyproject.toml - Dependencies (replaces requirements.txt)
- docs/mkdocs.yml - MkDocs config (moved from root)

All docs commands still work the same:
- script/docs [serve|build|clean-build]
- script/check-links
EOF
)"
```

## Rollback Plan

If anything goes wrong, rollback is simple since we haven't deployed anything:

```bash
git reset --hard HEAD
git clean -fd
```

Then manually restore `.venv/` if needed:
```bash
uv venv
uv pip install -r requirements.txt
```

## Success Criteria

* ✓ Root directory is clean (no Python files or .venv)
* ✓ All docs commands work (`script/docs`, `script/check-links`)
* ✓ Documentation builds correctly
* ✓ Documentation serves correctly at localhost:8000
* ✓ Link validation passes
* ✓ mise automatically activates Python environment in docs/
* ✓ Fresh clone can build and serve docs

## Notes

* **Why Python 3.14?** Latest stable release with performance improvements and new features
* **Why uv_venv_auto?** Simplifies setup - mise automatically creates .venv using uv when entering docs/
* **Why keep scripts in script/?** Maintains existing conventions and muscle memory
* **Why not move generate_pages_list.py?** It's already in docs/ and works fine there
* **CI Impact**: None - no CI workflows currently use Python/docs
* **Docker Impact**: None - no Dockerfiles reference requirements.txt

## References

* [mise documentation](https://mise.jdx.dev/)
* [mise + uv integration](https://mise.jdx.dev/mise-cookbook/python.html#mise-uv)
* [uv documentation](https://docs.astral.sh/uv/)
* [MkDocs documentation](https://www.mkdocs.org/)
