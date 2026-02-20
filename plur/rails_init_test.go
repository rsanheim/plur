package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransformDatabaseYml(t *testing.T) {
	t.Run("postgresql single database", func(t *testing.T) {
		input := `default: &default
  adapter: postgresql
  pool: 5

test:
  <<: *default
  database: myapp_test`

		result, skipped := transformDatabaseYml(input)
		assert.False(t, skipped)
		assert.Contains(t, result, `database: myapp_test<%= ENV['TEST_ENV_NUMBER'] %>`)
		assert.Contains(t, result, "adapter: postgresql")
	})

	t.Run("mysql single database", func(t *testing.T) {
		input := `default: &default
  adapter: mysql2
  pool: 5

test:
  <<: *default
  database: myapp_test`

		result, skipped := transformDatabaseYml(input)
		assert.False(t, skipped)
		assert.Contains(t, result, `database: myapp_test<%= ENV['TEST_ENV_NUMBER'] %>`)
	})

	t.Run("multi-database configuration", func(t *testing.T) {
		input := `default: &default
  adapter: postgresql
  pool: 5

test:
  primary:
    <<: *default
    database: myapp_test
  cache:
    <<: *default
    database: myapp_test_cache`

		result, skipped := transformDatabaseYml(input)
		assert.False(t, skipped)
		assert.Contains(t, result, `database: myapp_test<%= ENV['TEST_ENV_NUMBER'] %>`)
		assert.Contains(t, result, `database: myapp_test_cache<%= ENV['TEST_ENV_NUMBER'] %>`)
	})

	t.Run("already configured is idempotent", func(t *testing.T) {
		input := `test:
  <<: *default
  database: myapp_test<%= ENV['TEST_ENV_NUMBER'] %>`

		result, skipped := transformDatabaseYml(input)
		assert.True(t, skipped)
		assert.Equal(t, input, result)
	})

	t.Run("partially configured multi-db transforms remaining", func(t *testing.T) {
		input := `test:
  primary:
    database: myapp_test<%= ENV['TEST_ENV_NUMBER'] %>
  cache:
    database: myapp_test_cache`

		result, skipped := transformDatabaseYml(input)
		assert.False(t, skipped)
		assert.Contains(t, result, `database: myapp_test<%= ENV['TEST_ENV_NUMBER'] %>`)
		assert.Contains(t, result, `database: myapp_test_cache<%= ENV['TEST_ENV_NUMBER'] %>`)
	})

	t.Run("does not modify development or production sections", func(t *testing.T) {
		input := `development:
  database: myapp_dev

test:
  database: myapp_test

production:
  database: myapp_prod`

		result, skipped := transformDatabaseYml(input)
		assert.False(t, skipped)
		assert.Contains(t, result, "database: myapp_dev")
		assert.Contains(t, result, `database: myapp_test<%= ENV['TEST_ENV_NUMBER'] %>`)
		assert.Contains(t, result, "database: myapp_prod")
	})

	t.Run("skips sqlite3 databases", func(t *testing.T) {
		input := `test:
  <<: *default
  database: storage/test.sqlite3`

		result, skipped := transformDatabaseYml(input)
		assert.False(t, skipped)
		assert.Equal(t, input, result)
	})

	t.Run("handles comments in test section", func(t *testing.T) {
		input := `test:
  # Use test database
  <<: *default
  database: myapp_test`

		result, skipped := transformDatabaseYml(input)
		assert.False(t, skipped)
		assert.Contains(t, result, `database: myapp_test<%= ENV['TEST_ENV_NUMBER'] %>`)
		assert.Contains(t, result, "# Use test database")
	})

	t.Run("handles blank lines in test section", func(t *testing.T) {
		input := `test:
  <<: *default

  database: myapp_test`

		result, skipped := transformDatabaseYml(input)
		assert.False(t, skipped)
		assert.Contains(t, result, `database: myapp_test<%= ENV['TEST_ENV_NUMBER'] %>`)
	})

	t.Run("inserts TEST_ENV_NUMBER inside quoted database values", func(t *testing.T) {
		input := `test:
  database: "myapp_test"`

		result, skipped := transformDatabaseYml(input)
		assert.False(t, skipped)
		assert.Contains(t, result, `database: "myapp_test<%= ENV['TEST_ENV_NUMBER'] %>"`)
	})

	t.Run("preserves inline comments when transforming database values", func(t *testing.T) {
		input := `test:
  database: myapp_test # default db`

		result, skipped := transformDatabaseYml(input)
		assert.False(t, skipped)
		assert.Contains(t, result, `database: myapp_test<%= ENV['TEST_ENV_NUMBER'] %> # default db`)
	})

	t.Run("no test section returns content unchanged", func(t *testing.T) {
		input := `development:
  database: myapp_dev

production:
  database: myapp_prod`

		result, skipped := transformDatabaseYml(input)
		assert.False(t, skipped)
		assert.Equal(t, input, result)
	})

	t.Run("test section with no database line", func(t *testing.T) {
		input := `test:
  <<: *default
  adapter: postgresql`

		result, skipped := transformDatabaseYml(input)
		assert.False(t, skipped)
		assert.Equal(t, input, result)
	})
}

func TestInsertTestEnvNumber(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple database name",
			input:    "myapp_test",
			expected: "myapp_test<%= ENV['TEST_ENV_NUMBER'] %>",
		},
		{
			name:     "database name with hyphens",
			input:    "my-app-test",
			expected: "my-app-test<%= ENV['TEST_ENV_NUMBER'] %>",
		},
		{
			name:     "already has TEST_ENV_NUMBER",
			input:    "myapp_test<%= ENV['TEST_ENV_NUMBER'] %>",
			expected: "myapp_test<%= ENV['TEST_ENV_NUMBER'] %>",
		},
		{
			name:     "ERB database name",
			input:    "<%= ENV['DATABASE_NAME'] %>",
			expected: "<%= ENV['DATABASE_NAME'] %><%= ENV['TEST_ENV_NUMBER'] %>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := insertTestEnvNumber(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestTransformDatabaseLine(t *testing.T) {
	t.Run("transforms plain postgres database name", func(t *testing.T) {
		result := transformDatabaseLine("    database: myapp_test")
		assert.Equal(t, "    database: myapp_test<%= ENV['TEST_ENV_NUMBER'] %>", result)
	})

	t.Run("preserves indentation", func(t *testing.T) {
		result := transformDatabaseLine("      database: myapp_test")
		assert.Equal(t, "      database: myapp_test<%= ENV['TEST_ENV_NUMBER'] %>", result)
	})

	t.Run("inserts TEST_ENV_NUMBER inside double quotes", func(t *testing.T) {
		result := transformDatabaseLine(`    database: "myapp_test"`)
		assert.Equal(t, `    database: "myapp_test<%= ENV['TEST_ENV_NUMBER'] %>"`, result)
	})

	t.Run("inserts TEST_ENV_NUMBER inside single quotes", func(t *testing.T) {
		result := transformDatabaseLine("    database: 'myapp_test'")
		assert.Equal(t, "    database: 'myapp_test<%= ENV['TEST_ENV_NUMBER'] %>'", result)
	})

	t.Run("preserves inline comments", func(t *testing.T) {
		result := transformDatabaseLine("    database: myapp_test # default db")
		assert.Equal(t, "    database: myapp_test<%= ENV['TEST_ENV_NUMBER'] %> # default db", result)
	})

	t.Run("skips sqlite3", func(t *testing.T) {
		line := "    database: storage/test.sqlite3"
		result := transformDatabaseLine(line)
		assert.Equal(t, line, result)
	})

	t.Run("skips already configured", func(t *testing.T) {
		line := "    database: myapp_test<%= ENV['TEST_ENV_NUMBER'] %>"
		result := transformDatabaseLine(line)
		assert.Equal(t, line, result)
	})
}

func TestTransformCableYml(t *testing.T) {
	t.Run("transforms redis URL in test section", func(t *testing.T) {
		input := `development:
  adapter: redis
  url: redis://localhost:6379/1

test:
  adapter: redis
  url: redis://localhost:6379/0

production:
  adapter: redis
  url: redis://localhost:6379/1`

		result, skipped := transformCableYml(input)
		assert.False(t, skipped)
		assert.Contains(t, result, `redis://localhost:6379/<%= ENV.fetch('TEST_ENV_NUMBER', '0').to_i %>`)
		// Development and production should be unchanged
		lines := splitLines(result)
		devURL := findLineContaining(lines, "development", "url:")
		assert.Contains(t, devURL, "redis://localhost:6379/1")
	})

	t.Run("skips test adapter (no redis)", func(t *testing.T) {
		input := `test:
  adapter: test`

		result, skipped := transformCableYml(input)
		assert.False(t, skipped)
		assert.Equal(t, input, result)
	})

	t.Run("already configured is idempotent", func(t *testing.T) {
		input := `test:
  adapter: redis
  url: redis://localhost:6379/<%= ENV.fetch('TEST_ENV_NUMBER', '0').to_i %>`

		result, skipped := transformCableYml(input)
		assert.True(t, skipped)
		assert.Equal(t, input, result)
	})
}

func TestTransformRedisURLLine(t *testing.T) {
	t.Run("transforms redis URL with database number", func(t *testing.T) {
		result := transformRedisURLLine("  url: redis://localhost:6379/0")
		assert.Equal(t, "  url: redis://localhost:6379/<%= ENV.fetch('TEST_ENV_NUMBER', '0').to_i %>", result)
	})

	t.Run("transforms redis URL with higher database number", func(t *testing.T) {
		result := transformRedisURLLine("  url: redis://localhost:6379/5")
		assert.Equal(t, "  url: redis://localhost:6379/<%= ENV.fetch('TEST_ENV_NUMBER', '5').to_i %>", result)
	})

	t.Run("skips already configured", func(t *testing.T) {
		line := "  url: redis://localhost:6379/<%= ENV.fetch('TEST_ENV_NUMBER', '0').to_i %>"
		result := transformRedisURLLine(line)
		assert.Equal(t, line, result)
	})
}

func TestIsSQLiteOnlyTestDatabaseConfig(t *testing.T) {
	t.Run("returns true for sqlite-only test databases", func(t *testing.T) {
		content := `test:
  database: storage/test.sqlite3`

		assert.True(t, isSQLiteOnlyTestDatabaseConfig(content))
	})

	t.Run("returns false for postgres test databases", func(t *testing.T) {
		content := `test:
  database: myapp_test`

		assert.False(t, isSQLiteOnlyTestDatabaseConfig(content))
	})

	t.Run("returns false when no test database lines are present", func(t *testing.T) {
		content := `test:
  adapter: postgresql`

		assert.False(t, isSQLiteOnlyTestDatabaseConfig(content))
	})

	t.Run("returns false for mixed sqlite and non-sqlite database lines", func(t *testing.T) {
		content := `test:
  primary:
    database: storage/test.sqlite3
  cache:
    database: myapp_test_cache`

		assert.False(t, isSQLiteOnlyTestDatabaseConfig(content))
	})
}

func TestHasUnindexedRedisURLInTestSection(t *testing.T) {
	t.Run("returns true when test redis URL has no database index", func(t *testing.T) {
		content := `test:
  adapter: redis
  url: redis://localhost:6379`

		assert.True(t, hasUnindexedRedisURLInTestSection(content))
	})

	t.Run("returns false when test redis URL has database index", func(t *testing.T) {
		content := `test:
  adapter: redis
  url: redis://localhost:6379/0`

		assert.False(t, hasUnindexedRedisURLInTestSection(content))
	})

	t.Run("returns false when test redis URL already uses TEST_ENV_NUMBER", func(t *testing.T) {
		content := `test:
  adapter: redis
  url: redis://localhost:6379/<%= ENV.fetch('TEST_ENV_NUMBER', '0').to_i %>`

		assert.False(t, hasUnindexedRedisURLInTestSection(content))
	})

	t.Run("returns false when only non-test sections have unindexed redis URLs", func(t *testing.T) {
		content := `development:
  adapter: redis
  url: redis://localhost:6379

test:
  adapter: test`

		assert.False(t, hasUnindexedRedisURLInTestSection(content))
	})
}

func TestValidateYAMLWithERB(t *testing.T) {
	t.Run("accepts valid yaml with erb", func(t *testing.T) {
		content := `test:
  database: myapp_test<%= ENV['TEST_ENV_NUMBER'] %>`

		err := validateYAMLWithERB(content)
		require.NoError(t, err)
	})

	t.Run("rejects invalid yaml when erb is placed outside quotes", func(t *testing.T) {
		content := `test:
  database: "myapp_test"<%= ENV['TEST_ENV_NUMBER'] %>`

		err := validateYAMLWithERB(content)
		require.Error(t, err)
	})
}

func TestVerifyRailsProject(t *testing.T) {
	t.Run("succeeds with database.yml and Gemfile", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(dir, "config"), 0755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "config", "database.yml"), []byte("test:\n"), 0644))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "Gemfile"), []byte("source 'https://rubygems.org'\n"), 0644))

		origDir, _ := os.Getwd()
		require.NoError(t, os.Chdir(dir))
		defer os.Chdir(origDir)

		err := verifyRailsProject()
		assert.NoError(t, err)
	})

	t.Run("succeeds with database.yml and application.rb", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(dir, "config"), 0755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "config", "database.yml"), []byte("test:\n"), 0644))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "config", "application.rb"), []byte("# rails\n"), 0644))

		origDir, _ := os.Getwd()
		require.NoError(t, os.Chdir(dir))
		defer os.Chdir(origDir)

		err := verifyRailsProject()
		assert.NoError(t, err)
	})

	t.Run("fails without database.yml", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(dir, "Gemfile"), []byte("source 'https://rubygems.org'\n"), 0644))

		origDir, _ := os.Getwd()
		require.NoError(t, os.Chdir(dir))
		defer os.Chdir(origDir)

		err := verifyRailsProject()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "config/database.yml not found")
	})

	t.Run("fails without Gemfile or application.rb", func(t *testing.T) {
		dir := t.TempDir()
		require.NoError(t, os.MkdirAll(filepath.Join(dir, "config"), 0755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "config", "database.yml"), []byte("test:\n"), 0644))

		origDir, _ := os.Getwd()
		require.NoError(t, os.Chdir(dir))
		defer os.Chdir(origDir)

		err := verifyRailsProject()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no Gemfile or config/application.rb found")
	})
}

// helpers for cable.yml tests

func splitLines(s string) []string {
	return append([]string(nil), splitNonEmpty(s)...)
}

func splitNonEmpty(s string) []string {
	var result []string
	for _, line := range append([]string(nil), splitAll(s)...) {
		result = append(result, line)
	}
	return result
}

func splitAll(s string) []string {
	lines := make([]string, 0)
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func findLineContaining(lines []string, sectionHint, key string) string {
	inSection := false
	for _, line := range lines {
		trimmed := trimLeftSpaces(line)
		if trimmed == sectionHint+":" {
			inSection = true
			continue
		}
		if inSection && len(line) > 0 && line[0] != ' ' && line[0] != '\t' {
			inSection = false
		}
		if inSection && containsStr(trimmed, key) {
			return line
		}
	}
	return ""
}

func trimLeftSpaces(s string) string {
	i := 0
	for i < len(s) && s[i] == ' ' {
		i++
	}
	return s[i:]
}

func containsStr(s, sub string) bool {
	return len(s) >= len(sub) && searchStr(s, sub)
}

func searchStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
