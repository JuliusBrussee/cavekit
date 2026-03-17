package tmux

import (
	"os/exec"

	"golang.org/x/sys/unix"
)

// makeRaw sets the terminal to raw mode and returns the original state.
func makeRaw(fd uintptr) (*unix.Termios, error) {
	termios, err := unix.IoctlGetTermios(int(fd), unix.TIOCGETA)
	if err != nil {
		return nil, err
	}

	oldState := *termios

	// Set raw mode
	termios.Iflag &^= unix.IGNBRK | unix.BRKINT | unix.PARMRK | unix.ISTRIP | unix.INLCR | unix.IGNCR | unix.ICRNL | unix.IXON
	termios.Oflag &^= unix.OPOST
	termios.Lflag &^= unix.ECHO | unix.ECHONL | unix.ICANON | unix.ISIG | unix.IEXTEN
	termios.Cflag &^= unix.CSIZE | unix.PARENB
	termios.Cflag |= unix.CS8
	termios.Cc[unix.VMIN] = 1
	termios.Cc[unix.VTIME] = 0

	if err := unix.IoctlSetTermios(int(fd), unix.TIOCSETA, termios); err != nil {
		return nil, err
	}

	return &oldState, nil
}

// restoreTerminal restores the terminal to its original state.
func restoreTerminal(fd uintptr, state *unix.Termios) {
	if state != nil {
		unix.IoctlSetTermios(int(fd), unix.TIOCSETA, state)
	}
}

// buildCommand creates an exec.Cmd without using the executor (attach needs a raw process).
func buildCommand(name string, args ...string) *exec.Cmd {
	return exec.Command(name, args...)
}
