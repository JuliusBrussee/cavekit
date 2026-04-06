package session

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/JuliusBrussee/cavekit/internal/tmux"
	"github.com/JuliusBrussee/cavekit/internal/worktree"
)

// Manager orchestrates instance lifecycle operations.
type Manager struct {
	tmux     *tmux.Manager
	worktree *worktree.Manager
}

// NewManager creates a session manager.
func NewManager(tmuxMgr *tmux.Manager, wtMgr *worktree.Manager) *Manager {
	return &Manager{
		tmux:     tmuxMgr,
		worktree: wtMgr,
	}
}

// Create allocates a new instance with the given title and site info.
func (m *Manager) Create(title, sitePath, siteName, program string) *Instance {
	inst := NewInstance(title, sitePath, program)
	inst.TmuxSession = tmux.SessionName(siteName)
	return inst
}

// Start creates the worktree and tmux session, then sends the build command.
func (m *Manager) Start(ctx context.Context, inst *Instance, projectRoot, siteName string, startupDelay time.Duration) error {
	inst.Status = StatusLoading

	// Create worktree
	wtPath, err := m.worktree.Create(ctx, projectRoot, siteName)
	if err != nil {
		return fmt.Errorf("create worktree: %w", err)
	}
	inst.WorktreePath = wtPath

	// Create tmux session
	err = m.tmux.CreateSession(ctx, siteName, wtPath, inst.Program)
	if err != nil {
		return fmt.Errorf("create tmux session: %w", err)
	}
	inst.TmuxSession = tmux.SessionName(siteName)
	inst.Status = StatusRunning

	// Wait for startup, then send the build command
	if startupDelay > 0 {
		go func() {
			time.Sleep(startupDelay)
			cmd := fmt.Sprintf("/ck:make --filter %s", siteName)
			m.tmux.SendCommand(ctx, siteName, cmd)
		}()
	} else {
		cmd := fmt.Sprintf("/ck:make --filter %s", siteName)
		m.tmux.SendCommand(ctx, siteName, cmd)
	}

	return nil
}

// Pause detaches an instance from TUI tracking (session keeps running).
func (m *Manager) Pause(inst *Instance) {
	inst.Status = StatusPaused
}

// Resume re-attaches an instance to TUI tracking.
func (m *Manager) Resume(ctx context.Context, inst *Instance) {
	if m.tmux.Exists(ctx, inst.TmuxSession) {
		inst.Status = StatusRunning
	}
}

// Kill destroys the tmux session and optionally removes the worktree.
func (m *Manager) Kill(ctx context.Context, inst *Instance, projectRoot string, removeWorktree bool) error {
	// Kill tmux session
	if err := m.tmux.Kill(ctx, inst.TmuxSession); err != nil {
		// Non-fatal: session might already be gone
	}

	// Archive impl state before worktree removal (R5: Archive on Stop)
	if inst.WorktreePath != "" {
		archiveImplState(inst.WorktreePath, inst.TasksDone)
	}

	if removeWorktree && inst.WorktreePath != "" {
		// Derive site name from worktree path
		siteName := deriveSiteNameFromWorktree(inst.WorktreePath, projectRoot)
		if siteName != "" {
			m.worktree.Remove(ctx, projectRoot, siteName)
		}
	}

	inst.Status = StatusDone
	return nil
}

// archiveImplState copies loop log and impl files to an archive directory
// before a build session is torn down. Skips if no tasks were completed.
func archiveImplState(wtPath string, tasksDone int) {
	if tasksDone == 0 {
		return
	}

	implDir := filepath.Join(wtPath, "context", "impl")
	if _, err := os.Stat(implDir); os.IsNotExist(err) {
		return
	}

	archiveDir := filepath.Join(implDir, "archive", time.Now().UTC().Format("20060102-150405"))
	if err := os.MkdirAll(archiveDir, 0o755); err != nil {
		return
	}

	// Archive loop log
	loopLog := filepath.Join(implDir, "loop-log.md")
	if data, err := os.ReadFile(loopLog); err == nil {
		os.WriteFile(filepath.Join(archiveDir, "loop-log.md"), data, 0o644)
	}

	// Archive impl tracking files
	entries, err := os.ReadDir(implDir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		name := entry.Name()
		if !entry.IsDir() && len(name) > 5 && name[:5] == "impl-" {
			data, err := os.ReadFile(filepath.Join(implDir, name))
			if err == nil {
				os.WriteFile(filepath.Join(archiveDir, name), data, 0o644)
			}
		}
	}
}

func deriveSiteNameFromWorktree(wtPath, projectRoot string) string {
	// WorktreePath format: {root}/../{name}-cavekit-{site}
	// We need to extract the site name
	prefix := worktree.WorktreePath(projectRoot, "")
	if len(wtPath) > len(prefix) {
		return wtPath[len(prefix):]
	}
	return ""
}
