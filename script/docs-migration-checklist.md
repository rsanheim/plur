# Documentation Migration Checklist

This checklist outlines the manual steps needed to complete the documentation reorganization for MkDocs auto-navigation.

## Automated Steps (run `script/reorganize-docs`)

- [x] Create new directory structure (clean names, no numbers)
- [x] Move files to new locations with better names
- [x] Create index.md files for each section
- [x] Update mkdocs.yml to hybrid navigation (section order + auto-discovery)
- [x] Create new top-level files (getting-started, installation, usage, configuration)

## Manual Steps Required

### 1. Content Consolidation

#### Watch Mode Documentation
Consolidate these files into `docs/features/watch-mode.md`:
- [ ] `architecture/rux-watch-architecture.md`
- [ ] `architecture/watch-mode-concurrent-output-issue.md`
- [ ] `archive/2025-06-03-spike-into-adding-watcher.md` (key insights only)
- [ ] `archive/2025-06-04-rux-watch-multiple-watchers-plan.md` (future plans)

#### Backspin Documentation
Consolidate these files into `docs/internals/backspin-integration.md`:
- [ ] `research/backspin-api-analysis.md`
- [ ] `research/backspin-filter-vs-match-research.md`
- [ ] `research/backspin-io-capture-design.md`

### 2. Content Extraction

#### From `development/user-guide.md`:
- [ ] Extract remaining content not in getting-started/installation/usage
- [ ] Move troubleshooting section to relevant feature docs
- [ ] Create `docs/features/doctor-command.md` from doctor content

#### From `rux-optimization-plan.md`:
- [ ] Extract future plans to `docs/overview/roadmap.md`
- [ ] Move completed optimizations to relevant architecture docs
- [ ] Archive the original file

### 3. New Content Creation

#### Section Landing Pages
- [ ] `docs/architecture/concurrency-model.md` - Extract from project-status.md
- [ ] `docs/features/parallel-execution.md` - Core feature documentation
- [ ] `docs/development/contributing.md` - Contribution guidelines
- [ ] `docs/development/testing.md` - How to test Rux itself

### 4. Link Updates

After reorganization, update all internal links:
- [ ] Search for `](architecture/` and verify paths (some may stay the same)
- [ ] Search for `](development/` and verify paths (some may stay the same)
- [ ] Search for `](research/` and update to new locations in internals/
- [ ] Search for `](_archive/` and update to `](archive/`
- [ ] Update root-level doc references to new locations

### 5. Cleanup

- [ ] Remove empty directories after migration
- [ ] Delete the font-preview.html (already in .gitignore)
- [ ] Archive the reorganization-plan.md after completion
- [ ] Delete this checklist after completion

### 6. Validation

- [ ] Run `script/serve-docs build` to ensure no broken links
- [ ] Browse each section to verify auto-navigation works correctly
- [ ] Check that section order matches mkdocs.yml nav configuration
- [ ] Verify search functionality still works

## Commands to Run

```bash
# 1. First, do a dry run to see what will change
script/reorganize-docs --dry-run

# 2. Run the actual reorganization
script/reorganize-docs

# 3. Check the changes
git status

# 4. Complete manual consolidations (see above)

# 5. Update all internal links
# Use your editor's find/replace across files

# 6. Test the new structure
script/serve-docs build

# 7. Browse locally to verify
script/serve-docs

# 8. Commit the changes
git add -A
git commit -m "Reorganize docs for MkDocs auto-navigation"
```

## Benefits After Migration

1. **Auto-discovery**: New docs automatically appear within their sections
2. **Controlled ordering**: Section order defined in mkdocs.yml, files auto-discovered within
3. **Less maintenance**: Only need to update mkdocs.yml when adding new sections
4. **Better organization**: Related content grouped together
5. **Cleaner URLs**: Semantic paths without number prefixes (e.g., `/architecture/` not `/02-architecture/`)

## Understanding the Hybrid Approach

The new structure uses a hybrid navigation approach:

1. **Top-level items** are explicitly ordered in mkdocs.yml:
   ```yaml
   nav:
     - Home: index.md
     - Getting Started: getting-started.md
     - Installation: installation.md
     - Usage: usage.md
     - Configuration: configuration.md
     - Overview: overview/*        # Section 1
     - Architecture: architecture/*  # Section 2
     - Features: features/*         # Section 3
     # etc...
   ```

2. **Within each section**, MkDocs auto-discovers all markdown files
3. **Clean URLs**: `/features/watch-mode/` instead of `/03-features/watch-mode/`
4. **Easy additions**: Drop a new `.md` file in any section and it appears automatically