package cmd

import (
	"fmt"
	"os"

	"github.com/rsanheim/plur/internal/buildinfo"
)

type VersionCmd struct{}

func (v *VersionCmd) Run() error {
	fmt.Printf("plur version=%s", buildinfo.GetVersionInfo())
	os.Exit(0)
	return nil
}
