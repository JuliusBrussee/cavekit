---
name: ck-make
description: "Implement a build site or plan — automatically parallelizes independent tasks and progresses through tiers autonomously"
argument-hint: "[FILE] [--filter PATTERN] [--peer-review] [--max-iterations N] [--completion-promise TEXT]"
allowed-tools: ["Bash(${CLAUDE_PLUGIN_ROOT}/scripts/setup-build.sh:*)", "Bash(${CLAUDE_PLUGIN_ROOT}/scripts/bp-config.sh:*)", "Bash(node ${CLAUDE_PLUGIN_ROOT}/scripts/cavekit-tools.cjs:*)", "Bash(node ${CLAUDE_PLUGIN_ROOT}/scripts/cavekit-router.cjs:*)", "Bash(cavekit team:*)", "Bash(git *)"]
---

**What this does:** Runs the autonomous build loop from the build site — parallelizes ready tasks into coherent work packets, validates each against acceptance criteria, merges after every wave, progresses through tiers until all tasks are done.
**When to use it:** Right after `/ck:map`. Add `--peer-review` to enable Codex tier gates; `--max-iterations N` to cap the loop.

# Cavekit Make — Autonomous Implementation

This is the third phase of Cavekit. Execute the setup script:

```!
"${CLAUDE_PLUGIN_ROOT}/scripts/setup-build.sh" $ARGUMENTS
```

## Autonomous Runtime Mode (when `.cavekit/` is present)

`setup-build.sh` already calls `cavekit-tools setup-loop` when `node` is on
`$PATH`, which activates the **stop-hook** (`hooks/stop-hook.sh`). While
the hook is active, the session is automatically re-prompted after every
Stop event with the next wave — you do NOT need to re-read the build site
between tasks yourself. The hook does it for you.

Also load, once at the top of the run, the behavioral skills that every
task-builder must follow:

- `karpathy-guardrails` — think-before-code, simplicity, surgical changes,
  goal-driven execution. Applied per task.
- `autonomous-loop` — the state-machine contract (sentinels, phases, lock
  protocol). Informs how you emit status.
- `caveman-internal` — intensity mode (lite / full / ultra) for your own
  internal artifacts. Consult `cavekit-tools intensity` when writing
  artifact summaries or handoff memos.

When the stop-hook routes a wave prompt, the parent session executes the
wave according to the resolved execution mode (see "Resolve Execution
Profile" below):

- **Inline mode (`TB_ISOLATION=inline`, default):** the parent session
  implements each task directly — no subagent dispatch. The router below
  is not consulted because model selection is whatever the parent session
  is on.
- **Subagent mode (`TB_ISOLATION=worktree`, used by `/ck:make-parallel`
  and by users who explicitly configure it):** dispatch `ck:task-builder`
  subagents using the recommended model tier from:

```bash
node "${CLAUDE_PLUGIN_ROOT}/scripts/cavekit-router.cjs" classify-task \
  --role ck:task-builder \
  --files <N> --type <type> --judgment <j> --cross-component <c> --novelty <v> \
  --budget-pressure <pressure>
```

When a task completes, mark it complete in the registry so the next wave
can unblock downstream tasks:

```bash
node "${CLAUDE_PLUGIN_ROOT}/scripts/cavekit-tools.cjs" mark-complete --task T-XXX
```

Before writing artifact summaries, handoff memos, or wave logs, consult
the intensity resolver once per wave:

```bash
INTENSITY=$(node "${CLAUDE_PLUGIN_ROOT}/scripts/cavekit-tools.cjs" intensity)
# INTENSITY ∈ {lite, full, ultra}
```

Apply that intensity to internal artifacts per the `caveman-internal`
skill. User-facing wave status still respects the existing
`caveman-active build` check (the two are independent by design).

When the stop-hook reports `CAVEKIT_LOOP_DONE` (all tasks `complete`), emit
the completion sentinel on its own line as the final message:

```
<promise>CAVEKIT COMPLETE</promise>
```

If `.cavekit/task-status.json` does not exist (user did not run `/ck:init`
or `/ck:map` before this run), fall back to the legacy Ralph-loop flow
below. The two paths are mutually exclusive: either the stop-hook drives,
or the agent re-reads `.claude/ralph-loop.local.md` itself.

## Resolve Execution Profile

Before starting waves:

1. Run `"${CLAUDE_PLUGIN_ROOT}/scripts/bp-config.sh" summary` and report that exact line once.
2. Run `"${CLAUDE_PLUGIN_ROOT}/scripts/bp-config.sh" model execution` and treat the result as `EXECUTION_MODEL`.
3. Run `"${CLAUDE_PLUGIN_ROOT}/scripts/bp-config.sh" caveman-active build` and treat the result as `CAVEMAN_ACTIVE` (true/false).
4. Run `"${CLAUDE_PLUGIN_ROOT}/scripts/bp-config.sh" get task_builder_isolation` and treat the result as `TB_ISOLATION` (`inline` or `worktree`). Default is `inline`.
5. Run `"${CLAUDE_PLUGIN_ROOT}/scripts/bp-config.sh" get parallelism_max_per_repo` and treat the result as `MAX_PARALLEL` (positive integer; default `1`).
6. Use `EXECUTION_MODEL` in every `ck:task-builder` delegation below (only applies when `TB_ISOLATION=worktree`). Do not hard-code `opus`, `sonnet`, or `haiku` in this command.
7. If `CAVEMAN_ACTIVE` is `true`, all your own wave logs, iteration summaries, and status reports in this command should use caveman-speak (drop articles, filler, pleasantries — keep technical terms exact, code blocks unchanged). Spec artifacts (kits, build sites, impl tracking field values) stay in normal prose.

**To run with parallel subagents instead of inline**, use `/ck:make-parallel` (a separate command that defaults to `TB_ISOLATION=worktree` and `MAX_PARALLEL=3`).

**Execution mode matrix (after config + flag overrides):**
- `TB_ISOLATION=inline` and `MAX_PARALLEL=1` (default) — **no subagent is spawned.** The parent session implements each task directly: read context, edit files, run validation, commit, move to the next task. This is the confirmed-stable ralph-loop path — no Agent dispatch, no worktree, no merge. Trade-off: parent session runs under whatever model the user is on, not `EXECUTION_MODEL`.
- `TB_ISOLATION=worktree` and `MAX_PARALLEL=1` — one `ck:task-builder` subagent per wave, inside an isolated git worktree. Sequential; the parent merges after the packet returns.
- `TB_ISOLATION=worktree` and `MAX_PARALLEL>1` — up to `MAX_PARALLEL` `ck:task-builder` subagents per wave in isolated worktrees. Opt-in via `--parallel` or config. Can hit Claude-Code-harness worktree races on some builds.
- `TB_ISOLATION=inline` with `MAX_PARALLEL>1` is unsupported (inline is inherently sequential). Clamp `MAX_PARALLEL` to `1` and log a note.

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

   **1+ ready tasks** → Partition the frontier into coherent work packets (group tasks that touch the same subsystem/files, split large or file-disjoint work). Then execute packets according to `TB_ISOLATION` and `MAX_PARALLEL`:

   ---

   ### Mode A — Inline (default: `TB_ISOLATION=inline`, `MAX_PARALLEL=1`)

   **No subagent is spawned.** The parent session does the work directly. For each packet, in order (one at a time):

   1. Read the task entry from the build site, its cavekit requirements, acceptance criteria, and `context/impl/dead-ends.md`.
   2. If team mode is initialized (tracked `.cavekit/team/ledger.jsonl` and local `.cavekit/team/identity.json` exist), claim the packet's primary task before doing any work:
      ```bash
      cavekit team claim T-XXX --json
      ```
      - Exit `0` with `already=true` is fine — continue.
      - Exit `3`, `4`, `5`, or `6` means the task is unavailable right now; log it as skipped for this wave and move to the next packet.
      - On a successful fresh claim, immediately start the internal heartbeat loop in the background and capture its PID:
      ```bash
      CAVEKIT_INTERNAL=1 cavekit team heartbeat T-XXX >/tmp/cavekit-team-heartbeat-T-XXX.log 2>&1 &
      TEAM_HEARTBEAT_PID=$!
      ```
      - If team mode is absent, skip this entire claim/heartbeat step.
   3. If the packet contains UI tasks, read `DESIGN.md` and the `ck:ui-craft` skill.
   4. Implement the packet: edit files, write tests, run validation (build + tests).
   5. Commit on the current branch with a message naming the packet's primary task: `T-{ID}: {what was done}`. Do NOT push.
   6. If team mode is active:
      - On success, stop the heartbeat (`kill "$TEAM_HEARTBEAT_PID"`; if it ignores SIGTERM, wait up to 5s then SIGKILL) and run:
        ```bash
        cavekit team release T-XXX --complete
        ```
      - On failure or BLOCKED status, stop the heartbeat the same way and run:
        ```bash
        cavekit team release T-XXX --note "validation failure"
        ```
      - This release/complete step is mandatory in every exit path.
   7. Log one line in wave status:
      ```
      T-{ID}: {title} — COMPLETE | PARTIAL | BLOCKED. Files: {n}. Build {P/F}, Tests {P/F}.
      ```
   8. Move to the next packet. No merge, no worktree, no Agent dispatch.

   Inline mode runs under whatever model the parent session is using. It does **not** honor `EXECUTION_MODEL` — if the user needs a specific model for task implementation, they must switch to worktree mode (below) or set the parent session to that model.

   ---

   ### Mode B — Subagent (`TB_ISOLATION=worktree`, any `MAX_PARALLEL`)

   Dispatch `ck:task-builder` subagents. One packet per subagent. Each subagent runs in an isolated git worktree on a fresh branch; the parent merges after the packet returns.

   **Dispatch rule by `MAX_PARALLEL`:**
   - `MAX_PARALLEL=1` → emit one Agent call, wait for it to return, emit the next (sequential).
   - `MAX_PARALLEL>1` → emit up to `MAX_PARALLEL` Agent calls in a single assistant message (parallel). If the frontier has more packets than `MAX_PARALLEL`, pick the top-N highest-priority packets and defer the rest to the next wave.

   Before dispatching each packet, if team mode is active in the parent checkout, claim the packet's primary task with `cavekit team claim T-XXX --json`. Only dispatch packets whose claim succeeds (or is already held by this checkout). For each successfully claimed packet, start a background `CAVEKIT_INTERNAL=1 cavekit team heartbeat T-XXX` process and capture its PID alongside the packet metadata. If the claim exits `3`, `4`, `5`, or `6`, skip dispatch for that packet this wave.

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

   CAVEMAN MODE: {if CAVEMAN_ACTIVE is true, include: 'ON — apply caveman-speak ONLY to your final status report prose (summaries, issue notes). Drop articles/filler/pleasantries; keep technical terms exact. Pattern: [thing] [action] [reason]. Do NOT apply caveman to: (a) internal reasoning or thinking, (b) tool calls or tool arguments, (c) code, (d) git commit messages, (e) structured output fields (TASK RESULT keys and values like file lists). Think and call tools normally — compression applies to prose only.' If CAVEMAN_ACTIVE is false, include: 'OFF'}

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

   **Harness error recovery** (parallel mode, `MAX_PARALLEL>1`): if an Agent call returns `[Tool result missing due to internal error]`, no `agentId`, or otherwise reports a harness-level failure with no body, do NOT try to merge or clean up a worktree for it — there is none. Re-dispatch that packet **once sequentially** (on its own in a fresh message), then proceed. If the retry also errors, log the packet's tasks as BLOCKED with the harness error and move on. Do not retry a third time.

   ---

5. **After wave completes**:
   - **If `TB_ISOLATION=worktree`**: merge and clean up each subagent **one at a time**:
     1. `git merge <branch> --no-edit` — merge the subagent's branch
     2. `git worktree remove <worktree-path>` — remove the worktree directory (required before branch can be deleted)
     3. `git branch -D <branch>` — delete the branch
     Skip all three steps if the subagent reported no changes (Claude Code auto-cleans worktrees with no changes). If a merge conflicts, clean up the worktree (`git worktree remove <worktree-path> --force`) before reporting the conflict.
     If team mode is active for that packet, stop its heartbeat PID first (SIGTERM, wait up to 5s, then SIGKILL if needed), then:
     - on successful merge + validation, run `cavekit team release T-XXX --complete`
     - on failed merge, validation failure, or harness-level BLOCKED result, run `cavekit team release T-XXX --note "<reason>"`
   - **If `TB_ISOLATION=inline`**: no merge or worktree cleanup — the parent session's commits already landed on the current branch. Optionally run `git log --oneline {TIER_START_REF}..HEAD` to confirm the expected task commits are present.
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

Next: `/ck:check` to run gap analysis and peer review against the kits.
