package framework

import (
	"fmt"
	"strings"

	"github.com/rsanheim/plur/config"
)

// BuildRunArgs builds command arguments for run mode (plur spec).
// extraArgs follow the framework defaults: before the target files for
// frameworks that take files as arguments, and after the -e script that
// embeds them for TargetModeRubyRequire.
func (j Job) BuildRunArgs(files []string, cfg *config.GlobalConfig, extraArgs []string) ([]string, error) {
	fw := j.Framework
	if fw.Name == "" {
		return nil, fmt.Errorf("job %q has no resolved framework", j.Name)
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
			// ruby keeps parsing its own options after "-e script"; "--"
			// ends that so the extra args reach ARGV for minitest.
			args = append(args, "--")
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

// appendMinitestRequireArgs loads the target files via an -e script rather
// than script arguments, so the script can end with the plugin epilogue:
// minitest 5.x discovers plur's plugin automatically inside Minitest.run,
// but minitest 6 made plugin loading opt-in, so the worker asks for it via
// the documented Minitest.load - which loads only the named plugin. Using
// Minitest.load rather than Minitest.load_plugins matters: load_plugins
// re-enables the autodiscovery of every installed minitest/*_plugin.rb that
// minitest 6 deliberately retired, activating plugins the project left off
// by upgrading. Minitest.load is 6-only (added with the opt-in redesign),
// which makes respond_to?(:load) the capability gate; on 5.x minitest still
// discovers and loads plur itself. MT_NO_PLUGINS is minitest's own opt-out
// convention and doubles as plur's escape hatch.
const minitestPluginEpilogue = `Minitest.load "plur" if defined?(Minitest) && Minitest.respond_to?(:load) && !ENV["MT_NO_PLUGINS"]`

func appendMinitestRequireArgs(args []string, files []string) []string {
	requires := make([]string, 0, len(files))
	for _, file := range files {
		requires = append(requires, `"`+file+`"`)
	}

	script := `[` + strings.Join(requires, `, `) + `].each { |f| require File.expand_path(f) }; ` + minitestPluginEpilogue
	return append(args, "-e", script)
}
