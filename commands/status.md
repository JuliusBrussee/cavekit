---
name: ck-status
description: "Show build site progress and live runtime state. With --watch, tails the dashboard until interrupted. `--team` delegates to `cavekit team status`."
argument-hint: "[--watch] [--interval SECONDS] [--filter PATTERN] [--team] [--json]"
allowed-tools: ["Bash(node ${CLAUDE_PLUGIN_ROOT}/scripts/cavekit-tools.cjs:*)", "Bash(cavekit team:*)", "Bash(cat .cavekit/*)", "Bash(ls .cavekit/*)", "Bash(sleep *)", "Read(*)", "Glob(*)", "Grep(*)"]
---

**What this does:** Shows task frontier, completion, and runtime state. Default prints one snapshot. With `--watch`, refreshes every few seconds.
**When to use it:** During or after `/ck:make` to see where the loop is, which tasks are ready, and whether the session is still within budget.

## Parse arguments

- `--watch` — refresh repeatedly until interrupted (default interval: 3s)
- `--interval N` — seconds between frames (only meaningful with `--watch`)
- `--filter PATTERN` — scope the site match
- `--team` — delegate to `cavekit team status`
- `--json` — when used with `--team`, emit the machine-readable `cavekit.team.v1` payload

If `--team` is present, do not run the build-site progress flow below. Instead shell out directly:

```bash
cavekit team status $ARGUMENTS
```

Propagate its exit code unchanged.

## Step 0: Runtime state (when `.cavekit/` exists)

If `.cavekit/state.md` exists, print the runtime status block first — it is the authoritative live view:

```bash
node "${CLAUDE_PLUGIN_ROOT}/scripts/cavekit-tools.cjs" status
```

Then print the progress snapshot if one exists:

```bash
cat .cavekit/.progress.json 2>/dev/null || echo "(no progress snapshot yet)"
```

If `--watch` was passed and `.cavekit/.loop.json` is missing, print:

```
[ck:status] No active loop. Run /ck:make to start one.
```

…then exit the watch loop cleanly.

If `.cavekit/` is absent, skip Step 0 and run the impl-based flow below.

## Step 1: Find Site

Look in `context/plans/` then `context/sites/` for files matching `*site*`, `*plan*`, or `*frontier*` (exclude `*overview*`). Apply `--filter` from `$ARGUMENTS` if set.

If no site found: "No build site or plan found. Run `/ck:map` first."

## Step 2: Read State

1. Read the site — catalog every task (T-number), tier, kit/requirement, `blockedBy`.
2. Read all `context/impl/impl-*.md` — extract statuses (DONE / IN_PROGRESS / BLOCKED).
3. Read `context/impl/loop-log.md` if present — latest iteration, last task completed.

## Step 3: Classify Tasks

For each task:
- **DONE** — marked done in impl tracking
- **IN_PROGRESS** — marked in progress
- **BLOCKED** — unfinished `blockedBy`
- **READY** — all deps done, not started
- **WAITING** — deps not yet done, not directly blocked

## Step 4: Display Report

```markdown
## Cavekit Status

### Summary
| Status | Count | % |
|--------|-------|---|
| DONE | {n} | {%} |
| IN_PROGRESS | {n} | {%} |
| READY | {n} | {%} |
| BLOCKED | {n} | {%} |
| WAITING | {n} | {%} |

### Progress
[████████████░░░░░░░░] 58% (20/34 tasks)

### Current Tier: {n}

### Ready (next up)
| Task | Title | Kit | Requirement |
|------|-------|-----|-------------|

### Recently Completed
| Task | Title | Iteration |
|------|-------|-----------|

### Blocked
| Task | Title | Waiting On |
|------|-------|-----------|

### Dead Ends (if any)
| Task | Approach | Why Failed |

### Loop
- Iterations: {n}
- Last iteration: {timestamp}
- Active: {yes/no — `.cavekit/.loop.json` exists?}

### Runtime Budget (if .cavekit/token-ledger.json exists)
- Session tokens: {used} / {budget} ({pct}%)
- Per-task: {ok} / {warn} / {exhausted}
```

## Watch loop

When `--watch` is passed, loop Step 0 + Step 4:

```bash
while true; do
  clear
  # ... print status block + progress snapshot
  sleep "${INTERVAL:-3}"
done
```

Exit on Ctrl-C. Keep each frame short (≤ 12 lines after the runtime block) so scrollback stays usable.

## Critical rules

- Read-only. Never mutate any file under `.cavekit/`.
- Do not call `route` or `heartbeat` — that is the stop-hook's job.
- If `.cavekit/` is absent, fall back to impl-based classification without error.
