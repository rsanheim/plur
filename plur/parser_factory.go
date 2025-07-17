package main

import (
	"fmt"

	"github.com/rsanheim/plur/minitest"
	"github.com/rsanheim/plur/rspec"
	"github.com/rsanheim/plur/types"
)

// NewTestOutputParser creates the appropriate parser based on the test framework
func NewTestOutputParser(framework TestFramework) (types.TestOutputParser, error) {
	switch framework {
	case FrameworkRSpec:
		return &rspec.OutputParser{}, nil
	case FrameworkMinitest:
		return &minitest.OutputParser{}, nil
	default:
		return nil, fmt.Errorf("unsupported test framework: %s", framework)
	}
}
