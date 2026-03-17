package tmux

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"
)

// PaneStatus indicates what the tmux pane is doing.
type PaneStatus int

const (
	PaneUnknown PaneStatus = iota
	PaneActive             // Content is changing (Claude is working)
	PanePrompt             // Claude is waiting for permission approval
	PaneTrust              // Trust prompt ("Do you trust the files...")
	PaneIdle               // Content hasn't changed (might be waiting for input)
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

// Permission prompt markers in Claude Code output.
var permissionPromptMarkers = []string{
	"No, and tell Claude what to do differently",
	"Allow once",
	"Allow always",
	"(Y)es",
}

// Trust prompt markers.
var trustPromptMarkers = []string{
	"Do you trust the files in this folder?",
	"Trust this project",
}

// StatusDetector detects the status of a tmux pane by comparing content hashes.
type StatusDetector struct {
	mgr       *Manager
	lastHash  map[string]string // session name → last content hash
}

// NewStatusDetector creates a new status detector.
func NewStatusDetector(mgr *Manager) *StatusDetector {
	return &StatusDetector{
		mgr:      mgr,
		lastHash: make(map[string]string),
	}
}

// Detect captures pane content and determines the current status.
func (d *StatusDetector) Detect(ctx context.Context, name string) (PaneStatus, error) {
	content, err := d.mgr.CapturePane(ctx, name)
	if err != nil {
		return PaneUnknown, err
	}

	// Check for trust prompts first (highest priority)
	if containsAny(content, trustPromptMarkers) {
		return PaneTrust, nil
	}

	// Check for permission prompts
	if containsAny(content, permissionPromptMarkers) {
		return PanePrompt, nil
	}

	// Compare hash to detect activity
	hash := hashContent(content)
	sessionName := SanitizeName(name)
	prevHash, exists := d.lastHash[sessionName]
	d.lastHash[sessionName] = hash

	if !exists {
		return PaneActive, nil // First check, assume active
	}
	if hash != prevHash {
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
