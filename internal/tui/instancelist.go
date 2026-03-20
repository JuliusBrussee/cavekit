package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/julb/blueprint-monitor/internal/session"
)

// InstanceList renders the left panel with all active instances.
type InstanceList struct {
	instances     []*session.Instance
	selectedIndex int
	scrollOffset  int
	width         int
	height        int
}

// NewInstanceList creates a new instance list component.
func NewInstanceList() *InstanceList {
	return &InstanceList{}
}

// SetInstances updates the instance list.
func (l *InstanceList) SetInstances(instances []*session.Instance) {
	l.instances = instances
}

// SetSelected sets the selected index.
func (l *InstanceList) SetSelected(index int) {
	if index < 0 {
		index = 0
	}
	if len(l.instances) > 0 && index >= len(l.instances) {
		index = len(l.instances) - 1
	}
	l.selectedIndex = index
	l.ensureVisible()
}

// Selected returns the currently selected instance, or nil.
func (l *InstanceList) Selected() *session.Instance {
	if l.selectedIndex >= 0 && l.selectedIndex < len(l.instances) {
		return l.instances[l.selectedIndex]
	}
	return nil
}

// SelectedIndex returns the current selection index.
func (l *InstanceList) SelectedIndex() int {
	return l.selectedIndex
}

// SetSize updates the available dimensions.
func (l *InstanceList) SetSize(w, h int) {
	l.width = w
	l.height = h
}

// View renders the instance list.
func (l *InstanceList) View() string {
	if len(l.instances) == 0 {
		return l.renderEmptyState()
	}

	visibleRows := l.visibleRowCount()
	var rows []string

	// Scroll indicator top
	if l.scrollOffset > 0 {
		rows = append(rows, ScrollIndicatorStyle.Width(l.width).Render("▲"))
	}

	endIdx := min(l.scrollOffset+visibleRows, len(l.instances))
	for i := l.scrollOffset; i < endIdx; i++ {
		row := l.renderRow(l.instances[i], i == l.selectedIndex)
		rows = append(rows, row)
	}

	// Scroll indicator bottom
	if endIdx < len(l.instances) {
		rows = append(rows, ScrollIndicatorStyle.Width(l.width).Render("▼"))
	}

	return strings.Join(rows, "\n")
}

func (l *InstanceList) renderEmptyState() string {
	content := lipgloss.NewStyle().Foreground(ColorMuted).Bold(true).Render("No agents yet") + "\n\n" +
		lipgloss.NewStyle().Foreground(ColorSecondary).Render("Press ") +
		MenuKeyStyle.Render("n") +
		lipgloss.NewStyle().Foreground(ColorSecondary).Render(" to launch one")

	return lipgloss.Place(l.width, l.height, lipgloss.Center, lipgloss.Center, content)
}

func (l *InstanceList) renderRow(inst *session.Instance, selected bool) string {
	w := l.width
	if w < 10 {
		w = 10
	}

	// Status indicator
	statusIcon := statusIconFor(inst.Status)

	// Line 1: border + status + title (left) + progress fraction (right)
	title := inst.Title
	progressStr := ""
	if inst.TasksTotal > 0 {
		progressStr = fmt.Sprintf("%d/%d", inst.TasksDone, inst.TasksTotal)
	}

	titleMaxW := w - lipgloss.Width(statusIcon) - lipgloss.Width(progressStr) - 5
	if titleMaxW < 3 {
		titleMaxW = 3
	}
	if len(title) > titleMaxW {
		title = title[:titleMaxW]
	}

	gap1 := max(w-3-lipgloss.Width(statusIcon)-len(title)-lipgloss.Width(progressStr), 0)

	line1 := fmt.Sprintf(" %s %s%s%s", statusIcon, title, spaces(gap1), progressStr)

	// Line 2: progress bar + diff stats + branch
	barWidth := min(15, w/3)
	bar := ""
	if inst.TasksTotal > 0 {
		bar = RenderProgressBar(inst.TasksDone, inst.TasksTotal, barWidth)
	} else {
		bar = strings.Repeat("░", barWidth)
	}

	diffStr := ""
	if inst.DiffAdded > 0 || inst.DiffRemoved > 0 {
		diffStr = fmt.Sprintf("+%d/-%d", inst.DiffAdded, inst.DiffRemoved)
	}

	branchStr := ""
	if inst.BranchName != "" {
		branchStr = inst.BranchName
	}

	// Health indicator
	healthStr := ""
	if inst.HealthStatus == "error" {
		healthStr = lipgloss.NewStyle().Foreground(ColorDanger).Render("⊘")
	} else if inst.HealthStatus == "warning" {
		healthStr = lipgloss.NewStyle().Foreground(ColorWarning).Render("△")
	}

	line2Parts := " " + bar
	if healthStr != "" {
		line2Parts += "  " + healthStr
	}
	if diffStr != "" {
		line2Parts += "  " + diffStr
	}
	if branchStr != "" {
		remaining := w - lipgloss.Width(line2Parts) - 2
		if remaining > 3 {
			if len(branchStr) > remaining {
				branchStr = branchStr[:remaining-1] + "…"
			}
			line2Parts += "  " + lipgloss.NewStyle().Foreground(ColorMuted).Render(branchStr)
		}
	}

	if selected {
		border := InstanceSelectedBorder.String()
		l1 := SelectedItemStyle.Width(w - 1).Render(line1)
		l2 := SelectedItemStyle.Width(w - 1).Render(line2Parts)
		return border + l1 + "\n" + border + l2
	}

	border := InstanceNormalBorder.String()
	l1 := NormalItemStyle.Width(w - 1).Render(line1)
	l2 := NormalItemStyle.Width(w - 1).Render(line2Parts)
	return border + l1 + "\n" + border + l2
}

func (l *InstanceList) visibleRowCount() int {
	// Each row takes 2 lines
	if l.height <= 0 {
		return 0
	}
	return max(l.height/2, 1)
}

func (l *InstanceList) ensureVisible() {
	visible := l.visibleRowCount()
	if visible <= 0 {
		return
	}
	if l.selectedIndex < l.scrollOffset {
		l.scrollOffset = l.selectedIndex
	}
	if l.selectedIndex >= l.scrollOffset+visible {
		l.scrollOffset = l.selectedIndex - visible + 1
	}
}

func statusIconFor(status session.Status) string {
	switch status {
	case session.StatusRunning:
		return StatusRunning.String()
	case session.StatusReady:
		return StatusReady.String()
	case session.StatusLoading:
		return StatusLoading.String()
	case session.StatusPaused:
		return StatusPaused.String()
	case session.StatusDone:
		return StatusDone.String()
	default:
		return "?"
	}
}
