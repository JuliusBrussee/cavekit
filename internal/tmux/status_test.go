package tmux

import (
	"context"
	"testing"

	"github.com/julb/blueprint-monitor/internal/exec"
)

func TestStatusDetector_Active(t *testing.T) {
	callCount := 0
	mock := exec.NewMockExecutor()
	mock.OnCommand("tmux", func(c exec.Call) (exec.Result, error) {
		callCount++
		// Return different content each time
		return exec.Result{
			Stdout:   "Working on task " + string(rune('0'+callCount)) + "\n",
			ExitCode: 0,
		}, nil
	})

	mgr := NewManager(mock)
	det := NewStatusDetector(mgr)

	// First call — active (no previous hash)
	status, err := det.Detect(context.Background(), "test")
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if status != PaneActive {
		t.Errorf("first call should be Active, got %v", status)
	}

	// Second call — still active (content changed)
	status, err = det.Detect(context.Background(), "test")
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if status != PaneActive {
		t.Errorf("second call should be Active, got %v", status)
	}
}

func TestStatusDetector_Idle(t *testing.T) {
	mock := exec.NewMockExecutor()
	mock.OnCommand("tmux", func(c exec.Call) (exec.Result, error) {
		return exec.Result{Stdout: "$ \n", ExitCode: 0}, nil
	})

	mgr := NewManager(mock)
	det := NewStatusDetector(mgr)

	// First call — active (no previous)
	det.Detect(context.Background(), "test")

	// Second call — idle (same content)
	status, _ := det.Detect(context.Background(), "test")
	if status != PaneIdle {
		t.Errorf("should be Idle when content unchanged, got %v", status)
	}
}

func TestStatusDetector_Prompt(t *testing.T) {
	mock := exec.NewMockExecutor()
	mock.OnCommand("tmux", func(c exec.Call) (exec.Result, error) {
		return exec.Result{
			Stdout:   "Claude wants to edit file.go\nAllow once\nNo, and tell Claude what to do differently\n",
			ExitCode: 0,
		}, nil
	})

	mgr := NewManager(mock)
	det := NewStatusDetector(mgr)

	status, err := det.Detect(context.Background(), "test")
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if status != PanePrompt {
		t.Errorf("should detect Prompt, got %v", status)
	}
}

func TestStatusDetector_Trust(t *testing.T) {
	mock := exec.NewMockExecutor()
	mock.OnCommand("tmux", func(c exec.Call) (exec.Result, error) {
		return exec.Result{
			Stdout:   "Do you trust the files in this folder?\nTrust this project\n",
			ExitCode: 0,
		}, nil
	})

	mgr := NewManager(mock)
	det := NewStatusDetector(mgr)

	status, _ := det.Detect(context.Background(), "test")
	if status != PaneTrust {
		t.Errorf("should detect Trust, got %v", status)
	}
}

func TestPaneStatus_String(t *testing.T) {
	if PaneActive.String() != "active" {
		t.Errorf("PaneActive.String() = %q", PaneActive.String())
	}
	if PanePrompt.String() != "prompt" {
		t.Errorf("PanePrompt.String() = %q", PanePrompt.String())
	}
}
