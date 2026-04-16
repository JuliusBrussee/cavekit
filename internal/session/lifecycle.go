package session

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/JuliusBrussee/cavekit/internal/backend"
	"github.com/JuliusBrussee/cavekit/internal/worktree"
)

// Manager orchestrates instance lifecycle operations.
type Manager struct {
	sessionBackend backend.SessionBackend
	worktree       *worktree.Manager
}

// NewManager creates a session manager.
func NewManager(sessionBackend backend.SessionBackend, wtMgr *worktree.Manager) *Manager {
	return &Manager{
		sessionBackend: sessionBackend,
		worktree:       wtMgr,
	}
}

type backendMetadataProvider interface {
	BackendMetadata(name string) (kind string, processID int, logPath string, worktreePath string, ok bool)
}

// Create allocates a new instance with the given title and site info.
func (m *Manager) Create(title, sitePath, siteName, program string) *Instance {
	inst := NewInstance(title, sitePath, program)
	inst.SessionName = siteName
	return inst
}

// Start creates the worktree and runtime session, then sends the build command.
func (m *Manager) Start(ctx context.Context, inst *Instance, projectRoot, siteName string, startupDelay time.Duration) error {
	inst.Status = StatusLoading

	wtPath, err := m.worktree.Create(ctx, projectRoot, siteName)
	if err != nil {
		return fmt.Errorf("create worktree: %w", err)
	}
	inst.WorktreePath = wtPath

	if err := m.sessionBackend.CreateSession(ctx, inst.SessionName, wtPath, inst.Program); err != nil {
		return fmt.Errorf("create session: %w", err)
	}

	inst.Status = StatusRunning
	inst.BackendWorktree = wtPath
	if provider, ok := m.sessionBackend.(backendMetadataProvider); ok {
		kind, processID, logPath, worktreePath, found := provider.BackendMetadata(inst.SessionName)
		if found {
			inst.BackendKind = kind
			inst.BackendPID = processID
			inst.BackendLogPath = logPath
			if worktreePath != "" {
				inst.BackendWorktree = worktreePath
			}
		}
	}

	sendBuild := func() {
		cmd := fmt.Sprintf("/ck:make --filter %s", siteName)
		_ = m.sessionBackend.SendCommand(ctx, inst.SessionName, cmd)
	}

	if startupDelay > 0 {
		go func() {
			time.Sleep(startupDelay)
			sendBuild()
		}()
	} else {
		sendBuild()
	}

	return nil
}

// Pause detaches an instance from TUI tracking (session keeps running).
func (m *Manager) Pause(inst *Instance) {
	inst.Status = StatusPaused
}

// Resume re-attaches an instance to TUI tracking.
func (m *Manager) Resume(ctx context.Context, inst *Instance) {
	if m.sessionBackend.Exists(ctx, inst.SessionName) {
		inst.Status = StatusRunning
	}
}

// Kill destroys the runtime session and optionally removes the worktree.
func (m *Manager) Kill(ctx context.Context, inst *Instance, projectRoot string, removeWorktree bool) error {
	if err := m.sessionBackend.Kill(ctx, inst.SessionName); err != nil {
		// Non-fatal: session might already be gone.
	}

	if inst.WorktreePath != "" {
		archiveImplState(inst.WorktreePath, inst.TasksDone)
	}

	if removeWorktree && inst.WorktreePath != "" {
		siteName := deriveSiteNameFromWorktree(inst.WorktreePath, projectRoot)
		if siteName != "" {
			_ = m.worktree.Remove(ctx, projectRoot, siteName)
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

	loopLog := filepath.Join(implDir, "loop-log.md")
	if data, err := os.ReadFile(loopLog); err == nil {
		_ = os.WriteFile(filepath.Join(archiveDir, "loop-log.md"), data, 0o644)
	}

	entries, err := os.ReadDir(implDir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		name := entry.Name()
		if !entry.IsDir() && len(name) > 5 && name[:5] == "impl-" {
			data, err := os.ReadFile(filepath.Join(implDir, name))
			if err == nil {
				_ = os.WriteFile(filepath.Join(archiveDir, name), data, 0o644)
			}
		}
	}
}

func deriveSiteNameFromWorktree(wtPath, projectRoot string) string {
	prefix := worktree.WorktreePath(projectRoot, "")
	if len(wtPath) > len(prefix) {
		return wtPath[len(prefix):]
	}
	return ""
}
