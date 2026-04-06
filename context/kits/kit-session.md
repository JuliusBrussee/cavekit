---
created: "2026-03-17T00:00:00Z"
last_edited: "2026-03-19T00:00:00Z"
---

# Spec: Session Management

## Scope
The session (or "instance") model that ties together a tmux session, git worktree, and site into a single manageable unit. Handles persistence, state transitions, and lifecycle orchestration.

## Requirements

### R1: Instance Model
**Description:** A session instance represents one Claude Code agent working on one site.
**Acceptance Criteria:**
- [ ] Instance has: title, site path, worktree path, tmux session reference, status, program name, creation timestamp
- [ ] Status enum: Loading, Running, Ready, Paused, Done
- [ ] "Running" = Claude is actively generating output (pane content changing)
- [ ] "Ready" = Claude is waiting for input (permission prompt or idle)
- [ ] "Paused" = session checked out / detached from TUI management
**Dependencies:** none

### R2: Instance Lifecycle
**Description:** Create, start, pause, resume, and kill instances.
**Acceptance Criteria:**
- [ ] Creating an instance: allocates title, sets site path, derives worktree name
- [ ] Starting: creates worktree (if needed), creates tmux session, sends `/ck:make --filter {name}` after startup delay
- [ ] Pausing: detaches tmux session from TUI tracking (session keeps running)
- [ ] Resuming: re-attaches tmux session to TUI tracking
- [ ] Killing: kills tmux session, optionally removes worktree and branch
**Dependencies:** R1, cavekit-tmux R1, cavekit-worktree R1

### R3: Persistence
**Description:** Save and restore instance state across TUI restarts.
**Acceptance Criteria:**
- [ ] Instances are saved to `~/.cavekit/state.json` (or configurable path)
- [ ] Saved state includes: title, site path, worktree path, program, status
- [ ] On load, validates that tmux sessions and worktrees still exist
- [ ] Stale instances (tmux session gone) are marked accordingly
**Dependencies:** R1

### R4: Staggered Launch
**Description:** When multiple instances are created at once, stagger their `/ck:make` commands to avoid resource contention.
**Acceptance Criteria:**
- [ ] Configurable delay between launches (default 5 seconds)
- [ ] First instance starts immediately, subsequent ones wait
- [ ] Launch happens in background — TUI remains responsive
**Dependencies:** R2

### R5: Auto-Yes Mode
**Description:** Automatically approve Claude Code permission prompts.
**Acceptance Criteria:**
- [ ] When enabled, monitors pane content for permission prompts
- [ ] Sends Enter keystroke to approve
- [ ] Also handles trust prompts and MCP server prompts
**Dependencies:** R2, cavekit-tmux R4

### R6: Cavekit Progress Integration
**Description:** Each instance tracks its site progress for display.
**Acceptance Criteria:**
- [ ] Instance exposes: tasks done, tasks total, current tier, current task ID
- [ ] Progress is updated periodically (every 500ms metadata tick)
- [ ] Progress data comes from cavekit-site task status tracking
**Dependencies:** R1, cavekit-site R3

## Out of Scope
- Multiple programs per instance (each instance runs one program)
- Instance-to-instance communication
- Automatic scaling based on system resources

## Cross-References
- See also: cavekit-tmux.md (tmux session backend)
- See also: cavekit-worktree.md (worktree creation)
- See also: cavekit-site.md (progress tracking)
- See also: cavekit-tui.md (displays and controls instances)
- See also: cavekit-build-lifecycle.md (instance lifecycle triggers build lifecycle operations)
