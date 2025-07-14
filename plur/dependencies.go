package main

import (
	"fmt"
	"os"
	"os/exec"
)

// DependencyManager handles dependency installation
type DependencyManager struct{}

// NewDependencyManager creates a new dependency manager
func NewDependencyManager() *DependencyManager {
	return &DependencyManager{}
}

// InstallDependencies runs bundle install if needed
func (dm *DependencyManager) InstallDependencies() error {

	fmt.Println("Installing dependencies...")
	bundleCmd := exec.Command("bundle", "install")
	bundleCmd.Stdout = os.Stdout
	bundleCmd.Stderr = os.Stderr

	if err := bundleCmd.Run(); err != nil {
		return fmt.Errorf("error running bundle install: %v", err)
	}

	return nil
}
