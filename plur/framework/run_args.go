package framework

import (
	"strings"

	"github.com/rsanheim/plur/config"
	"github.com/rsanheim/plur/job"
)

// BuildRunArgs builds command arguments for run mode (plur spec).
// It ignores any {{target}} tokens in job.Cmd and appends targets at the end.
// extraArgs are inserted after framework defaults and before target files.
func BuildRunArgs(j job.Job, files []string, cfg *config.GlobalConfig, extraArgs []string) ([]string, error) {
	spec, err := Get(j.Framework)
	if err != nil {
		return nil, err
	}

	args := stripTargetTokens(j.Cmd)

	if spec.DefaultArgs != nil {
		defaultArgs, err := spec.DefaultArgs(cfg)
		if err != nil {
			return nil, err
		}
		args = append(args, defaultArgs...)
	}

	if len(extraArgs) > 0 {
		args = append(args, extraArgs...)
	}

	switch spec.TargetMode {
	case TargetModeRubyRequire:
		args = appendMinitestRequireArgs(args, files)
	default:
		args = append(args, files...)
	}

	return args, nil
}

func stripTargetTokens(args []string) []string {
	filtered := make([]string, 0, len(args))
	for _, arg := range args {
		if strings.Contains(arg, "{{target}}") {
			continue
		}
		filtered = append(filtered, arg)
	}
	return filtered
}

func appendMinitestRequireArgs(args []string, files []string) []string {
	if len(files) <= 1 {
		return append(args, files...)
	}

	requires := make([]string, 0, len(files))
	for _, file := range files {
		testFile := strings.TrimPrefix(file, "test/")
		testFile = strings.TrimSuffix(testFile, ".rb")
		requires = append(requires, testFile)
	}

	requireList := `"` + strings.Join(requires, `", "`) + `"`
	script := `[` + requireList + `].each { |f| require f }`
	return append(args, "-e", script)
}
