package session

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/JuliusBrussee/cavekit/internal/exec"
	"github.com/JuliusBrussee/cavekit/internal/worktree"
)

type fakeSessionBackend struct {
	created  []string
	killed   []string
	commands []string
	exists   map[string]bool
}

func newFakeSessionBackend() *fakeSessionBackend {
	return &fakeSessionBackend{exists: make(map[string]bool)}
}

func (f *fakeSessionBackend) CreateSession(_ context.Context, name, _, _ string) error {
	f.created = append(f.created, name)
	f.exists[name] = true
	return nil
}

func (f *fakeSessionBackend) Exists(_ context.Context, name string) bool {
	return f.exists[name]
}

func (f *fakeSessionBackend) Kill(_ context.Context, name string) error {
	f.killed = append(f.killed, name)
	delete(f.exists, name)
	return nil
}

func (f *fakeSessionBackend) ListSessions(context.Context) ([]string, error)        { return nil, nil }
func (f *fakeSessionBackend) CapturePane(context.Context, string) (string, error)    { return "", nil }
func (f *fakeSessionBackend) CaptureScrollback(context.Context, string) (string, error) {
	return "", nil
}
func (f *fakeSessionBackend) SendKeys(context.Context, string, ...string) error { return nil }
func (f *fakeSessionBackend) SendText(context.Context, string, string) error     { return nil }
func (f *fakeSessionBackend) SendCommand(_ context.Context, name, cmd string) error {
	f.commands = append(f.commands, fmt.Sprintf("%s:%s", name, cmd))
	return nil
}

func newTestManager() (*Manager, *exec.MockExecutor, *fakeSessionBackend) {
	mock := exec.NewMockExecutor()
	mock.OnCommand("git", func(c exec.Call) (exec.Result, error) {
		args := strings.Join(c.Args, " ")
		if strings.Contains(args, "worktree list") {
			return exec.Result{Stdout: "", ExitCode: 0}, nil
		}
		if strings.Contains(args, "rev-parse --verify") {
			return exec.Result{ExitCode: 1}, nil
		}
		return exec.Result{ExitCode: 0}, nil
	})

	sessionBackend := newFakeSessionBackend()
	wtMgr := worktree.NewManager(mock)
	return NewManager(sessionBackend, wtMgr), mock, sessionBackend
}

func TestManager_Create(t *testing.T) {
	mgr, _, _ := newTestManager()
	inst := mgr.Create("auth", "/path/site.md", "auth", "claude")

	if inst.Title != "auth" {
		t.Errorf("Title = %q", inst.Title)
	}
	if inst.SitePath != "/path/site.md" {
		t.Errorf("SitePath = %q", inst.SitePath)
	}
	if inst.SessionName != "auth" {
		t.Errorf("SessionName = %q", inst.SessionName)
	}
	if inst.Status != StatusLoading {
		t.Errorf("Status = %v, want Loading", inst.Status)
	}
}

func TestManager_Start(t *testing.T) {
	mgr, _, sessionBackend := newTestManager()
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

	foundBuild := false
	for _, cmd := range sessionBackend.commands {
		if strings.Contains(cmd, "/ck:make") && strings.Contains(cmd, "auth") {
			foundBuild = true
		}
	}
	if !foundBuild {
		t.Error("should have sent /ck:make command to the session backend")
	}
}

func TestManager_Pause(t *testing.T) {
	mgr, _, _ := newTestManager()
	inst := mgr.Create("auth", "", "auth", "claude")
	inst.Status = StatusRunning

	mgr.Pause(inst)
	if inst.Status != StatusPaused {
		t.Errorf("Status = %v, want Paused", inst.Status)
	}
}

func TestManager_Kill(t *testing.T) {
	mgr, _, _ := newTestManager()
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
