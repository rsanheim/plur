package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// DependencyManager handles dependency installation
type DependencyManager struct {
	dryRun bool
}

// NewDependencyManager creates a new dependency manager
func NewDependencyManager(dryRun bool) *DependencyManager {
	return &DependencyManager{dryRun: dryRun}
}

// InstallDependencies runs bundle install if needed
func (dm *DependencyManager) InstallDependencies() error {
	cmd := exec.Command("bundle", "install")

	if dm.dryRun {
		fmt.Fprintf(os.Stderr, "[dry-run] %s\n", strings.Join(cmd.Args, " "))
		return nil
	}

	fmt.Println("Installing dependencies...")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("error running bundle install: %v", err)
	}

	return nil
}
