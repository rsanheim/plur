package runtime

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuiltinDefaultsLoad(t *testing.T) {
	assert.NotEmpty(t, builtinDefaults.Defaults.Jobs)
	assert.Contains(t, builtinDefaults.Defaults.Jobs, "rspec")
	assert.Contains(t, builtinDefaults.Defaults.Jobs, "minitest")
	assert.Contains(t, builtinDefaults.Defaults.Jobs, "go-test")
}
