---
name: ck-make
description: "Implement a build site or plan — automatically parallelizes independent tasks and progresses through tiers autonomously"
argument-hint: "[FILE] [--filter PATTERN] [--peer-review] [--max-iterations N] [--completion-promise TEXT]"
allowed-tools: ["Bash(${CLAUDE_PLUGIN_ROOT}/scripts/setup-build.sh*)", "Bash(${CLAUDE_PLUGIN_ROOT}/scripts/bp-config.sh*)", "Bash(git *)"]
---

> **Note:** `/bp:build`, `/ck:build`, `/bp:make` are deprecated aliases. Use `/ck:make` instead.

# Cavekit Build — Autonomous Implementation

This is the third phase of Cavekit. Run `"${CLAUDE_PLUGIN_ROOT}/scripts/setup-build.sh" $ARGUMENTS`.

## Resolve Execution Profile

Before starting waves:

1. Run `"${CLAUDE_PLUGIN_ROOT}/scripts/bp-config.sh" summary` and report that exact line once.
2. Run `"${CLAUDE_PLUGIN_ROOT}/scripts/bp-config.sh" model execution` and treat the result as `EXECUTION_MODEL`.
3. Run `"${CLAUDE_PLUGIN_ROOT}/scripts/bp-config.sh" caveman-active build` and treat the result as `CAVEMAN_ACTIVE` (true/false).
4. Use that exact `EXECUTION_MODEL` string in every `ck:task-builder` delegation below. Do not hard-code `opus`, `sonnet`, or `haiku` in this command.
5. If `CAVEMAN_ACTIVE` is `true`, all your own wave logs, iteration summaries, and status reports in this command should use caveman-speak (drop articles, filler, pleasantries — keep technical terms exact, code blocks unchanged). Spec artifacts (kits, build sites, impl tracking field values) stay in normal prose.

## Pre-flight Coverage Check

Before entering the execution loop, validate that the build site covers all cavekit requirements:

1. Read the build site and all cavekit files referenced in it
2. If the build site contains a **Coverage Matrix** section, scan it for any rows with status `GAP`
3. If no Coverage Matrix exists, perform a quick manual check: for each cavekit requirement, confirm at least one task in the build site references it
4. **If gaps are found**, report them before starting:
   ```
   ⚠ COVERAGE GAPS DETECTED — {n} acceptance criteria have no assigned task:
     - cavekit-{domain}.md R{n}: {criterion text}
     - cavekit-{domain}.md R{n}: {criterion text}
   
   Run `/ck:map` to regenerate the build site with full coverage, or continue with known gaps.
   ```
   Ask the user whether to proceed or stop. Do NOT silently continue with gaps.
5. If no gaps are found, log: `✓ Pre-flight coverage check passed — all criteria mapped to tasks.`

## If site selection is required

If the output contains `CAVEKIT_SITE_SELECTION_REQUIRED=true`, multiple build sites/plans were found. **Ask the user which one to implement.** Then re-run with `--filter <their-choice>`.

## Execution Loop

Once the setup script completes (outputs the ralph prompt), you run the execution loop autonomously. Progress through all tiers without stopping.

### Each Wave

1. **Read state**: Read the build site/plan + scoped `context/impl/impl-*.md` files + `context/impl/dead-ends.md`. **Scoping rule:** only read impl files that contain `Build site: <this site's path>` (or matching basename). Ignore impl files declaring a different build site. If no scoped files exist, fall back to reading all impl files. If this is the first wave of a new tier, capture the tier start ref: `TIER_START_REF=$(git rev-parse HEAD)`
2. **Compute frontier**: Find all tasks that are NOT done AND whose `blockedBy` dependencies are ALL done
3. **Report**:
   ```
   ═══ Wave {N} ═══
   {count} task(s) ready:
     {task_id}: {title} (tier {N}, deps: {deps})
   ```

4. **Execute based on frontier size**:

   **0 ready tasks** → Check if ALL tasks are done. If yes → completion. If not → report blockage and stop.

   **1 ready task** → Delegate it as a single-task `ck:task-builder` packet using `EXECUTION_MODEL`. This keeps execution model selection explicit even when only one task is ready.

   **2+ ready tasks** → Partition the frontier into a small set of coherent work packets, then parallelize those. Optimize for throughput, not raw agent count.

   **Build work packets using these rules:**
   - Group tasks when they touch the same subsystem, cavekit/domain, or expected file set
   - Group small related tasks when the combined scope is still coherent for one agent
   - Split large, risky, or file-disjoint tasks into separate packets
   - Keep ownership clean: each packet should have a clear primary file/module surface
   - Prefer a few meaningful packets, usually 2-4 concurrent subagents, rather than one agent per task
   - Prefer delegated execution whenever model selection matters; only keep work inline if you are certain the current parent model already matches `EXECUTION_MODEL`

   ```
   Agent(
     subagent_type: "ck:task-builder",
     model: "{EXECUTION_MODEL}",
     isolation: "worktree",
     prompt: "TASKS:
   - {task_id}: {title}
   - {task_id}: {title}

   SHARED CONTEXT:
   - Domain/spec: {spec_name}
   - Requirement IDs: {requirement_ids}
   BUILD SITE: {path to build site}
   CAVEKITS: {paths to relevant cavekit files}
   DESIGN SYSTEM: {path to DESIGN.md if it exists and packet contains UI tasks, or 'None — no design system'}
   DESIGN REFERENCES: {specific DESIGN.md sections relevant to this packet's UI tasks, or 'N/A'}
   EXPECTED FILE OWNERSHIP: {files or modules this packet should own}

   ACCEPTANCE CRITERIA (from kits):
   {paste the acceptance criteria for each task in this packet}

   DEAD ENDS TO AVOID:
   {paste relevant dead ends, or 'None'}

   CAVEMAN MODE: {if CAVEMAN_ACTIVE is true, include: 'ON — all status reports, logs, and reasoning use caveman-speak: drop articles/filler/pleasantries, keep technical terms exact, code blocks unchanged. Pattern: [thing] [action] [reason]. Do NOT apply caveman to code, git commits, or structured output fields.' If CAVEMAN_ACTIVE is false, include: 'OFF'}

   INSTRUCTIONS:
   1. Read each listed cavekit requirement for full context
   2. Implement the packet as one coherent slice of work
   3. Keep changes inside the owned files/modules unless a requirement forces expansion
   4. Write tests as needed
   5. Run validation: build must pass, tests must pass
   6. Commit with a message that names the packet's primary task BEFORE reporting
   7. Report result:
      TASK RESULT:
      - Tasks: {ids and titles}
      - Status: COMPLETE | PARTIAL | BLOCKED
      - Files changed: {list}
      - Issues: {any}"
   )
   ```

   Dispatch all packets for the wave in a single message with multiple Agent tool calls.

5. **After wave completes**:
   - For any delegated wave (single-agent or parallel), merge and clean up each subagent **one at a time**:
     1. `git merge <branch> --no-edit` — merge the subagent's branch
     2. `git worktree remove <worktree-path>` — remove the worktree directory (required before branch can be deleted)
     3. `git branch -D <branch>` — delete the branch
     Skip all three steps if the subagent reported no changes (Claude Code auto-cleans worktrees with no changes). If a merge conflicts, clean up the worktree (`git worktree remove <worktree-path> --force`) before reporting the conflict.
   - Update `context/impl/impl-*.md` with status for each completed task
   - Record any dead ends in `context/impl/dead-ends.md`
   - Update `context/impl/loop-log.md` with an iteration entry. **If `CAVEMAN_ACTIVE` is true**, compress the loop-log entry to a dense one-liner per task using caveman-speak. Instead of verbose iteration summaries, write compact entries like:
     ```
     ### Iteration {N} — {date}
     - T-{id}: {title} — DONE. Files: {list}. Build P, Tests P. Next: T-{ids}
     ```
     The log stays searchable but uses a fraction of the context window. Field names (Task, Status, Files, Validation, Next) can be abbreviated. If `CAVEMAN_ACTIVE` is false, use the standard verbose format.

6. **Tier boundary check** — after updating impl tracking, check whether all tasks in the current tier are now done. If the current tier still has undone tasks, skip this step. If the tier is complete, run the Codex tier gate review (the `TIER_START_REF` was captured in step 1 at the start of this tier):

   a. Source `codex-config.sh` and check `tier_gate_mode` via `bp_config_get tier_gate_mode`. If the value is `"off"`, skip the review and log:
      ```
      [ck:tier-gate] Tier gate review disabled (tier_gate_mode=off). Skipping.
      ```

   b. Source `codex-detect.sh` and check `codex_available`. If `false`, log a note and continue:
      ```
      [ck:tier-gate] Codex unavailable — skipping tier boundary review. Continuing to next tier.
      ```

   c. Otherwise, run the review inline (wait for it to complete before advancing):
      ```
      scripts/codex-review.sh --base $TIER_START_REF
      ```

   d. **Severity-based gating** — after the review, source `scripts/codex-gate.sh` and run `bp_tier_gate`:
      - If `GATE_RESULT=proceed`: log the tier review summary and advance.
      - If `GATE_RESULT=blocked`: the tier has P0/P1 findings (or all findings in `strict` mode) that must be fixed before advancing.

   e. **When blocked** — run the review-fix cycle using `bp_review_fix_cycle $TIER_START_REF 2`:
      - The cycle function runs the review, evaluates the gate, and if blocked returns exit code 2 with `AWAITING_FIXES` and the fix task list
      - For each fix task in the output: read the finding's file and description, implement the fix, commit
      - After fixes, mark each fixed finding: `bp_findings_update_status <F-ID> FIXED`
      - Call `bp_review_fix_cycle` again for the re-review (it tracks the cycle count internally)
      - **Maximum 2 review-fix cycles per tier** — after 2 cycles, the function returns exit code 1 and logs a warning; advance to the next tier regardless
      - If the function returns 0, all blocking findings are resolved — advance normally

   ```
   ═══ Tier {N} Complete — Codex Review ═══
   Review: {CLEAN | N findings (M blocking, K deferred)}
   Gate: {PROCEED | BLOCKED → fix cycle {1|2}}
   ```

7. **Immediately proceed to next wave** — do NOT wait for user input between waves.

### Completion

When all tasks in the build site are done:

```
═══ BUILD COMPLETE ═══
Waves executed: {N}
Tasks completed: {done}/{total}
```

### Post-Build: Cavekit Verification

Before updating CLAUDE.md, verify that the build actually satisfies the kits:

1. Read all cavekit files and the build site's Coverage Matrix (if present)
2. For each cavekit requirement and its acceptance criteria, cross-reference against impl tracking:
   - Is the task marked DONE in impl tracking?
   - Does the task's scope actually cover this specific criterion? (A task being DONE does not mean every criterion it was supposed to cover is actually met)
3. Produce a brief coverage summary:
   ```
   ═══ Cavekit Verification ═══
   Requirements: {done}/{total}
   Acceptance Criteria: {verified}/{total}
   Gaps: {list any unmet criteria, or "None"}
   ```
4. If gaps are found (criteria not covered by completed tasks):
   - Log each gap with its cavekit reference
   - Add the gaps as new tasks to the build site (append to the highest tier + 1)
   - Report: `{n} gap(s) found — {n} remediation tasks added to build site. Run /ck:make again to address.`
5. If no gaps: proceed to CLAUDE.md hierarchy update

### Post-Build: Update CLAUDE.md Hierarchy

After BUILD COMPLETE and before the completion promise, update the context hierarchy:

1. **Read the build site** to get task-to-cavekit-requirement mappings
2. **Read `git diff --name-only` against the pre-build ref** to identify which source files were created/modified during the build
3. **For each source directory that was touched** (e.g., `src/auth/`, `src/api/`):
   - If no `CLAUDE.md` exists in that directory: create one with cavekit/plan references derived from the tasks that touched those files:
     ```markdown
     # {Module Name}

     Implements:
     - cavekit-{domain}.md R{n} ({Requirement Name})

     Build tasks: T-{ids} (build-site.md)
     ```
   - If `CLAUDE.md` already exists: append any new cavekit references not already listed (never remove existing content)
   - For UI component directories: if `DESIGN.md` exists at project root, include `Visual design: follows DESIGN.md Section {N} ({section name})` in the CLAUDE.md
4. **Update `context/impl/impl-overview.md`** with current domain statuses (tasks done/total per domain)
5. **Update `context/plans/plan-overview.md`** (or `context/sites/` equivalent if legacy) with build site completion status

**Constraints:**
- Only write mappings you are certain about — tasks you completed and files you created
- Never remove existing content from a CLAUDE.md
- Source-tree CLAUDE.md files are kept minimal (references only, no duplicated content)

Then output the completion promise from the ralph prompt.

## Circuit Breakers

- **3 consecutive test failures on same task** → mark BLOCKED, document in dead-ends.md, skip
- **Merge conflict unresolvable** → clean up remaining worktrees (`git worktree remove <path> --force` for each), stop the wave, report which branches conflict
- **All remaining tasks blocked** → report the dependency chain and stop

## Critical Rules

- Parallelize by work packet, not blindly by task count
- Group related small/medium tasks when they share files or context
- Split large or file-disjoint work so concurrent agents have clean ownership
- Merge after EVERY wave — do not accumulate unmerged branches
- Update impl tracking after EVERY wave — next wave reads it for frontier computation
- Progress through tiers autonomously — never pause between waves
- NEVER output completion promise unless ALL tasks are genuinely DONE
- NEVER mark a task DONE because existing code "looks related" — verify each acceptance criterion
