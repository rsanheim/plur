package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/pelletier/go-toml"
	"github.com/rsanheim/plur/logger"
	"github.com/rsanheim/plur/watch"
)

// WatchFindCmd implements the 'plur watch find' command
type WatchFindCmd struct {
	Interactive bool     `help:"Interactive mode - prompt to add mappings" short:"i"`
	DryRun      bool     `help:"Show what would be added without modifying config" short:"d"`
	Files       []string `arg:"" help:"Files to find mappings for" type:"path"`
}

func (cmd *WatchFindCmd) Run(parent *WatchCmd, globals *PlurCLI) error {
	// Load existing mapping configuration
	mappingConfig, err := watch.LoadMappingConfig("")
	if err != nil {
		logger.LogDebug("Failed to load mapping config, using defaults", "error", err)
		mappingConfig = watch.NewMappingConfig()
	}

	// Process each file
	suggestedRules := []watch.MappingRule{}
	
	for _, file := range cmd.Files {
		// Normalize the path - make it relative if possible
		file = filepath.Clean(file)
		if cwd, err := os.Getwd(); err == nil {
			if rel, err := filepath.Rel(cwd, file); err == nil && !strings.HasPrefix(rel, "..") {
				file = rel
			}
		}
		
		// Check if file already has a mapping
		rule, vars := mappingConfig.MatchFile(file)
		
		if rule != nil {
			// File has a mapping - check if the target actually exists
			target := watch.ExpandTarget(rule.Target, vars)
			
			// Check if the mapped spec file exists
			targetExists := false
			if !strings.Contains(target, "*") {
				if _, err := os.Stat(target); err == nil {
					targetExists = true
				}
			}
			
			if targetExists {
				fmt.Printf("✓ %s → %s (exists via: %s)\n", file, target, rule.Description)
				continue
			} else {
				// Mapping exists but target doesn't - look for alternatives
				fmt.Printf("✗ %s → %s (mapping exists but spec not found)\n", file, target)
				
				// Search for alternative specs
				alternatives := findAlternativeSpecs(file)
				if len(alternatives) > 0 {
					fmt.Println("  Found alternative specs:")
					for i, alt := range alternatives {
						if i >= 3 {
							fmt.Printf("  ... and %d more\n", len(alternatives)-3)
							break
						}
						fmt.Printf("  - %s\n", alt)
					}
					
					// Suggest a new rule based on the first alternative
					if len(alternatives) > 0 {
						pattern, targetPattern := detectPatternFromAlternative(file, alternatives[0])
						newRule := watch.MappingRule{
							Pattern:     pattern,
							Target:      targetPattern,
							Description: fmt.Sprintf("Custom mapping for %s structure", filepath.Dir(file)),
							Priority:    60,
							Type:        "glob",
						}
						
						fmt.Println("\n  Suggested rule based on found specs:")
						fmt.Printf("    pattern = \"%s\"\n", newRule.Pattern)
						fmt.Printf("    target = \"%s\"\n", newRule.Target)
						fmt.Printf("    description = \"%s\"\n", newRule.Description)
						
						suggestedRules = append(suggestedRules, newRule)
					}
				}
				continue
			}
		}
		
		// No mapping found - search for alternatives
		fmt.Printf("\n✗ No mapping for: %s\n", file)
		
		// First, search for actual spec files that might match
		alternatives := findAlternativeSpecs(file)
		
		if len(alternatives) > 0 {
			fmt.Println("  Found potential specs:")
			for i, alt := range alternatives {
				if i >= 5 {
					fmt.Printf("  ... and %d more\n", len(alternatives)-5)
					break
				}
				fmt.Printf("  - %s\n", alt)
			}
			
			// Suggest a rule based on the first alternative
			pattern, targetPattern := detectPatternFromAlternative(file, alternatives[0])
			newRule := watch.MappingRule{
				Pattern:     pattern,
				Target:      targetPattern,
				Description: fmt.Sprintf("Custom mapping for %s files", filepath.Dir(file)),
				Priority:    60,
				Type:        "glob",
			}
			
			if cmd.Interactive || cmd.DryRun {
				fmt.Println("\n  Suggested rule based on found specs:")
				fmt.Printf("    pattern = \"%s\"\n", newRule.Pattern)
				fmt.Printf("    target = \"%s\"\n", newRule.Target)
				fmt.Printf("    description = \"%s\"\n", newRule.Description)
			}
			
			suggestedRules = append(suggestedRules, newRule)
		} else {
			// No existing specs found - use generic suggestions
			suggestions := watch.GenerateSuggestions(file)
			if len(suggestions) == 0 {
				fmt.Println("  No suggestions available")
			} else {
				fmt.Println("  No existing specs found. Suggested locations for new specs:")
				for i, suggestion := range suggestions {
					if i >= 3 {
						break
					}
					fmt.Printf("  %d. %s\n", i+1, suggestion)
				}
				
				// Create a rule for the most likely location
				if len(suggestions) > 0 && !strings.Contains(suggestions[0], "*") {
					newRule := createRuleForFile(file, suggestions[0])
					
					if cmd.Interactive || cmd.DryRun {
						fmt.Println("\n  Proposed rule for new specs:")
						fmt.Printf("    pattern = \"%s\"\n", newRule.Pattern)
						fmt.Printf("    target = \"%s\"\n", newRule.Target)
						fmt.Printf("    description = \"%s\"\n", newRule.Description)
					}
					
					suggestedRules = append(suggestedRules, newRule)
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
			fmt.Printf("description = \"%s\"\n", rule.Description)
			fmt.Printf("priority = %d\n", rule.Priority)
		}
	}
	
	return nil
}

// findAlternativeSpecs searches for spec files that might match the given source file
func findAlternativeSpecs(sourceFile string) []string {
	var alternatives []string
	
	// Extract the base name without extension
	base := filepath.Base(sourceFile)
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)
	
	// Don't search for specs if this is already a spec file
	if strings.HasSuffix(name, "_spec") {
		return alternatives
	}
	
	// Search patterns to try using doublestar
	patterns := []string{
		fmt.Sprintf("spec/**/%s_spec.rb", name),      // Exact name match anywhere
		fmt.Sprintf("spec/**/*%s_spec.rb", name),     // Name as suffix  
		fmt.Sprintf("spec/**/%s*_spec.rb", name),     // Name as prefix
		fmt.Sprintf("spec/**/*%s*_spec.rb", name),    // Partial name match
	}
	
	// Use a map to avoid duplicates
	found := make(map[string]bool)
	
	for _, pattern := range patterns {
		// Use doublestar which supports ** for recursive matching
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

// detectPatternFromAlternative analyzes an alternative spec path and the source file
// to detect a pattern that could be used as a mapping rule
func detectPatternFromAlternative(sourceFile, specFile string) (pattern, target string) {
	// Clean up paths
	sourceFile = filepath.Clean(sourceFile)
	specFile = filepath.Clean(specFile)
	
	// Get directory components
	sourceDir := filepath.Dir(sourceFile)
	specDir := filepath.Dir(specFile)
	
	// Case 1: lib/example-project/cli.rb -> spec/lib/example-project/cli_spec.rb (lib preserved in spec)
	if strings.HasPrefix(sourceFile, "lib/") && strings.Contains(specDir, "/lib/") {
		pattern = "lib/**/*.rb"
		target = "spec/lib/{path}/{name}_spec.rb"
		return
	}
	
	// Case 2: lib/example-project/cli.rb -> spec/example-project/cli_spec.rb (standard lib to spec)
	if strings.HasPrefix(sourceFile, "lib/") && !strings.Contains(specDir, "/lib/") {
		if strings.HasPrefix(specFile, "spec/") {
			// Check if the structure aligns after removing lib/ and spec/
			sourcePath := strings.TrimPrefix(sourceFile, "lib/")
			specPath := strings.TrimPrefix(specFile, "spec/")
			specPath = strings.TrimSuffix(specPath, "_spec.rb") + ".rb"
			
			if sourcePath == specPath {
				pattern = "lib/**/*.rb"
				target = "spec/{path}/{name}_spec.rb"
				return
			}
		}
	}
	
	// Case 3: app/services/foo.rb -> spec/services/foo_spec.rb
	if strings.HasPrefix(sourceFile, "app/") {
		pattern = "app/**/*.rb"
		target = "spec/{path}/{name}_spec.rb"
		return
	}
	
	// Case 4: Generic pattern for other directories
	if sourceDir != "." && sourceDir != "" {
		pattern = fmt.Sprintf("%s/**/*.rb", sourceDir)
		if strings.HasPrefix(specFile, "spec/") {
			// Try to detect the pattern in the spec path
			specRelative := strings.TrimPrefix(specFile, "spec/")
			specRelativeDir := filepath.Dir(specRelative)
			
			if specRelativeDir == sourceDir {
				// Direct mapping: config/foo.rb -> spec/config/foo_spec.rb
				target = fmt.Sprintf("spec/%s/{name}_spec.rb", sourceDir)
			} else {
				// Complex mapping, use generic pattern
				target = "spec/**/{name}_spec.rb"
			}
		} else {
			target = "{dir}/{name}_spec.rb"
		}
		return
	}
	
	// Default fallback
	pattern = "**/*.rb"
	target = "spec/**/{name}_spec.rb"
	return
}

// createRuleForFile creates a mapping rule for a file based on its path
func createRuleForFile(file, target string) watch.MappingRule {
	// Determine the pattern based on the file structure
	dir := filepath.Dir(file)
	
	// Create a pattern that matches similar files in the same directory
	pattern := filepath.Join(dir, "*.rb")
	
	// Extract the target pattern
	targetDir := filepath.Dir(target)
	targetPattern := filepath.Join(targetDir, "{name}_spec.rb")
	
	// Create description
	description := fmt.Sprintf("Map %s files to %s specs", dir, targetDir)
	
	return watch.MappingRule{
		Pattern:     pattern,
		Target:      targetPattern,
		Description: description,
		Priority:    55,
		Type:        "glob",
	}
}

// addRulesToConfig adds rules to the .plur.toml config file
func addRulesToConfig(rules []watch.MappingRule) error {
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
			"pattern":     rule.Pattern,
			"target":      rule.Target,
			"description": rule.Description,
			"priority":    rule.Priority,
			"type":        rule.Type,
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