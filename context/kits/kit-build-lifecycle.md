---
created: "2026-03-20T00:00:00Z"
last_edited: "2026-03-20T00:00:00Z"
---

# Cavekit: Build Lifecycle

## Scope
Robust worktree lifecycle management for the build system. Covers keeping worktrees fresh with main, forwarding environment variables, and providing structured recovery options when builds fail or get interrupted.

## Requirements

### R1: Auto-Merge on Resume
**Description:** Before an agent starts or resumes work in a worktree, the system merges the latest `main` (or the branch the worktree was created from) into the worktree branch. This ensures the agent always has access to the latest code.
**Acceptance Criteria:**
- [ ] On build start, if the worktree already exists, main is merged into the worktree branch before the agent begins work
- [ ] If the merge has conflicts, the build does not proceed — the user is notified with the conflicting files and offered options (resolve manually, abort, recreate worktree from fresh main)
- [ ] On first-time worktree creation, the branch is created from the current HEAD of main
- [ ] The merge result is logged so the user can see what was pulled in
**Dependencies:** none

### R2: Environment Variable Forwarding
**Description:** Environment files from the project root are made available in the worktree so that builds, tests, and dev servers work correctly in the isolated environment.
**Acceptance Criteria:**
- [ ] On worktree creation, `.env*` files from the project root are made available in the worktree
- [ ] On resume (when worktree already exists), env file availability is verified and restored if broken
- [ ] Changes to env vars in the project root are reflected in the worktree without manual intervention
- [ ] Gitignored env files (like `.env.local`) are also forwarded to the worktree
**Dependencies:** none

### R3: Failure Recovery
**Description:** When a build fails or is interrupted, the system offers structured recovery options instead of leaving an orphaned worktree.
**Acceptance Criteria:**
- [ ] On build failure or interruption, the user is presented with three options: (a) resume — rebase on main and restart the agent, (b) abandon — remove the worktree and its branch, (c) merge — merge whatever was completed back to main via the normal `/ck:merge` flow
- [ ] The TUI surfaces worktree state for failed/interrupted instances (branch name, last commit, diff stats) so the user can make an informed choice
- [ ] Abandoned worktrees are fully cleaned up (worktree removed, branch deleted if unmerged work is confirmed disposable)
- [ ] Resume re-runs R1 (auto-rebase) and R2 (env var verification) before restarting the agent
**Dependencies:** R1, R2

### R4: Worktree Health Check
**Description:** Before any build operation, the system validates that the worktree is in a healthy state — clean git status, no detached HEAD, env vars present, branch exists on remote if expected.
**Acceptance Criteria:**
- [ ] Health check runs automatically before build start and resume
- [ ] Reports issues as warnings (non-blocking) or errors (blocking) with clear descriptions
- [ ] Warnings: uncommitted changes in worktree, env files missing
- [ ] Errors: detached HEAD, worktree path doesn't exist, branch deleted
- [ ] Health check results are visible in the TUI instance detail view
**Dependencies:** R1, R2

### R5: Archive on Stop
**Description:** When a build session is stopped (via the Stop hook, TUI kill, or CLI kill), the system archives the build's implementation state before teardown. This preserves progress so nothing is silently lost.
**Acceptance Criteria:**
- [ ] On stop, the current loop log and impl tracking files are archived to `context/impl/archive/{timestamp}/`
- [ ] The archive includes: loop log, impl status files, and a summary of tasks completed vs remaining
- [ ] Archive happens before worktree removal (if the user chose to abandon)
- [ ] If the build had no progress (zero tasks completed), no archive is created
**Dependencies:** R3

## Out of Scope
- Changing the worktree directory convention (`../{project}-cavekit-{name}`)
- Multi-repo worktree support (only single-repo projects)
- Automatic conflict resolution during rebase (conflicts require user intervention)
- Worktree creation for non-build purposes

## Cross-References
- See also: cavekit-spec-sync.md (worktree changes must be on main before `/ck:revise` can scan them)
- See also: cavekit-worktree.md (existing worktree creation and discovery primitives this domain builds on)
- See also: cavekit-session.md (instance lifecycle triggers build lifecycle operations)

## Changelog
