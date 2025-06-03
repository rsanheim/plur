package watch

import (
	"reflect"
	"testing"
)

func TestFileMapper_MapFileToSpecs(t *testing.T) {
	fm := NewFileMapper()

	tests := []struct {
		name        string
		changedFile string
		want        []string
	}{
		// Spec files
		{
			name:        "spec file returns itself",
			changedFile: "spec/models/user_spec.rb",
			want:        []string{"spec/models/user_spec.rb"},
		},
		{
			name:        "nested spec file returns itself",
			changedFile: "spec/lib/validators/email_validator_spec.rb",
			want:        []string{"spec/lib/validators/email_validator_spec.rb"},
		},

		// spec_helper.rb
		{
			name:        "spec_helper.rb runs all specs",
			changedFile: "spec/spec_helper.rb",
			want:        []string{"spec"},
		},
		{
			name:        "rails_helper.rb runs all specs",
			changedFile: "spec/rails_helper.rb",
			want:        []string{"spec"},
		},

		// lib/ -> spec/ mapping
		{
			name:        "lib file maps to spec",
			changedFile: "lib/calculator.rb",
			want:        []string{"spec/calculator_spec.rb"},
		},
		{
			name:        "nested lib file maps to nested spec",
			changedFile: "lib/validators/email_validator.rb",
			want:        []string{"spec/validators/email_validator_spec.rb"},
		},

		// Rails app/ -> spec/ mapping
		{
			name:        "model file maps to model spec",
			changedFile: "app/models/user.rb",
			want:        []string{"spec/models/user_spec.rb"},
		},
		{
			name:        "controller file maps to controller spec",
			changedFile: "app/controllers/users_controller.rb",
			want:        []string{"spec/controllers/users_controller_spec.rb"},
		},
		{
			name:        "nested Rails file maps correctly",
			changedFile: "app/models/concerns/validatable.rb",
			want:        []string{"spec/models/concerns/validatable_spec.rb"},
		},
		{
			name:        "service file maps to service spec",
			changedFile: "app/services/user_creator.rb",
			want:        []string{"spec/services/user_creator_spec.rb"},
		},

		// No mapping
		{
			name:        "non-ruby file returns nil",
			changedFile: "README.md",
			want:        nil,
		},
		{
			name:        "random ruby file outside known dirs returns nil",
			changedFile: "random.rb",
			want:        nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := fm.MapFileToSpecs(tt.changedFile)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MapFileToSpecs(%q) = %v, want %v", tt.changedFile, got, tt.want)
			}
		})
	}
}

func TestFileMapper_ShouldWatchFile(t *testing.T) {
	fm := NewFileMapper()

	tests := []struct {
		name     string
		filePath string
		want     bool
	}{
		{
			name:     "ruby file",
			filePath: "lib/calculator.rb",
			want:     true,
		},
		{
			name:     "erb template",
			filePath: "app/views/users/index.html.erb",
			want:     true,
		},
		{
			name:     "haml template",
			filePath: "app/views/users/show.html.haml",
			want:     true,
		},
		{
			name:     "slim template",
			filePath: "app/views/users/edit.html.slim",
			want:     true,
		},
		{
			name:     "markdown file",
			filePath: "README.md",
			want:     false,
		},
		{
			name:     "yaml file",
			filePath: "config/database.yml",
			want:     false,
		},
		{
			name:     "javascript file",
			filePath: "app/assets/javascripts/application.js",
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := fm.ShouldWatchFile(tt.filePath); got != tt.want {
				t.Errorf("ShouldWatchFile(%q) = %v, want %v", tt.filePath, got, tt.want)
			}
		})
	}
}