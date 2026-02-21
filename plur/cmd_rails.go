package main

type RailsInitCmd struct{}

func (r *RailsInitCmd) Run(parent *PlurCLI) error {
	return runRailsInit(parent.globalConfig)
}
