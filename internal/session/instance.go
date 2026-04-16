// Package session defines the instance model that ties together a runtime session,
// git worktree, and site into a single manageable unit.
package session

import (
	"encoding/json"
	"time"
)

// Status represents the current state of an instance.
type Status int

const (
	StatusLoading Status = iota
	StatusRunning
	StatusReady
	StatusPaused
	StatusDone
)

func (s Status) String() string {
	switch s {
	case StatusLoading:
		return "Loading"
	case StatusRunning:
		return "Running"
	case StatusReady:
		return "Ready"
	case StatusPaused:
		return "Paused"
	case StatusDone:
		return "Done"
	default:
		return "Unknown"
	}
}

// Icon returns a display icon for the status.
func (s Status) Icon() string {
	switch s {
	case StatusLoading:
		return "◌"
	case StatusRunning:
		return "⟳"
	case StatusReady:
		return "●"
	case StatusPaused:
		return "⏸"
	case StatusDone:
		return "✓"
	default:
		return "?"
	}
}

// Instance represents one agent working on one site.
type Instance struct {
	Title           string    `json:"title"`
	SitePath        string    `json:"site_path"`
	WorktreePath    string    `json:"worktree_path"`
	SessionName     string    `json:"session_name"`
	BackendKind     string    `json:"backend_kind,omitempty"`
	BackendPID      int       `json:"backend_pid,omitempty"`
	BackendLogPath  string    `json:"backend_log_path,omitempty"`
	BackendWorktree string    `json:"backend_worktree,omitempty"`
	Status          Status    `json:"status"`
	Program         string    `json:"program"`
	CreatedAt       time.Time `json:"created_at"`

	// Progress fields (updated periodically from site tracking).
	TasksDone   int    `json:"tasks_done"`
	TasksTotal  int    `json:"tasks_total"`
	CurrentTier int    `json:"current_tier"`
	CurrentTask string `json:"current_task"`

	// Diff stats (updated periodically from worktree).
	BranchName  string `json:"branch_name"`
	DiffAdded   int    `json:"diff_added"`
	DiffRemoved int    `json:"diff_removed"`

	// Health check state (from .claude/health-check.json in worktree).
	HealthStatus   string `json:"health_status"` // "healthy", "warning", "error"
	HealthErrors   int    `json:"health_errors"`
	HealthWarnings int    `json:"health_warnings"`
}

// NewInstance creates a new instance with default values.
func NewInstance(title, sitePath, program string) *Instance {
	return &Instance{
		Title:     title,
		SitePath:  sitePath,
		Status:    StatusLoading,
		Program:   program,
		CreatedAt: time.Now(),
	}
}

// IsActive returns true if the instance is running or ready (not paused/done).
func (i *Instance) IsActive() bool {
	return i.Status == StatusRunning || i.Status == StatusReady || i.Status == StatusLoading
}

// ProgressString returns a compact progress display.
func (i *Instance) ProgressString() string {
	if i.TasksTotal == 0 {
		return ""
	}
	return i.Status.Icon() + " " + i.Title + " " + formatFraction(i.TasksDone, i.TasksTotal)
}

func formatFraction(a, b int) string {
	return itoa(a) + "/" + itoa(b)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}

type instanceJSON struct {
	Title           string    `json:"title"`
	SitePath        string    `json:"site_path"`
	WorktreePath    string    `json:"worktree_path"`
	SessionName     string    `json:"session_name"`
	LegacySession   string    `json:"tmux_session"`
	BackendKind     string    `json:"backend_kind,omitempty"`
	BackendPID      int       `json:"backend_pid,omitempty"`
	BackendLogPath  string    `json:"backend_log_path,omitempty"`
	BackendWorktree string    `json:"backend_worktree,omitempty"`
	Status          Status    `json:"status"`
	Program         string    `json:"program"`
	CreatedAt       time.Time `json:"created_at"`
	TasksDone       int       `json:"tasks_done"`
	TasksTotal      int       `json:"tasks_total"`
	CurrentTier     int       `json:"current_tier"`
	CurrentTask     string    `json:"current_task"`
	BranchName      string    `json:"branch_name"`
	DiffAdded       int       `json:"diff_added"`
	DiffRemoved     int       `json:"diff_removed"`
	HealthStatus    string    `json:"health_status"`
	HealthErrors    int       `json:"health_errors"`
	HealthWarnings  int       `json:"health_warnings"`
}

func (i Instance) MarshalJSON() ([]byte, error) {
	return json.Marshal(instanceJSON{
		Title:           i.Title,
		SitePath:        i.SitePath,
		WorktreePath:    i.WorktreePath,
		SessionName:     i.SessionName,
		BackendKind:     i.BackendKind,
		BackendPID:      i.BackendPID,
		BackendLogPath:  i.BackendLogPath,
		BackendWorktree: i.BackendWorktree,
		Status:          i.Status,
		Program:         i.Program,
		CreatedAt:       i.CreatedAt,
		TasksDone:       i.TasksDone,
		TasksTotal:      i.TasksTotal,
		CurrentTier:     i.CurrentTier,
		CurrentTask:     i.CurrentTask,
		BranchName:      i.BranchName,
		DiffAdded:       i.DiffAdded,
		DiffRemoved:     i.DiffRemoved,
		HealthStatus:    i.HealthStatus,
		HealthErrors:    i.HealthErrors,
		HealthWarnings:  i.HealthWarnings,
	})
}

func (i *Instance) UnmarshalJSON(data []byte) error {
	var decoded instanceJSON
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}

	i.Title = decoded.Title
	i.SitePath = decoded.SitePath
	i.WorktreePath = decoded.WorktreePath
	i.SessionName = decoded.SessionName
	if i.SessionName == "" {
		i.SessionName = decoded.LegacySession
	}
	i.BackendKind = decoded.BackendKind
	i.BackendPID = decoded.BackendPID
	i.BackendLogPath = decoded.BackendLogPath
	i.BackendWorktree = decoded.BackendWorktree
	i.Status = decoded.Status
	i.Program = decoded.Program
	i.CreatedAt = decoded.CreatedAt
	i.TasksDone = decoded.TasksDone
	i.TasksTotal = decoded.TasksTotal
	i.CurrentTier = decoded.CurrentTier
	i.CurrentTask = decoded.CurrentTask
	i.BranchName = decoded.BranchName
	i.DiffAdded = decoded.DiffAdded
	i.DiffRemoved = decoded.DiffRemoved
	i.HealthStatus = decoded.HealthStatus
	i.HealthErrors = decoded.HealthErrors
	i.HealthWarnings = decoded.HealthWarnings
	return nil
}
