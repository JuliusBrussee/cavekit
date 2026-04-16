package tui

import (
	"context"

	"github.com/JuliusBrussee/cavekit/internal/backend"
)

// PreviewTab captures and renders session content for the selected instance.
type PreviewTab struct {
	sessionBackend backend.SessionBackend
	content        string
	scrollMode     bool
}

// NewPreviewTab creates a preview tab.
func NewPreviewTab(sessionBackend backend.SessionBackend) *PreviewTab {
	return &PreviewTab{sessionBackend: sessionBackend}
}

// Capture refreshes the pane content for the given session.
func (p *PreviewTab) Capture(ctx context.Context, sessionName string) {
	if sessionName == "" {
		p.content = ""
		return
	}

	var (
		content string
		err     error
	)
	if p.scrollMode {
		content, err = p.sessionBackend.CaptureScrollback(ctx, sessionName)
	} else {
		content, err = p.sessionBackend.CapturePane(ctx, sessionName)
	}
	if err != nil {
		p.content = "Failed to capture: " + err.Error()
		return
	}
	p.content = content
}

// Content returns the current captured content.
func (p *PreviewTab) Content() string {
	if p.content == "" {
		return "No instance selected or instance is paused."
	}
	return p.content
}

// SetScrollMode toggles full scrollback capture.
func (p *PreviewTab) SetScrollMode(enabled bool) {
	p.scrollMode = enabled
}

// IsScrollMode returns true if in scroll mode.
func (p *PreviewTab) IsScrollMode() bool {
	return p.scrollMode
}
