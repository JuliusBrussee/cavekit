package worktree

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/JuliusBrussee/cavekit/internal/exec"
)

func testPath(parts ...string) string {
	return filepath.Join(parts...)
}

func TestWorktreePath(t *testing.T) {
	tests := []struct {
		root, site, want string
	}{
		{testPath("home", "user", "myproject"), "auth", testPath("home", "user", "myproject-cavekit-auth")},
		{testPath("code", "cavekit"), "build", testPath("code", "cavekit-cavekit-build")},
	}
	for _, tt := range tests {
		got := WorktreePath(tt.root, tt.site)
		if got != tt.want {
			t.Errorf("WorktreePath(%q, %q) = %q, want %q", tt.root, tt.site, got, tt.want)
		}
	}
}

func TestBranchName(t *testing.T) {
	if got := BranchName("auth"); got != "cavekit/auth" {
		t.Errorf("BranchName(auth) = %q, want %q", got, "cavekit/auth")
	}
}

func TestManager_Create_NewBranch(t *testing.T) {
	mock := exec.NewMockExecutor()
	callIdx := 0
	mock.OnCommand("git", func(c exec.Call) (exec.Result, error) {
		callIdx++
		args := strings.Join(c.Args, " ")

		// worktree list (for Exists check) — worktree doesn't exist
		if strings.Contains(args, "worktree list") {
			return exec.Result{Stdout: "worktree /other\n", ExitCode: 0}, nil
		}
		// rev-parse --verify — branch doesn't exist
		if strings.Contains(args, "rev-parse --verify") {
			return exec.Result{ExitCode: 1, Stderr: "unknown revision"}, nil
		}
		// branch create
		if args == "branch cavekit/auth" {
			return exec.Result{ExitCode: 0}, nil
		}
		// worktree add
		if strings.Contains(args, "worktree add") {
			return exec.Result{ExitCode: 0}, nil
		}
		return exec.Result{ExitCode: 0}, nil
	})

	mgr := NewManager(mock)
	projectRoot := testPath("code", "myproject")
	path, err := mgr.Create(context.Background(), projectRoot, "auth")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if path != testPath("code", "myproject-cavekit-auth") {
		t.Errorf("path = %q, unexpected", path)
	}
}

func TestManager_Create_ExistingWorktree(t *testing.T) {
	projectRoot := testPath("code", "myproject")
	existing := testPath("code", "myproject-cavekit-auth")
	mock := exec.NewMockExecutor()
	mock.OnCommand("git", func(c exec.Call) (exec.Result, error) {
		if strings.Contains(strings.Join(c.Args, " "), "worktree list") {
			return exec.Result{
				Stdout:   "worktree " + existing + "\nbranch refs/heads/cavekit/auth\n",
				ExitCode: 0,
			}, nil
		}
		return exec.Result{ExitCode: 0}, nil
	})

	mgr := NewManager(mock)
	path, err := mgr.Create(context.Background(), projectRoot, "auth")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if path != existing {
		t.Errorf("path = %q, unexpected", path)
	}
	// Should have only called worktree list (no create commands)
	for _, c := range mock.Calls {
		if strings.Contains(strings.Join(c.Args, " "), "worktree add") {
			t.Error("should not call worktree add for existing worktree")
		}
	}
}

func TestManager_Remove(t *testing.T) {
	mock := exec.NewMockExecutor()
	mock.OnCommand("git", func(c exec.Call) (exec.Result, error) {
		return exec.Result{ExitCode: 0}, nil
	})

	mgr := NewManager(mock)
	err := mgr.Remove(context.Background(), testPath("code", "myproject"), "auth")
	if err != nil {
		t.Fatalf("Remove: %v", err)
	}

	// Should have: prune, remove, branch -D
	cmds := make([]string, 0)
	for _, c := range mock.Calls {
		cmds = append(cmds, strings.Join(c.Args, " "))
	}
	if len(mock.Calls) != 3 {
		t.Fatalf("expected 3 calls, got %d: %v", len(mock.Calls), cmds)
	}
}

func TestManager_ProjectRoot(t *testing.T) {
	mock := exec.NewMockExecutor()
	mock.OnCommand("git", func(c exec.Call) (exec.Result, error) {
		return exec.Result{Stdout: testPath("code", "myproject") + "\n", ExitCode: 0}, nil
	})

	mgr := NewManager(mock)
	root, err := mgr.ProjectRoot(context.Background(), testPath("code", "myproject", "src"))
	if err != nil {
		t.Fatalf("ProjectRoot: %v", err)
	}
	if root != testPath("code", "myproject") {
		t.Errorf("root = %q, want %q", root, testPath("code", "myproject"))
	}
}
