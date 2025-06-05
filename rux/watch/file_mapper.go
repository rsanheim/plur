package watch

import (
	"path/filepath"
	"strings"
)

// FileMapper handles mapping between source files and their corresponding spec files
type FileMapper struct {
	// Add configuration options here in the future
}

// NewFileMapper creates a new FileMapper instance
func NewFileMapper() *FileMapper {
	return &FileMapper{}
}

// MapFileToSpecs maps a changed file to the spec files that should be run
func (fm *FileMapper) MapFileToSpecs(changedFile string) []string {
	// Normalize the path
	changedFile = filepath.Clean(changedFile)

	// Case 1: The changed file is already a spec file
	if strings.HasSuffix(changedFile, "_spec.rb") {
		return []string{changedFile}
	}

	// Case 2: spec_helper.rb or rails_helper.rb - run all specs
	baseName := filepath.Base(changedFile)
	if baseName == "spec_helper.rb" || baseName == "rails_helper.rb" {
		return []string{"spec"} // Run all specs
	}

	// Case 3: lib/ -> spec/ mapping
	if strings.HasPrefix(changedFile, "lib/") {
		specPath := fm.mapLibToSpec(changedFile)
		if specPath != "" {
			return []string{specPath}
		}
	}

	// Case 4: Rails app/ -> spec/ mapping
	if strings.HasPrefix(changedFile, "app/") {
		specPath := fm.mapAppToSpec(changedFile)
		if specPath != "" {
			return []string{specPath}
		}
	}

	// No mapping found
	return nil
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
