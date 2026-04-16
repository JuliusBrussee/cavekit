//go:build !windows

package main

func addDirToUserPath(string) error {
	return nil
}
