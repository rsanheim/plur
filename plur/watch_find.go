package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/pelletier/go-toml"
	"github.com/rsanheim/plur/internal/task"
	"github.com/rsanheim/plur/logger"
)

// mappingRule represents a simple mapping rule for TOML output
type mappingRule struct {
	Pattern string
	Target  string
}

// WatchFindCmd implements the 'plur watch find' command
type WatchFindCmd struct {
	Interactive bool     `help:"Interactive mode - prompt to add mappings" short:"i"`
	DryRun      bool     `help:"Show what would be added without modifying config" short:"d"`
	Files       []string `arg:"" help:"Files to find mappings for" type:"path"`
}

func (cmd *WatchFindCmd) Run(parent *WatchCmd, globals *PlurCLI) error {
	// Get the current task (which has all the mapping rules)
	currentTask := task.DetectFramework()

	// Process each file
	suggestedRules := []mappingRule{}

	for _, file := range cmd.Files {
		// Normalize the path - make it relative if possible
		file = filepath.Clean(file)
		if cwd, err := os.Getwd(); err == nil {
			if rel, err := filepath.Rel(cwd, file); err == nil && !strings.HasPrefix(rel, "..") {
				file = rel
			}
		}

		// Use Task's MapFilesToTarget method
		mappedSpecs := currentTask.MapFilesToTarget([]string{file})

		if len(mappedSpecs) > 0 {
			// Check if the mapped specs actually exist
			allExist := true
			for _, spec := range mappedSpecs {
				if _, err := os.Stat(spec); err == nil {
					fmt.Printf("✓ %s → %s (exists)\n", file, spec)
				} else {
					fmt.Printf("✗ %s → %s (mapping exists but spec not found)\n", file, spec)
					allExist = false
				}
			}

			// If mapped specs don't exist, search for alternatives
			if !allExist {
				alternatives := findAlternativeSpecs(file, currentTask)
				if len(alternatives) > 0 {
					fmt.Println("  Found alternative specs:")
					for i, alt := range alternatives {
						if i >= 3 {
							fmt.Printf("  ... and %d more\n", len(alternatives)-3)
							break
						}
						fmt.Printf("  - %s\n", alt)
					}

					// Simple suggestion based on task patterns
					newRule := createSimpleRule(file, alternatives[0], currentTask)

					fmt.Println("\n  Suggested rule based on found specs:")
					fmt.Printf("    pattern = \"%s\"\n", newRule.Pattern)
					fmt.Printf("    target = \"%s\"\n", newRule.Target)

					suggestedRules = append(suggestedRules, newRule)
				}
			}
		} else {
			// No mapping found - search for alternatives
			fmt.Printf("✗ No mapping for: %s\n", file)

			// Search for actual spec files that might match
			alternatives := findAlternativeSpecs(file, currentTask)

			if len(alternatives) > 0 {
				fmt.Println("  Found potential specs:")
				for i, alt := range alternatives {
					if i >= 5 {
						fmt.Printf("  ... and %d more\n", len(alternatives)-5)
						break
					}
					fmt.Printf("  - %s\n", alt)
				}

				// Create simple rule
				newRule := createSimpleRule(file, alternatives[0], currentTask)

				if cmd.Interactive || cmd.DryRun {
					fmt.Println("\n  Suggested rule based on found specs:")
					fmt.Printf("    pattern = \"%s\"\n", newRule.Pattern)
					fmt.Printf("    target = \"%s\"\n", newRule.Target)
				}

				suggestedRules = append(suggestedRules, newRule)
			} else {
				// No existing specs found - suggest based on task patterns
				fmt.Println("  No existing specs found")
				suggestion := generateSuggestionFromTask(file, currentTask)
				if suggestion != "" {
					fmt.Printf("  Suggested location for new spec: %s\n", suggestion)
					newRule := createSimpleRule(file, suggestion, currentTask)

					if cmd.Interactive || cmd.DryRun {
						fmt.Println("\n  Proposed rule for new specs:")
						fmt.Printf("    pattern = \"%s\"\n", newRule.Pattern)
						fmt.Printf("    target = \"%s\"\n", newRule.Target)
					}

					suggestedRules = append(suggestedRules, newRule)
				} else {
					fmt.Println("  No suggestions available")
				}
			}
		}
	}

	// If no suggested rules, we're done
	if len(suggestedRules) == 0 {
		return nil
	}

	// Handle interactive mode
	if cmd.Interactive && !cmd.DryRun {
		fmt.Printf("\nAdd %d mapping rule(s) to .plur.toml? [y/N]: ", len(suggestedRules))

		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))

		if response == "y" || response == "yes" {
			if err := addRulesToConfig(suggestedRules); err != nil {
				return fmt.Errorf("failed to update config: %w", err)
			}
			fmt.Println("✓ Config updated successfully")
		} else {
			fmt.Println("No changes made")
		}
	} else if cmd.DryRun {
		fmt.Println("\n--- Dry run mode - no changes made ---")
		fmt.Println("Would add the following to .plur.toml:")
		for _, rule := range suggestedRules {
			fmt.Println("\n[[watch.mappings.rules]]")
			fmt.Printf("pattern = \"%s\"\n", rule.Pattern)
			fmt.Printf("target = \"%s\"\n", rule.Target)
		}
	}

	return nil
}

// findAlternativeSpecs searches for spec/test files that might match the given source file
func findAlternativeSpecs(sourceFile string, currentTask *task.Task) []string {
	var alternatives []string

	// Extract the base name without extension
	base := filepath.Base(sourceFile)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)

	// Don't search for specs if this is already a spec/test file
	if strings.HasSuffix(name, "_spec") || strings.HasSuffix(name, "_test") {
		return alternatives
	}

	// Use Task's TestGlob pattern to search for potential matches
	testGlob := currentTask.GetTestPattern()

	// Get the suffix from the task (e.g., "_spec.rb" or "_test.rb")
	suffix := currentTask.GetTestSuffix()

	// Try different name variations using more specific patterns
	patterns := []string{
		strings.Replace(testGlob, "*"+suffix, name+suffix, 1),         // Exact name match
		strings.Replace(testGlob, "*"+suffix, "*"+name+suffix, 1),     // Name as suffix
		strings.Replace(testGlob, "*"+suffix, name+"*"+suffix, 1),     // Name as prefix
		strings.Replace(testGlob, "*"+suffix, "*"+name+"*"+suffix, 1), // Partial match
	}

	// Use a map to avoid duplicates
	found := make(map[string]bool)

	for _, pattern := range patterns {
		matches, err := doublestar.FilepathGlob(pattern)
		if err != nil {
			logger.LogDebug("Error matching pattern", "pattern", pattern, "error", err)
			continue
		}
		for _, match := range matches {
			if !found[match] {
				found[match] = true
				alternatives = append(alternatives, match)
			}
		}
	}

	return alternatives
}

// createSimpleRule creates a mapping rule based on existing task patterns
func createSimpleRule(sourceFile, targetFile string, currentTask *task.Task) mappingRule {
	// Check if we can infer a pattern from the sourceFile -> targetFile relationship
	if customPattern, customTarget := inferPatternFromMapping(sourceFile, targetFile); customPattern != "" {
		return mappingRule{Pattern: customPattern, Target: customTarget}
	}

	// Find a similar pattern from the task's existing mappings
	sourceDir := filepath.Dir(sourceFile)

	// Look for a mapping that matches this source file
	for _, mapping := range currentTask.Mappings {
		if matched, err := doublestar.Match(mapping.Pattern, sourceFile); err == nil && matched {
			// Use this pattern and target as-is
			pattern := mapping.Pattern
			target := mapping.Target
			return mappingRule{Pattern: pattern, Target: target}
		}
	}

	// Default: create a generic pattern
	pattern := filepath.Join(sourceDir, "*.rb")
	target := generateTargetFromPath(targetFile)
	return mappingRule{Pattern: pattern, Target: target}
}

// inferPatternFromMapping tries to infer a mapping pattern from source->target example
func inferPatternFromMapping(sourceFile, targetFile string) (pattern, target string) {
	// For lib/example-project/cli.rb -> spec/lib/example-project/cli_spec.rb, we want:
	// pattern = "lib/**/*.rb", target = "spec/lib/{{path}}/{{name}}_spec.rb"

	if strings.HasPrefix(sourceFile, "lib/") && strings.HasPrefix(targetFile, "spec/lib/") {
		return "lib/**/*.rb", "spec/lib/{{path}}/{{name}}_spec.rb"
	}

	// For lib/foo.rb -> spec/lib/foo_spec.rb (direct lib mapping)
	if strings.HasPrefix(sourceFile, "lib/") && strings.Contains(targetFile, "/lib/") {
		return "lib/**/*.rb", "spec/lib/{{path}}/{{name}}_spec.rb"
	}

	// Similar logic for minitest
	if strings.HasPrefix(sourceFile, "lib/") && strings.HasPrefix(targetFile, "test/lib/") {
		return "lib/**/*.rb", "test/lib/{{path}}/{{name}}_test.rb"
	}

	return "", ""
}

// generateSuggestionFromTask generates a suggestion based on task patterns
func generateSuggestionFromTask(sourceFile string, currentTask *task.Task) string {
	// Try applying task mappings to see where this file would map
	targets := currentTask.MapFilesToTarget([]string{sourceFile})
	if len(targets) > 0 {
		return targets[0]
	}

	// If no mapping, generate based on file location and task type
	base := filepath.Base(sourceFile)
	name := strings.TrimSuffix(base, filepath.Ext(base))

	if currentTask.Name == "minitest" {
		return fmt.Sprintf("test/%s_test.rb", name)
	} else {
		return fmt.Sprintf("spec/%s_spec.rb", name)
	}
}

// generateTargetFromPath creates a target pattern from an example target path
func generateTargetFromPath(targetPath string) string {
	dir := filepath.Dir(targetPath)
	base := filepath.Base(targetPath)

	// Replace the specific filename with pattern (use {} for TOML output)
	if strings.Contains(base, "_spec.rb") {
		return filepath.Join(dir, "{name}_spec.rb")
	} else if strings.Contains(base, "_test.rb") {
		return filepath.Join(dir, "{name}_test.rb")
	} else {
		return filepath.Join(dir, "{name}.rb")
	}
}

// addRulesToConfig adds rules to the .plur.toml config file
func addRulesToConfig(rules []mappingRule) error {
	configPath := ".plur.toml"

	// Read existing config or create new one
	var configData map[string]interface{}

	if data, err := os.ReadFile(configPath); err == nil {
		// Parse existing config
		if err := toml.Unmarshal(data, &configData); err != nil {
			return fmt.Errorf("failed to parse existing config: %w", err)
		}
	} else {
		// Create new config
		configData = make(map[string]interface{})
	}

	// Ensure watch.mappings.rules exists
	if _, ok := configData["watch"]; !ok {
		configData["watch"] = make(map[string]interface{})
	}

	watchConfig := configData["watch"].(map[string]interface{})
	if _, ok := watchConfig["mappings"]; !ok {
		watchConfig["mappings"] = make(map[string]interface{})
	}

	mappingsConfig := watchConfig["mappings"].(map[string]interface{})

	// Get existing rules or create new array
	var existingRules []interface{}
	if rulesRaw, ok := mappingsConfig["rules"]; ok {
		if rules, ok := rulesRaw.([]interface{}); ok {
			existingRules = rules
		}
	}

	// Add new rules
	for _, rule := range rules {
		ruleMap := map[string]interface{}{
			"pattern": rule.Pattern,
			"target":  rule.Target,
		}
		existingRules = append(existingRules, ruleMap)
	}

	mappingsConfig["rules"] = existingRules

	// Write back to file
	data, err := toml.Marshal(configData)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}
