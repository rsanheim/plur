package railsinit

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/rsanheim/plur/config"
	"github.com/rsanheim/plur/internal/fsutil"
	"gopkg.in/yaml.v3"
)

// initResult tracks what the rails:init command found and did
type initResult struct {
	changesApplied int
	warnings       []string
	alreadyDone    []string
}

func Run(cfg *config.GlobalConfig) error {
	if err := verifyRailsProject(); err != nil {
		return err
	}

	if cfg.DryRun {
		fmt.Println("[dry-run] Plur Rails Init")
	} else {
		fmt.Println("Plur Rails Init")
	}
	fmt.Println()

	result := &initResult{}

	if err := setupDatabaseYml(cfg, result); err != nil {
		return err
	}

	if err := setupCableYml(cfg, result); err != nil {
		return err
	}

	scanForWarnings(result)
	printInitSummary(cfg, result)

	return nil
}

func verifyRailsProject() error {
	if !fsutil.FileExists("config/database.yml") {
		return fmt.Errorf("plur rails:init requires a Rails project (config/database.yml not found)")
	}
	hasGemfile := fsutil.FileExists("Gemfile")
	hasAppRb := fsutil.FileExists("config/application.rb")
	if !hasGemfile && !hasAppRb {
		return fmt.Errorf("plur rails:init requires a Rails project (no Gemfile or config/application.rb found)")
	}
	return nil
}

// setupDatabaseYml reads config/database.yml and transforms database names
// in the test section to include TEST_ENV_NUMBER for parallel test isolation.
func setupDatabaseYml(cfg *config.GlobalConfig, result *initResult) error {
	path := "config/database.yml"
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", path, err)
	}

	original := string(content)

	modified, skipped := transformDatabaseYml(original)

	if skipped {
		result.alreadyDone = append(result.alreadyDone,
			path+": already configured for parallel testing")
		return nil
	}

	if modified == original {
		if isSQLiteOnlyTestDatabaseConfig(original) {
			result.warnings = append(result.warnings,
				path+": sqlite3 test database detected; parallel database transformation is not currently supported for sqlite3")
		} else {
			result.warnings = append(result.warnings,
				path+": could not find test database name to modify (you may need to configure this manually)")
		}
		return nil
	}

	if err := validateYAMLWithERB(modified); err != nil {
		return fmt.Errorf("refusing to update %s: transformed content failed validation: %w", path, err)
	}

	showFileDiff(cfg.DryRun, path, original, modified)

	if !cfg.DryRun {
		if err := os.WriteFile(path, []byte(modified), 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", path, err)
		}
		fmt.Printf("  ✓ Updated %s\n", path)
	}
	result.changesApplied++
	return nil
}

// transformDatabaseYml processes the content of a database.yml file,
// adding TEST_ENV_NUMBER to database names in the test section.
// Returns the modified content and whether it was already configured (skipped).
func transformDatabaseYml(content string) (string, bool) {
	lines := strings.Split(content, "\n")

	// Check if any database line in the test section already has TEST_ENV_NUMBER
	allConfigured := true
	hasDbLine := false
	iterateTestSection(lines, func(_ int, line, trimmed string) {
		if isDatabaseLine(trimmed) {
			hasDbLine = true
			if !strings.Contains(line, "TEST_ENV_NUMBER") {
				allConfigured = false
			}
		}
	})

	if hasDbLine && allConfigured {
		return content, true
	}

	// Now do the actual transformation
	iterateTestSection(lines, func(i int, line, trimmed string) {
		if isDatabaseLine(trimmed) {
			lines[i] = transformDatabaseLine(line)
		}
	})

	return strings.Join(lines, "\n"), false
}

// iterateTestSection calls fn for each non-blank, non-comment line in the test: section.
// The index i is the original line index, usable for in-place modification of lines.
func iterateTestSection(lines []string, fn func(i int, line, trimmed string)) {
	inTestSection := false
	testSectionIndent := 0

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}

		currentIndent := len(line) - len(strings.TrimLeft(line, " "))

		if isTopLevelKey(trimmed, currentIndent, testSectionIndent) && inTestSection {
			break
		}

		if isTestSectionStart(trimmed, currentIndent) {
			inTestSection = true
			testSectionIndent = currentIndent
			continue
		}

		if inTestSection {
			fn(i, line, trimmed)
		}
	}
}

// isTestKey returns true if trimmed is the YAML key named "test" (not test_foo, testing, etc.).
func isTestKey(trimmed string) bool {
	return trimmed == "test:" || strings.HasPrefix(trimmed, "test: ") || strings.HasPrefix(trimmed, "test:\t")
}

// isTestSectionStart returns true if the line starts the test: section.
func isTestSectionStart(trimmed string, indent int) bool {
	return indent == 0 && isTestKey(trimmed)
}

// isTopLevelKey returns true if the line is a top-level YAML key (not test:).
func isTopLevelKey(trimmed string, currentIndent, testSectionIndent int) bool {
	if currentIndent > testSectionIndent {
		return false
	}
	if strings.HasPrefix(trimmed, "#") {
		return false
	}
	// A top-level key that isn't the test section
	return currentIndent == 0 && strings.Contains(trimmed, ":") && !isTestKey(trimmed)
}

// isDatabaseLine returns true if the trimmed line is a database: key.
func isDatabaseLine(trimmed string) bool {
	return strings.HasPrefix(trimmed, "database:")
}

// sqliteRe matches sqlite3 database file paths.
var sqliteRe = regexp.MustCompile(`\.sqlite3`)

// dbLineRe captures the prefix and value of a database: YAML line.
var dbLineRe = regexp.MustCompile(`^(\s*database:\s*)(.+)$`)

// transformDatabaseLine adds TEST_ENV_NUMBER to a database: line.
// Skips sqlite3 databases. Skips lines that already have TEST_ENV_NUMBER.
func transformDatabaseLine(line string) string {
	if strings.Contains(line, "TEST_ENV_NUMBER") {
		return line
	}

	match := dbLineRe.FindStringSubmatch(line)
	if match == nil {
		return line
	}

	prefix := match[1]
	value, comment := splitYAMLValueAndComment(match[2])

	// Skip sqlite3 databases
	if sqliteRe.MatchString(value) {
		return line
	}

	transformed := insertTestEnvNumber(value)
	if comment != "" {
		return prefix + transformed + " " + comment
	}
	return prefix + transformed
}

// insertTestEnvNumber appends the TEST_ENV_NUMBER ERB tag to a database name.
// Handles plain names and ERB-wrapped values.
func insertTestEnvNumber(dbValue string) string {
	if strings.Contains(dbValue, "TEST_ENV_NUMBER") {
		return dbValue
	}

	if len(dbValue) >= 2 {
		first := dbValue[0]
		last := dbValue[len(dbValue)-1]
		if (first == '"' && last == '"') || (first == '\'' && last == '\'') {
			return dbValue[:len(dbValue)-1] + `<%= ENV['TEST_ENV_NUMBER'] %>` + dbValue[len(dbValue)-1:]
		}
	}

	return dbValue + `<%= ENV['TEST_ENV_NUMBER'] %>`
}

// splitYAMLValueAndComment splits a YAML scalar value from an inline comment.
// The returned comment does not include leading whitespace.
func splitYAMLValueAndComment(valueWithComment string) (string, string) {
	inSingleQuotes := false
	inDoubleQuotes := false
	escaped := false

	for i := 0; i < len(valueWithComment); i++ {
		ch := valueWithComment[i]

		if inDoubleQuotes {
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == '"' {
				inDoubleQuotes = false
			}
			continue
		}

		if inSingleQuotes {
			if ch == '\'' {
				inSingleQuotes = false
			}
			continue
		}

		if ch == '"' {
			inDoubleQuotes = true
			continue
		}
		if ch == '\'' {
			inSingleQuotes = true
			continue
		}

		if ch == '#' && (i == 0 || valueWithComment[i-1] == ' ' || valueWithComment[i-1] == '\t') {
			value := strings.TrimSpace(valueWithComment[:i])
			comment := strings.TrimLeft(valueWithComment[i:], " \t")
			return value, comment
		}
	}

	return strings.TrimSpace(valueWithComment), ""
}

// setupCableYml checks config/cable.yml for redis URLs in the test section
// that may need TEST_ENV_NUMBER isolation.
func setupCableYml(cfg *config.GlobalConfig, result *initResult) error {
	path := "config/cable.yml"
	if !fsutil.FileExists(path) {
		return nil
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", path, err)
	}

	original := string(content)

	modified, skipped := transformCableYml(original)

	if skipped {
		result.alreadyDone = append(result.alreadyDone,
			path+": already configured for parallel testing")
		return nil
	}

	if hasUnindexedRedisURLInTestSection(modified) {
		result.warnings = append(result.warnings,
			path+": found redis URL in test section without a /N database index; configure TEST_ENV_NUMBER isolation manually")
	}

	if modified == original {
		return nil
	}

	if err := validateYAMLWithERB(modified); err != nil {
		return fmt.Errorf("refusing to update %s: transformed content failed validation: %w", path, err)
	}

	showFileDiff(cfg.DryRun, path, original, modified)

	if !cfg.DryRun {
		if err := os.WriteFile(path, []byte(modified), 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", path, err)
		}
		fmt.Printf("  ✓ Updated %s\n", path)
	}
	result.changesApplied++
	return nil
}

// transformCableYml processes config/cable.yml, adding TEST_ENV_NUMBER
// to redis URLs in the test section.
func transformCableYml(content string) (string, bool) {
	lines := strings.Split(content, "\n")

	hasRedisURL := false
	allConfigured := true
	iterateTestSection(lines, func(_ int, line, trimmed string) {
		if isRedisURLLine(trimmed) {
			hasRedisURL = true
			if !strings.Contains(line, "TEST_ENV_NUMBER") {
				allConfigured = false
			}
		}
	})

	if hasRedisURL && allConfigured {
		return content, true
	}

	if !hasRedisURL {
		return content, false
	}

	// Transform redis URLs
	iterateTestSection(lines, func(i int, line, trimmed string) {
		if isRedisURLLine(trimmed) {
			lines[i] = transformRedisURLLine(line)
		}
	})

	return strings.Join(lines, "\n"), false
}

// isRedisURLLine returns true if the line contains a redis:// URL.
func isRedisURLLine(trimmed string) bool {
	return strings.Contains(trimmed, "redis://")
}

// redisDBRe matches a redis URL with a numeric database index (e.g. redis://host:6379/0).
var redisDBRe = regexp.MustCompile(`(redis://[^/]+/)(\d+)`)

// transformRedisURLLine replaces the redis database number with a
// TEST_ENV_NUMBER-based ERB expression.
func transformRedisURLLine(line string) string {
	if strings.Contains(line, "TEST_ENV_NUMBER") {
		return line
	}

	// Replace redis://host:port/N with redis://host:port/<%= ENV.fetch('TEST_ENV_NUMBER', N) %>
	return redisDBRe.ReplaceAllString(line,
		`${1}<%= ENV.fetch('TEST_ENV_NUMBER', '${2}').to_i %>`)
}

// warningFilesOrder lists config files that may need manual parallel isolation,
// in the order they are checked and reported.
var warningFilesOrder = []struct {
	path string
	msg  string
}{
	{"config/initializers/sidekiq.rb", "Sidekiq may need Redis URL isolation for parallel testing"},
	{"config/sidekiq.yml", "Sidekiq may need Redis URL isolation for parallel testing"},
	{"config/initializers/elasticsearch.rb", "Elasticsearch may need index prefix isolation for parallel testing"},
	{"config/initializers/searchkick.rb", "Searchkick may need index prefix isolation for parallel testing"},
	{"config/initializers/opensearch.rb", "OpenSearch may need index prefix isolation for parallel testing"},
}

// scanForWarnings checks for files that may need manual configuration
// for parallel test isolation.
func scanForWarnings(result *initResult) {
	envFiles := []string{".env.test", ".env.test.local"}
	for _, path := range envFiles {
		if !fsutil.FileExists(path) {
			continue
		}
		content, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		if containsServiceURLs(string(content)) {
			result.warnings = append(result.warnings,
				path+": contains service URLs that may need TEST_ENV_NUMBER isolation (REDIS_URL, ELASTICSEARCH_URL, etc.)")
		}
	}

	for _, wf := range warningFilesOrder {
		if fsutil.FileExists(wf.path) {
			result.warnings = append(result.warnings, wf.path+": "+wf.msg)
		}
	}
}

func containsServiceURLs(content string) bool {
	patterns := []string{
		"REDIS_URL",
		"ELASTICSEARCH_URL",
		"OPENSEARCH_URL",
		"MEMCACHE",
	}
	for _, pattern := range patterns {
		if strings.Contains(content, pattern) {
			return true
		}
	}
	return false
}

func isSQLiteOnlyTestDatabaseConfig(content string) bool {
	values := testSectionDatabaseValues(content)
	if len(values) == 0 {
		return false
	}

	for _, value := range values {
		if !sqliteRe.MatchString(value) {
			return false
		}
	}

	return true
}

func testSectionDatabaseValues(content string) []string {
	lines := strings.Split(content, "\n")
	values := make([]string, 0)

	iterateTestSection(lines, func(_ int, line, trimmed string) {
		if !isDatabaseLine(trimmed) {
			return
		}
		match := dbLineRe.FindStringSubmatch(line)
		if match == nil {
			return
		}
		value, _ := splitYAMLValueAndComment(match[2])
		values = append(values, value)
	})

	return values
}

func hasUnindexedRedisURLInTestSection(content string) bool {
	lines := strings.Split(content, "\n")
	found := false

	iterateTestSection(lines, func(_ int, line, trimmed string) {
		if found || !isRedisURLLine(trimmed) {
			return
		}
		if !strings.Contains(line, "TEST_ENV_NUMBER") && !redisDBRe.MatchString(line) {
			found = true
		}
	})

	return found
}

// erbTagRe matches ERB tags (both <%= %> and <% %>).
var erbTagRe = regexp.MustCompile(`(?s)<%=?[\s\S]*?%>`)

// validateYAMLWithERB validates YAML content that may include ERB tags by
// replacing ERB segments with placeholders before parsing.
func validateYAMLWithERB(content string) error {
	sanitized := erbTagRe.ReplaceAllStringFunc(content, func(match string) string {
		newlineCount := strings.Count(match, "\n")
		if newlineCount == 0 {
			return "0"
		}
		return "0" + strings.Repeat("\n", newlineCount)
	})

	var parsed any
	return yaml.Unmarshal([]byte(sanitized), &parsed)
}

// showFileDiff displays a simple diff of changed lines between two file versions.
func showFileDiff(dryRun bool, path, original, modified string) {
	prefix := ""
	if dryRun {
		prefix = "[dry-run] "
	}
	fmt.Printf("%s%s:\n", prefix, path)

	origLines := strings.Split(original, "\n")
	modLines := strings.Split(modified, "\n")

	maxLen := len(origLines)
	if len(modLines) > maxLen {
		maxLen = len(modLines)
	}

	for i := 0; i < maxLen; i++ {
		origLine := ""
		modLine := ""
		if i < len(origLines) {
			origLine = origLines[i]
		}
		if i < len(modLines) {
			modLine = modLines[i]
		}
		if origLine != modLine {
			fmt.Printf("  - %s\n", origLine)
			fmt.Printf("  + %s\n", modLine)
		}
	}
	fmt.Println()
}

func printInitSummary(cfg *config.GlobalConfig, result *initResult) {
	for _, msg := range result.alreadyDone {
		fmt.Printf("  ✓ %s\n", msg)
	}

	if len(result.warnings) > 0 {
		fmt.Println()
		fmt.Println("Warnings (may need manual configuration):")
		for _, w := range result.warnings {
			fmt.Printf("  ✗ %s\n", w)
		}
	}

	fmt.Println()
	if result.changesApplied > 0 && !cfg.DryRun {
		fmt.Println("Next steps:")
		fmt.Println("  1. Review the changes above")
		fmt.Println("  2. plur db:create    - Create parallel test databases")
		fmt.Println("  3. plur db:migrate   - Run migrations on all test databases")
		fmt.Println("  4. plur spec         - Run your tests in parallel")
	} else if cfg.DryRun {
		fmt.Println("Run 'plur rails:init' without --dry-run to apply these changes.")
	} else if result.changesApplied == 0 && len(result.alreadyDone) > 0 {
		fmt.Println("Your project appears ready for parallel testing.")
		fmt.Println()
		fmt.Println("Next steps:")
		fmt.Println("  1. plur db:create    - Create parallel test databases")
		fmt.Println("  2. plur db:migrate   - Run migrations on all test databases")
		fmt.Println("  3. plur spec         - Run your tests in parallel")
	}
}
