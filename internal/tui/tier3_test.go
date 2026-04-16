package tui

import (
	"context"
	"strings"
	"testing"

	"github.com/JuliusBrussee/cavekit/internal/backend"
	"github.com/JuliusBrussee/cavekit/internal/exec"
	"github.com/JuliusBrussee/cavekit/internal/site"
	"github.com/JuliusBrussee/cavekit/internal/worktree"
)

type fakeSessionBackend struct {
	panes       map[string]string
	scrollbacks map[string]string
	created     []string
}

func newFakeSessionBackend() *fakeSessionBackend {
	return &fakeSessionBackend{
		panes:       make(map[string]string),
		scrollbacks: make(map[string]string),
	}
}

func (f *fakeSessionBackend) CreateSession(_ context.Context, name, _, _ string) error {
	f.created = append(f.created, name)
	return nil
}

func (f *fakeSessionBackend) Exists(context.Context, string) bool                   { return true }
func (f *fakeSessionBackend) Kill(context.Context, string) error                    { return nil }
func (f *fakeSessionBackend) ListSessions(context.Context) ([]string, error)        { return nil, nil }
func (f *fakeSessionBackend) SendKeys(context.Context, string, ...string) error     { return nil }
func (f *fakeSessionBackend) SendText(context.Context, string, string) error        { return nil }
func (f *fakeSessionBackend) SendCommand(context.Context, string, string) error     { return nil }
func (f *fakeSessionBackend) CapturePane(_ context.Context, name string) (string, error) {
	return f.panes[name], nil
}
func (f *fakeSessionBackend) CaptureScrollback(_ context.Context, name string) (string, error) {
	return f.scrollbacks[name], nil
}

func TestPreviewTab_Content(t *testing.T) {
	backend := newFakeSessionBackend()
	backend.panes["test"] = "$ claude\nWorking...\n"
	preview := NewPreviewTab(backend)

	preview.Capture(nil, "test")
	content := preview.Content()
	if !strings.Contains(content, "Working") {
		t.Errorf("preview should contain captured content: %q", content)
	}
}

func TestPreviewTab_Empty(t *testing.T) {
	preview := NewPreviewTab(newFakeSessionBackend())

	content := preview.Content()
	if !strings.Contains(content, "No instance") {
		t.Errorf("empty preview should show fallback: %q", content)
	}
}

func TestDiffTab_Content(t *testing.T) {
	mock := exec.NewMockExecutor()
	callIdx := 0
	mock.OnCommand("git", func(c exec.Call) (exec.Result, error) {
		callIdx++
		args := strings.Join(c.Args, " ")
		if strings.Contains(args, "--stat") {
			return exec.Result{Stdout: " 3 files changed, 42 insertions(+)\n", ExitCode: 0}, nil
		}
		return exec.Result{Stdout: "+added line\n-removed line\n", ExitCode: 0}, nil
	})
	wtMgr := worktree.NewManager(mock)
	diff := NewDiffTab(wtMgr)

	diff.Refresh(nil, "/tmp/wt")
	content := diff.Content()
	if !strings.Contains(content, "added line") {
		t.Errorf("diff should contain diff output: %q", content)
	}
}

func TestTerminalTab_NoSession(t *testing.T) {
	term := NewTerminalTab(newFakeSessionBackend(), backend.Capabilities{SupportsShellTerminal: true})

	if term.HasSession("test") {
		t.Error("should have no session initially")
	}
	content := term.Content()
	if !strings.Contains(content, "Enter") {
		t.Errorf("should show Enter prompt: %q", content)
	}
}

func TestSitePicker_View(t *testing.T) {
	picker := NewSitePicker()
	picker.SetItems([]SitePickerItem{
		{Name: "auth", Status: site.SiteAvailable, TasksDone: 0, TasksTotal: 12},
		{Name: "payments", Status: site.SiteInProgress, TasksDone: 5, TasksTotal: 10},
		{Name: "done", Status: site.SiteDone, TasksDone: 8, TasksTotal: 8},
	})
	picker.Show()

	view := picker.View()
	if !strings.Contains(view, "auth") {
		t.Error("should show auth")
	}
	if !strings.Contains(view, "payments") {
		t.Error("should show payments")
	}
}

func TestSitePicker_Selection(t *testing.T) {
	picker := NewSitePicker()
	picker.SetItems([]SitePickerItem{
		{Name: "auth", Status: site.SiteAvailable},
		{Name: "payments", Status: site.SiteAvailable},
	})

	// Single select
	items := picker.SelectedItems()
	if len(items) != 1 || items[0].Name != "auth" {
		t.Error("should select first item by default")
	}

	// Multi-select
	picker.ToggleSelect()
	picker.MoveDown()
	picker.ToggleSelect()
	items = picker.SelectedItems()
	if len(items) != 2 {
		t.Errorf("should have 2 selected, got %d", len(items))
	}
}

func TestSitePicker_CantSelectDone(t *testing.T) {
	picker := NewSitePicker()
	picker.SetItems([]SitePickerItem{
		{Name: "done", Status: site.SiteDone},
	})

	picker.ToggleSelect()
	if picker.multiSelect[0] {
		t.Error("should not be able to select done sites")
	}
}

func TestMapKey_Normal(t *testing.T) {
	tests := []struct {
		key  string
		want KeyAction
	}{
		{"n", ActionNew},
		{"D", ActionKill},
		{"q", ActionQuit},
		{"tab", ActionSwitchTab},
		{"?", ActionHelp},
		{"j", ActionNavigateDown},
		{"k", ActionNavigateUp},
		{"enter", ActionOpen},
	}
	for _, tt := range tests {
		got := MapKey(tt.key, false, OverlayNone, false)
		if got != tt.want {
			t.Errorf("MapKey(%q, false) = %d, want %d", tt.key, got, tt.want)
		}
	}
}

func TestMapKey_Overlay(t *testing.T) {
	// In overlay mode, 'n' maps to ConfirmNo in confirmation
	got := MapKey("n", true, OverlayConfirmation, false)
	if got != ActionConfirmNo {
		t.Errorf("in confirmation overlay, 'n' should be ConfirmNo, got %d", got)
	}

	// Esc always cancels
	got = MapKey("esc", true, OverlayHelp, false)
	if got != ActionCancel {
		t.Errorf("esc should cancel in overlay, got %d", got)
	}

	// Regular keys don't work in overlay
	got = MapKey("D", true, OverlayHelp, false)
	if got != ActionNone {
		t.Errorf("D should be None in overlay, got %d", got)
	}
}
