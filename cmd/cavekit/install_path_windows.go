//go:build windows

package main

import (
	"strings"

	"golang.org/x/sys/windows/registry"
)

func addDirToUserPath(dir string) error {
	key, _, err := registry.CreateKey(registry.CURRENT_USER, `Environment`, registry.SET_VALUE|registry.QUERY_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()

	current, _, err := key.GetStringValue("Path")
	if err != nil && err != registry.ErrNotExist {
		return err
	}

	for _, entry := range strings.Split(current, ";") {
		if strings.EqualFold(strings.TrimSpace(entry), dir) {
			return nil
		}
	}

	updated := dir
	if strings.TrimSpace(current) != "" {
		updated = current + ";" + dir
	}
	return key.SetStringValue("Path", updated)
}

