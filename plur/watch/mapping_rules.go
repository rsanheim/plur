package watch

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/rsanheim/plur/internal/task"
)

// ExpandTarget expands a target pattern with variables using {{key}} syntax
func ExpandTarget(target string, vars map[string]string) string {
	result := target

	// Replace variables in the target using {{key}} syntax only
	for key, value := range vars {
		placeholder := "{{" + key + "}}"
		result = strings.ReplaceAll(result, placeholder, value)
	}

	return result
}

// GenerateSuggestions generates spec file suggestions for an unmapped file
func GenerateSuggestions(filePath string) []string {
	currentTask := task.DetectFramework()
	return GenerateSuggestionsForFramework(filePath, currentTask.Name)
}

// GenerateSuggestionsForFramework generates spec/test file suggestions for an unmapped file
func GenerateSuggestionsForFramework(filePath string, framework string) []string {
	suggestions := []string{}

	// Extract base name from the file path
	base := filepath.Base(filePath)
	name := strings.TrimSuffix(base, filepath.Ext(base))
	dir := filepath.Dir(filePath)

	// Skip if this is already a spec/test file
	if strings.HasSuffix(name, "_spec") || strings.HasSuffix(name, "_test") {
		return suggestions
	}

	if framework == "minitest" {
		// For minitest, suggest test locations
		if strings.HasPrefix(filePath, "lib/") {
			// lib/foo.rb -> test/foo_test.rb or test/lib/foo_test.rb
			relativePath := strings.TrimPrefix(filePath, "lib/")
			baseName := strings.TrimSuffix(relativePath, ".rb")
			suggestions = append(suggestions, fmt.Sprintf("test/%s_test.rb", baseName))
			suggestions = append(suggestions, fmt.Sprintf("test/lib/%s_test.rb", baseName))
		} else if strings.HasPrefix(filePath, "app/") {
			// app/models/user.rb -> test/models/user_test.rb
			relativePath := strings.TrimPrefix(filePath, "app/")
			baseName := strings.TrimSuffix(relativePath, ".rb")
			suggestions = append(suggestions, fmt.Sprintf("test/%s_test.rb", baseName))
		} else {
			// Generic file -> test/foo_test.rb
			suggestions = append(suggestions, fmt.Sprintf("test/%s_test.rb", name))
			if dir != "." {
				suggestions = append(suggestions, fmt.Sprintf("test/%s/%s_test.rb", dir, name))
			}
		}
	} else {
		// For RSpec, suggest spec locations
		if strings.HasPrefix(filePath, "lib/") {
			// lib/foo.rb -> spec/foo_spec.rb or spec/lib/foo_spec.rb
			relativePath := strings.TrimPrefix(filePath, "lib/")
			baseName := strings.TrimSuffix(relativePath, ".rb")
			suggestions = append(suggestions, fmt.Sprintf("spec/%s_spec.rb", baseName))
			suggestions = append(suggestions, fmt.Sprintf("spec/lib/%s_spec.rb", baseName))
		} else if strings.HasPrefix(filePath, "app/") {
			// app/models/user.rb -> spec/models/user_spec.rb
			relativePath := strings.TrimPrefix(filePath, "app/")
			baseName := strings.TrimSuffix(relativePath, ".rb")
			suggestions = append(suggestions, fmt.Sprintf("spec/%s_spec.rb", baseName))
		} else {
			// Generic file -> spec/foo_spec.rb
			suggestions = append(suggestions, fmt.Sprintf("spec/%s_spec.rb", name))
			if dir != "." {
				suggestions = append(suggestions, fmt.Sprintf("spec/%s/%s_spec.rb", dir, name))
			}
		}
	}

	return suggestions
}
