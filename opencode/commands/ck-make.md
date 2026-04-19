---
description: Sequential scoped implementation from a build-site task or requirement for the OpenCode Cavekit portable workflow
argument-hint: <task-id | requirement-ref | filter>
---

You are running **/ck-make** for the OpenCode Cavekit portable port.

Hard rules:
- This is sequential, bounded implementation only.
- No autonomous loop, no subagents, no worktrees, no hidden batching.
- Do not commit unless the user explicitly asks for a commit.
- Do not edit kit requirements as part of implementation. Track execution in plan/impl docs instead.

## Step 1: Resolve target
Interpret `$ARGUMENTS` as one of:
- a task ID like `T-001`
- a requirement reference like `cavekit-auth.md:R3`
- a narrow filter string that identifies one coherent task slice

If no clear target is provided:
- inspect `context/plans/build-site.md`
- if exactly one task is clearly READY, use it
- otherwise stop and ask the user to choose a target

## Step 2: Gather only relevant context
Use Read/Glob/Grep to load:
- `context/plans/build-site.md`
- the specific kit files tied to the target
- `context/impl/` notes relevant to that target, if present
- `DESIGN.md` if the target affects UI
- the exact source/test files likely involved

Use `lsp_diagnostics` before editing when supported.

## Step 3: Implement the smallest coherent slice
- Satisfy the target's acceptance criteria
- Keep changes scoped
- Add or update tests when the target changes behavior
- Prefer surgical edits over broad refactors

Use the Bash tool to run the smallest relevant validation commands for the repo (tests, build, lint, typecheck as appropriate).

## Step 4: Update execution tracking
If the repo uses `context/impl/`, update one relevant tracking file with:
- target implemented
- files changed
- validation commands and pass/fail status
- remaining blockers or follow-up work

Do not mark kits as complete by editing their acceptance-criteria checkboxes unless the repo already treats kits as live progress trackers and the user expects that behavior.

## Step 5: Report
Return:
- target implemented
- files changed
- validation run and results
- remaining gaps, if any
- exact next step: another `/ck-make <target>`, `/ck-status`, or `/ck-check`

If validation fails, stop with a precise blocker report instead of claiming success.
