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
* `../mkdocs.yml` - MkDocs configuration (in project root)

mise automatically creates and activates `.venv` using uv when you cd into docs/.

### Adding Dependencies
```bash
cd docs
uv add package-name
uv sync
```

### Building Documentation
```bash
script/docs build        # Build to site/
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

## Python Environment

This directory uses Python 3.14.0 managed by mise with uv for dependencies:

* When you `cd` into `docs/`, mise automatically activates the Python environment
* Dependencies are locked in `uv.lock` (generated automatically)
* The `.venv` directory is created automatically by mise's `uv_venv_auto` setting

## Notes

* MkDocs configuration file (`mkdocs.yml`) must stay in project root due to MkDocs requirements
* Built documentation goes to `site/` in the project root

### Cairo Imaging Setup (macOS)

The social plugin requires cairo graphics library for generating social cards (preview images for social media).

**Already installed:**
* Cairo is installed via Homebrew at `/opt/homebrew/lib`
* Python imaging extras (`mkdocs-material[imaging]`) are in pyproject.toml

**Environment variable fix:**
* `DYLD_FALLBACK_LIBRARY_PATH=/opt/homebrew/lib` is set in `.mise.toml` and in scripts
* This tells Python where to find the Homebrew-installed cairo library
* Without this, you'll see: "no library called cairo-2 was found"

**If you encounter issues:**
1. Ensure cairo is installed: `brew install cairo freetype libffi libjpeg libpng zlib`
2. Trust the mise config: `cd docs && mise trust`
3. Enable experimental features: `mise settings experimental=true`

See: https://squidfunk.github.io/mkdocs-material/plugins/requirements/image-processing/
