# Documentation Reorganization Plan for MkDocs Auto-Navigation

## Goal
Restructure documentation to work seamlessly with MkDocs auto-navigation, minimizing manual nav configuration while maintaining clear organization.

## Proposed Structure

```
docs/
├── index.md                          # Home page (renamed from README.md)
├── getting-started.md                # Quick start guide (extracted from user-guide.md)
├── installation.md                   # Installation instructions (extracted from user-guide.md)
├── usage.md                         # How to use rux (extracted from user-guide.md)
├── configuration.md                 # Configuration options
├── changelog.md                     # Link to ../CHANGELOG.md
│
├── overview/                        # Project overview and status
│   ├── index.md                     # Section landing page
│   ├── project-status.md            # Current status
│   └── roadmap.md                   # Future plans (extracted from optimization-plan.md)
│
├── architecture/                    # Technical architecture
│   ├── index.md                     # Architecture overview
│   ├── cli-design.md               # CLI context comparison → renamed
│   ├── concurrency-model.md        # Worker pool and goroutines
│   ├── output-design.md            # Output and logging design
│   └── performance-tracing.md       # Performance debugging
│
├── features/                        # Feature documentation
│   ├── index.md                     # Features overview
│   ├── parallel-execution.md        # Parallel test execution
│   ├── watch-mode.md               # Watch mode (consolidated)
│   └── doctor-command.md           # Doctor command details
│
├── development/                     # Development guides
│   ├── index.md                     # Development overview
│   ├── contributing.md              # How to contribute
│   ├── testing.md                   # Testing the project
│   ├── release-process.md           # Release process (moved from root)
│   └── go-vendoring.md             # Go vendoring and CI
│
├── internals/                       # Internal implementation details
│   ├── index.md                     # Internals overview
│   ├── backspin-integration.md      # Backspin usage (consolidated)
│   ├── file-mapping.md             # File mapping formats
│   └── snapshot-testing.md          # Snapshot testing approaches
│
└── archive/                         # Historical documents (remove underscore)
    ├── index.md                     # Archive overview
    └── [existing archive files]     # Keep as-is
```

## Key Changes

### 1. Rename Files for Better Auto-Navigation
- `README.md` → `index.md` (MkDocs convention for landing pages)
- `user-guide.md` → Split into: `getting-started.md`, `installation.md`, `usage.md`
- `cli-context-comparison.md` → `cli-design.md`
- `rux-output-and-design.md` → `output-design.md`
- `go-vendoring-and-ci.md` → `go-vendoring.md`

### 2. Add Index Files
Each directory gets an `index.md` that serves as a landing page for that section.

### 3. Control Order via mkdocs.yml
Use clean directory names and control the order through mkdocs.yml nav configuration.

### 4. Consolidate Related Content
- Merge all watch-mode related docs into `features/watch-mode.md`
- Merge backspin research into `internals/backspin-integration.md`
- Extract roadmap/future plans into dedicated `roadmap.md`

### 5. Flatten Research Directory
Move research content into appropriate sections:
- Backspin research → `internals/backspin-integration.md`
- File mapping → `internals/file-mapping.md`
- Snapshot testing → `internals/snapshot-testing.md`

## Updated mkdocs.yml Configuration

```yaml
site_name: Rux Documentation
# ... theme and other settings ...

# Hybrid navigation - explicit section ordering with auto-discovery within sections
nav:
  - Home: index.md
  - Getting Started: getting-started.md
  - Installation: installation.md
  - Usage: usage.md
  - Configuration: configuration.md
  - Overview:
    - overview/*
  - Architecture:
    - architecture/*
  - Features:
    - features/*
  - Development:
    - development/*
  - Internals:
    - internals/*
  - Archive:
    - archive/*
```

## Benefits
1. **Low Maintenance**: Only need to maintain section order, not individual files
2. **Clear Structure**: Explicit section ordering with clean directory names
3. **Better URLs**: Cleaner, more semantic URLs without number prefixes
4. **Easier to Find**: Related content is grouped together
5. **Future-Proof**: Easy to add new docs within sections without updating config

## Migration Steps
1. Create new directory structure
2. Move and rename files
3. Create index.md files for each section
4. Update internal links
5. Simplify mkdocs.yml nav configuration
6. Test with `script/serve-docs build`