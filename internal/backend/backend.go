package backend

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"
)

// SessionBackend abstracts the runtime used to host agent sessions.
type SessionBackend interface {
	CreateSession(ctx context.Context, name, workDir, program string) error
	Exists(ctx context.Context, name string) bool
	Kill(ctx context.Context, name string) error
	ListSessions(ctx context.Context) ([]string, error)
	CapturePane(ctx context.Context, name string) (string, error)
	CaptureScrollback(ctx context.Context, name string) (string, error)
	SendKeys(ctx context.Context, name string, keys ...string) error
	SendText(ctx context.Context, name string, text string) error
	SendCommand(ctx context.Context, name, cmd string) error
}

// Capabilities describe optional runtime features used by the TUI.
type Capabilities struct {
	SupportsAttach       bool
	SupportsShellTerminal bool
}

// PaneStatus indicates what the active session appears to be doing.
type PaneStatus int

const (
	PaneUnknown PaneStatus = iota
	PaneActive
	PanePrompt
	PaneTrust
	PaneIdle
)

func (s PaneStatus) String() string {
	switch s {
	case PaneActive:
		return "active"
	case PanePrompt:
		return "prompt"
	case PaneTrust:
		return "trust"
	case PaneIdle:
		return "idle"
	default:
		return "unknown"
	}
}

var permissionPromptMarkers = []string{
	"No, and tell Claude what to do differently",
	"Allow once",
	"Allow always",
	"(Y)es",
}

var trustPromptMarkers = []string{
	"Do you trust the files in this folder?",
	"Trust this project",
}

// StatusDetector detects prompt/idle/active state using captured session content.
type StatusDetector struct {
	backend  SessionBackend
	lastHash map[string]string
}

func NewStatusDetector(sessionBackend SessionBackend) *StatusDetector {
	return &StatusDetector{
		backend:  sessionBackend,
		lastHash: make(map[string]string),
	}
}

func (d *StatusDetector) Detect(ctx context.Context, name string) (PaneStatus, error) {
	content, err := d.backend.CapturePane(ctx, name)
	if err != nil {
		return PaneUnknown, err
	}

	if containsAny(content, trustPromptMarkers) {
		return PaneTrust, nil
	}
	if containsAny(content, permissionPromptMarkers) {
		return PanePrompt, nil
	}

	hash := hashContent(content)
	prevHash, exists := d.lastHash[name]
	d.lastHash[name] = hash
	if !exists || hash != prevHash {
		return PaneActive, nil
	}
	return PaneIdle, nil
}

func hashContent(content string) string {
	h := sha256.Sum256([]byte(content))
	return fmt.Sprintf("%x", h[:8])
}

func containsAny(content string, markers []string) bool {
	for _, marker := range markers {
		if strings.Contains(content, marker) {
			return true
		}
	}
	return false
}
