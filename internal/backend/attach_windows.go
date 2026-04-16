//go:build windows

package backend

import (
	"fmt"
	osexec "os/exec"
)

func AttachCommand(name string) (*osexec.Cmd, error) {
	return nil, fmt.Errorf("full-screen attach is not supported on Windows yet")
}

func DefaultTerminalProgram() string {
	return ""
}

func UnsupportedPlatformMessage() string {
	return "full-screen attach is not supported on Windows yet"
}
