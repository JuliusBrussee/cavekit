//go:build windows

package tmux

import (
	"context"
	"fmt"
)

const (
	// DetachKey is Ctrl+Q (ASCII 17).
	DetachKey = 17
)

type Attacher struct {
	mgr *Manager
}

func NewAttacher(mgr *Manager) *Attacher {
	return &Attacher{mgr: mgr}
}

func (a *Attacher) Attach(ctx context.Context, name string) (<-chan struct{}, error) {
	_ = ctx
	_ = name
	done := make(chan struct{})
	close(done)
	return done, fmt.Errorf("tmux attach is not supported on Windows")
}
