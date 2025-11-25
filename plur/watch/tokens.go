package watch

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"
	"text/template"

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

// RenderTemplate renders a template string with the given tokens
// Uses Go's text/template with custom functions for each token
func RenderTemplate(tmpl string, tok Tokens) (string, error) {
	// Create function map with all token accessors
	funcs := template.FuncMap{
		"match":        func() string { return tok.Match },
		"path":         func() string { return tok.Path },
		"dir":          func() string { return tok.Dir },
		"dir_relative": func() string { return tok.DirRelative },
		"base":         func() string { return tok.Base },
		"name":         func() string { return tok.Name },
		"ext":          func() string { return tok.Ext },
		"ext_no_dot":   func() string { return tok.ExtNoDot },
	}

	// Parse and execute template
	t := template.New("target").Funcs(funcs)
	t, err := t.Parse(tmpl)
	if err != nil {
		// Try to provide helpful error message for common mistakes
		if strings.Contains(err.Error(), "function") {
			// Extract the function name from error
			return "", fmt.Errorf("invalid token in template %q: %w\nAvailable tokens: {{match}}, {{path}}, {{dir}}, {{dir_relative}}, {{base}}, {{name}}, {{ext}}, {{ext_no_dot}}", tmpl, err)
		}
		return "", fmt.Errorf("failed to parse template %q: %w", tmpl, err)
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, nil); err != nil {
		return "", fmt.Errorf("failed to execute template %q: %w", tmpl, err)
	}

	result := buf.String()

	// Convert back to native path separators
	return filepath.FromSlash(result), nil
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
