---
description: Show the OpenCode Cavekit portable workflow and recommend the next command
---

You are serving the **Cavekit OpenCode portable port**.

Hard rules:
- Be explicit that this is **not** the upstream Claude/Codex Cavekit runtime.
- No autonomous loop, no hook-driven re-entry, no subagent fan-out, no worktree orchestration, no hidden runtime state.
- Use OpenCode's normal tools only.

First inspect the current repo with read-only tools:
- project root
- `README.md` if present
- `AGENTS.md` if present
- `context/kits/`, `context/plans/`, `context/impl/` if present

Then return a concise guide with these sections:

## Workflow
- `/ck-init` → bootstrap `context/`
- `/ck-sketch` → write or refine kits after explicit approval
- `/ck-map` → generate `context/plans/build-site.md`
- `/ck-make <target>` → implement one bounded task or requirement at a time
- `/ck-check` → read-only verification against kits and code
- `/ck-status` → snapshot of kit/plan/impl/git state

## Supported in OpenCode port
- Sequential file-based workflow
- Kit drafting and refinement
- Build-site generation with tier tables + coverage matrix
- Scoped implementation with normal test/build commands
- Read-only verification and status reporting

## Not supported in OpenCode port
- parallel build orchestration
- hook-driven autonomous loop
- subagent dispatch APIs
- `.cavekit/tasks.json` runtime registry
- team mode, visual companion, auto-backprop, live dashboards

## Recommended next step
- Recommend exactly one command for the current repo: `/ck-init`, `/ck-sketch`, `/ck-map`, `/ck-make <target>`, or `/ck-check`.

Keep it short and operational.
