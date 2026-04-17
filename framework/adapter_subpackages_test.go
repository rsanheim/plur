package framework

import (
	"testing"

	frameworkminitest "github.com/rsanheim/plur/framework/minitest"
	frameworkpassthrough "github.com/rsanheim/plur/framework/passthrough"
	frameworkrspec "github.com/rsanheim/plur/framework/rspec"
	"github.com/stretchr/testify/require"
)

func TestAdapterSubpackagesExportParsers(t *testing.T) {
	require.NotNil(t, frameworkminitest.NewOutputParser())
	require.NotNil(t, frameworkpassthrough.NewOutputParser())
	require.NotNil(t, frameworkrspec.NewOutputParser())
}
