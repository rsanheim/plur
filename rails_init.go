package main

import "github.com/rsanheim/plur/internal/railsinit"

type RailsInitCmd struct{}

func (r *RailsInitCmd) Run(parent *PlurCLI) error {
	return railsinit.Run(parent.globalConfig)
}
