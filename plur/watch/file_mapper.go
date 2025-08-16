package watch

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rsanheim/plur/logger"
)

// FileMapper handles mapping between source files and their corresponding spec files
type FileMapper struct {
	// Configuration for mapping rules
	config *MappingConfig
}

// NewFileMapper creates a new FileMapper instance
func NewFileMapper() *FileMapper {
	return &FileMapper{
		config: NewMappingConfig(),
	}
}

// NewFileMapperWithConfig creates a FileMapper with custom configuration
func NewFileMapperWithConfig(config *MappingConfig) *FileMapper {
	return &FileMapper{
		config: config,
	}
}

// MapFileToSpecs maps a changed file to the spec files that should be run
func (fm *FileMapper) MapFileToSpecs(changedFile string) []string {
	// Normalize the path
	changedFile = filepath.Clean(changedFile)

	// Try to match using the mapping rules
	rule, vars := fm.config.MatchFile(changedFile)

	if rule != nil {
		// Found a matching rule
		target := ExpandTarget(rule.Target, vars)

		// If target is the file itself, return it
		if target == "{file}" || target == changedFile {
			return []string{changedFile}
		}

		// Target doesn't exist, but we still have a rule match
		// This might be a directory (like "spec") or a pattern
		if target == "spec" || strings.HasSuffix(target, "/") {
			return []string{target}
		}

		// Check if the target spec file exists
		if _, err := os.Stat(target); err == nil {
			logger.LogDebug("Mapping found", "file", changedFile, "target", target, "rule", rule.Description)
			return []string{target}
		} else {
			// For existing tests, still return the target even if it doesn't exist
			// to maintain backward compatibility
			if !fm.config.ProvideFeedback {
				// Running in test mode or with feedback disabled
				return []string{target}
			}

			// Provide feedback about missing spec file
			fm.provideMappingFeedback(changedFile, target, false)
			return nil
		}
	}

	// No mapping found - provide feedback
	if fm.config.ProvideFeedback {
		fm.provideNoMappingFeedback(changedFile)
	}

	return nil
}

// provideMappingFeedback provides user feedback about mapping results
func (fm *FileMapper) provideMappingFeedback(changedFile, target string, exists bool) {
	if exists {
		fmt.Printf("\n>>> File '%s' mapped to '%s'\n", changedFile, target)
	} else {
		fmt.Printf("\n>>> File '%s' would map to '%s', but that spec doesn't exist\n", changedFile, target)

		// Show suggestions if enabled
		if fm.config.ShowSuggestions {
			suggestions := GenerateSuggestions(changedFile)
			if len(suggestions) > 0 {
				fmt.Println("    Suggestions:")
				for i, suggestion := range suggestions {
					if i >= 3 {
						break // Limit to 3 suggestions
					}
					// Check if suggested file exists
					if _, err := os.Stat(suggestion); err == nil {
						fmt.Printf("    • %s (exists)\n", suggestion)
					} else if !strings.Contains(suggestion, "*") {
						fmt.Printf("    • %s (would need to be created)\n", suggestion)
					} else {
						fmt.Printf("    • %s (pattern)\n", suggestion)
					}
				}
			}
		}
	}
}

// provideNoMappingFeedback provides feedback when no mapping is found
func (fm *FileMapper) provideNoMappingFeedback(changedFile string) {
	// Don't provide feedback for non-Ruby files or spec files
	if !strings.HasSuffix(changedFile, ".rb") || strings.HasSuffix(changedFile, "_spec.rb") {
		return
	}

	fmt.Printf("\n>>> No mapping found for '%s'\n", changedFile)
	fmt.Println("    This file doesn't match any configured mapping rules.")

	// Show suggestions if enabled
	if fm.config.ShowSuggestions {
		suggestions := GenerateSuggestions(changedFile)
		if len(suggestions) > 0 {
			fmt.Println("    Possible spec files to create or run:")
			for i, suggestion := range suggestions {
				if i >= 3 {
					break // Limit to 3 suggestions
				}
				// Check if suggested file exists
				if _, err := os.Stat(suggestion); err == nil {
					fmt.Printf("    • %s (exists - consider adding a mapping rule)\n", suggestion)
				} else if !strings.Contains(suggestion, "*") {
					fmt.Printf("    • %s (create this spec file)\n", suggestion)
				} else {
					fmt.Printf("    • Run: plur %s\n", suggestion)
				}
			}
		}
	}

	fmt.Println("    To add a custom mapping, create a .plur.toml file with:")
	fmt.Println("    [[watch.mappings.rules]]")
	fmt.Printf("    pattern = \"%s\"\n", changedFile)
	fmt.Println("    target = \"spec/path/to/your_spec.rb\"")
	fmt.Println()
}

// mapLibToSpec maps lib/foo.rb to spec/foo_spec.rb
func (fm *FileMapper) mapLibToSpec(libFile string) string {
	if !strings.HasSuffix(libFile, ".rb") {
		return ""
	}

	// Remove "lib/" prefix and ".rb" suffix
	relativePath := strings.TrimPrefix(libFile, "lib/")
	baseName := strings.TrimSuffix(relativePath, ".rb")

	// Construct spec path
	specPath := filepath.Join("spec", baseName+"_spec.rb")
	return specPath
}

// mapAppToSpec maps Rails app files to their corresponding specs
// e.g., app/models/user.rb -> spec/models/user_spec.rb
// e.g., app/controllers/users_controller.rb -> spec/controllers/users_controller_spec.rb
func (fm *FileMapper) mapAppToSpec(appFile string) string {
	if !strings.HasSuffix(appFile, ".rb") {
		return ""
	}

	// Remove "app/" prefix and ".rb" suffix
	relativePath := strings.TrimPrefix(appFile, "app/")
	baseName := strings.TrimSuffix(relativePath, ".rb")

	// Construct spec path
	specPath := filepath.Join("spec", baseName+"_spec.rb")
	return specPath
}

// ShouldWatchFile determines if a file should trigger spec runs
func (fm *FileMapper) ShouldWatchFile(filePath string) bool {
	// Watch Ruby files
	if strings.HasSuffix(filePath, ".rb") {
		return true
	}

	// Watch ERB templates in Rails apps
	if strings.HasSuffix(filePath, ".erb") {
		return true
	}

	// Watch Haml templates
	if strings.HasSuffix(filePath, ".haml") {
		return true
	}

	// Watch Slim templates
	if strings.HasSuffix(filePath, ".slim") {
		return true
	}

	return false
}
