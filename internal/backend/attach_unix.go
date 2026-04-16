//go:build !windows

package backend

import (
	"fmt"
	osexec "os/exec"

	"github.com/JuliusBrussee/cavekit/internal/tmux"
)

func AttachCommand(name string) (*osexec.Cmd, error) {
	return osexec.Command("tmux", "attach-session", "-t", tmux.SessionName(name)), nil
}

func DefaultTerminalProgram() string {
	return "zsh"
}

func UnsupportedPlatformMessage() string {
	return fmt.Sprintf("attach unsupported on %s", "this platform")
}
