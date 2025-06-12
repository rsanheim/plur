package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/rsanheim/rux/tracing"
)

// DependencyManager handles dependency installation
type DependencyManager struct{}

// NewDependencyManager creates a new dependency manager
func NewDependencyManager() *DependencyManager {
	return &DependencyManager{}
}

// InstallDependencies runs bundle install if needed
func (dm *DependencyManager) InstallDependencies() error {
	defer tracing.StartRegion(context.Background(), "bundle_install")()

	fmt.Println("Installing dependencies...")
	bundleCmd := exec.Command("bundle", "install")
	bundleCmd.Stdout = os.Stdout
	bundleCmd.Stderr = os.Stderr

	if err := bundleCmd.Run(); err != nil {
		return fmt.Errorf("error running bundle install: %v", err)
	}

	return nil
}