package framework

import (
	"fmt"
	"strings"

	"github.com/rsanheim/plur/config"
	"github.com/rsanheim/plur/internal/framework/minitest"
	"github.com/rsanheim/plur/internal/framework/passthrough"
	"github.com/rsanheim/plur/internal/framework/rspec"
	"github.com/rsanheim/plur/types"
)

type TargetMode int

const (
	TargetModeAppend TargetMode = iota
	TargetModeRubyRequire
)

type Framework struct {
	Name           string
	Parser         func() types.TestOutputParser
	DefaultArgs    func(*config.GlobalConfig) ([]string, error)
	DetectPatterns []string
	TargetMode     TargetMode
}

var registry = map[string]Framework{
	"rspec": {
		Name:           "rspec",
		Parser:         rspec.NewOutputParser,
		DefaultArgs:    rspecDefaultArgs,
		DetectPatterns: []string{"**/*_spec.rb"},
		TargetMode:     TargetModeAppend,
	},
	"minitest": {
		Name:           "minitest",
		Parser:         minitest.NewOutputParser,
		DetectPatterns: []string{"**/*_test.rb"},
		TargetMode:     TargetModeRubyRequire,
	},
	"passthrough": {
		Name:       "passthrough",
		Parser:     passthrough.NewOutputParser,
		TargetMode: TargetModeAppend,
	},
	"go-test": {
		Name:           "go-test",
		Parser:         passthrough.NewOutputParser,
		DetectPatterns: []string{"**/*_test.go"},
		TargetMode:     TargetModeAppend,
	},
}

func Get(name string) (Framework, error) {
	normalized := Normalize(name)
	if normalized == "" {
		return Framework{}, fmt.Errorf("framework is required")
	}
	if fw, ok := registry[normalized]; ok {
		return fw, nil
	}
	return Framework{}, fmt.Errorf("unknown framework %q", name)
}

func IsKnown(name string) bool {
	normalized := Normalize(name)
	_, ok := registry[normalized]
	return ok
}

func Normalize(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func DetectPatterns(name string) []string {
	fw, ok := registry[Normalize(name)]
	if !ok {
		return nil
	}
	return fw.DetectPatterns
}

func rspecDefaultArgs(cfg *config.GlobalConfig) ([]string, error) {
	if cfg == nil || cfg.ConfigPaths == nil {
		return nil, fmt.Errorf("config paths are required for rspec formatter")
	}

	args := []string{}
	formatterPath, err := rspec.GetFormatterPath(cfg.ConfigPaths.FormatterDir)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize RSpec formatter: %w", err)
	}
	if formatterPath != "" {
		args = append(args, "-r", formatterPath, "--format", "Plur::JsonRowsFormatter")
	}

	if !cfg.ColorOutput {
		args = append(args, "--no-color")
	} else {
		args = append(args, "--force-color")
	}

	return args, nil
}
