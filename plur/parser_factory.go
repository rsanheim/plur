package main

import (
	"fmt"

	"github.com/rsanheim/plur/config"
	"github.com/rsanheim/plur/minitest"
	"github.com/rsanheim/plur/rspec"
	"github.com/rsanheim/plur/types"
)

// NewTestOutputParser creates the appropriate parser based on the test framework
func NewTestOutputParser(framework config.TestFramework) (types.TestOutputParser, error) {
	switch framework {
	case config.FrameworkRSpec:
		return rspec.NewOutputParser(), nil
	case config.FrameworkMinitest:
		return minitest.NewOutputParser(), nil
	default:
		return nil, fmt.Errorf("unsupported test framework: %s", framework)
	}
}
