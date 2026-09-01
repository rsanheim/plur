package cmd

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestVersionCmdRunReturns(t *testing.T) {
	require.NoError(t, (&VersionCmd{}).Run())
}
