---
name: ck-make-parallel
description: "Like /ck:make but uses parallel ck:task-builder subagents in isolated git worktrees (opt-in; the safe default is /ck:make)"
argument-hint: "[FILE] [--filter PATTERN] [--peer-review] [--concurrency N] [--max-iterations N] [--completion-promise TEXT]"
allowed-tools: ["Bash(${CLAUDE_PLUGIN_ROOT}/scripts/setup-build.sh:*)", "Bash(${CLAUDE_PLUGIN_ROOT}/scripts/bp-config.sh:*)", "Bash(node ${CLAUDE_PLUGIN_ROOT}/scripts/cavekit-tools.cjs:*)", "Bash(node ${CLAUDE_PLUGIN_ROOT}/scripts/cavekit-router.cjs:*)", "Bash(cavekit team:*)", "Bash(git *)"]
---

**What this does:** Runs the same build loop as `/ck:make`, but overrides the execution mode for this run: `TB_ISOLATION=worktree` and `MAX_PARALLEL=N` (default 3). Task-builder subagents dispatch in parallel inside isolated git worktrees; the parent session merges and cleans up each worktree after the wave.
**When to use it:** When you want the throughput of parallel subagents and your environment is known-good with parallel worktree dispatch. The default `/ck:make` (inline, sequential, no subagent) is safer and is the recommended path.

# Cavekit Make (Parallel) — Opt-In Subagent Execution

## Why this is a separate command

The bug report from Cavekit v3.0.0 showed that parallel `ck:task-builder` subagents in isolated worktrees can hit a Claude-Code-harness worktree race on some builds, returning `[Tool result missing due to internal error]` for one or more packets. The default `/ck:make` path avoids this entirely by running inline in the parent session with no subagent dispatch. This command exists for users who explicitly want parallel execution and accept that trade-off.

## Step 0: Setup

Execute the setup script — same as `/ck:make`:

```!
"${CLAUDE_PLUGIN_ROOT}/scripts/setup-build.sh" $ARGUMENTS
```

## Step 1: Apply parallel overrides

After running setup, **override the two execution-mode values** for this run only. Do NOT write these to config — they are per-run.

1. Set `TB_ISOLATION=worktree` (regardless of `task_builder_isolation` config).
2. Parse `$ARGUMENTS` for `--concurrency N` (positive integer). If present and valid, use that value for `MAX_PARALLEL`. Otherwise default to `3`. Ignore `parallelism_max_per_repo` from config — this command forces the override.
3. Log once:
   ```
   /ck:make-parallel: TB_ISOLATION=worktree, MAX_PARALLEL={N} (this run only; config unchanged)
   ```

## Step 2: Run the standard make flow with these overrides

Follow the remainder of `commands/make.md` starting from **"Resolve Execution Profile"** (steps 1–3 and 6–7 of that section — skip steps 4 and 5 since we just set the values directly).

When you reach the **"Execute based on frontier size"** section:
- Skip **Mode A (Inline)**.
- Use **Mode B (Subagent)** with `MAX_PARALLEL>1`:
  - Partition the frontier into up to `MAX_PARALLEL` coherent work packets per wave.
  - Dispatch them in a single assistant message with multiple `Agent` tool calls.
  - Use the dispatch template from `commands/make.md` verbatim (with `isolation: "worktree"` included).
  - Apply the **Harness error recovery** rule if any packet returns a harness-level failure: retry that packet once sequentially; mark BLOCKED if it errors a second time.

Post-wave cleanup follows the `TB_ISOLATION=worktree` branch in `commands/make.md`:
1. `git merge <branch> --no-edit`
2. `git worktree remove <worktree-path>`
3. `git branch -D <branch>`

Continue through all tiers until the build site is complete. All other rules — pre-flight coverage, tier gate review, post-build verification, CLAUDE.md hierarchy update, completion sentinel — apply unchanged.

## Step 3: Completion

Same completion criteria as `/ck:make`: emit `<promise>CAVEKIT COMPLETE</promise>` when all tasks are genuinely DONE.

## Circuit breakers

Same as `/ck:make`:
- 3 consecutive test failures on one task → mark BLOCKED, document in `dead-ends.md`, skip.
- Unresolvable merge conflict → clean up remaining worktrees, stop the wave, report which branches conflict.
- All remaining tasks blocked → report the dependency chain and stop.

## Trade-offs vs. `/ck:make`

| | `/ck:make` (default) | `/ck:make-parallel` |
|---|---|---|
| Dispatch | Parent session, no subagent | `ck:task-builder` subagent per packet |
| Isolation | None — edits the working tree directly | Each packet in a separate git worktree |
| Concurrency | Sequential (one packet at a time) | Up to `MAX_PARALLEL` packets per wave |
| Model | Whatever model the parent session uses | `EXECUTION_MODEL` from the preset |
| Failure modes | Simple — parent controls everything | Possible worktree race, merge conflicts between packets |
| Recommended for | Every run by default | Large builds where throughput matters and the environment is known-good |

Next: `/ck:check` to run gap analysis and peer review against the kits.
