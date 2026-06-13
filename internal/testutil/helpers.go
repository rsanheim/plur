// Package testutil provides shared helpers for tests across the module.
package testutil

import "os"

// IsCI reports whether the suite is running in a CI environment.
func IsCI() bool {
	return os.Getenv("CI") != ""
}
