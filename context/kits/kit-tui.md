---
created: "2026-03-17T00:00:00Z"
last_edited: "2026-03-19T00:00:00Z"
---

# Spec: Terminal User Interface

## Scope
The bubbletea-based TUI that replaces the current tmux split-pane layout. Renders a session list, preview/diff/terminal tabs, and a bottom menu — all in a single process that controls rendering (no flicker).

## Requirements

### R1: Application Shell
**Description:** The main bubbletea application with alt-screen, mouse support, and component layout.
**Acceptance Criteria:**
- [ ] Uses `tea.WithAltScreen()` for full-screen takeover
- [ ] Uses `tea.WithMouseCellMotion()` for mouse scroll support
- [ ] Layout: left panel (30% width) = instance list, right panel (70%) = tabbed content, bottom = menu bar
- [ ] Responds to terminal resize events
- [ ] Clean exit saves state and restores terminal
**Dependencies:** none

### R2: Instance List
**Description:** Left panel showing all active instances with status indicators.
**Acceptance Criteria:**
- [ ] Each row shows: index number, title, branch name, diff stats (+N, -N), status indicator
- [ ] Status indicators: green dot = Running, yellow dot = Ready, spinner = Loading, grey = Paused
- [ ] Cavekit-specific: shows `{done}/{total} tasks` next to each instance
- [ ] Navigate with j/k or arrow keys
- [ ] Selected instance highlighted with border/background
- [ ] Mouse click selects instance
**Dependencies:** R1, cavekit-session R1

### R3: Tabbed Content Window
**Description:** Right panel with Preview, Diff, and Terminal tabs.
**Acceptance Criteria:**
- [ ] Tab bar at top with Preview | Diff | Terminal labels
- [ ] Switch tabs with Tab key
- [ ] Active tab is visually highlighted
- [ ] Each tab renders independently based on the selected instance
**Dependencies:** R1

### R4: Preview Tab
**Description:** Shows a snapshot of the selected instance's tmux pane content.
**Acceptance Criteria:**
- [ ] Captures tmux pane content every 100ms for the selected instance only
- [ ] Renders content with ANSI color preservation
- [ ] Supports scroll mode (Shift+Up/Down to enter, Esc to exit)
- [ ] In scroll mode, captures full scrollback history
- [ ] Shows fallback message when no instance is selected or instance is paused
**Dependencies:** R3, cavekit-tmux R2

### R5: Diff Tab
**Description:** Shows git diff between the instance's branch and main.
**Acceptance Criteria:**
- [ ] Renders `git diff main...HEAD` output with syntax highlighting
- [ ] Scrollable with Shift+Up/Down
- [ ] Shows diff stats summary at top (files changed, +insertions, -deletions)
- [ ] Updates when instance changes or on metadata tick
**Dependencies:** R3, cavekit-worktree R2

### R6: Terminal Tab
**Description:** An independent terminal session in the instance's worktree directory.
**Acceptance Criteria:**
- [ ] Creates a separate tmux session per instance for shell access in the worktree
- [ ] Displays captured pane content (same as preview but for the terminal session)
- [ ] Press Enter to attach full-screen (interact directly)
- [ ] Sessions are cached per instance and preserved when switching between instances
**Dependencies:** R3, cavekit-tmux R1

### R7: Bottom Menu
**Description:** Shows available keyboard shortcuts.
**Acceptance Criteria:**
- [ ] Displays: `n new | D kill | Enter/o open | p push branch | c checkout | tab switch tab | ? help | q quit`
- [ ] Highlights the active shortcut briefly when pressed (visual feedback)
- [ ] Menu adapts to current state (different options when creating new instance vs. default)
**Dependencies:** R1

### R8: New Instance Flow
**Description:** Interactive flow for creating a new agent instance.
**Acceptance Criteria:**
- [ ] Press `n` to start: shows name input field at the bottom of the instance list
- [ ] Press `N` (Shift+N) to start with prompt: shows name input, then prompt+branch picker overlay
- [ ] Name input validates: non-empty, max 32 chars
- [ ] On Enter: creates instance, starts worktree+tmux, sends `/ck:make --filter {name}`
- [ ] Esc cancels and removes the pending instance
**Dependencies:** R2, R7, cavekit-session R2

### R9: Cavekit Progress Display
**Description:** Cavekit-specific progress information integrated into the UI.
**Acceptance Criteria:**
- [ ] Instance list shows task progress: `3/12` or `✓` if complete
- [ ] Progress bar or fraction visible per-instance
- [ ] Current working task ID shown when instance is Running
- [ ] Tier completion markers (Tier 0 ✓, Tier 1 >, Tier 2 -)
**Dependencies:** R2, cavekit-site R3, cavekit-site R5

### R10: Overlays
**Description:** Modal overlays for text input, confirmations, and help screens.
**Acceptance Criteria:**
- [ ] Text input overlay: centered modal with text field (for prompts)
- [ ] Confirmation overlay: yes/no modal for destructive actions (kill, push)
- [ ] Help overlay: shows keyboard shortcuts and usage info
- [ ] Overlays capture all input while active
- [ ] Esc or Ctrl+C closes overlays
**Dependencies:** R1

### R11: Site Picker Integration
**Description:** On startup (or when pressing `n`), show available sites for selection.
**Acceptance Criteria:**
- [ ] Lists sites from `context/frontiers/` with status (available/in-progress/done)
- [ ] Shows task count per site
- [ ] Done sites shown as struck-through/disabled
- [ ] In-progress sites shown with resume indicator
- [ ] Multi-select support for launching multiple agents at once
**Dependencies:** R8, cavekit-site R1, cavekit-site R4

## Out of Scope
- Web-based UI
- Multiple project support (single project root per TUI instance)
- Theming/color customization (uses sensible defaults)

### R12: Tick-Driven Data Flow
**Description:** The TUI must actively poll and update all data on each tick cycle, not just render static content.
**Acceptance Criteria:**
- [ ] On each 500ms tick: capture tmux pane content for selected instance's preview tab
- [ ] On each tick: refresh diff stats for selected instance's diff tab
- [ ] On each tick: call UpdateProgress for all active instances
- [ ] On each tick: run AutoYes.Check for all active instances if auto-yes enabled
- [ ] On each tick: update instance status via StatusDetector for all active instances
- [ ] PreviewTab, DiffTab, TerminalTab are instantiated in App and their output piped to TabContent
**Dependencies:** R1, R4, R5, cavekit-session R5, cavekit-session R6

### R13: Action Handler Completeness
**Description:** All mapped key actions must have corresponding handlers in App.Update.
**Acceptance Criteria:**
- [ ] ActionOpen: attaches to selected instance's tmux session via PTY, suspends TUI, returns on Ctrl+Q
- [ ] ActionPush: pushes selected instance's worktree branch to remote (with confirmation overlay)
- [ ] ActionCheckout: opens a shell in the selected instance's worktree directory
- [ ] ActionResume: calls sessionMgr.Resume for paused instances
- [ ] ActionScrollUp/Down: scrolls preview or diff tab content
**Dependencies:** R1, cavekit-tmux R3

## Cross-References
- See also: cavekit-session.md (TUI controls sessions)
- See also: cavekit-tmux.md (preview captures)
- See also: cavekit-site.md (progress data)
- See also: cavekit-cli.md (TUI is launched by CLI)

## Changes
- 2026-03-17: Added R12 (Tick-Driven Data Flow) — discovered during inspection (finding F-001, F-002, F-004, F-005)
- 2026-03-17: Added R13 (Action Handler Completeness) — discovered during inspection (finding F-003, F-006, F-008)
