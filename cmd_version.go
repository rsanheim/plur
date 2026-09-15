package main

import (
	"fmt"

	"github.com/rsanheim/plur/internal/buildinfo"
)

type VersionCmd struct{}

func (v *VersionCmd) Run() {
	fmt.Printf("plur version=%s", buildinfo.GetVersionInfo())
}
