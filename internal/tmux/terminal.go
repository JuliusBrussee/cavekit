package tmux

import (
	"os/exec"

	"golang.org/x/term"
)

// makeRaw sets the terminal to raw mode and returns the original state.
func makeRaw(fd uintptr) (*term.State, error) {
	return term.MakeRaw(int(fd))
}

// restoreTerminal restores the terminal to its original state.
func restoreTerminal(fd uintptr, state *term.State) {
	if state != nil {
		_ = term.Restore(int(fd), state)
	}
}

// buildCommand creates an exec.Cmd without using the executor (attach needs a raw process).
func buildCommand(name string, args ...string) *exec.Cmd {
	return exec.Command(name, args...)
}