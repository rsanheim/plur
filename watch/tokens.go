package watch

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

// Tokens represents all available template variables for path transformations
type Tokens struct {
	Match       string // filename without extension relative to source pattern base
	Path        string // full path
	Dir         string // directory with trailing slash
	DirRelative string // directory relative to source pattern base
	Base        string // basename (filename with extension)
	Name        string // filename without extension
	Ext         string // extension with dot (.rb, .go)
	ExtNoDot    string // extension without dot (rb, go)
}

// BuildTokens creates a Tokens struct from a file path and source pattern
// The sourcePattern is used to determine relative paths and the base directory
func BuildTokens(path string, sourcePattern string) Tokens {
	// Normalize to forward slashes for consistent behavior
	path = filepath.ToSlash(path)
	sourcePattern = filepath.ToSlash(sourcePattern)

	// Get the base directory from the source pattern
	baseDir, _ := doublestar.SplitPattern(sourcePattern)
	if baseDir == "" {
		baseDir = "."
	}

	// Calculate relative path from base directory
	relativePath := path
	if baseDir != "." && strings.HasPrefix(path, baseDir+"/") {
		relativePath = strings.TrimPrefix(path, baseDir+"/")
	} else if baseDir != "." && strings.HasPrefix(path, baseDir) {
		relativePath = strings.TrimPrefix(path, baseDir)
	}

	// Extract path components
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)
	dir := filepath.Dir(path)
	if dir == "." {
		dir = "./"
	} else if !strings.HasSuffix(dir, "/") {
		dir = dir + "/"
	}

	// Calculate directory relative to source base
	dirRelative := filepath.Dir(relativePath)
	if dirRelative == "." {
		dirRelative = ""
	} else if !strings.HasSuffix(dirRelative, "/") {
		dirRelative = dirRelative + "/"
	}

	// Calculate match (relative path without extension)
	match := strings.TrimSuffix(relativePath, ext)

	extNoDot := strings.TrimPrefix(ext, ".")

	return Tokens{
		Match:       match,
		Path:        path,
		Dir:         dir,
		DirRelative: dirRelative,
		Base:        base,
		Name:        name,
		Ext:         ext,
		ExtNoDot:    extNoDot,
	}
}

// RenderTemplate renders a template string with the given tokens.
// Templates only support simple {{token}} substitution, so this uses a small
// hand-rolled scanner instead of text/template (which costs ~270kB of binary).
func RenderTemplate(tmpl string, tok Tokens) (string, error) {
	values := map[string]string{
		"match":        tok.Match,
		"path":         tok.Path,
		"dir":          tok.Dir,
		"dir_relative": tok.DirRelative,
		"base":         tok.Base,
		"name":         tok.Name,
		"ext":          tok.Ext,
		"ext_no_dot":   tok.ExtNoDot,
	}

	var out strings.Builder
	rest := tmpl
	for {
		start := strings.Index(rest, "{{")
		if start == -1 {
			out.WriteString(rest)
			break
		}
		out.WriteString(rest[:start])
		end := strings.Index(rest[start:], "}}")
		if end == -1 {
			return "", fmt.Errorf("failed to parse template %q: unclosed {{", tmpl)
		}
		token := strings.TrimSpace(rest[start+2 : start+end])
		value, ok := values[token]
		if !ok {
			return "", fmt.Errorf("invalid token %q in template %q\nAvailable tokens: {{match}}, {{path}}, {{dir}}, {{dir_relative}}, {{base}}, {{name}}, {{ext}}, {{ext_no_dot}}", token, tmpl)
		}
		out.WriteString(value)
		rest = rest[start+end+2:]
	}

	// Convert back to native path separators
	return filepath.FromSlash(out.String()), nil
}

// ValidateTemplate checks if a template string is valid without executing it
func ValidateTemplate(tmpl string) error {
	// Create dummy tokens for validation
	dummyTokens := Tokens{
		Match:       "dummy",
		Path:        "dummy",
		Dir:         "dummy/",
		DirRelative: "dummy/",
		Base:        "dummy",
		Name:        "dummy",
		Ext:         ".dummy",
		ExtNoDot:    "dummy",
	}

	_, err := RenderTemplate(tmpl, dummyTokens)
	return err
}
