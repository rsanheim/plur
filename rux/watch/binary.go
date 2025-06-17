package watch

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// LogDebug logs a debug message with key-value pairs
func LogDebug(msg string, args ...interface{}) {
	// For now, just print to stdout
	fmt.Printf("[DEBUG] %s", msg)
	for i := 0; i < len(args); i += 2 {
		if i+1 < len(args) {
			fmt.Printf(" %v=%v", args[i], args[i+1])
		}
	}
	fmt.Println()
}

// GetWatcherBinaryPath returns the path to the installed watcher binary
// It checks if the binary exists and returns an error with helpful message if not
func GetWatcherBinaryPath(binDir string) (string, error) {
	binaryPath, err := getBinaryPath(binDir)
	if err != nil {
		return "", err
	}

	if _, err := os.Stat(binaryPath); err == nil {
		return binaryPath, nil
	}

	// Binary not found, suggest running 'rux watch install'
	return "", fmt.Errorf("watcher binary not found at %s. Please run 'rux watch install' to install it", binaryPath)
}

// InstallBinary extracts the embedded watcher binary and installs it to RUX_HOME/bin
func InstallBinary(watcherBinaries embed.FS, binDir, ruxHome string, force bool) error {
	binaryPath, err := getBinaryPath(binDir)
	if err != nil {
		return fmt.Errorf("failed to determine binary path: %v", err)
	}
	if !force {
		if _, err := os.Stat(binaryPath); err == nil {
			LogDebug("watcher binary already installed at", "path", binaryPath)
			return nil
		}
	}

	embeddedPath, err := getEmbeddedBinaryPath()
	if err != nil {
		return err
	}

	data, err := watcherBinaries.ReadFile(embeddedPath)
	if err != nil {
		return fmt.Errorf("watcher binary not embedded for this platform: %v", err)
	}

	if err := os.WriteFile(binaryPath, data, 0755); err != nil {
		return fmt.Errorf("failed to write watcher binary: %v", err)
	}

	// Print success message
	relPath, _ := filepath.Rel(ruxHome, binaryPath)
	fmt.Printf("installed watcher binary path=%s\n", relPath)

	return nil
}

// getBinaryPath determines the platform-specific watcher binary path
func getBinaryPath(binDir string) (string, error) {
	binaryName, err := getPlatformBinaryName()
	if err != nil {
		return "", err
	}
	return filepath.Join(binDir, binaryName), nil
}

// getPlatformBinaryName returns the platform-specific binary name
func getPlatformBinaryName() (string, error) {
	switch runtime.GOOS {
	case "darwin":
		switch runtime.GOARCH {
		case "arm64", "aarch64":
			return "watcher-aarch64-apple-darwin", nil
		default:
			return "", fmt.Errorf("unsupported macOS architecture: %s", runtime.GOARCH)
		}
	case "linux":
		switch runtime.GOARCH {
		case "arm64", "aarch64":
			return "watcher-aarch64-unknown-linux-gnu", nil
		case "amd64":
			return "watcher-x86_64-unknown-linux-gnu", nil
		default:
			return "", fmt.Errorf("unsupported Linux architecture: %s", runtime.GOARCH)
		}
	default:
		return "", fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}
}

// getEmbeddedBinaryPath returns the path within the embedded filesystem for the current platform
func getEmbeddedBinaryPath() (string, error) {
	binaryName, err := getPlatformBinaryName()
	if err != nil {
		return "", err
	}
	return filepath.Join("vendor/watcher", binaryName), nil
}
