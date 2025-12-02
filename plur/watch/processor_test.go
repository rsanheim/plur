package watch

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/rsanheim/plur/job"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEventProcessorBasicMapping(t *testing.T) {
	jobs := map[string]job.Job{
		"rspec": {
			Name: "rspec",
			Cmd:  []string{"bundle", "exec", "rspec", "{{target}}"},
		},
	}

	watches := []WatchMapping{
		{
			Name:    "lib-to-spec",
			Source:  "lib/**/*.rb",
			Targets: []string{"spec/{{match}}_spec.rb"},
			Jobs:    []string{"rspec"},
		},
	}

	processor := NewEventProcessor(jobs, watches)

	tests := []struct {
		name           string
		path           string
		expectedJobs   []string
		expectedTarget string
	}{
		{
			name:           "simple lib file",
			path:           "lib/user.rb",
			expectedJobs:   []string{"rspec"},
			expectedTarget: "spec/user_spec.rb",
		},
		{
			name:           "nested lib file",
			path:           "lib/models/user.rb",
			expectedJobs:   []string{"rspec"},
			expectedTarget: "spec/models/user_spec.rb",
		},
		{
			name:         "non-matching file",
			path:         "app/models/user.rb",
			expectedJobs: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := processor.ProcessPath(tt.path)
			require.NoError(t, err)

			if len(tt.expectedJobs) == 0 {
				assert.Empty(t, result)
				return
			}

			for _, jobName := range tt.expectedJobs {
				assert.Contains(t, result, jobName)
				if tt.expectedTarget != "" {
					assert.Contains(t, result[jobName], filepath.FromSlash(tt.expectedTarget))
				}
			}
		})
	}
}

func TestEventProcessorNoTargets(t *testing.T) {
	jobs := map[string]job.Job{
		"rspec": {
			Name: "rspec",
			Cmd:  []string{"bundle", "exec", "rspec", "{{target}}"},
		},
	}

	// No targets means use the source file itself
	watches := []WatchMapping{
		{
			Name:    "spec-files",
			Source:  "spec/**/*_spec.rb",
			Targets: nil,
			Jobs:    []string{"rspec"},
		},
	}

	processor := NewEventProcessor(jobs, watches)

	result, err := processor.ProcessPath("spec/models/user_spec.rb")
	require.NoError(t, err)

	assert.Contains(t, result, "rspec")
	assert.Equal(t, []string{filepath.FromSlash("spec/models/user_spec.rb")}, result["rspec"])
}

func TestEventProcessorMultipleJobs(t *testing.T) {
	jobs := map[string]job.Job{
		"rspec": {
			Name: "rspec",
			Cmd:  []string{"bundle", "exec", "rspec", "{{target}}"},
		},
		"rubocop": {
			Name: "rubocop",
			Cmd:  []string{"bundle", "exec", "rubocop", "{{target}}"},
		},
	}

	watches := []WatchMapping{
		{
			Name:    "lib-to-both",
			Source:  "lib/**/*.rb",
			Targets: []string{"spec/{{match}}_spec.rb"},
			Jobs:    []string{"rspec", "rubocop"},
		},
	}

	processor := NewEventProcessor(jobs, watches)

	result, err := processor.ProcessPath("lib/user.rb")
	require.NoError(t, err)

	assert.Contains(t, result, "rspec")
	assert.Contains(t, result, "rubocop")
	assert.Equal(t, []string{filepath.FromSlash("spec/user_spec.rb")}, result["rspec"])
	assert.Equal(t, []string{filepath.FromSlash("spec/user_spec.rb")}, result["rubocop"])
}

func TestEventProcessorMultipleTargets(t *testing.T) {
	jobs := map[string]job.Job{
		"rspec": {
			Name: "rspec",
			Cmd:  []string{"bundle", "exec", "rspec", "{{target}}"},
		},
	}

	// One source file can map to multiple targets
	watches := []WatchMapping{
		{
			Name:    "lib-to-multiple-specs",
			Source:  "lib/**/*.rb",
			Targets: []string{"spec/{{match}}_spec.rb", "spec/lib/{{match}}_spec.rb"},
			Jobs:    []string{"rspec"},
		},
	}

	processor := NewEventProcessor(jobs, watches)

	result, err := processor.ProcessPath("lib/user.rb")
	require.NoError(t, err)

	assert.Contains(t, result, "rspec")
	assert.Len(t, result["rspec"], 2)
	assert.Contains(t, result["rspec"], filepath.FromSlash("spec/user_spec.rb"))
	assert.Contains(t, result["rspec"], filepath.FromSlash("spec/lib/user_spec.rb"))
}

func TestEventProcessorIgnorePatterns(t *testing.T) {
	jobs := map[string]job.Job{
		"rspec": {
			Name: "rspec",
			Cmd:  []string{"bundle", "exec", "rspec", "{{target}}"},
		},
	}

	watches := []WatchMapping{
		{
			Name:    "lib-to-spec",
			Source:  "lib/**/*.rb",
			Targets: []string{"spec/{{match}}_spec.rb"},
			Jobs:    []string{"rspec"},
			Ignore:  []string{"lib/generators/**", "lib/vendor/**"},
		},
	}

	processor := NewEventProcessor(jobs, watches)

	tests := []struct {
		name        string
		path        string
		shouldMatch bool
	}{
		{
			name:        "normal lib file",
			path:        "lib/user.rb",
			shouldMatch: true,
		},
		{
			name:        "generators file (ignored)",
			path:        "lib/generators/model.rb",
			shouldMatch: false,
		},
		{
			name:        "vendor file (ignored)",
			path:        "lib/vendor/gem.rb",
			shouldMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := processor.ProcessPath(tt.path)
			require.NoError(t, err)

			if tt.shouldMatch {
				assert.NotEmpty(t, result)
			} else {
				assert.Empty(t, result)
			}
		})
	}
}

func TestEventProcessorMultipleWatches(t *testing.T) {
	jobs := map[string]job.Job{
		"rspec": {
			Name: "rspec",
			Cmd:  []string{"bundle", "exec", "rspec", "{{target}}"},
		},
	}

	watches := []WatchMapping{
		{
			Name:    "lib-to-spec",
			Source:  "lib/**/*.rb",
			Targets: []string{"spec/{{match}}_spec.rb"},
			Jobs:    []string{"rspec"},
		},
		{
			Name:    "app-to-spec",
			Source:  "app/**/*.rb",
			Targets: []string{"spec/{{match}}_spec.rb"},
			Jobs:    []string{"rspec"},
		},
		{
			Name:    "spec-files",
			Source:  "spec/**/*_spec.rb",
			Targets: nil,
			Jobs:    []string{"rspec"},
		},
	}

	processor := NewEventProcessor(jobs, watches)

	tests := []struct {
		name           string
		path           string
		expectedTarget string
	}{
		{
			name:           "lib file",
			path:           "lib/user.rb",
			expectedTarget: "spec/user_spec.rb",
		},
		{
			name:           "app file",
			path:           "app/models/post.rb",
			expectedTarget: "spec/models/post_spec.rb",
		},
		{
			name:           "spec file",
			path:           "spec/models/user_spec.rb",
			expectedTarget: "spec/models/user_spec.rb",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := processor.ProcessPath(tt.path)
			require.NoError(t, err)

			assert.Contains(t, result, "rspec")
			assert.Contains(t, result["rspec"], filepath.FromSlash(tt.expectedTarget))
		})
	}
}

func TestEventProcessorDeduplication(t *testing.T) {
	jobs := map[string]job.Job{
		"rspec": {
			Name: "rspec",
			Cmd:  []string{"bundle", "exec", "rspec", "{{target}}"},
		},
	}

	watches := []WatchMapping{
		{
			Name:    "duplicate-targets",
			Source:  "lib/user.rb",
			Targets: []string{"spec/user_spec.rb", "spec/user_spec.rb"}, // Duplicate targets
			Jobs:    []string{"rspec"},
		},
	}

	processor := NewEventProcessor(jobs, watches)

	result, err := processor.ProcessPath("lib/user.rb")
	require.NoError(t, err)

	assert.Contains(t, result, "rspec")
	assert.Len(t, result["rspec"], 1, "Should deduplicate targets")
}

func TestEventProcessorUndefinedJob(t *testing.T) {
	jobs := map[string]job.Job{
		"rspec": {
			Name: "rspec",
			Cmd:  []string{"bundle", "exec", "rspec", "{{target}}"},
		},
	}

	watches := []WatchMapping{
		{
			Name:    "lib-to-spec",
			Source:  "lib/**/*.rb",
			Targets: []string{"spec/{{match}}_spec.rb"},
			Jobs:    []string{"minitest"}, // Job doesn't exist
		},
	}

	processor := NewEventProcessor(jobs, watches)

	_, err := processor.ProcessPath("lib/user.rb")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "undefined job")
}

func TestValidateConfig(t *testing.T) {
	jobs := map[string]job.Job{
		"rspec": {
			Name: "rspec",
			Cmd:  []string{"bundle", "exec", "rspec", "{{target}}"},
		},
	}

	tests := []struct {
		name    string
		watches []WatchMapping
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			watches: []WatchMapping{
				{
					Name:    "lib-to-spec",
					Source:  "lib/**/*.rb",
					Targets: []string{"spec/{{match}}_spec.rb"},
					Jobs:    []string{"rspec"},
				},
			},
			wantErr: false,
		},
		{
			name: "undefined job",
			watches: []WatchMapping{
				{
					Name:    "lib-to-spec",
					Source:  "lib/**/*.rb",
					Targets: []string{"spec/{{match}}_spec.rb"},
					Jobs:    []string{"nonexistent"},
				},
			},
			wantErr: true,
			errMsg:  "undefined job",
		},
		{
			name: "invalid template",
			watches: []WatchMapping{
				{
					Name:    "lib-to-spec",
					Source:  "lib/**/*.rb",
					Targets: []string{"spec/{{invalid}}_spec.rb"},
					Jobs:    []string{"rspec"},
				},
			},
			wantErr: true,
			errMsg:  "invalid target template",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfig(jobs, tt.watches)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestEventProcessorComplexity detects if path processing degrades beyond O(n)
func TestEventProcessorComplexity(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping complexity test in short mode")
	}

	sizes := []int{1000, 2000, 4000, 8000} // number of watch rules
	iterations := 20

	times := make([]time.Duration, len(sizes))

	for idx, size := range sizes {
		watches := generateWatchMappings(size)
		jobs := map[string]job.Job{"rspec": {Name: "rspec"}}
		processor := NewEventProcessor(jobs, watches)

		// Use a path that won't match any rules to test full iteration
		testPath := "lib/foo/bar.rb"

		start := time.Now()
		for i := 0; i < iterations; i++ {
			processor.ProcessPath(testPath)
		}
		times[idx] = time.Since(start) / time.Duration(iterations)

		t.Logf("EventProcessor: rules=%5d, time=%v", size, times[idx])
	}

	checkLinearScaling(t, "EventProcessor", sizes, times)
}

func generateWatchMappings(n int) []WatchMapping {
	mappings := make([]WatchMapping, n)
	for i := 0; i < n; i++ {
		mappings[i] = WatchMapping{
			Name:    fmt.Sprintf("rule-%d", i),
			Source:  fmt.Sprintf("src/dir%d/**/*.rb", i),
			Targets: []string{"spec/{{match}}_spec.rb"},
			Jobs:    []string{"rspec"},
		}
	}
	return mappings
}

// checkLinearScaling verifies that time grows at most linearly with input size.
// If time ratio exceeds 2x the size ratio, we likely have O(n²) behavior.
func checkLinearScaling(t *testing.T, name string, sizes []int, times []time.Duration) {
	t.Helper()

	for i := 1; i < len(sizes); i++ {
		sizeRatio := float64(sizes[i]) / float64(sizes[i-1])
		timeRatio := float64(times[i]) / float64(times[i-1])

		// Allow 2x tolerance over linear scaling to account for system noise
		threshold := sizeRatio * 2.0

		if timeRatio > threshold {
			t.Errorf("%s: potential O(n²) detected between size %d and %d: "+
				"size ratio=%.2f, time ratio=%.2f (threshold=%.2f)",
				name, sizes[i-1], sizes[i], sizeRatio, timeRatio, threshold)
		}
	}
}
