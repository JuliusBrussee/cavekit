package tui

import (
	"context"

	"github.com/JuliusBrussee/cavekit/internal/backend"
)

// TerminalTab manages terminal content for the selected instance.
type TerminalTab struct {
	sessionBackend backend.SessionBackend
	caps           backend.Capabilities
	sessions       map[string]string
	content        string
}

// NewTerminalTab creates a terminal tab.
func NewTerminalTab(sessionBackend backend.SessionBackend, caps backend.Capabilities) *TerminalTab {
	return &TerminalTab{
		sessionBackend: sessionBackend,
		caps:           caps,
		sessions:       make(map[string]string),
	}
}

// EnsureSession creates a terminal session for the instance if the backend supports one.
func (t *TerminalTab) EnsureSession(ctx context.Context, instanceTitle, worktreePath, primarySessionName string) string {
	if !t.caps.SupportsShellTerminal {
		return primarySessionName
	}

	sessionName := "term_" + instanceTitle
	if _, exists := t.sessions[instanceTitle]; !exists {
		if err := t.sessionBackend.CreateSession(ctx, sessionName, worktreePath, backend.DefaultTerminalProgram()); err == nil {
			t.sessions[instanceTitle] = sessionName
		}
	}

	return t.sessions[instanceTitle]
}

// Capture updates the terminal content.
func (t *TerminalTab) Capture(ctx context.Context, instanceTitle, primarySessionName string) {
	if !t.caps.SupportsShellTerminal {
		if primarySessionName == "" {
			t.content = "Open a running agent to view full scrollback."
			return
		}
		content, err := t.sessionBackend.CaptureScrollback(ctx, primarySessionName)
		if err != nil {
			t.content = "Terminal session error: " + err.Error()
			return
		}
		t.content = content
		return
	}

	sessionName, exists := t.sessions[instanceTitle]
	if !exists {
		t.content = "Press Enter to open terminal."
		return
	}

	content, err := t.sessionBackend.CapturePane(ctx, sessionName)
	if err != nil {
		t.content = "Terminal session error: " + err.Error()
		return
	}
	t.content = content
}

// Content returns the current terminal content.
func (t *TerminalTab) Content() string {
	if t.content == "" {
		if t.caps.SupportsShellTerminal {
			return "Press Enter to open terminal."
		}
		return "Select an agent and open the Terminal tab to view scrollback."
	}
	return t.content
}

// HasSession returns true if the terminal content is available.
func (t *TerminalTab) HasSession(instanceTitle string) bool {
	if !t.caps.SupportsShellTerminal {
		return true
	}
	_, exists := t.sessions[instanceTitle]
	return exists
}

// SessionName returns the session name backing the instance's terminal.
func (t *TerminalTab) SessionName(instanceTitle string) string {
	return t.sessions[instanceTitle]
}
