package cmd

import (
	"fmt"

	"github.com/rsanheim/plur/internal/buildinfo"
)

type VersionCmd struct{}

func (v *VersionCmd) Run() error {
	fmt.Printf("plur version=%s", buildinfo.GetVersionInfo())
	return nil
}
