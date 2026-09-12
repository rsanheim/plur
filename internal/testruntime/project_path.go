package testruntime

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func computeRuntimeFilePath(runtimeDir string) (string, string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", "", err
	}
	cwd = canonicalPath(cwd)
	identity := "cwd\x00" + cwd

	// Share across worktrees, keeping project subdirectories separate.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "--no-optional-locks", "-C", cwd, "rev-parse", "--path-format=absolute", "--git-common-dir", "--show-toplevel")
	cmd.WaitDelay = 100 * time.Millisecond
	output, err := cmd.Output()
	if err == nil {
		paths := strings.Split(strings.TrimSuffix(string(output), "\n"), "\n")
		if len(paths) == 2 && filepath.IsAbs(paths[0]) && filepath.IsAbs(paths[1]) {
			root := canonicalPath(paths[1])
			rel, err := filepath.Rel(root, cwd)
			if err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
				identity = "git\x00" + canonicalPath(paths[0]) + "\x00" + filepath.ToSlash(rel)
			}
		}
	}

	hash := sha256.Sum256([]byte(identity))
	return filepath.Join(runtimeDir, hex.EncodeToString(hash[:8])+".json"), cwd, nil
}

func canonicalPath(path string) string {
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		return resolved
	}
	return filepath.Clean(path)
}
