package session

import (
	"context"
	"strings"
	"testing"

	"github.com/JuliusBrussee/cavekit/internal/exec"
	"github.com/JuliusBrussee/cavekit/internal/tmux"
	"github.com/JuliusBrussee/cavekit/internal/worktree"
)

func newTestManager() (*Manager, *exec.MockExecutor) {
	mock := exec.NewMockExecutor()
	mock.OnCommand("tmux", func(c exec.Call) (exec.Result, error) {
		return exec.Result{ExitCode: 0}, nil
	})
	mock.OnCommand("git", func(c exec.Call) (exec.Result, error) {
		args := strings.Join(c.Args, " ")
		if strings.Contains(args, "worktree list") {
			return exec.Result{Stdout: "", ExitCode: 0}, nil
		}
		if strings.Contains(args, "rev-parse --verify") {
			return exec.Result{ExitCode: 1}, nil // branch doesn't exist
		}
		return exec.Result{ExitCode: 0}, nil
	})

	tmuxMgr := tmux.NewManager(mock)
	wtMgr := worktree.NewManager(mock)
	return NewManager(tmuxMgr, wtMgr), mock
}

func TestManager_Create(t *testing.T) {
	mgr, _ := newTestManager()
	inst := mgr.Create("auth", "/path/site.md", "auth", "claude")

	if inst.Title != "auth" {
		t.Errorf("Title = %q", inst.Title)
	}
	if inst.SitePath != "/path/site.md" {
		t.Errorf("SitePath = %q", inst.SitePath)
	}
	if inst.TmuxSession != "bp_auth" {
		t.Errorf("TmuxSession = %q", inst.TmuxSession)
	}
	if inst.Status != StatusLoading {
		t.Errorf("Status = %v, want Loading", inst.Status)
	}
}

func TestManager_Start(t *testing.T) {
	mgr, mock := newTestManager()
	inst := mgr.Create("auth", "/path/site.md", "auth", "claude")

	err := mgr.Start(context.Background(), inst, "/code/project", "auth", 0)
	if err != nil {
		t.Fatalf("Start: %v", err)
	}

	if inst.Status != StatusRunning {
		t.Errorf("Status = %v, want Running", inst.Status)
	}
	if inst.WorktreePath == "" {
		t.Error("WorktreePath should be set")
	}

	// Verify tmux send-keys was called with the build command
	foundBuild := false
	for _, c := range mock.Calls {
		if c.Name == "tmux" {
			args := strings.Join(c.Args, " ")
			if strings.Contains(args, "/ck:make") && strings.Contains(args, "auth") {
				foundBuild = true
			}
		}
	}
	if !foundBuild {
		t.Error("should have sent /ck:make command to tmux")
	}
}

func TestManager_Pause(t *testing.T) {
	mgr, _ := newTestManager()
	inst := mgr.Create("auth", "", "auth", "claude")
	inst.Status = StatusRunning

	mgr.Pause(inst)
	if inst.Status != StatusPaused {
		t.Errorf("Status = %v, want Paused", inst.Status)
	}
}

func TestManager_Kill(t *testing.T) {
	mgr, _ := newTestManager()
	inst := mgr.Create("auth", "", "auth", "claude")
	inst.Status = StatusRunning

	err := mgr.Kill(context.Background(), inst, "/code/project", false)
	if err != nil {
		t.Fatalf("Kill: %v", err)
	}
	if inst.Status != StatusDone {
		t.Errorf("Status = %v, want Done", inst.Status)
	}
}
