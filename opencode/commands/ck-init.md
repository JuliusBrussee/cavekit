---
description: Bootstrap context directories for the OpenCode Cavekit portable workflow
argument-hint: [--tools-only]
---

You are running **/ck-init** for the OpenCode Cavekit portable port.

Hard rules:
- Do not claim this initializes the upstream `.cavekit/` runtime.
- Do not call upstream runtime scripts, router scripts, hook scripts, tmux helpers, or subagent APIs.
- Idempotent only. Create missing files. Do not overwrite existing project docs.

Interpret `$ARGUMENTS`:
- `--tools-only` → capability summary only, no file edits
- empty → bootstrap `context/` and minimal CLAUDE docs

## If `--tools-only`
Use the Bash tool to check common tools with `command -v` and summarize availability for:
- git
- gh
- node
- npm
- python3
- docker
- opencode
- codex

Return a short table and a brief note that capability discovery is advisory only.

## Default mode

1. Inspect repo root first with Read/Glob:
   - existing `context/`
   - `README.md`
   - `AGENTS.md`
   - `.gitignore`

2. Create these directories if missing:
   - `context/`
   - `context/refs/`
   - `context/kits/`
   - `context/designs/`
   - `context/plans/`
   - `context/impl/`
   - `context/impl/archive/`

3. Create these files only if missing:

### `context/CLAUDE.md`
```markdown
# Context Hierarchy

This repo uses the Cavekit OpenCode portable workflow.

- refs/ — source material and external references
- kits/ — implementation-agnostic requirements
- designs/ — visual system and design references
- plans/ — build-site and task graphs
- impl/ — execution notes, progress, dead ends
```

### `context/refs/CLAUDE.md`
```markdown
# References

Source material used to derive kits. Read-only inputs.
```

### `context/kits/CLAUDE.md`
```markdown
# Kits

Kits describe what must be true, not how to implement it.
Use `cavekit-overview.md` as the entry point.
```

### `context/plans/CLAUDE.md`
```markdown
# Plans

Plans translate kits into tasks, tiers, and dependencies.
Use `build-site.md` as the primary execution plan.
```

### `context/impl/CLAUDE.md`
```markdown
# Implementation Tracking

Record what was changed, what passed, what failed, and what remains.
```

### `context/designs/CLAUDE.md`
```markdown
# Design System

Store design references here or point to a root DESIGN.md.
```

4. If source directories like `src/`, `app/`, `lib/`, `tests/`, or `scripts/` exist and already contain no `CLAUDE.md`, you may add a minimal local `CLAUDE.md` only when it helps navigation. Keep it tiny.

5. Report:
   - directories created
   - files created
   - files left untouched
   - exact next step: usually `/ck-sketch`

Keep the bootstrap minimal and honest.
