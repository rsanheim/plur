package main

import (
	"fmt"
	"github.com/rsanheim/rux/types"
)

// NewTestOutputParser creates the appropriate parser based on the test framework
func NewTestOutputParser(framework TestFramework) (types.TestOutputParser, error) {
	switch framework {
	case FrameworkRSpec:
		return NewRSpecParser(), nil
	case FrameworkMinitest:
		return NewMinitestParser(), nil
	default:
		return nil, fmt.Errorf("unsupported test framework: %s", framework)
	}
}
