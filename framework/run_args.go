package framework

import (
	"strings"

	"github.com/rsanheim/plur/config"
	"github.com/rsanheim/plur/job"
)

// BuildRunArgs builds command arguments for run mode (plur spec).
// extraArgs are inserted after framework defaults and before target files.
func BuildRunArgs(j job.Job, files []string, cfg *config.GlobalConfig, extraArgs []string) ([]string, error) {
	fw, err := Get(j.FrameworkName)
	if err != nil {
		return nil, err
	}

	args := append([]string{}, j.Cmd...)

	if fw.DefaultArgs != nil {
		defaultArgs, err := fw.DefaultArgs(cfg)
		if err != nil {
			return nil, err
		}
		args = append(args, defaultArgs...)
	}

	switch fw.TargetMode {
	case TargetModeRubyRequire:
		args = appendMinitestRequireArgs(args, files)
		if len(extraArgs) > 0 {
			args = append(args, extraArgs...)
		}
	default:
		if len(extraArgs) > 0 {
			args = append(args, extraArgs...)
		}
		args = append(args, files...)
	}

	return args, nil
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
