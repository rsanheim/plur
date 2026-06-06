package watch

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAdmitEvent(t *testing.T) {
	cwd := filepath.Join("repo", "project")

	tests := []struct {
		name     string
		event    Event
		ignores  []string
		expected AdmissionResult
	}{
		{
			name: "admits modify event",
			event: Event{
				PathType:   "file",
				PathName:   filepath.Join(cwd, "lib", "user.rb"),
				EffectType: "modify",
			},
			expected: AdmissionResult{
				Path:     filepath.Join("lib", "user.rb"),
				Admitted: true,
			},
		},
		{
			name: "admits create event",
			event: Event{
				PathType:   "file",
				PathName:   filepath.Join(cwd, "spec", "user_spec.rb"),
				EffectType: "create",
			},
			expected: AdmissionResult{
				Path:     filepath.Join("spec", "user_spec.rb"),
				Admitted: true,
			},
		},
		{
			name: "rejects watcher event",
			event: Event{
				PathType:   "watcher",
				PathName:   filepath.Join(cwd, "lib", "user.rb"),
				EffectType: "modify",
			},
			expected: AdmissionResult{Admitted: false, Reason: "watcher"},
		},
		{
			name: "rejects ignored path",
			event: Event{
				PathType:   "file",
				PathName:   filepath.Join(cwd, "node_modules", "pkg", "index.js"),
				EffectType: "modify",
			},
			ignores: []string{"node_modules/**"},
			expected: AdmissionResult{
				Path:     filepath.Join("node_modules", "pkg", "index.js"),
				Admitted: false,
				Reason:   "ignored",
			},
		},
		{
			name: "rejects non-runnable effect",
			event: Event{
				PathType:   "file",
				PathName:   filepath.Join(cwd, "lib", "user.rb"),
				EffectType: "delete",
			},
			expected: AdmissionResult{
				Path:     filepath.Join("lib", "user.rb"),
				Admitted: false,
				Reason:   "effect",
			},
		},
		{
			name: "rejects relative path failure",
			event: Event{
				PathType:   "file",
				PathName:   filepath.Join(cwd, "lib", "user.rb"),
				EffectType: "modify",
			},
			expected: AdmissionResult{Admitted: false, Reason: "relative_path"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testCWD := cwd
			if tt.name == "rejects relative path failure" {
				testCWD = ""
			}

			result := AdmitEvent(tt.event, testCWD, tt.ignores)

			assert.Equal(t, tt.expected, result)
		})
	}
}
