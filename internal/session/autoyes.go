package session

import (
	"context"

	"github.com/JuliusBrussee/cavekit/internal/backend"
)

// AutoYes monitors pane content and auto-approves permission prompts.
type AutoYes struct {
	sessionBackend backend.SessionBackend
	detector       *backend.StatusDetector
	enabled        bool
}

// NewAutoYes creates an auto-yes handler.
func NewAutoYes(sessionBackend backend.SessionBackend, enabled bool) *AutoYes {
	return &AutoYes{
		sessionBackend: sessionBackend,
		detector:       backend.NewStatusDetector(sessionBackend),
		enabled:        enabled,
	}
}

// Check examines pane status and auto-approves if enabled.
// Returns true if an approval was sent.
func (a *AutoYes) Check(ctx context.Context, name string) bool {
	if !a.enabled {
		return false
	}

	status, err := a.detector.Detect(ctx, name)
	if err != nil {
		return false
	}

	switch status {
	case backend.PanePrompt, backend.PaneTrust:
		_ = a.sessionBackend.SendKeys(ctx, name, "Enter")
		return true
	}

	return false
}

// SetEnabled toggles auto-yes mode.
func (a *AutoYes) SetEnabled(enabled bool) {
	a.enabled = enabled
}

// IsEnabled returns the current state.
func (a *AutoYes) IsEnabled() bool {
	return a.enabled
}
