package minitest

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildCommand(t *testing.T) {
	tests := []struct {
		name     string
		files    []string
		options  BuildOptions
		expected []string
	}{
		{
			name:    "single file",
			files:   []string{"test/models/user_test.rb"},
			options: BuildOptions{},
			expected: []string{
				"ruby", "-Itest", "test/models/user_test.rb",
			},
		},
		{
			name:    "multiple files",
			files:   []string{"test/models/user_test.rb", "test/models/post_test.rb"},
			options: BuildOptions{},
			expected: []string{
				"ruby", "-Itest", "-e",
				"['test/models/user_test.rb', 'test/models/post_test.rb'].each { |f| require f }",
			},
		},
		{
			name:  "single file with verbose",
			files: []string{"test/models/user_test.rb"},
			options: BuildOptions{
				Verbose: true,
			},
			expected: []string{
				"ruby", "-Itest", "-v", "test/models/user_test.rb",
			},
		},
		{
			name:  "with test options",
			files: []string{"test/models/user_test.rb"},
			options: BuildOptions{
				TestOptions: []string{"-n", "test_name_validation"},
			},
			expected: []string{
				"ruby", "-Itest", "test/models/user_test.rb",
				"--", "-n", "test_name_validation",
			},
		},
		{
			name:  "multiple files with all options",
			files: []string{"test/models/user_test.rb", "test/models/post_test.rb"},
			options: BuildOptions{
				Verbose:     true,
				TestOptions: []string{"--seed", "1234"},
			},
			expected: []string{
				"ruby", "-Itest", "-v", "-e",
				"['test/models/user_test.rb', 'test/models/post_test.rb'].each { |f| require f }",
				"--", "--seed", "1234",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BuildCommand(tt.files, tt.options)
			assert.Equal(t, tt.expected, result)
		})
	}
}
