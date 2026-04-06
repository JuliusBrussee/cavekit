---
created: "2026-03-17T00:00:00Z"
last_edited: "2026-03-17T00:00:00Z"
---

# Spec: Tmux Session Management

## Scope
Managing detached tmux sessions as the execution backend for Claude Code instances. Each agent runs in its own isolated tmux session. The TUI never renders tmux panes directly — it captures snapshots for preview and attaches for full interaction.

## Requirements

### R1: Session Lifecycle
**Description:** Create, restore, and destroy detached tmux sessions running Claude Code (or other programs).
**Acceptance Criteria:**
- [ ] Can create a new detached tmux session with a given name, working directory, and program
- [ ] Session names are sanitized (no whitespace, dots replaced with underscores, prefixed with `sdd_`)
- [ ] Can check if a session exists via `tmux has-session`
- [ ] Can kill a session and clean up its PTY
- [ ] Can restore an attached PTY to an existing session (for detach/reattach cycles)
- [ ] Sets history-limit to 10000 and enables mouse on created sessions
**Dependencies:** none

### R2: Pane Content Capture
**Description:** Capture the visible content of a detached tmux session's pane for rendering in the TUI preview.
**Acceptance Criteria:**
- [ ] Can capture current visible pane content with ANSI escape sequences preserved (`tmux capture-pane -p -e -J`)
- [ ] Can capture full scrollback history (start="-", end="-") for scroll mode
- [ ] Capture is non-blocking and returns string content
**Dependencies:** R1

### R3: Full-Screen Attach/Detach
**Description:** Take over the terminal to interact directly with a tmux session, then return to the TUI.
**Acceptance Criteria:**
- [ ] Attach copies tmux PTY output to stdout and stdin to the PTY
- [ ] Detach via Ctrl+Q returns control to the TUI
- [ ] Window size is forwarded to the tmux session while attached
- [ ] Attach returns a channel that closes when detach completes
- [ ] Abnormal termination (Ctrl+D) prints a warning
**Dependencies:** R1

### R4: Status Detection
**Description:** Detect whether Claude Code is actively working or waiting for input.
**Acceptance Criteria:**
- [ ] Detects "active" state by hashing pane content and comparing to previous hash
- [ ] Detects "prompt" state by checking for Claude Code permission prompts ("No, and tell Claude what to do differently")
- [ ] Detects trust prompts ("Do you trust the files in this folder?") and auto-dismisses them
**Dependencies:** R2

### R5: Input Injection
**Description:** Send keystrokes and prompts to a detached tmux session.
**Acceptance Criteria:**
- [ ] Can send Enter keystroke
- [ ] Can send arbitrary key sequences (for `/ck:make` command injection)
- [ ] Can send multi-line prompt text
**Dependencies:** R1

## Out of Scope
- Tmux installation/version checking (handled by CLI preflight)
- Tmux configuration beyond session-level settings
- Multiple panes within a single tmux session

## Cross-References
- See also: cavekit-session.md (consumes tmux sessions)
