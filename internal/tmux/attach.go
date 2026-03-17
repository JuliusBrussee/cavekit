package tmux

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/creack/pty"
)

const (
	// DetachKey is Ctrl+Q (ASCII 17).
	DetachKey = 17
)

// Attacher handles full-screen attach/detach to tmux sessions via PTY.
type Attacher struct {
	mgr *Manager
}

// NewAttacher creates an attacher for the given tmux manager.
func NewAttacher(mgr *Manager) *Attacher {
	return &Attacher{mgr: mgr}
}

// Attach takes over the terminal to interact with a tmux session.
// Returns a channel that closes when detach completes.
// Detach occurs when user presses Ctrl+Q.
func (a *Attacher) Attach(ctx context.Context, name string) (<-chan struct{}, error) {
	sessionName := SanitizeName(name)
	done := make(chan struct{})

	// Start tmux attach-session in a PTY
	cmd := buildCommand("tmux", "attach-session", "-t", sessionName)

	ptmx, err := pty.Start(cmd)
	if err != nil {
		close(done)
		return done, fmt.Errorf("pty.Start: %w", err)
	}

	// Forward window size changes
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGWINCH)
	go func() {
		for range sigCh {
			if err := pty.InheritSize(os.Stdin, ptmx); err != nil {
				// Best effort
			}
		}
	}()
	// Set initial size
	pty.InheritSize(os.Stdin, ptmx)

	// Save and restore terminal state
	oldState, err := makeRaw(os.Stdin.Fd())
	if err != nil {
		ptmx.Close()
		cmd.Process.Kill()
		close(done)
		return done, fmt.Errorf("terminal raw mode: %w", err)
	}

	go func() {
		defer func() {
			restoreTerminal(os.Stdin.Fd(), oldState)
			signal.Stop(sigCh)
			close(sigCh)
			ptmx.Close()
			cmd.Wait()
			close(done)
		}()

		// Copy PTY output to stdout
		go io.Copy(os.Stdout, ptmx)

		// Read stdin, looking for Ctrl+Q to detach
		buf := make([]byte, 1024)
		for {
			n, err := os.Stdin.Read(buf)
			if err != nil {
				return
			}
			for i := 0; i < n; i++ {
				if buf[i] == DetachKey {
					// Detach: send tmux detach command
					a.mgr.exec.Run(ctx, "tmux", "detach-client", "-s", sessionName)
					return
				}
			}
			ptmx.Write(buf[:n])
		}
	}()

	return done, nil
}
