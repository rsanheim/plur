#!/usr/bin/env python3
"""
Generate a markdown file listing all documentation pages sorted by last modified date.
This script should be run from the project root directory.

Usage:
    python3 docs/generate_pages_list.py

The script will create/update docs/recent-pages.md with a sorted list of all documentation.
"""

import subprocess
import os
from pathlib import Path
from datetime import datetime
import re

def get_git_timestamp(file_path):
    """Get the last git commit timestamp for a file."""
    try:
        result = subprocess.run(
            ["git", "log", "-1", "--format=%at", str(file_path)],
            stdout=subprocess.PIPE,
            stderr=subprocess.DEVNULL,
            text=True,
            cwd="."
        )
        if result.stdout.strip():
            return int(result.stdout.strip())
    except:
        pass
    return None

def get_title_from_markdown(file_path):
    """Extract the title from a markdown file."""
    try:
        with open(file_path, 'r', encoding='utf-8') as f:
            content = f.read()
            
            # Try to find YAML frontmatter title
            if content.startswith('---'):
                yaml_match = re.search(r'^---\s*\ntitle:\s*(.+?)\n', content, re.MULTILINE)
                if yaml_match:
                    return yaml_match.group(1).strip('"\'')
            
            # Try to find first H1 heading
            h1_match = re.search(r'^#\s+(.+)$', content, re.MULTILINE)
            if h1_match:
                return h1_match.group(1).strip()
                
    except:
        pass
    
    # Fallback to filename
    return file_path.stem.replace('-', ' ').replace('_', ' ').title()

def main():
    docs_dir = Path("docs")
    entries = []
    
    # Collect all markdown files
    for md_file in docs_dir.rglob("*.md"):
        # Skip some files
        rel_path = md_file.relative_to(docs_dir)
        path_str = str(rel_path)

        # Skip WIP, overrides, README files (conflicts with index.md), and the recent-pages file itself
        if 'wip/' in path_str or 'overrides/' in path_str or md_file.name == 'recent-pages.md' or md_file.name == 'README.md':
            continue
        
        timestamp = get_git_timestamp(md_file)
        if timestamp:
            dt = datetime.fromtimestamp(timestamp)
            title = get_title_from_markdown(md_file)
            
            # Determine section
            parts = rel_path.parts
            if len(parts) > 1:
                section = parts[0].replace('-', ' ').replace('_', ' ').title()
            else:
                section = "Root"
            
            # Create relative URL path (keep .md extension for MkDocs)
            url_path = str(rel_path)
            # For index files, link to the directory
            if url_path.endswith('/index.md'):
                url_path = url_path.replace('/index.md', '/')
            
            entries.append({
                'date': dt,
                'date_str': dt.strftime('%Y-%m-%d'),
                'title': title,
                'path': url_path,
                'section': section,
                'is_index': md_file.name == 'index.md'
            })
    
    # Sort by date (newest first)
    entries.sort(key=lambda x: x['date'], reverse=True)
    
    # Generate markdown content
    content = """# Recent Documentation Updates

This page lists all documentation pages sorted by their last git modification date (newest first).

Last generated: {}

## All Pages by Last Modified Date

| Page | Path | Section | Last Modified |
|------|------|---------|---------------|
""".format(datetime.now().strftime('%Y-%m-%d %H:%M'))
    
    for entry in entries:
        # Skip index pages in the table (they're usually less interesting)
        if not entry['is_index']:
            # Truncate long paths for display
            display_path = entry['path']
            if len(display_path) > 40:
                display_path = '...' + display_path[-37:]
            
            content += "| [{}]({}) | `{}` | {} | {} |\n".format(
                entry['title'],
                entry['path'],
                display_path,
                entry['section'],
                entry['date_str']
            )
    
    content += """

---

*This page is automatically generated from git history. Rebuild the docs to update.*
"""
    
    # Write the output file
    output_file = docs_dir / "recent-pages.md"
    with open(output_file, 'w', encoding='utf-8') as f:
        f.write(content)
    
    print(f"Generated {output_file} with {len(entries)} pages")

if __name__ == "__main__":
    main()