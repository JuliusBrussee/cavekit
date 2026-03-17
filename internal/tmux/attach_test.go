package tmux

import (
	"testing"

	"github.com/julb/blueprint-monitor/internal/exec"
)

func TestNewAttacher(t *testing.T) {
	mock := exec.NewMockExecutor()
	mgr := NewManager(mock)
	attacher := NewAttacher(mgr)
	if attacher == nil {
		t.Error("NewAttacher should not return nil")
	}
	if attacher.mgr != mgr {
		t.Error("attacher should reference the manager")
	}
}

func TestDetachKey(t *testing.T) {
	// Ctrl+Q is ASCII 17
	if DetachKey != 17 {
		t.Errorf("DetachKey should be 17 (Ctrl+Q), got %d", DetachKey)
	}
}
