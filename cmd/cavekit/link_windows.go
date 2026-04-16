//go:build windows

package main

import (
	"fmt"
	"os"
	"os/exec"
)

func createDirLink(target, linkPath string) error {
	if err := removeIfExists(linkPath); err != nil {
		return err
	}

	if err := os.Symlink(target, linkPath); err == nil {
		return nil
	}

	cmd := exec.Command("cmd", "/c", "mklink", "/J", linkPath, target)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("create junction: %w (%s)", err, string(output))
	}
	return nil
}
