---
name: ck-progress
description: "Show progress against the build site or plan — tasks done, in progress, blocked, remaining"
argument-hint: "[--filter PATTERN]"
---

> **Note:** `/bp:progress` is deprecated and will be removed in a future version. Use `/ck:progress` instead.

# Cavekit Progress

Show the user a progress report by comparing the build site against implementation tracking.

## Step 1: Find Site

Look in `context/plans/` then `context/sites/` for `*site*`, `*plan*`, or `*frontier*` files (exclude `*overview*`). If `--filter` is set (parse from `$ARGUMENTS`), match against it.

If no site/plan found: "No build site or plan found. Run `/ck:map` first."

## Step 2: Read State

1. Read the site file — catalog every task (T-number), its tier, cavekit requirement, and blockedBy
2. Read all `context/impl/impl-*.md` files — extract task statuses (DONE, IN_PROGRESS, BLOCKED)
3. Read `context/impl/loop-log.md` if it exists — get the latest iteration number and last task completed

## Step 3: Classify Tasks

For each task in the site:
- **DONE** — marked done in impl tracking
- **IN_PROGRESS** — marked in progress
- **BLOCKED** — has unfinished blockedBy dependencies
- **READY** — all dependencies done, not started yet (next up)
- **WAITING** — dependencies not yet done, not directly blocked

## Step 4: Display Report

```markdown
## Cavekit Progress

### Summary
| Status | Count | % |
|--------|-------|---|
| DONE | {n} | {%} |
| IN_PROGRESS | {n} | {%} |
| READY | {n} | {%} |
| BLOCKED | {n} | {%} |
| WAITING | {n} | {%} |

### Progress Bar
[████████████░░░░░░░░] 58% (20/34 tasks)

### Current Tier: {n}
{tier name if any}

### Ready to Implement (next up)
| Task | Title | Cavekit | Requirement |
|------|-------|------|------------|
| T-{id} | {title} | cavekit-{domain}.md | R{n} |

### Recently Completed
| Task | Title | Iteration |
|------|-------|-----------|
| T-{id} | {title} | {n} |

### Blocked
| Task | Title | Waiting On |
|------|-------|-----------|
| T-{id} | {title} | T-{id} (status) |

### Dead Ends (if any)
| Task | Approach | Why Failed |
|------|----------|-----------|
| T-{id} | {what was tried} | {why it failed} |

### Loop Status
- Iterations completed: {n}
- Last iteration: {timestamp}
- Active: {yes/no — .claude/ralph-loop.local.md exists?}
```

Display this to the user.
