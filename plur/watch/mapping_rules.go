package watch

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// MappingRule represents a configurable mapping from source files to test files
type MappingRule struct {
	// Pattern to match source files (can be glob or regex)
	Pattern string `toml:"pattern"`
	// Target spec file or pattern to run (can include variables)
	Target string `toml:"target"`
	// Description for debugging and user feedback
	Description string `toml:"description"`
	// Priority for rule ordering (higher = evaluated first)
	Priority int `toml:"priority"`
	// Type of pattern: "glob" or "regex"
	Type string `toml:"type"`

	// Compiled regex (if Type is regex)
	compiledRegex *regexp.Regexp
}

// MappingConfig holds all mapping rules and settings
type MappingConfig struct {
	// Built-in rules that are always applied
	BuiltinRules []MappingRule
	// User-defined rules from config file
	CustomRules []MappingRule `toml:"rules"`
	// Whether to show suggestions for unmapped files
	ShowSuggestions bool `toml:"show_suggestions"`
	// Whether to provide feedback when no mapping is found
	ProvideFeedback bool `toml:"provide_feedback"`
}

// NewMappingConfig creates a default mapping configuration
func NewMappingConfig() *MappingConfig {
	return &MappingConfig{
		BuiltinRules:    getBuiltinRules(),
		CustomRules:     []MappingRule{},
		ShowSuggestions: true,
		ProvideFeedback: true,
	}
}

// getBuiltinRules returns the default built-in mapping rules
func getBuiltinRules() []MappingRule {
	return []MappingRule{
		// Direct spec file mapping
		{
			Pattern:     "**/*_spec.rb",
			Target:      "{file}",
			Description: "Run spec files directly",
			Priority:    100,
			Type:        "glob",
		},
		// spec_helper.rb triggers all specs
		{
			Pattern:     "spec/spec_helper.rb",
			Target:      "spec",
			Description: "Run all specs when spec_helper changes",
			Priority:    90,
			Type:        "glob",
		},
		// rails_helper.rb triggers all specs
		{
			Pattern:     "spec/rails_helper.rb",
			Target:      "spec",
			Description: "Run all specs when rails_helper changes",
			Priority:    90,
			Type:        "glob",
		},
		// lib/ -> spec/ mapping (with subdirectories)
		{
			Pattern:     "lib/**/*.rb",
			Target:      "spec/{path}/{name}_spec.rb",
			Description: "Map lib files to corresponding specs",
			Priority:    50,
			Type:        "glob",
		},
		// lib/ -> spec/ mapping (direct files)
		{
			Pattern:     "lib/*.rb",
			Target:      "spec/{name}_spec.rb",
			Description: "Map lib files to corresponding specs",
			Priority:    50,
			Type:        "glob",
		},
		// app/ -> spec/ mapping for Rails
		{
			Pattern:     "app/**/*.rb",
			Target:      "spec/{path}/{name}_spec.rb",
			Description: "Map Rails app files to corresponding specs",
			Priority:    50,
			Type:        "glob",
		},
	}
}

// CompileRules prepares rules for matching (compiles regexes, etc)
func (mc *MappingConfig) CompileRules() error {
	allRules := append(mc.BuiltinRules, mc.CustomRules...)
	for i := range allRules {
		if allRules[i].Type == "regex" {
			re, err := regexp.Compile(allRules[i].Pattern)
			if err != nil {
				return fmt.Errorf("invalid regex pattern '%s': %v", allRules[i].Pattern, err)
			}
			allRules[i].compiledRegex = re
		}
	}
	return nil
}

// MatchFile attempts to find a mapping rule for the given file
func (mc *MappingConfig) MatchFile(filePath string) (*MappingRule, map[string]string) {
	// Normalize the path
	filePath = filepath.Clean(filePath)

	// Try all rules in priority order
	allRules := mc.GetSortedRules()

	for _, rule := range allRules {
		vars := mc.matchRule(rule, filePath)
		if vars != nil {
			return &rule, vars
		}
	}

	return nil, nil
}

// GetSortedRules returns all rules sorted by priority (highest first)
func (mc *MappingConfig) GetSortedRules() []MappingRule {
	// Combine builtin and custom rules
	allRules := append([]MappingRule{}, mc.BuiltinRules...)
	allRules = append(allRules, mc.CustomRules...)

	// Sort by priority (highest first)
	// Simple bubble sort for now (small number of rules)
	for i := 0; i < len(allRules); i++ {
		for j := i + 1; j < len(allRules); j++ {
			if allRules[j].Priority > allRules[i].Priority {
				allRules[i], allRules[j] = allRules[j], allRules[i]
			}
		}
	}

	return allRules
}

// matchRule checks if a file matches a rule and returns extracted variables
func (mc *MappingConfig) matchRule(rule MappingRule, filePath string) map[string]string {
	switch rule.Type {
	case "regex":
		return mc.matchRegexRule(rule, filePath)
	default: // "glob" or unspecified
		return mc.matchGlobRule(rule, filePath)
	}
}

// matchGlobRule matches using glob patterns
func (mc *MappingConfig) matchGlobRule(rule MappingRule, filePath string) map[string]string {
	matched, vars := matchGlobPattern(rule.Pattern, filePath)
	if matched {
		return vars
	}
	return nil
}

// matchRegexRule matches using regex patterns
func (mc *MappingConfig) matchRegexRule(rule MappingRule, filePath string) map[string]string {
	if rule.compiledRegex == nil {
		return nil
	}

	matches := rule.compiledRegex.FindStringSubmatch(filePath)
	if matches == nil {
		return nil
	}

	// Create variables from named groups
	vars := make(map[string]string)
	vars["file"] = filePath

	// Extract named groups
	for i, name := range rule.compiledRegex.SubexpNames() {
		if i > 0 && name != "" && i < len(matches) {
			vars[name] = matches[i]
		}
	}

	return vars
}

// matchGlobPattern matches a glob pattern and extracts variables
func matchGlobPattern(pattern, filePath string) (bool, map[string]string) {
	// Check if pattern matches
	matched := false

	// First try simple glob matching
	if simpleMatch, err := filepath.Match(pattern, filePath); err == nil && simpleMatch {
		matched = true
	}

	// If no match and pattern contains **, try double wildcard matching
	if !matched && strings.Contains(pattern, "**") {
		matched = matchDoubleWildcard(pattern, filePath)
	}

	// If no match and pattern contains simple wildcard at the beginning
	if !matched && strings.HasPrefix(pattern, "*") && !strings.HasPrefix(pattern, "**") {
		// Try matching just the filename for patterns like "*_spec.rb"
		base := filepath.Base(filePath)
		if simpleMatch, err := filepath.Match(pattern, base); err == nil && simpleMatch {
			matched = true
		}
	}

	if !matched {
		return false, nil
	}

	// Extract variables for template expansion
	vars := make(map[string]string)
	vars["file"] = filePath

	// Extract directory and filename parts
	dir := filepath.Dir(filePath)
	base := filepath.Base(filePath)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)

	// Remove leading directory prefixes for path variable
	path := dir
	if strings.HasPrefix(path, "lib/") {
		path = strings.TrimPrefix(path, "lib/")
	} else if strings.HasPrefix(path, "app/") {
		path = strings.TrimPrefix(path, "app/")
	} else if strings.HasPrefix(path, "spec/") {
		path = strings.TrimPrefix(path, "spec/")
	}

	// Clean up path - remove trailing slash and handle "."
	if path == "." || path == "lib" || path == "app" || path == "spec" {
		path = ""
	}
	path = strings.TrimSuffix(path, "/")

	vars["path"] = path
	vars["dir"] = dir
	vars["name"] = name
	vars["ext"] = strings.TrimPrefix(ext, ".")

	return true, vars
}

// matchDoubleWildcard handles ** in glob patterns
func matchDoubleWildcard(pattern, filePath string) bool {
	// Convert ** to match any number of directories
	if strings.Contains(pattern, "**") {
		// Convert pattern to regex
		regexPattern := regexp.QuoteMeta(pattern)
		// Replace escaped \*\* with .* to match any path
		regexPattern = strings.ReplaceAll(regexPattern, `\*\*`, `.*`)
		// Replace escaped \* with [^/]* to match within a directory
		regexPattern = strings.ReplaceAll(regexPattern, `\*`, `[^/]*`)
		// Replace escaped \? with . to match single character
		regexPattern = strings.ReplaceAll(regexPattern, `\?`, `.`)
		// Anchor the pattern
		regexPattern = "^" + regexPattern + "$"

		if matched, _ := regexp.MatchString(regexPattern, filePath); matched {
			return true
		}
	}

	return false
}

// ExpandTarget expands a target pattern using extracted variables
func ExpandTarget(target string, vars map[string]string) string {
	result := target

	// Replace variables in the target
	for key, value := range vars {
		placeholder := "{" + key + "}"
		result = strings.ReplaceAll(result, placeholder, value)
	}

	return result
}

// GenerateSuggestions generates spec file suggestions for an unmapped file
func GenerateSuggestions(filePath string) []string {
	suggestions := []string{}

	// Clean up the file path
	filePath = filepath.Clean(filePath)
	dir := filepath.Dir(filePath)
	base := filepath.Base(filePath)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)

	// Don't suggest for spec files themselves
	if strings.HasSuffix(filePath, "_spec.rb") {
		return suggestions
	}

	// Don't suggest for non-Ruby files
	if ext != ".rb" {
		return suggestions
	}

	// Suggestion 1: Direct spec file in spec/ directory
	if strings.HasPrefix(filePath, "lib/") {
		specPath := strings.Replace(filePath, "lib/", "spec/", 1)
		specPath = strings.TrimSuffix(specPath, ".rb") + "_spec.rb"
		suggestions = append(suggestions, specPath)
	}

	// Suggestion 2: Rails app/ to spec/ mapping
	if strings.HasPrefix(filePath, "app/") {
		specPath := strings.Replace(filePath, "app/", "spec/", 1)
		specPath = strings.TrimSuffix(specPath, ".rb") + "_spec.rb"
		suggestions = append(suggestions, specPath)
	}

	// Suggestion 3: Look for specs with similar names
	// This would require filesystem access, so we'll suggest patterns
	if !strings.HasPrefix(filePath, "spec/") {
		suggestions = append(suggestions, fmt.Sprintf("spec/**/*%s*_spec.rb", name))
	}

	// Suggestion 4: Integration or request specs for controllers
	if strings.Contains(dir, "controllers") {
		controllerName := strings.TrimSuffix(name, "_controller")
		suggestions = append(suggestions, fmt.Sprintf("spec/requests/%s_spec.rb", controllerName))
		suggestions = append(suggestions, fmt.Sprintf("spec/integration/%s_spec.rb", controllerName))
	}

	// Suggestion 5: System specs for views
	if strings.Contains(dir, "views") {
		suggestions = append(suggestions, fmt.Sprintf("spec/system/%s_spec.rb", name))
	}

	return suggestions
}
