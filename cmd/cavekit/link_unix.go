//go:build !windows

package main

import "os"

func createDirLink(target, linkPath string) error {
	if err := removeIfExists(linkPath); err != nil {
		return err
	}
	return os.Symlink(target, linkPath)
}
