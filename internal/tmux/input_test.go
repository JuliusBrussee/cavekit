package tmux

import (
	"context"
	"strings"
	"testing"

	"github.com/JuliusBrussee/cavekit/internal/exec"
)

func TestManager_SendEnter(t *testing.T) {
	mock := exec.NewMockExecutor()
	mock.OnCommand("tmux", func(c exec.Call) (exec.Result, error) {
		return exec.Result{ExitCode: 0}, nil
	})

	mgr := NewManager(mock)
	err := mgr.SendEnter(context.Background(), "test")
	if err != nil {
		t.Fatalf("SendEnter: %v", err)
	}

	if len(mock.Calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(mock.Calls))
	}
	args := strings.Join(mock.Calls[0].Args, " ")
	if !strings.Contains(args, "send-keys") || !strings.Contains(args, "Enter") {
		t.Errorf("expected send-keys with Enter, got: %s", args)
	}
}

func TestManager_SendKeys(t *testing.T) {
	mock := exec.NewMockExecutor()
	mock.OnCommand("tmux", func(c exec.Call) (exec.Result, error) {
		return exec.Result{ExitCode: 0}, nil
	})

	mgr := NewManager(mock)
	err := mgr.SendKeys(context.Background(), "test", "/ck:make", "Enter")
	if err != nil {
		t.Fatalf("SendKeys: %v", err)
	}

	args := mock.Calls[0].Args
	// Should include: send-keys -t bp_test /ck:make Enter
	if args[0] != "send-keys" {
		t.Errorf("first arg should be send-keys, got %s", args[0])
	}
}

func TestManager_SendText_MultiLine(t *testing.T) {
	mock := exec.NewMockExecutor()
	mock.OnCommand("tmux", func(c exec.Call) (exec.Result, error) {
		return exec.Result{ExitCode: 0}, nil
	})

	mgr := NewManager(mock)
	err := mgr.SendText(context.Background(), "test", "line1\nline2\nline3")
	if err != nil {
		t.Fatalf("SendText: %v", err)
	}

	// Should have sent 3 lines, each with Enter
	if len(mock.Calls) != 3 {
		t.Errorf("expected 3 calls, got %d", len(mock.Calls))
	}
}

func TestManager_SendCommand(t *testing.T) {
	mock := exec.NewMockExecutor()
	mock.OnCommand("tmux", func(c exec.Call) (exec.Result, error) {
		return exec.Result{ExitCode: 0}, nil
	})

	mgr := NewManager(mock)
	err := mgr.SendCommand(context.Background(), "test", "/ck:make --filter auth")
	if err != nil {
		t.Fatalf("SendCommand: %v", err)
	}

	args := mock.Calls[0].Args
	found := false
	for _, a := range args {
		if a == "/ck:make --filter auth" {
			found = true
		}
	}
	if !found {
		t.Errorf("command not found in args: %v", args)
	}
}
