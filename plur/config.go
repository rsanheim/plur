package main

import (
	"github.com/rsanheim/plur/config"
)

// ParseFrameworkType converts a string type to TestFramework enum
func ParseFrameworkType(frameworkType string) config.TestFramework {
	if frameworkType == "" {
		return config.DetectTestFramework()
	}
	switch frameworkType {
	case "rspec":
		return config.FrameworkRSpec
	case "minitest":
		return config.FrameworkMinitest
	default:
		// Default to RSpec for backward compatibility
		return config.FrameworkRSpec
	}
}
