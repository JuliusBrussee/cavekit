---
created: "2026-03-17T00:00:00Z"
last_edited: "2026-03-19T00:00:00Z"
---

# Spec: Site Discovery and Tracking

## Scope
Finding site files, parsing their task structure, and tracking task completion from implementation files. This is the Cavekit-specific intelligence that claude-squad doesn't have.

## Requirements

### R1: Site Discovery
**Description:** Find site files in the project's `context/plans/` directory (with fallback to legacy `context/sites/` and `context/frontiers/`).
**Acceptance Criteria:**
- [ ] Scans `context/plans/` first, then `context/sites/`, then `context/frontiers/` for `*.md` files containing "site" or "frontier" in the name
- [ ] Excludes files in archive subdirectories
- [ ] Returns list of site paths with derived names using the canonical derivation: strip `plan-`, `build-site-`, `feature-` prefixes, then strip `-?frontier-?` or `-?site-?` anywhere, then strip leading/trailing hyphens. Empty result defaults to `execute`
- [ ] Name derivation logic MUST be identical across all scripts that map site filenames to worktree paths
- [ ] Works from both project root and worktree paths — all path variables must be explicitly initialized via `git rev-parse --show-toplevel`, never assumed from undefined variables
**Dependencies:** none

### R2: Site Parsing
**Description:** Parse a site file to extract its task dependency graph.
**Acceptance Criteria:**
- [ ] Extracts all task IDs matching pattern `T-([A-Za-z0-9]+-)*[A-Za-z0-9]+`
- [ ] Parses tier structure (Tier 0, Tier 1, etc.) from markdown headers
- [ ] Extracts task title, spec reference, requirement, blockedBy, and effort from table rows
- [ ] Returns total task count and per-tier breakdown
**Dependencies:** R1

### R3: Task Status Tracking
**Description:** Determine which tasks are done, in-progress, or blocked by reading implementation tracking files.
**Acceptance Criteria:**
- [ ] Scans `context/impl/impl-*.md` files for task IDs with status markers (DONE, IN PROGRESS, PARTIAL, BLOCKED, DEAD END)
- [ ] Also scans worktree impl directories for cross-worktree awareness
- [ ] Task ID matching uses word boundaries to prevent prefix collisions (e.g., `T-1` must NOT match `T-10 DONE`)
- [ ] Returns per-task status map
- [ ] Computes aggregate: total, done, in-progress, blocked, remaining
**Dependencies:** R1, R2

### R4: Site Status Classification
**Description:** Classify each site's overall status for display.
**Acceptance Criteria:**
- [ ] "done" — all tasks complete
- [ ] "in-progress" — has an active worktree with Ralph Loop running
- [ ] "available" — has incomplete tasks, no active worktree
- [ ] Status detection checks for `.claude/ralph-loop.local.md` in the site's worktree
**Dependencies:** R3

### R5: Progress Summary
**Description:** Generate a compact progress string for status bar display.
**Acceptance Criteria:**
- [ ] Format: `{icon} {name} {done}/{total}` (e.g., `⟳ auth 3/12`)
- [ ] Icon: `⟳` for in-progress, `✓` for done, `·` for available
- [ ] Includes current task ID if in-progress
**Dependencies:** R3, R4

### R6: Site Selection (Multi-Candidate Ranking)
**Description:** When multiple sites exist, deterministically select the best one.
**Acceptance Criteria:**
- [ ] If `--filter` is set and matches zero sites, hard-fail with `exit 1` and list available sites — never silently fall back to unfiltered
- [ ] Ranking priority: active worktree with Ralph Loop (score 3) > worktree exists or has incomplete tasks (score 2) > base (score 1)
- [ ] Ties break deterministically: first candidate in alphabetical order wins (use `>` not `>=` in score comparison)
- [ ] All task ID grep patterns use `$TASK_ID_PATTERN` variable, never hardcoded subsets
- [ ] When listing candidates, mark the selected one with `→` for Claude visibility
**Dependencies:** R1, R3

## Back-Propagated

The following requirements were added after tracing manual bug fixes back to spec gaps (2026-03-17):

- **R1 (name derivation)**: Unified canonical sed chain after launch scripts used different derivations, causing worktree lookup failures
- **R1 (variable initialization)**: Added explicit `git rev-parse` requirement after scripts used undefined variables, silently breaking all worktree path lookups
- **R3 (word boundaries)**: Added after `T-1` false-matched `T-10 DONE` in done-count grep, inflating completion counts
- **R6 (filter fail-fast)**: Added after silent filter fallback caused wrong site selection on filter typos
- **R6 (deterministic ties)**: Added to prevent non-deterministic ranking when multiple candidates score equally

## Out of Scope
- Site creation (handled by `/ck:map`)
- Site modification/updating
- Spec file parsing (site references specs but doesn't need their content)

## Cross-References
- See also: cavekit-session.md (sessions are tied to sites)
- See also: cavekit-tui.md (TUI displays site progress)
- See also: cavekit-spec-sync.md (drift detection reads task-to-requirement mappings)
