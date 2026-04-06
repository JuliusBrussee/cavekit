---
created: "2026-03-17T00:00:00Z"
last_edited: "2026-03-19T00:00:00Z"
---

# Spec: Git Worktree Management

## Scope
Creating and managing git worktrees for isolated agent execution. Each agent gets its own worktree on a dedicated branch so multiple agents can work on the same repo simultaneously without conflicts.

## Requirements

### R1: Worktree Lifecycle
**Description:** Create, detect, and remove git worktrees for Cavekit agents.
**Acceptance Criteria:**
- [ ] Creates worktree at `{project_root}/../{project_name}-cavekit-{site_name}` on branch `cavekit/{site_name}`
- [ ] Creates the branch from HEAD if it doesn't exist
- [ ] Detects if a worktree already exists and reuses it
- [ ] Can remove a worktree and its branch via `git worktree remove --force` + `git branch -D`
- [ ] Runs `git worktree prune` before removal to handle stale entries
**Dependencies:** none

### R2: Diff Stats
**Description:** Compute diff statistics between the worktree branch and main for display in the TUI.
**Acceptance Criteria:**
- [ ] Returns files changed count, insertions, and deletions (`git diff --stat main...HEAD`)
- [ ] Returns the raw diff output for the diff pane (`git diff main...HEAD`)
- [ ] Handles case where main branch doesn't exist or worktree has no commits yet
**Dependencies:** R1

### R3: Worktree Discovery
**Description:** Find all existing Cavekit worktrees for the current project.
**Acceptance Criteria:**
- [ ] Scans `{project_root}/../{project_name}-cavekit-*` directories
- [ ] Returns worktree path, branch name, and derived site name for each
- [ ] Detects if a worktree has an active Ralph Loop (`.claude/ralph-loop.local.md` exists)
**Dependencies:** R1

### R4: Branch Push
**Description:** Push worktree branch changes to remote.
**Acceptance Criteria:**
- [ ] Commits all changes with a descriptive message
- [ ] Pushes to remote with `--set-upstream` on first push
- [ ] Reports success/failure
**Dependencies:** R1

## Out of Scope
- Git authentication/credentials
- Merge conflict resolution (handled by `/ck:merge`)
- Non-Cavekit worktree management

## Cross-References
- See also: cavekit-session.md (creates worktrees per session)
- See also: cavekit-site.md (site name drives worktree naming)
- See also: cavekit-build-lifecycle.md (builds on worktree primitives for lifecycle management)
