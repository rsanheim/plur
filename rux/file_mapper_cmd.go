package main

import (
	"fmt"
	"strings"

	"github.com/rsanheim/rux/watch"
	"github.com/urfave/cli/v2"
)

// runFileMapper is a development command to test file mapping
func runFileMapper(ctx *cli.Context) error {
	// Get files from command line arguments
	files := ctx.Args().Slice()

	if len(files) == 0 {
		return fmt.Errorf("please provide one or more files to map")
	}

	// Create file mapper
	mapper := watch.NewFileMapper()

	// Process each file
	for _, file := range files {
		// Normalize the file path
		file = strings.TrimSpace(file)

		// Get the mapped specs
		specs := mapper.MapFileToSpecs(file)

		// Output the results
		if len(specs) == 0 {
			fmt.Printf("%s -> (no mapping)\n", file)
		} else {
			fmt.Printf("%s -> %s\n", file, strings.Join(specs, ", "))
		}
	}

	return nil
}
