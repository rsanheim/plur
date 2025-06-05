package main

import (
	"fmt"
	"os"

	"github.com/rsanheim/rux"
	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:    "rux-watch",
		Usage:   "Watch for file changes and run tests automatically",
		Version: main.GetVersionInfo(),
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:  "timeout",
				Usage: "Exit after specified seconds (default: run until Ctrl-C)",
			},
			&cli.IntFlag{
				Name:  "debounce",
				Usage: "Debounce delay in milliseconds (default: 100)",
				Value: 100,
			},
		},
		Action: func(ctx *cli.Context) error {
			return main.runWatch(ctx)
		},
	}

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}