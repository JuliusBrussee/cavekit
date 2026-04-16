//go:build windows

package tmux

import "os/exec"

func buildCommand(name string, args ...string) *exec.Cmd {
	return exec.Command(name, args...)
}
